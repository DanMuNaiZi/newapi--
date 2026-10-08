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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
  RouterProvider,
} from '@tanstack/react-router'
import { act, cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, expect, test, vi } from 'vitest'

import { UserPreviewBanner } from '@/components/user-preview-banner'
import { useAuthStore } from '@/stores/auth-store'
import { useUserPreviewStore } from '@/stores/user-preview-store'

afterEach(() => {
  cleanup()
  vi.useRealTimers()
  vi.unstubAllGlobals()
  useUserPreviewStore.getState().clear()
  useAuthStore.getState().auth.reset()
})

async function renderPreview(expiresInSeconds = 900, beforeMount?: () => void) {
  vi.stubGlobal('scrollTo', () => undefined)
  const client = new QueryClient()
  client.setQueryData(['target-balance'], { quota: 123 })
  useAuthStore.getState().auth.setBundle({
    access_token: 'actor-access',
    token_type: 'Bearer',
    access_expires_at: 2_000_000_000,
    user: { id: 1, username: 'root', role: 100 },
    session: {
      sid: 'actor-session',
      current: true,
      login_method: 'password',
      ip: '',
      user_agent: '',
      created_at: 1,
      last_active_at: 1,
      expires_at: 2_000_000_000,
    },
  })
  const root = createRootRoute({
    component: () => (
      <>
        <UserPreviewBanner />
        <Outlet />
      </>
    ),
  })
  const profile = createRoute({
    getParentRoute: () => root,
    path: '/profile',
    component: () => <p>Target profile</p>,
  })
  const dashboard = createRoute({
    getParentRoute: () => root,
    path: '/dashboard',
    component: () => <p>Administrator dashboard</p>,
  })
  const router = createRouter({
    routeTree: root.addChildren([profile, dashboard]),
    history: createMemoryHistory({ initialEntries: ['/profile'] }),
  })
  await router.load()
  beforeMount?.()
  useUserPreviewStore
    .getState()
    .start('preview-token', Math.floor(Date.now() / 1000) + expiresInSeconds, {
      id: 2,
      username: 'target',
    })
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  return { client, router }
}

test('expiry clears target data and restores the administrator route without changing the session', async () => {
  const { client, router } = await renderPreview(1, () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-10-08T00:00:00.000Z'))
  })
  await act(async () => {
    await vi.advanceTimersByTimeAsync(1100)
  })
  expect(useUserPreviewStore.getState().token).toBeNull()
  expect(client.getQueryData(['target-balance'])).toBeUndefined()
  expect(router.state.location.pathname).toBe('/dashboard')
  expect(useAuthStore.getState().auth.user?.id).toBe(1)
  expect(useAuthStore.getState().auth.accessToken).toBe('actor-access')
})

test('manual exit clears cached target data', async () => {
  const { client, router } = await renderPreview()
  fireEvent.click(screen.getByRole('button', { name: 'Exit preview' }))
  await act(async () => {
    await router.load()
  })
  expect(client.getQueryData(['target-balance'])).toBeUndefined()
  expect(router.state.location.pathname).toBe('/dashboard')
})

test('a server-rejected preview restores the administrator view', async () => {
  const { client, router } = await renderPreview()
  act(() => useUserPreviewStore.getState().clear())
  await act(async () => {
    await router.load()
  })
  expect(client.getQueryData(['target-balance'])).toBeUndefined()
  expect(router.state.location.pathname).toBe('/dashboard')
  expect(useAuthStore.getState().auth.accessToken).toBe('actor-access')
})
