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
import test from 'node:test'

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { AxiosAdapter, AxiosResponse } from 'axios'
import { createInstance } from 'i18next'
import { createElement, Profiler } from 'react'
import { flushSync } from 'react-dom'
import { createRoot } from 'react-dom/client'
import { I18nextProvider, initReactI18next } from 'react-i18next'

class TestNode {
  nodeType: number
  nodeName: string
  tagName?: string
  namespaceURI = 'http://www.w3.org/1999/xhtml'
  ownerDocument: unknown
  parentNode: TestNode | null = null
  childNodes: TestNode[] = []
  style: Record<string, string> = {}
  attributes: Record<string, string> = {}
  nodeValue = ''
  private ownText = ''

  constructor(nodeType: number, nodeName: string, ownerDocument: unknown) {
    this.nodeType = nodeType
    this.nodeName = nodeName
    this.tagName = nodeType === 1 ? nodeName : undefined
    this.ownerDocument = ownerDocument
  }

  appendChild(node: TestNode) {
    node.parentNode = this
    this.childNodes.push(node)
    return node
  }

  insertBefore(node: TestNode, before: TestNode) {
    node.parentNode = this
    const index = this.childNodes.indexOf(before)
    this.childNodes.splice(index < 0 ? this.childNodes.length : index, 0, node)
    return node
  }

  removeChild(node: TestNode) {
    const index = this.childNodes.indexOf(node)
    if (index >= 0) this.childNodes.splice(index, 1)
    node.parentNode = null
    return node
  }

  setAttribute(name: string, value: unknown) {
    this.attributes[name] = String(value)
  }

  removeAttribute(name: string) {
    delete this.attributes[name]
  }

  getAttribute(name: string) {
    return this.attributes[name] ?? null
  }

  addEventListener() {}
  removeEventListener() {}
  focus() {}
  blur() {}

  contains(node: TestNode | null) {
    let current = node
    while (current) {
      if (current === this) return true
      current = current.parentNode
    }
    return false
  }

  getRootNode() {
    return this.ownerDocument
  }

  set textContent(value: string) {
    this.ownText = String(value)
    this.childNodes = []
  }

  get textContent(): string {
    return (
      this.ownText +
      this.nodeValue +
      this.childNodes.map((node) => node.textContent).join('')
    )
  }
}

function findNode(
  node: TestNode,
  predicate: (candidate: TestNode) => boolean
): TestNode | undefined {
  if (predicate(node)) return node
  for (const child of node.childNodes) {
    const match = findNode(child, predicate)
    if (match) return match
  }
  return undefined
}

function getReactProps(node: TestNode) {
  const propsKey = Object.keys(node).find((key) =>
    key.startsWith('__reactProps$')
  )
  assert.ok(propsKey, 'expected React props on rendered node')
  return (node as unknown as Record<string, Record<string, unknown>>)[propsKey]
}

function installTestDom() {
  const globalKeys = [
    'window',
    'document',
    'localStorage',
    'sessionStorage',
    'Node',
    'Element',
    'HTMLElement',
    'navigator',
    'MutationObserver',
    'ResizeObserver',
    'requestAnimationFrame',
    'cancelAnimationFrame',
  ] as const
  const originalGlobals = new Map(
    globalKeys.map((key) => [
      key,
      Object.getOwnPropertyDescriptor(globalThis, key),
    ])
  )
  const document = {
    nodeType: 9,
    nodeName: '#document',
    activeElement: null,
    defaultView: null as unknown,
    body: null as unknown as TestNode,
    head: null as unknown as TestNode,
    documentElement: null as unknown as TestNode,
    addEventListener() {},
    removeEventListener() {},
    createElement(name: string) {
      return new TestNode(1, name.toUpperCase(), document)
    },
    createElementNS(namespace: string, name: string) {
      const node = new TestNode(1, name, document)
      node.namespaceURI = namespace
      return node
    },
    createTextNode(value: string) {
      const node = new TestNode(3, '#text', document)
      node.nodeValue = String(value)
      return node
    },
    createComment(value: string) {
      const node = new TestNode(8, '#comment', document)
      node.nodeValue = String(value)
      return node
    },
    getElementsByTagName(name: string) {
      if (name === 'head') return [document.head]
      if (name === 'body') return [document.body]
      return []
    },
  }
  document.body = new TestNode(1, 'BODY', document)
  document.head = new TestNode(1, 'HEAD', document)
  document.documentElement = new TestNode(1, 'HTML', document)

  const storage = {
    getItem: () => null,
    setItem: () => undefined,
    removeItem: () => undefined,
    clear: () => undefined,
  }
  const window = {
    document,
    localStorage: storage,
    sessionStorage: storage,
    location: { href: 'http://localhost/' },
    Node: TestNode,
    Element: TestNode,
    HTMLElement: TestNode,
    HTMLIFrameElement: class {},
    addEventListener() {},
    removeEventListener() {},
    getComputedStyle: () => ({ display: 'block' }),
    matchMedia: () => ({
      matches: false,
      addEventListener() {},
      removeEventListener() {},
    }),
    requestAnimationFrame(callback: FrameRequestCallback) {
      return setTimeout(() => callback(Date.now()), 0) as unknown as number
    },
    cancelAnimationFrame(handle: number) {
      clearTimeout(handle)
    },
  }
  document.defaultView = window

  Object.assign(globalThis, {
    window,
    document,
    localStorage: storage,
    sessionStorage: storage,
    Node: TestNode,
    Element: TestNode,
    HTMLElement: TestNode,
    navigator: { userAgent: 'bun' },
    MutationObserver: class {
      observe() {}
      disconnect() {}
    },
    ResizeObserver: class {
      observe() {}
      disconnect() {}
    },
    requestAnimationFrame: window.requestAnimationFrame,
    cancelAnimationFrame: window.cancelAnimationFrame,
  })

  return {
    document,
    restore() {
      for (const [key, descriptor] of originalGlobals) {
        if (descriptor) {
          Object.defineProperty(globalThis, key, descriptor)
        } else {
          Reflect.deleteProperty(globalThis, key)
        }
      }
    },
  }
}

test('whitelist renders loading, failed, and empty queries without a render loop', async () => {
  const testDom = installTestDom()
  try {
    const { document } = testDom
    const [{ GitHubRegistrationWhitelist }, { api }] = await Promise.all([
      import('./github-registration-whitelist'),
      import('@/lib/api'),
    ])
    const originalAdapter = api.defaults.adapter
    const i18n = createInstance()
    await i18n.use(initReactI18next).init({
      lng: 'en',
      fallbackLng: 'en',
      resources: {
        en: {
          translation: {
            'No GitHub age exemptions': 'No GitHub age exemptions',
            'Request failed': 'Request failed',
            Retry: 'Retry',
          },
        },
      },
    })

    const cases: Array<{
      name: string
      adapter: AxiosAdapter
      expectedStatus: 'pending' | 'error' | 'success'
      expectedText?: RegExp
    }> = [
      {
        name: 'loading',
        adapter: () => new Promise<AxiosResponse>(() => undefined),
        expectedStatus: 'pending',
      },
      {
        name: 'failed',
        adapter: async () => {
          throw new Error('request failed')
        },
        expectedStatus: 'error',
        expectedText: /Request failed.*Retry/,
      },
      {
        name: 'empty',
        adapter: async (config) => ({
          data: { success: true, message: '', data: [] },
          status: 200,
          statusText: 'OK',
          headers: {},
          config,
        }),
        expectedStatus: 'success',
        expectedText: /No GitHub age exemptions/,
      },
    ]

    try {
      for (const testCase of cases) {
        api.defaults.adapter = testCase.adapter
        const queryClient = new QueryClient({
          defaultOptions: {
            queries: {
              enabled: testCase.name !== 'loading',
              retry: false,
            },
          },
        })
        const container = new TestNode(1, 'DIV', document)
        const root = createRoot(container as unknown as Element)
        let commitCount = 0
        const waitForTerminalStatus =
          testCase.expectedStatus === 'pending'
            ? Promise.resolve()
            : new Promise<void>((resolve) => {
                const unsubscribe = queryClient
                  .getQueryCache()
                  .subscribe(() => {
                    const state = queryClient.getQueryState([
                      'github-registration-whitelist',
                    ])
                    if (state?.status === testCase.expectedStatus) {
                      unsubscribe()
                      resolve()
                    }
                  })
              })

        flushSync(() => {
          root.render(
            createElement(
              QueryClientProvider,
              { client: queryClient },
              createElement(
                I18nextProvider,
                { i18n },
                createElement(
                  Profiler,
                  {
                    id: testCase.name,
                    onRender: () => {
                      commitCount += 1
                      if (commitCount > 5) {
                        throw new Error(
                          `${testCase.name} query entered a render loop`
                        )
                      }
                    },
                  },
                  createElement(GitHubRegistrationWhitelist)
                )
              )
            )
          )
        })
        await waitForTerminalStatus
        for (let flush = 0; flush < 10; flush += 1) {
          await new Promise<void>((resolve) => setImmediate(resolve))
        }
        const renderedText = container.textContent
        assert.equal(
          queryClient.getQueryState(['github-registration-whitelist'])?.status,
          testCase.expectedStatus
        )
        flushSync(() => root.unmount())
        queryClient.clear()
        for (let flush = 0; flush < 10; flush += 1) {
          await new Promise<void>((resolve) => setImmediate(resolve))
        }

        assert.ok(
          commitCount <= 5,
          `${testCase.name} query rendered ${commitCount} times`
        )
        if (testCase.expectedText) {
          assert.match(renderedText, testCase.expectedText)
        }
      }
    } finally {
      api.defaults.adapter = originalAdapter
    }
  } finally {
    testDom.restore()
  }
})

test('whitelist renders entry and resolved identity dates for project Chinese locale codes', async () => {
  const testDom = installTestDom()
  try {
    const { document } = testDom
    const [{ GitHubRegistrationWhitelist }, { api }] = await Promise.all([
      import('./github-registration-whitelist'),
      import('@/lib/api'),
    ])
    const originalAdapter = api.defaults.adapter
    let activeRoot: ReturnType<typeof createRoot> | undefined
    let activeQueryClient: QueryClient | undefined

    try {
      for (const locale of ['zhCN', 'zhTW']) {
        let resolveRequested = false
        api.defaults.adapter = async (config) => {
          if (config.url === '/api/github-registration/resolve') {
            resolveRequested = true
            return {
              data: {
                success: true,
                message: '',
                data: {
                  id: 2002,
                  login: 'hubot',
                  created_at: 1704153600,
                },
              },
              status: 200,
              statusText: 'OK',
              headers: {},
              config,
            }
          }
          return {
            data: {
              success: true,
              message: '',
              data: [
                {
                  id: 1,
                  github_id: '1001',
                  github_login: 'octocat',
                  github_created_at: 1704067200,
                  remark: 'trusted',
                  created_by: 1,
                  updated_by: 1,
                  created_at: 1704067200,
                  updated_at: 1704067200,
                },
              ],
            },
            status: 200,
            statusText: 'OK',
            headers: {},
            config,
          }
        }

        const i18n = createInstance()
        await i18n.use(initReactI18next).init({
          lng: locale,
          fallbackLng: locale,
          resources: {
            [locale]: { translation: {} },
          },
        })
        const queryClient = new QueryClient({
          defaultOptions: { queries: { retry: false } },
        })
        const container = new TestNode(1, 'DIV', document)
        const root = createRoot(container as unknown as Element)
        activeRoot = root
        activeQueryClient = queryClient
        const queryCompleted = new Promise<void>((resolve) => {
          const unsubscribe = queryClient.getQueryCache().subscribe(() => {
            if (
              queryClient.getQueryState(['github-registration-whitelist'])
                ?.status === 'success'
            ) {
              unsubscribe()
              resolve()
            }
          })
        })

        flushSync(() => {
          root.render(
            createElement(
              QueryClientProvider,
              { client: queryClient },
              createElement(
                I18nextProvider,
                { i18n },
                createElement(GitHubRegistrationWhitelist)
              )
            )
          )
        })
        await queryCompleted
        for (let flush = 0; flush < 10; flush += 1) {
          await new Promise<void>((resolve) => setImmediate(resolve))
        }

        assert.match(container.textContent, /octocat/)
        const usernameInput = findNode(
          container,
          (node) => node.getAttribute('id') === 'github-whitelist-username'
        )
        assert.ok(usernameInput)
        const inputProps = getReactProps(usernameInput)
        flushSync(() => {
          const onChange = inputProps.onChange as (event: {
            target: { value: string }
            currentTarget: { value: string }
            nativeEvent: { defaultPrevented: boolean }
          }) => void
          const inputTarget = { value: 'hubot' }
          onChange({
            target: inputTarget,
            currentTarget: inputTarget,
            nativeEvent: { defaultPrevented: false },
          })
        })

        const resolveButton = findNode(
          container,
          (node) =>
            node.tagName === 'BUTTON' &&
            /Resolve identity/.test(node.textContent)
        )
        assert.ok(resolveButton)
        const buttonProps = getReactProps(resolveButton)
        flushSync(() => {
          const onClick = buttonProps.onClick as () => void
          onClick()
        })
        for (let flush = 0; flush < 10; flush += 1) {
          await new Promise<void>((resolve) => setImmediate(resolve))
        }

        assert.equal(resolveRequested, true)
        assert.match(container.textContent, /hubot/)
        assert.match(container.textContent, /2002/)

        flushSync(() => root.unmount())
        queryClient.clear()
        activeRoot = undefined
        activeQueryClient = undefined
        for (let flush = 0; flush < 10; flush += 1) {
          await new Promise<void>((resolve) => setImmediate(resolve))
        }
      }
    } finally {
      if (activeRoot) flushSync(() => activeRoot?.unmount())
      activeQueryClient?.clear()
      for (let flush = 0; flush < 10; flush += 1) {
        await new Promise<void>((resolve) => setImmediate(resolve))
      }
      api.defaults.adapter = originalAdapter
    }
  } finally {
    testDom.restore()
  }
})
