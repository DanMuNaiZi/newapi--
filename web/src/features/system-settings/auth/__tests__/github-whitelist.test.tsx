/* Copyright (C) 2023-2026 QuantumNous */
import { QueryClientProvider, type QueryClient } from '@tanstack/react-query'
import { act, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import axios, { type Axios, type AxiosRequestConfig } from 'axios'
import { Toaster, toast } from 'sonner'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api, applyAuthBundle } from '@/lib/api'
import { createAppQueryClient } from '@/lib/query-client'
import { useAuthStore, type AuthBundle } from '@/stores/auth-store'

import { createGitHubRegistrationWhitelist } from '../../api'
import type { GitHubRegistrationWhitelistEntry } from '../../types'
import { GitHubRegistrationWhitelist } from '../github-registration-whitelist'

const entry: GitHubRegistrationWhitelistEntry = {
  id: 5,
  github_id: '12345',
  github_login: 'octocat',
  github_created_at: 1_600_000_000,
  remark: 'Approved community contributor',
  created_by: 1,
  updated_by: 1,
  created_at: 1_800_000_000,
  updated_at: 1_800_000_000,
}
let originalAdapter: typeof api.defaults.adapter
let client: QueryClient
let reads: string[]
let writes: Array<{ method: string; url: string; data: unknown }>
let entries: GitHubRegistrationWhitelistEntry[]
let listResponse: unknown
let resolveResponse: unknown
let createResponse: unknown
let resolveResponses: Record<string, unknown>
let createResponses: Record<string, unknown>
let updateResponse: unknown
let deleteResponse: unknown
let finishResolve: ((value: unknown) => void) | undefined
let delayResolve: boolean
let delayCreate: boolean
let finishCreate: ((status: number) => void) | undefined
let createSignal: AxiosRequestConfig['signal']
let refreshRequests: number
let importAuthorizations: string[]

function authBundle(id: number, sid = `whitelist-session-${id}`): AuthBundle {
  const now = Math.floor(Date.now() / 1000)
  return {
    access_token: `whitelist-fixture-${id}`,
    token_type: 'Bearer',
    access_expires_at: now + 3600,
    user: {
      id,
      username: `reviewer${id}`,
      role: 10,
      permissions: {
        admin_permissions: { oauth: { read: true, write: true } },
      },
    },
    session: {
      sid,
      current: true,
      login_method: 'password',
      ip: '192.0.2.1',
      user_agent: 'whitelist-regression',
      created_at: now,
      last_active_at: now,
      expires_at: now + 3600,
    },
  }
}

function observeImportCompletion() {
  let finish: (() => void) | undefined
  const completed = new Promise<void>((resolve) => {
    finish = resolve
  })
  const interceptor = api.interceptors.response.use(
    (response) => {
      if (
        response.config.url === '/api/github-registration/whitelist' &&
        response.config.method === 'post'
      ) {
        finish?.()
      }
      return response
    },
    (error) => {
      if (
        axios.isAxiosError(error) &&
        error.config?.url === '/api/github-registration/whitelist' &&
        error.config.method === 'post'
      ) {
        finish?.()
      }
      throw error
    }
  )
  return {
    completed,
    dispose: () => api.interceptors.response.eject(interceptor),
  }
}

function setPermissions(permissions: Record<string, boolean>) {
  useAuthStore.getState().auth.setUser({
    id: 1,
    username: 'reviewer',
    role: 10,
    permissions: { admin_permissions: { oauth: permissions } },
  })
}

function renderWhitelist() {
  return render(
    <QueryClientProvider client={client}>
      <GitHubRegistrationWhitelist />
      <Toaster />
    </QueryClientProvider>
  )
}

beforeEach(() => {
  applyAuthBundle(authBundle(1), false)
  setPermissions({ read: true, write: true })
  reads = []
  writes = []
  entries = [entry]
  listResponse = undefined
  resolveResponse = {
    success: true,
    data: { id: 12345, login: 'octocat', created_at: 1_600_000_000 },
  }
  createResponse = {
    success: false,
    message: 'Account identity changed; verify again',
  }
  resolveResponses = {}
  createResponses = {}
  updateResponse = { success: false, message: 'Remark cannot be saved' }
  deleteResponse = {
    success: false,
    message: 'Exemption removal is temporarily unavailable',
  }
  finishResolve = undefined
  delayResolve = false
  delayCreate = false
  finishCreate = undefined
  createSignal = undefined
  refreshRequests = 0
  importAuthorizations = []
  client = createAppQueryClient()
  client.setDefaultOptions({
    queries: { retry: false },
    mutations: { retry: false },
  })
  originalAdapter = api.defaults.adapter
  const originalRequest = axios.Axios.prototype.request
  vi.spyOn(axios.Axios.prototype, 'request').mockImplementation(function (
    this: Axios,
    config: string | AxiosRequestConfig,
    options?: AxiosRequestConfig
  ) {
    const request =
      typeof config === 'string' ? { ...options, url: config } : config
    if (request.url === '/api/user/auth/refresh') {
      request.adapter = async (refreshConfig) => {
        refreshRequests++
        const auth = useAuthStore.getState().auth
        const refreshed = {
          ...authBundle(auth.user?.id ?? 1, auth.session?.sid),
          user: auth.user ?? authBundle(1).user,
        }
        return {
          config: refreshConfig,
          status: 200,
          statusText: 'OK',
          headers: {},
          data: { success: true, data: refreshed },
        }
      }
    }
    return originalRequest.call(this, request)
  } as typeof originalRequest)
  api.defaults.adapter = async (config) => {
    const url = config.url ?? ''
    let data: unknown
    if (config.method === 'get') {
      reads.push(url)
      data = listResponse ?? { success: true, data: entries }
    } else {
      const payload = config.data ? JSON.parse(config.data) : undefined
      const requestedUsername =
        (payload as { username?: string } | undefined)?.username ?? ''
      writes.push({
        method: config.method ?? '',
        url,
        data: payload,
      })
      if (url.endsWith('/resolve')) {
        data = delayResolve
          ? await new Promise((resolve) => {
              finishResolve = resolve
            })
          : (resolveResponses[requestedUsername] ?? resolveResponse)
      } else if (config.method === 'patch') data = updateResponse
      else if (config.method === 'delete') {
        data = deleteResponse
        if (
          typeof data === 'object' &&
          data !== null &&
          'success' in data &&
          data.success
        ) {
          entries = []
        }
      } else {
        importAuthorizations.push(String(config.headers.Authorization ?? ''))
        if (delayCreate) {
          createSignal = config.signal
          await new Promise<void>((resolve, reject) => {
            finishCreate = (status) => {
              if (status === 401) {
                reject(
                  new axios.AxiosError(
                    'Expired fixture token',
                    undefined,
                    config,
                    undefined,
                    {
                      config,
                      status: 401,
                      statusText: 'Unauthorized',
                      headers: {},
                      data: { code: 'AUTH_SESSION_REVOKED' },
                    }
                  )
                )
              } else resolve()
            }
          })
        }
        data = createResponses[requestedUsername] ?? createResponse
        if (
          typeof data === 'object' &&
          data !== null &&
          'success' in data &&
          data.success &&
          'data' in data
        ) {
          const created = data.data as GitHubRegistrationWhitelistEntry
          entries = [
            ...entries.filter((item) => item.id !== created.id),
            created,
          ]
        }
      }
    }
    return { data, status: 200, statusText: 'OK', headers: {}, config }
  }
})

afterEach(() => {
  vi.restoreAllMocks()
  api.defaults.adapter = originalAdapter
  client.clear()
  useAuthStore.getState().auth.reset()
  toast.dismiss()
})

describe('GitHub registration age exemption controls', () => {
  it('does not fetch or expose exemptions without OAuth read access', async () => {
    setPermissions({})
    renderWhitelist()
    expect(
      await screen.findByText('You do not have permission to view this page')
    ).toBeVisible()
    expect(reads).toEqual([])
  })

  it('shows exemption details without editing controls for a read-only administrator', async () => {
    setPermissions({ read: true })
    renderWhitelist()
    expect(await screen.findByText(entry.github_login)).toBeVisible()
    expect(screen.getByText(entry.remark)).toBeVisible()
    expect(
      screen.queryByRole('button', { name: 'Resolve identity' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Save' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: /Remove exemption/ })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('textbox', { name: 'GitHub usernames' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Resolve batch' })
    ).not.toBeInTheDocument()
  })

  it('locks the username during identity resolution and retains the verified ID on a failed add', async () => {
    const user = userEvent.setup()
    delayResolve = true
    renderWhitelist()
    await user.type(
      screen.getByRole('textbox', { name: 'GitHub username' }),
      'octocat'
    )
    await user.click(screen.getByRole('button', { name: 'Resolve identity' }))
    await waitFor(() => expect(finishResolve).toBeDefined())
    expect(
      screen.getByRole('textbox', { name: 'GitHub username' })
    ).toBeDisabled()
    finishResolve?.(resolveResponse)
    await user.type(
      await screen.findByRole('textbox', { name: 'Remark' }),
      'Trusted contributor'
    )
    await user.click(
      screen.getByRole('button', { name: 'Confirm and add exemption' })
    )
    expect(
      await screen.findByText('Account identity changed; verify again')
    ).toBeVisible()
    expect(screen.getByRole('textbox', { name: 'Remark' })).toHaveValue(
      'Trusted contributor'
    )
    expect(writes[1]).toEqual({
      method: 'post',
      url: '/api/github-registration/whitelist',
      data: {
        username: 'octocat',
        github_id: '12345',
        remark: 'Trusted contributor',
      },
    })
    createResponse = { success: true, data: entry }
    await user.click(
      screen.getByRole('button', { name: 'Confirm and add exemption' })
    )
    await waitFor(() =>
      expect(
        screen.queryByRole('button', { name: 'Confirm and add exemption' })
      ).not.toBeInTheDocument()
    )
    expect(
      screen.getByRole('textbox', { name: 'GitHub username' })
    ).toHaveValue('')
  })

  it('keeps the draft remark when its update fails', async () => {
    const user = userEvent.setup()
    renderWhitelist()
    const input = await screen.findByRole('textbox', {
      name: 'Remark for octocat',
    })
    await user.clear(input)
    await user.type(input, 'Revised explanation')
    await user.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('Remark cannot be saved')).toBeVisible()
    expect(input).toHaveValue('Revised explanation')
    expect(writes[0]).toEqual({
      method: 'patch',
      url: '/api/github-registration/whitelist/5',
      data: { remark: 'Revised explanation' },
    })
  })

  it('keeps deletion open on failure and removes the row after a successful retry', async () => {
    const user = userEvent.setup()
    renderWhitelist()
    await user.click(
      await screen.findByRole('button', {
        name: 'Remove exemption for octocat',
      })
    )
    const dialog = await screen.findByRole('alertdialog', {
      name: 'Remove GitHub age exemption?',
    })
    await user.click(within(dialog).getByRole('button', { name: 'Remove' }))
    expect(
      await screen.findByText('Exemption removal is temporarily unavailable')
    ).toBeVisible()
    expect(dialog).toBeVisible()
    deleteResponse = { success: true }
    await user.click(within(dialog).getByRole('button', { name: 'Remove' }))
    await waitFor(() =>
      expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    )
    expect(await screen.findByText('No GitHub age exemptions')).toBeVisible()
  })

  it('offers a retry when whitelist loading returns success false', async () => {
    listResponse = {
      success: false,
      message: 'GitHub exemption list unavailable',
    }
    renderWhitelist()
    expect(await screen.findByRole('button', { name: 'Retry' })).toBeVisible()
    listResponse = undefined
    await userEvent.setup().click(screen.getByRole('button', { name: 'Retry' }))
    expect(await screen.findByText('octocat')).toBeVisible()
  })
})

describe('GitHub whitelist batch preview and import', () => {
  const alice = { ...entry, id: 6, github_login: 'alice', github_id: '1001' }
  const bob = { ...entry, id: 7, github_login: 'bob', github_id: '1002' }

  beforeEach(() => {
    resolveResponses = {
      alice: {
        success: true,
        data: { id: 1001, login: 'alice', created_at: entry.github_created_at },
      },
      bob: {
        success: true,
        data: { id: 1002, login: 'bob', created_at: entry.github_created_at },
      },
    }
    createResponses = {
      alice: { success: true, data: alice },
      bob: { success: true, data: bob },
    }
  })

  it('deduplicates case-insensitive usernames from lines, commas and spaces without creating before confirmation', async () => {
    const user = userEvent.setup()
    renderWhitelist()
    const input = await screen.findByRole('textbox', {
      name: 'GitHub usernames',
    })
    await user.click(input)
    await user.paste('ALICE, alice\nBob\t bob')
    await user.click(screen.getByRole('button', { name: 'Resolve batch' }))
    const preview = await screen.findByRole('table', {
      name: 'GitHub identity preview',
    })
    await waitFor(() =>
      expect(within(preview).getAllByText('Verified')).toHaveLength(2)
    )
    expect(writes.map((write) => write.data)).toEqual([
      { username: 'alice' },
      { username: 'bob' },
    ])
    expect(within(preview).getByText('1001')).toBeVisible()
    expect(within(preview).getByText('1002')).toBeVisible()
    await user.click(
      screen.getByRole('button', { name: 'Import verified accounts' })
    )
    const dialog = await screen.findByRole('alertdialog', {
      name: 'Confirm GitHub whitelist import?',
    })
    expect(writes.every((write) => write.url.endsWith('/resolve'))).toBe(true)
    await user.click(within(dialog).getByRole('button', { name: 'Cancel' }))
    await waitFor(() =>
      expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    )
    expect(writes).toHaveLength(2)
    expect(within(preview).getAllByText('Verified')).toHaveLength(2)
  })

  it('rejects more than 20 unique usernames without issuing resolve requests', async () => {
    const user = userEvent.setup()
    renderWhitelist()
    const input = await screen.findByRole('textbox', {
      name: 'GitHub usernames',
    })
    await user.click(input)
    await user.paste(
      Array.from({ length: 21 }, (_, index) => `account${index}`).join('\n')
    )
    expect(
      screen.getByText(
        'A batch can contain at most 20 unique GitHub usernames.'
      )
    ).toBeVisible()
    expect(input).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByRole('button', { name: 'Resolve batch' })).toBeDisabled()
    expect(writes).toEqual([])
    await user.clear(input)
    await user.paste(
      Array.from({ length: 20 }, (_, index) => `account${index}`).join(', ')
    )
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'Resolve batch' })
      ).toBeEnabled()
    )
  })

  it('retains verification failures and retries only failed usernames', async () => {
    const user = userEvent.setup()
    resolveResponses.bob = {
      success: false,
      message: 'GitHub account unavailable',
    }
    renderWhitelist()
    await user.type(
      await screen.findByRole('textbox', { name: 'GitHub usernames' }),
      'alice bob'
    )
    await user.click(screen.getByRole('button', { name: 'Resolve batch' }))
    const preview = await screen.findByRole('table', {
      name: 'GitHub identity preview',
    })
    expect(
      await within(preview).findByText('GitHub account unavailable')
    ).toBeVisible()
    expect(within(preview).getByText('Verified')).toBeVisible()
    expect(within(preview).getByText('Verification failed')).toBeVisible()
    resolveResponses.bob = {
      success: true,
      data: { id: 1002, login: 'bob', created_at: entry.github_created_at },
    }
    await user.click(
      screen.getByRole('button', { name: 'Retry failed resolutions' })
    )
    await waitFor(() =>
      expect(within(preview).getAllByText('Verified')).toHaveLength(2)
    )
    expect(writes.map((write) => write.data)).toEqual([
      { username: 'alice' },
      { username: 'bob' },
      { username: 'bob' },
    ])
    expect(writes.every((write) => write.url.endsWith('/resolve'))).toBe(true)
  })

  it('skips existing numeric IDs and preserves successful imports when retrying a partial failure', async () => {
    const user = userEvent.setup()
    createResponses.bob = {
      success: false,
      message: 'Bob exemption could not be saved',
    }
    renderWhitelist()
    const input = await screen.findByRole('textbox', {
      name: 'GitHub usernames',
    })
    await user.type(input, 'octocat alice bob')
    await user.type(
      screen.getByRole('textbox', { name: 'Batch remark' }),
      'Trusted maintainers'
    )
    await user.click(screen.getByRole('button', { name: 'Resolve batch' }))
    const preview = await screen.findByRole('table', {
      name: 'GitHub identity preview',
    })
    await waitFor(() =>
      expect(within(preview).getAllByText('Verified')).toHaveLength(2)
    )
    expect(within(preview).getByText('Already in whitelist')).toBeVisible()
    await user.click(
      screen.getByRole('button', { name: 'Import verified accounts' })
    )
    const dialog = await screen.findByRole('alertdialog', {
      name: 'Confirm GitHub whitelist import?',
    })
    expect(input).toBeDisabled()
    await user.click(
      within(dialog).getByRole('button', { name: 'Import accounts' })
    )
    expect(
      await within(preview).findByText('Bob exemption could not be saved')
    ).toBeVisible()
    expect(within(preview).getByText('Imported')).toBeVisible()
    await waitFor(() =>
      expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    )
    createResponses.bob = { success: true, data: bob }
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'Retry failed imports' })
      ).toBeEnabled()
    )
    await user.click(
      screen.getByRole('button', { name: 'Retry failed imports' })
    )
    await user.click(
      within(await screen.findByRole('alertdialog')).getByRole('button', {
        name: 'Import accounts',
      })
    )
    await waitFor(() =>
      expect(within(preview).getAllByText('Imported')).toHaveLength(2)
    )
    const imports = writes.filter(
      (write) => write.url === '/api/github-registration/whitelist'
    )
    expect(imports.map((write) => write.data)).toEqual([
      { username: 'alice', github_id: '1001', remark: 'Trusted maintainers' },
      { username: 'bob', github_id: '1002', remark: 'Trusted maintainers' },
      { username: 'bob', github_id: '1002', remark: 'Trusted maintainers' },
    ])
  })

  it('removes the batch confirmation and prevents importing when OAuth write permission is revoked', async () => {
    const user = userEvent.setup()
    renderWhitelist()
    await user.type(
      await screen.findByRole('textbox', { name: 'GitHub usernames' }),
      'alice'
    )
    await user.click(screen.getByRole('button', { name: 'Resolve batch' }))
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'Import verified accounts' })
      ).toBeEnabled()
    )
    await user.click(
      screen.getByRole('button', { name: 'Import verified accounts' })
    )
    expect(await screen.findByRole('alertdialog')).toBeVisible()
    act(() => setPermissions({ read: true }))
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    expect(
      screen.queryByRole('textbox', { name: 'GitHub usernames' })
    ).not.toBeInTheDocument()
    expect(writes.every((write) => write.url.endsWith('/resolve'))).toBe(true)
  })

  it('does not import identities whose numeric IDs cannot be represented safely', async () => {
    const user = userEvent.setup()
    resolveResponses.alice = {
      success: true,
      data: {
        id: Number.MAX_SAFE_INTEGER + 1,
        login: 'alice',
        created_at: entry.github_created_at,
      },
    }
    renderWhitelist()
    await user.type(
      await screen.findByRole('textbox', { name: 'GitHub usernames' }),
      'alice'
    )
    await user.click(screen.getByRole('button', { name: 'Resolve batch' }))
    const preview = await screen.findByRole('table', {
      name: 'GitHub identity preview',
    })
    expect(
      await within(preview).findByText('Unable to verify GitHub account')
    ).toBeVisible()
    expect(within(preview).getByText('Verification failed')).toBeVisible()
    expect(
      screen.getByRole('button', { name: 'Import verified accounts' })
    ).toBeDisabled()
    expect(writes.map((write) => write.data)).toEqual([{ username: 'alice' }])
  })

  it('stops resolving the remaining usernames if OAuth write permission is revoked during a request', async () => {
    const user = userEvent.setup()
    delayResolve = true
    renderWhitelist()
    const input = await screen.findByRole('textbox', {
      name: 'GitHub usernames',
    })
    await user.type(input, 'alice bob')
    await user.click(screen.getByRole('button', { name: 'Resolve batch' }))
    await waitFor(() => expect(finishResolve).toBeDefined())
    expect(input).toBeDisabled()
    await act(async () => {
      setPermissions({ read: true })
      finishResolve?.(resolveResponses.alice)
    })
    expect(
      screen.queryByRole('textbox', { name: 'GitHub usernames' })
    ).not.toBeInTheDocument()
    expect(writes.map((write) => write.data)).toEqual([{ username: 'alice' }])
  })

  it.each(['actor', 'session', 'permission', 'unmount'] as const)(
    'cancels pending imports without refreshing or replaying after %s changes',
    async (change) => {
      const user = userEvent.setup()
      delayCreate = true
      const view = renderWhitelist()
      await user.type(
        await screen.findByRole('textbox', { name: 'GitHub usernames' }),
        'alice bob'
      )
      await user.click(screen.getByRole('button', { name: 'Resolve batch' }))
      await waitFor(() =>
        expect(
          screen.getByRole('button', { name: 'Import verified accounts' })
        ).toBeEnabled()
      )
      await user.click(
        screen.getByRole('button', { name: 'Import verified accounts' })
      )
      const completion = observeImportCompletion()
      try {
        await user.click(
          within(await screen.findByRole('alertdialog')).getByRole('button', {
            name: 'Import accounts',
          })
        )
        await waitFor(() => expect(finishCreate).toBeDefined())
        delayCreate = false
        await act(async () => {
          if (change === 'actor') applyAuthBundle(authBundle(2), false)
          else if (change === 'session') {
            applyAuthBundle(authBundle(1, 'replacement-session'), false)
          } else if (change === 'permission') setPermissions({ read: true })
          else view.unmount()
          finishCreate?.(401)
          await completion.completed
        })
        expect(importAuthorizations).toEqual(['Bearer whitelist-fixture-1'])
        expect(refreshRequests).toBe(0)
        expect(
          writes.filter(
            (write) => write.url === '/api/github-registration/whitelist'
          )
        ).toHaveLength(1)
        expect(createSignal?.aborted).toBe(true)
        expect(screen.queryByText('Session expired!')).not.toBeInTheDocument()
        expect(
          screen.queryByText('Failed to add exemption')
        ).not.toBeInTheDocument()
      } finally {
        completion.dispose()
      }
    }
  )

  it('does not dispatch an import under a replacement actor while Axios is preparing the request', async () => {
    const user = userEvent.setup()
    renderWhitelist()
    await user.type(
      await screen.findByRole('textbox', { name: 'GitHub usernames' }),
      'alice'
    )
    await user.click(screen.getByRole('button', { name: 'Resolve batch' }))
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'Import verified accounts' })
      ).toBeEnabled()
    )
    await user.click(
      screen.getByRole('button', { name: 'Import verified accounts' })
    )
    let releaseDispatch: (() => void) | undefined
    let blockedSignal: AxiosRequestConfig['signal']
    const beforeDispatch = api.interceptors.request.use(async (config) => {
      if (
        config.url === '/api/github-registration/whitelist' &&
        config.method === 'post'
      ) {
        blockedSignal = config.signal
        await new Promise<void>((resolve) => {
          releaseDispatch = resolve
        })
      }
      return config
    })
    const completion = observeImportCompletion()
    try {
      await user.click(
        within(await screen.findByRole('alertdialog')).getByRole('button', {
          name: 'Import accounts',
        })
      )
      await waitFor(() => expect(releaseDispatch).toBeDefined())
      await act(async () => {
        applyAuthBundle(authBundle(2), false)
        releaseDispatch?.()
        await completion.completed
      })
      expect(
        writes.filter(
          (write) => write.url === '/api/github-registration/whitelist'
        )
      ).toEqual([])
      expect(blockedSignal?.aborted).toBe(true)
      expect(refreshRequests).toBe(0)
      expect(
        screen.queryByText('Failed to add exemption')
      ).not.toBeInTheDocument()
    } finally {
      api.interceptors.request.eject(beforeDispatch)
      completion.dispose()
    }
  })

  it('completes a confirmed import after normal token rotation in the same actor session', async () => {
    const user = userEvent.setup()
    delayCreate = true
    renderWhitelist()
    await user.type(
      await screen.findByRole('textbox', { name: 'GitHub usernames' }),
      'alice'
    )
    await user.click(screen.getByRole('button', { name: 'Resolve batch' }))
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'Import verified accounts' })
      ).toBeEnabled()
    )
    await user.click(
      screen.getByRole('button', { name: 'Import verified accounts' })
    )
    await user.click(
      within(await screen.findByRole('alertdialog')).getByRole('button', {
        name: 'Import accounts',
      })
    )
    await waitFor(() => expect(finishCreate).toBeDefined())
    await act(async () => {
      applyAuthBundle(
        { ...authBundle(1), access_token: 'rotated-whitelist-fixture-1' },
        false
      )
      finishCreate?.(200)
    })
    expect(await screen.findByText('Imported')).toBeVisible()
    expect(createSignal?.aborted).not.toBe(true)
    expect(refreshRequests).toBe(0)
    expect(
      writes.filter(
        (write) => write.url === '/api/github-registration/whitelist'
      )
    ).toHaveLength(1)
  })

  it('never refreshes or replays a whitelist API write under the replacement session', async () => {
    delayCreate = true
    const result = createGitHubRegistrationWhitelist(
      {
        username: 'alice',
        github_id: '1001',
        remark: 'Approved by first actor',
      },
      { skipErrorHandler: true }
    ).catch((error: unknown) => error)
    await waitFor(() => expect(finishCreate).toBeDefined())
    applyAuthBundle(authBundle(2), false)
    delayCreate = false
    finishCreate?.(401)
    const error = await result
    expect(axios.isAxiosError(error)).toBe(true)
    expect(importAuthorizations).toEqual(['Bearer whitelist-fixture-1'])
    expect(refreshRequests).toBe(0)
    expect(
      writes.filter(
        (write) => write.url === '/api/github-registration/whitelist'
      )
    ).toHaveLength(1)
  })
})
