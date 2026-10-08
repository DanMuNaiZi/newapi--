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
import { spawnSync } from 'node:child_process'
import {
  mkdirSync,
  mkdtempSync,
  readdirSync,
  readFileSync,
  rmSync,
  statSync,
  writeFileSync,
} from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, extname, join, relative, resolve } from 'node:path'

const mode = process.argv[2]

if (mode !== '--check' && mode !== '--write') {
  console.error(
    'Usage: node scripts/format-with-protected-headers.mjs --check|--write'
  )
  process.exit(2)
}

const root = process.cwd()
const excludedDirs = new Set([
  '.git',
  '.tanstack',
  'build',
  'coverage',
  'dist',
  'node_modules',
])
const headerExtensions = new Set([
  '.cjs',
  '.cts',
  '.js',
  '.jsx',
  '.mjs',
  '.mts',
  '.ts',
  '.tsx',
])
const protectedHeaderPattern =
  /^\/\*\r?\nCopyright \(C\) [^\r\n]*QuantumNous\r?\n[\s\S]*?\*\/\r?\n+/

function walk(dir, files = []) {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (entry.isDirectory()) {
      if (!excludedDirs.has(entry.name)) {
        walk(join(dir, entry.name), files)
      }
      continue
    }
    if (entry.isFile()) {
      files.push(join(dir, entry.name))
    }
  }
  return files
}

const files = walk(root).filter(
  (file) => statSync(file).size < 10 * 1024 * 1024
)
const stagingRoot = mkdtempSync(join(tmpdir(), 'newapi-format-'))
const snapshots = new Map()
let exitCode = 0

try {
  // Format copies so check mode never strips live source headers or rewrites
  // files held open by Bun/Node on Windows, including this running script.
  for (const file of files) {
    const original = readFileSync(file)
    const stagedFile = join(stagingRoot, relative(root, file))
    let header = ''
    let body = original
    if (headerExtensions.has(extname(file))) {
      const source = original.toString('utf8')
      const match = source.match(protectedHeaderPattern)
      if (match) {
        header = match[0]
        body = Buffer.from(source.slice(header.length))
      }
    }
    snapshots.set(file, { original, header, stagedFile })
    mkdirSync(dirname(stagedFile), { recursive: true })
    writeFileSync(stagedFile, body)
  }

  const result = spawnSync(
    'oxfmt',
    ['-c', '.oxfmtrc.json', '--ignore-path', '.gitignore', '--write', '.'],
    { cwd: stagingRoot, stdio: 'inherit' }
  )
  exitCode = result.status ?? 1
  if (result.error) {
    console.error(result.error.message)
  }
  if (exitCode === 0) {
    const changed = []
    for (const [file, snapshot] of snapshots) {
      const body = readFileSync(snapshot.stagedFile)
      const formatted = snapshot.header
        ? Buffer.from(
            snapshot.header + body.toString('utf8').replace(/^\n+/, '')
          )
        : body
      if (snapshot.original.equals(formatted)) {
        continue
      }
      changed.push(relative(root, file))
      if (mode === '--write') {
        writeFileSync(file, formatted)
      }
    }
    if (mode === '--check' && changed.length > 0) {
      console.error('Format issues found in protected-header-safe check:')
      for (const file of changed) {
        console.error(file)
      }
      exitCode = 1
    }
  }
} finally {
  const ownedTempPrefix = resolve(tmpdir(), 'newapi-format-')
  if (!resolve(stagingRoot).startsWith(ownedTempPrefix)) {
    throw new Error('Refusing to remove an unexpected formatter directory')
  }
  rmSync(stagingRoot, { recursive: true, force: true })
}

process.exit(exitCode)
