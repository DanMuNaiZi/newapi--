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

import { QueryClientProvider, type QueryClient } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import { createElement } from 'react'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'

import { Lotteries } from '@/features/lottery'
import { LotteryDashboardPanel } from '@/features/lottery/components/lottery-dashboard-panel'
import type { LotteryPlan, LotterySelfResult } from '@/features/lottery/types'
import { api } from '@/lib/api'
import { createAppQueryClient } from '@/lib/query-client'
import { useAuthStore, type AuthUser } from '@/stores/auth-store'
import { useUserPreviewStore } from '@/stores/user-preview-store'

import { Route } from './index'

test('parses lottery plan IDs from URL search strings', () => {
  const validateSearch = Route.options.validateSearch as unknown as {
    parse: (input: unknown) => { plan?: number }
  }

  assert.deepEqual(validateSearch.parse({ plan: '20' }), { plan: 20 })
})

const originalAdapter = api.defaults.adapter
const clients: QueryClient[] = []
const requests: Array<{
  method: string
  url: string
  data: unknown
  preview: unknown
}> = []
let includePlans: boolean
let unread: boolean
let participantWeight: number

function planForUser(): LotteryPlan {
  const userId =
    useUserPreviewStore.getState().target?.id ??
    useAuthStore.getState().auth.user?.id ??
    0
  return {
    id: 41,
    title: `Lottery for user ${userId}`,
    icon: '',
    description: 'Public lottery description',
    status: 'open',
    eligibility_mode: 'all',
    max_participants: 100,
    registration_start_time: 1_700_000_000,
    draw_time: 1_800_000_000,
    participant_count: 2,
    winner_count: 1,
    joined: false,
  }
}

const claimable: LotterySelfResult = {
  id: 7,
  plan_id: 41,
  user_id: 10,
  prize_id: 1,
  reward_type: 'quota',
  quota: 500,
  subscription_plan_id: 0,
  fulfillment_mode: 'self_claim',
  fulfillment_status: 'pending',
  claim_status: 'pending',
  claimable: true,
  claim_expires_at: 2_000_000_000,
  claimed_at: 0,
  redemption_code: '',
  created_at: 1_700_000_000,
}

beforeEach(() => {
  vi.stubGlobal('scrollTo', () => undefined)
  useAuthStore.getState().auth.setUser({ id: 10, username: 'reader', role: 1 })
  useUserPreviewStore.getState().clear()
  requests.length = 0
  includePlans = true
  unread = true
  participantWeight = 9
  api.defaults.adapter = async (config) => {
    const url = config.url ?? ''
    const method = config.method ?? 'get'
    const data =
      typeof config.data === 'string' ? JSON.parse(config.data) : config.data
    requests.push({
      method,
      url,
      data,
      preview: config.headers.get('New-Api-User-Preview'),
    })
    let payload: unknown
    if (method === 'post') {
      if (url.endsWith('/read')) unread = false
      payload = undefined
    } else if (method === 'put') {
      participantWeight = data.weight
      payload = undefined
    } else if (url === '/api/lottery/self') {
      payload = includePlans
        ? [
            planForUser(),
            {
              ...planForUser(),
              id: 42,
              title: 'Already joined lottery',
              joined: true,
            },
          ]
        : []
    } else if (url.endsWith('/results/self/pending')) {
      payload = [claimable]
    } else if (url.endsWith('/results/self/page')) {
      payload = { items: [claimable], has_more: false }
    } else if (url.endsWith('/notifications/self/page')) {
      const cursor = config.params?.cursor
      payload = {
        items: unread
          ? [
              {
                id: cursor ? 102 : 101,
                user_id: 10,
                plan_id: 41,
                type: 'lottery_result',
                content: '',
                read_at: 0,
                created_at: 1_700_000_000,
              },
            ]
          : [],
        has_more: unread && !cursor,
        next_cursor: unread && !cursor ? 'next-notification' : undefined,
        unread_total: unread ? 2 : 0,
      }
    } else if (url === '/api/lottery/admin/plans/41') {
      payload = { ...planForUser(), title: 'Administrator lottery details' }
    } else if (url.endsWith('/admin/plans/41/participants')) {
      payload = [
        {
          id: 8,
          plan_id: 41,
          user_id: 17,
          username: 'private-user',
          display_name: 'Administrator participant',
          user_group: 'default',
          weight: participantWeight,
          preset_prize_id: 0,
          status: 'joined',
        },
      ]
    } else if (url.includes('/admin/')) {
      payload = []
    } else if (url === '/api/lottery/plans/41') {
      payload = planForUser()
    } else if (url.endsWith('/participants/page')) {
      payload = {
        items: [
          {
            id: 8,
            username: 'target-user',
            display_name: 'Public participant',
            joined_at: 1_700_000_000,
            is_self: false,
          },
        ],
        has_more: false,
      }
    } else if (url.endsWith('/results/page')) {
      payload = { items: [], has_more: false }
    } else {
      throw new Error(`Unexpected request: ${method} ${url}`)
    }
    return {
      config,
      status: 200,
      statusText: 'OK',
      headers: {},
      data: { success: true, message: '', data: payload },
    }
  }
})

afterEach(() => {
  api.defaults.adapter = originalAdapter
  for (const client of clients.splice(0)) client.clear()
  useUserPreviewStore.getState().clear()
  useAuthStore.getState().auth.reset()
  vi.unstubAllGlobals()
})

async function renderLotteryRoute(path: string) {
  const client = createAppQueryClient()
  client.setDefaultOptions({ queries: { retry: false, staleTime: Infinity } })
  clients.push(client)
  const root = createRootRoute()
  const auth = createRoute({ getParentRoute: () => root, id: '_authenticated' })
  const lottery = createRoute({
    getParentRoute: () => auth,
    path: 'lotteries/',
    component: Lotteries,
    validateSearch: (search: Record<string, unknown>) => {
      const schema = Route.options.validateSearch as unknown as {
        parse: (input: unknown) => { plan?: number }
      }
      return schema.parse(search)
    },
  })
  const dashboard = createRoute({
    getParentRoute: () => auth,
    path: 'dashboard',
    component: LotteryDashboardPanel,
  })
  const router = createRouter({
    routeTree: root.addChildren([auth.addChildren([lottery, dashboard])]),
    history: createMemoryHistory({ initialEntries: [path] }),
    scrollRestoration: false,
  })
  await router.load()
  render(
    createElement(
      QueryClientProvider,
      { client },
      createElement(RouterProvider, { router })
    )
  )
  return { client, router }
}

test('dashboard notification summary and paginated notifications coexist across navigation and marking read', async () => {
  const { client, router } = await renderLotteryRoute('/dashboard')
  await screen.findByText('2 lottery result notifications')
  fireEvent.click(screen.getByRole('button', { name: 'View all' }))
  await screen.findByText('Lottery notifications')
  fireEvent.click(screen.getByRole('button', { name: 'Load more' }))
  await waitFor(() =>
    expect(screen.getAllByText('Lottery result available')).toHaveLength(2)
  )
  const notifications = client
    .getQueryCache()
    .findAll({ queryKey: ['lottery', 'notifications'] })
    .map((query) => query.state.data)
  expect(notifications).toEqual(
    expect.arrayContaining([
      expect.objectContaining({
        data: expect.objectContaining({ unread_total: 2 }),
      }),
      expect.objectContaining({
        pages: expect.any(Array),
        pageParams: expect.any(Array),
      }),
    ])
  )
  fireEvent.click(
    screen.getByRole('button', { name: 'Mark displayed as read' })
  )
  await waitFor(() =>
    expect(screen.queryByText('Lottery notifications')).not.toBeInTheDocument()
  )
  expect(requests.filter((request) => request.method === 'post')).toEqual([
    expect.objectContaining({
      url: '/api/lottery/notifications/self/read',
      data: { ids: [101, 102] },
    }),
  ])
  await act(async () => {
    await router.navigate({ to: '/dashboard' })
  })
  await screen.findByRole('heading', { name: 'Lotteries' })
  await waitFor(() =>
    expect(
      screen.queryByText('2 lottery result notifications')
    ).not.toBeInTheDocument()
  )
})

test('dashboard caches follow the current account and preview target without reusing another user data', async () => {
  await renderLotteryRoute('/dashboard')
  await screen.findByText('Lottery for user 10')
  act(() =>
    useAuthStore
      .getState()
      .auth.setUser({ id: 11, username: 'root', role: 100 })
  )
  await screen.findByText('Lottery for user 11')
  act(() =>
    useUserPreviewStore
      .getState()
      .start('preview-17', 2_000_000_000, { id: 17, username: 'target' })
  )
  await screen.findByText('Lottery for user 17')
  expect(
    screen.queryByRole('button', { name: 'Claim lottery reward' })
  ).not.toBeInTheDocument()
  expect(screen.getByText('User preview is read-only')).toBeInTheDocument()
  act(() =>
    useUserPreviewStore
      .getState()
      .start('preview-18', 2_000_000_000, { id: 18, username: 'other-target' })
  )
  await screen.findByText('Lottery for user 18')
  expect(
    requests.some(
      (request) =>
        request.url === '/api/lottery/self' && request.preview === 'preview-18'
    )
  ).toBe(true)
})

test.each([
  [{ id: 10, username: 'root', role: 100 }, true, true],
  [
    {
      id: 10,
      username: 'reader',
      role: 10,
      permissions: {
        admin_permissions: { lottery: { read: true, operate: false } },
      },
    },
    true,
    false,
  ],
  [
    {
      id: 10,
      username: 'operator-without-read',
      role: 10,
      permissions: {
        admin_permissions: { lottery: { read: false, operate: true } },
      },
    },
    false,
    false,
  ],
  [{ id: 10, username: 'member', role: 1 }, false, false],
] as const)(
  'deep-link lottery details respect LotteryRead and LotteryOperate for %j',
  async (user, adminDetails, canOperate) => {
    useAuthStore.getState().auth.setUser(user as AuthUser)
    includePlans = false
    await renderLotteryRoute('/lotteries/?plan=41')
    const dialog = await screen.findByRole('dialog', {
      name: adminDetails
        ? 'Administrator lottery details'
        : 'Lottery for user 10',
    })
    if (adminDetails) {
      expect(
        requests.some(
          (request) => request.url === '/api/lottery/admin/plans/41'
        )
      ).toBe(true)
      fireEvent.click(within(dialog).getByRole('tab', { name: 'Participants' }))
      await within(dialog).findByText('Administrator participant')
      expect(
        within(dialog).queryByRole('spinbutton', { name: 'Lottery weight' }) !==
          null
      ).toBe(canOperate)
      expect(
        within(dialog).queryByRole('combobox', { name: 'Preset prize' }) !==
          null
      ).toBe(canOperate)
    } else {
      expect(requests.some((request) => request.url.includes('/admin/'))).toBe(
        false
      )
      expect(
        within(dialog).queryByRole('tab', { name: 'Prizes' })
      ).not.toBeInTheDocument()
    }
  }
)

test('root preview uses user details and suppresses join, leave, claim, and mark-read actions', async () => {
  useAuthStore.getState().auth.setUser({ id: 10, username: 'root', role: 100 })
  useUserPreviewStore
    .getState()
    .start('preview-17', 2_000_000_000, { id: 17, username: 'target' })
  const { router } = await renderLotteryRoute('/lotteries/?plan=41')
  const dialog = await screen.findByRole('dialog', {
    name: 'Lottery for user 17',
  })
  fireEvent.click(within(dialog).getByRole('tab', { name: /Participants/ }))
  await within(dialog).findByText('Public participant')
  expect(requests.some((request) => request.url.includes('/admin/'))).toBe(
    false
  )
  expect(within(dialog).queryByRole('spinbutton')).not.toBeInTheDocument()
  fireEvent.click(within(dialog).getByRole('button', { name: 'Close' }))
  await waitFor(() =>
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  )
  await screen.findByText('Lottery notifications')
  for (const name of [
    'Join lottery',
    'Leave lottery',
    'Claim lottery reward',
    'Mark displayed as read',
  ]) {
    expect(screen.queryByRole('button', { name })).not.toBeInTheDocument()
  }
  await act(async () => {
    await router.navigate({ to: '/dashboard' })
  })
  await screen.findByText('User preview is read-only')
  expect(
    screen.queryByRole('button', { name: 'Claim lottery reward' })
  ).not.toBeInTheDocument()
  expect(requests.filter((request) => request.method !== 'get')).toHaveLength(0)
  expect(useAuthStore.getState().auth.user?.role).toBe(100)
})

test('LotteryOperate permits participant edits from the administrator detail view', async () => {
  useAuthStore.getState().auth.setUser({
    id: 10,
    username: 'operator',
    role: 10,
    permissions: {
      admin_permissions: { lottery: { read: true, operate: true } },
    },
  })
  await renderLotteryRoute('/lotteries/?plan=41')
  const dialog = await screen.findByRole('dialog', {
    name: 'Administrator lottery details',
  })
  fireEvent.click(within(dialog).getByRole('tab', { name: 'Participants' }))
  const weight = await within(dialog).findByRole('spinbutton', {
    name: 'Lottery weight',
  })
  fireEvent.change(weight, { target: { value: '3' } })
  fireEvent.blur(weight)
  await waitFor(() =>
    expect(requests.filter((request) => request.method === 'put')).toEqual([
      expect.objectContaining({
        url: '/api/lottery/admin/plans/41/participants',
        data: { user_id: 17, weight: 3 },
      }),
    ])
  )
})
