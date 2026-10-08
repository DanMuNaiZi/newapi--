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
import {
  QueryClient,
  QueryClientProvider,
  isCancelledError,
} from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
  RouterProvider,
} from '@tanstack/react-router'
import { act, cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AxiosError, type InternalAxiosRequestConfig } from 'axios'
import { toast } from 'sonner'
import { afterEach, expect, it, vi } from 'vitest'

import { TooltipProvider } from '@/components/ui/tooltip'
import { Profile } from '@/features/profile'
import { api } from '@/lib/api'
import { createAppQueryClient } from '@/lib/query-client'
import { statusQueryOptions } from '@/lib/status-query'
import { useAuthStore } from '@/stores/auth-store'
import { useUserPreviewStore } from '@/stores/user-preview-store'

import type { User } from '../../types'
import { UsersMutateDrawer } from '../users-mutate-drawer'
import { UsersProvider } from '../users-provider'
import { UsersTable } from '../users-table'

const clients: QueryClient[] = []
const originalAdapter = api.defaults.adapter

const target: User = {
  id: 2,
  username: 'managed-admin',
  display_name: 'Managed admin',
  role: 10,
  status: 1,
  quota: 0,
  used_quota: 0,
  request_count: 0,
  group: 'default',
}
const label = "View other accounts' audit logs"
const description =
  'View audit records from user and admin roles. Root records are always excluded.'

function renderPermissions(viewerRole: number, allowed?: boolean) {
  useAuthStore
    .getState()
    .auth.setUser({ id: 1, username: 'operator', role: viewerRole })
  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/authz/catalog') {
      return {
        data: {
          success: true,
          data: {
            resources: [
              {
                resource: 'audit',
                label_key: 'Audit Logs',
                actions: [
                  {
                    action: 'read',
                    label_key: label,
                    description_key: description,
                  },
                ],
              },
            ],
            roles: [{ key: 'admin', grants: { audit: { read: false } } }],
          },
        },
      }
    }
    if (url === '/api/group/') {
      return { data: { success: true, data: ['default'] } }
    }
    return {
      data: {
        success: true,
        data: {
          ...target,
          admin_permissions:
            allowed === undefined ? {} : { audit: { read: allowed } },
        },
      },
    }
  })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  clients.push(client)
  return render(
    <QueryClientProvider client={client}>
      <UsersProvider>
        <UsersMutateDrawer
          open
          onOpenChange={() => undefined}
          currentRow={target}
        />
      </UsersProvider>
    </QueryClientProvider>
  )
}

afterEach(() => {
  cleanup()
  for (const client of clients.splice(0)) client.clear()
  api.defaults.adapter = originalAdapter
  vi.restoreAllMocks()
  useUserPreviewStore.getState().clear()
  useAuthStore.getState().auth.reset()
  localStorage.clear()
})

it.each([undefined, true])(
  'root can save an audit grant or revocation from the existing editor (previous=%s)',
  async (allowed) => {
    const put = vi
      .spyOn(api, 'put')
      .mockResolvedValue({ data: { success: true } })
    renderPermissions(100, allowed)
    await screen.findByDisplayValue('Managed admin')
    const checkbox = await screen.findByRole('checkbox', {
      name: new RegExp(label),
    })
    await waitFor(() =>
      expect(checkbox).toHaveAttribute('aria-checked', String(!!allowed))
    )
    expect(screen.getByText(description)).toBeVisible()
    await userEvent.click(checkbox)
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() =>
      expect(put).toHaveBeenCalledWith(
        '/api/user/',
        expect.objectContaining({
          id: 2,
          admin_permissions: { audit: { read: !allowed } },
        })
      )
    )
  }
)

it('admin cannot edit the audit permission even when the catalog is available', async () => {
  renderPermissions(10)
  await screen.findByDisplayValue('Managed admin')
  expect(
    screen.queryByRole('checkbox', { name: new RegExp(label) })
  ).not.toBeInTheDocument()
})

function PreviewUsersPage() {
  return (
    <TooltipProvider>
      <UsersProvider>
        <UsersTable />
      </UsersProvider>
    </TooltipProvider>
  )
}

it('entering a target preview cancels actor queries and keeps the mounted admin list from refetching', async () => {
  const actor = {
    id: 1,
    username: 'root',
    display_name: 'Root Actor',
    role: 100,
  }
  const session = {
    sid: 'preview-transition-actor-session',
    current: true,
    login_method: 'password',
    ip: '127.0.0.1',
    user_agent: 'test-browser',
    created_at: 1,
    last_active_at: 1,
    expires_at: 2_000_000_000,
  }
  useAuthStore.getState().auth.setBundle({
    access_token: 'preview-transition-actor-access',
    token_type: 'Bearer',
    access_expires_at: 2_000_000_000,
    user: actor,
    session,
  })
  const previewTarget = {
    ...target,
    username: 'preview-target',
    display_name: 'Preview Target',
    role: 1,
    quota: 1_000_000,
    aff_count: 0,
    aff_quota: 0,
    aff_history_quota: 0,
    created_time: 0,
  }
  const requests: InternalAxiosRequestConfig[] = []
  const cancellationPreviewTokens: (string | null)[] = []
  let finishActorRead: () => void = () => undefined
  const actorRead = new Promise<void>((resolve) => {
    finishActorRead = resolve
  })
  api.defaults.adapter = async (config) => {
    requests.push(config)
    const preview = config.headers.get('New-Api-User-Preview')
    let data: unknown
    if (config.url === '/api/user/') {
      if (preview) {
        throw new AxiosError(
          'User preview is read-only',
          'ERR_BAD_REQUEST',
          config,
          undefined,
          {
            data: { success: false, message: 'User preview is read-only' },
            status: 403,
            statusText: 'Forbidden',
            headers: {},
            config,
          }
        )
      }
      data = { success: true, data: { items: [previewTarget], total: 1 } }
    } else if (config.url === '/api/user/2/preview') {
      data = {
        success: true,
        data: {
          token: 'target-preview-token',
          expires_at: 2_000_000_000,
          target: previewTarget,
        },
      }
    } else if (config.url === '/api/user/self') {
      if (!preview) await actorRead
      data = { success: true, data: preview ? previewTarget : actor }
    } else if (config.url === '/api/status') {
      data = { success: true, data: { checkin_enabled: true } }
    } else {
      throw new Error(`Unexpected preview request: ${config.url}`)
    }
    return { data, status: 200, statusText: 'OK', headers: {}, config }
  }
  const errorToast = vi.spyOn(toast, 'error')
  const client = createAppQueryClient()
  clients.push(client)
  client.setQueryData(['user-self'], actor)

  let finishProfileLoad: () => void = () => undefined
  const profileLoad = new Promise<void>((resolve) => {
    finishProfileLoad = resolve
  })
  const root = createRootRoute({ component: Outlet })
  const auth = createRoute({ getParentRoute: () => root, id: '_authenticated' })
  const users = createRoute({
    getParentRoute: () => auth,
    path: 'users/',
    component: PreviewUsersPage,
  })
  const profile = createRoute({
    getParentRoute: () => auth,
    path: 'profile/',
    beforeLoad: () => profileLoad,
    component: Profile,
  })
  const router = createRouter({
    routeTree: root.addChildren([auth.addChildren([users, profile])]),
    history: createMemoryHistory({ initialEntries: ['/users/'] }),
    defaultPendingMs: 0,
    defaultPendingMinMs: 0,
  })
  await router.load()
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  await screen.findByText('preview-target')
  const pendingActor = client
    .fetchQuery({
      queryKey: ['actor-pending-self'],
      queryFn: async ({ signal }) => {
        signal.addEventListener(
          'abort',
          () => {
            cancellationPreviewTokens.push(useUserPreviewStore.getState().token)
          },
          { once: true }
        )
        return (await api.get('/api/user/self', { signal })).data.data
      },
    })
    .catch((error: unknown) => error)
  await waitFor(() =>
    expect(requests.some((request) => request.url === '/api/user/self')).toBe(
      true
    )
  )

  try {
    await userEvent.click(screen.getByRole('button', { name: 'View as user' }))
    await waitFor(() => expect(router.state.isLoading).toBe(true))
    expect(screen.getByText('preview-target')).toBeVisible()
    // The app shell refreshes public status after the cache is cleared;
    // applying its currency config re-renders the still-mounted user table.
    await act(async () => {
      await client.fetchQuery(statusQueryOptions)
    })
    await act(async () => {
      finishProfileLoad()
    })
    expect(
      await screen.findByRole('heading', { name: 'Preview Target' })
    ).toBeVisible()
    expect(
      requests.filter((request) => request.url === '/api/user/')
    ).toHaveLength(1)
    expect(errorToast).not.toHaveBeenCalled()
    expect(screen.queryByText('Settings & Preferences')).not.toBeInTheDocument()
    expect(screen.queryByText('Daily Check-in')).not.toBeInTheDocument()
    expect(
      requests
        .filter((request) => request.url === '/api/user/self')
        .map((request) => request.headers.get('New-Api-User-Preview'))
    ).toEqual([undefined, 'target-preview-token'])
    expect(useAuthStore.getState().auth.user).toBe(actor)
    expect(useAuthStore.getState().auth.session).toBe(session)
    expect(useAuthStore.getState().auth.accessToken).toBe(
      'preview-transition-actor-access'
    )
    expect(isCancelledError(await pendingActor)).toBe(true)
    expect(cancellationPreviewTokens).toEqual([null])
  } finally {
    await act(async () => {
      finishProfileLoad()
      finishActorRead()
    })
    await pendingActor
  }
})
