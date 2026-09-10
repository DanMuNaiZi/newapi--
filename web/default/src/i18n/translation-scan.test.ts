/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import assert from 'node:assert/strict'
import { readFile, readdir } from 'node:fs/promises'
import path from 'node:path'
import { describe, test } from 'node:test'

import { parseSync } from 'oxc-parser'

const localeCodes = ['en', 'zh', 'zh-TW', 'fr', 'ja', 'ru', 'vi'] as const
const sourcePaths = [
  'src/features/lottery',
  'src/features/public-pool',
  'src/features/referral-campaigns',
  'src/components/reward-amount-field.tsx',
  'src/hooks/use-reward-quota-config.ts',
  'src/lib/reward-amount.ts',
] as const

type AstNode = Record<string, unknown>

function isAstNode(value: unknown): value is AstNode {
  return typeof value === 'object' && value !== null
}

function nodeType(node: AstNode): string {
  return typeof node.type === 'string' ? node.type : ''
}

function literalStrings(
  node: unknown,
  stringConstants: ReadonlyMap<string, string[]> = new Map()
): string[] {
  if (!isAstNode(node)) return []
  if (nodeType(node) === 'Identifier' && typeof node.name === 'string') {
    return stringConstants.get(node.name) ?? []
  }
  if (nodeType(node) === 'Literal' && typeof node.value === 'string') {
    return [node.value]
  }
  if (nodeType(node) === 'TemplateLiteral') {
    const expressions = node.expressions
    const quasis = node.quasis
    if (
      Array.isArray(expressions) &&
      expressions.length === 0 &&
      Array.isArray(quasis)
    ) {
      return quasis.flatMap((quasi) => {
        if (!isAstNode(quasi) || !isAstNode(quasi.value)) return []
        return typeof quasi.value.raw === 'string' ? [quasi.value.raw] : []
      })
    }
  }
  if (nodeType(node) === 'ConditionalExpression') {
    return [
      ...literalStrings(node.consequent, stringConstants),
      ...literalStrings(node.alternate, stringConstants),
    ]
  }
  if (nodeType(node) === 'LogicalExpression') {
    return [
      ...literalStrings(node.left, stringConstants),
      ...literalStrings(node.right, stringConstants),
    ]
  }
  return []
}

function propertyName(node: AstNode): string | null {
  const key = node.key
  if (!isAstNode(key)) return null
  if (nodeType(key) === 'Identifier' && typeof key.name === 'string') {
    return key.name
  }
  return nodeType(key) === 'Literal' && typeof key.value === 'string'
    ? key.value
    : null
}

function calleeName(node: AstNode): string | null {
  const callee = node.callee
  if (!isAstNode(callee)) return null
  if (nodeType(callee) === 'Identifier' && typeof callee.name === 'string') {
    return callee.name
  }
  if (nodeType(callee) !== 'MemberExpression' || !isAstNode(callee.property)) {
    return null
  }
  return nodeType(callee.property) === 'Identifier' &&
    typeof callee.property.name === 'string'
    ? callee.property.name
    : null
}

function walkAst(value: unknown, visit: (node: AstNode) => void): void {
  if (Array.isArray(value)) {
    value.forEach((item) => walkAst(item, visit))
    return
  }
  if (!isAstNode(value)) return
  visit(value)
  for (const [key, child] of Object.entries(value)) {
    if (key !== 'start' && key !== 'end' && key !== 'type') {
      walkAst(child, visit)
    }
  }
}

function stringConstants(program: unknown): Map<string, string[]> {
  const constants = new Map<string, string[]>()
  walkAst(program, (node) => {
    if (nodeType(node) !== 'VariableDeclarator' || !isAstNode(node.id)) {
      return
    }
    if (
      nodeType(node.id) !== 'Identifier' ||
      typeof node.id.name !== 'string'
    ) {
      return
    }
    const values = literalStrings(node.init)
    if (values.length > 0) {
      constants.set(node.id.name, values)
    }
  })
  return constants
}

function scanTranslationKeys(filename: string, source: string): Set<string> {
  const keys = new Set<string>()
  const ast = parseSync(filename, source)
  const constants = stringConstants(ast.program)
  const add = (values: string[]): void => {
    values.filter(Boolean).forEach((value) => keys.add(value))
  }

  walkAst(ast.program, (node) => {
    if (nodeType(node) === 'CallExpression' && calleeName(node) === 't') {
      const [argument] = Array.isArray(node.arguments) ? node.arguments : []
      add(literalStrings(argument, constants))
    }
    if (
      nodeType(node) === 'Property' &&
      ['error', 'message'].includes(propertyName(node) ?? '')
    ) {
      add(literalStrings(node.value, constants))
    }
    if (nodeType(node) === 'NewExpression' && calleeName(node) === 'Error') {
      const [argument] = Array.isArray(node.arguments) ? node.arguments : []
      add(literalStrings(argument, constants))
    }
    if (
      nodeType(node) === 'CallExpression' &&
      [
        'min',
        'max',
        'int',
        'positive',
        'nonnegative',
        'finite',
        'refine',
      ].includes(calleeName(node) ?? '')
    ) {
      const arguments_ = Array.isArray(node.arguments) ? node.arguments : []
      const messages =
        calleeName(node) === 'refine' ? arguments_.slice(1) : arguments_
      messages.forEach((argument) => add(literalStrings(argument, constants)))
    }
    if (nodeType(node) === 'FunctionDeclaration' && isAstNode(node.id)) {
      if (typeof node.id.name !== 'string' || !node.id.name.endsWith('Error')) {
        return
      }
      walkAst(node.body, (child) => {
        if (nodeType(child) === 'ReturnStatement') {
          add(literalStrings(child.argument, constants))
        }
      })
    }
  })
  return keys
}

async function sourceFiles(): Promise<string[]> {
  const files = new Set<string>()
  const walkDirectory = async (directory: string): Promise<void> => {
    const entries = await readdir(directory, { withFileTypes: true })
    for (const entry of entries) {
      const fullPath = path.join(directory, entry.name)
      if (entry.isDirectory()) {
        await walkDirectory(fullPath)
      } else if (
        /\.(ts|tsx)$/.test(entry.name) &&
        !/\.test\.(ts|tsx)$/.test(entry.name)
      ) {
        files.add(path.relative('.', fullPath))
      }
    }
  }
  for (const sourcePath of sourcePaths) {
    if (
      /\.(ts|tsx)$/.test(sourcePath) &&
      !/\.test\.(ts|tsx)$/.test(sourcePath)
    ) {
      files.add(sourcePath)
    } else {
      await walkDirectory(sourcePath)
    }
  }
  return [...files].sort((first, second) => first.localeCompare(second))
}

function interpolationVariables(value: string): string[] {
  return [...value.matchAll(/{{\s*([^{}]+?)\s*}}/g)]
    .map((match) => match[1].trim())
    .sort((first, second) => first.localeCompare(second))
}

describe('translation source scan', () => {
  test('extracts direct, conditional, logical, validation, and shared error keys', () => {
    const source = `
      const invalidCampaignMessage = 'Invalid campaign range'
      t('Direct key')
      t(active ? 'Conditional true' : 'Conditional false')
      t(active && 'Logical key')
      schema.min(1, 'Validation key')
      schema.finite('Finite number key')
      schema.refine(valid, 'Refinement key')
      schema.refine(valid, { message: invalidCampaignMessage })
      const parsed = { error: 'Schema error key' }
      function getSharedError() { return 'Shared error key' }
      throw new Error('Configuration error key')
    `
    assert.deepEqual([...scanTranslationKeys('sample.ts', source)].sort(), [
      'Conditional false',
      'Conditional true',
      'Configuration error key',
      'Direct key',
      'Finite number key',
      'Invalid campaign range',
      'Logical key',
      'Refinement key',
      'Schema error key',
      'Shared error key',
      'Validation key',
    ])
  })

  test('requires every discovered scoped key in each supported locale', async () => {
    const files = await sourceFiles()
    const discovered = new Map<string, string[]>()
    for (const file of files) {
      const source = await readFile(file, 'utf8')
      for (const key of scanTranslationKeys(file, source)) {
        const paths = discovered.get(key) ?? []
        paths.push(file)
        discovered.set(key, paths)
      }
    }

    const locales = await Promise.all(
      localeCodes.map(async (locale) => {
        const json = JSON.parse(
          await readFile(`src/i18n/locales/${locale}.json`, 'utf8')
        ) as { translation: Record<string, string> }
        return [locale, json.translation] as const
      })
    )
    const missing = [...discovered]
      .flatMap(([key, paths]) =>
        locales
          .filter(([, translation]) => !(key in translation))
          .map(([locale]) => `${locale}: ${key} (${paths.join(', ')})`)
      )
      .sort()

    assert.deepEqual(missing, [])

    const invalidValues = [...discovered].flatMap(([key]) =>
      locales.flatMap(([locale, translation]) => {
        const value = translation[key]
        const expectedVariables = interpolationVariables(key)
        const actualVariables = interpolationVariables(value ?? '')
        const errors: string[] = []
        if (!value?.trim()) {
          errors.push(`${locale}: ${key} has an empty translation`)
        }
        try {
          assert.deepEqual(actualVariables, expectedVariables)
        } catch {
          errors.push(
            `${locale}: ${key} placeholders ${actualVariables.join(',')} do not match ${expectedVariables.join(',')}`
          )
        }
        if (
          (locale === 'zh' || locale === 'zh-TW') &&
          key.includes('platform quota') &&
          /platform quota/i.test(value ?? '')
        ) {
          errors.push(`${locale}: ${key} still contains platform quota`)
        }
        return errors
      })
    )

    assert.deepEqual(invalidValues.sort(), [])
  })
})
