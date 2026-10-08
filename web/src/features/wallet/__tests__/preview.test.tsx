/* Copyright (C) 2023-2026 QuantumNous */
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'
import { useUserPreviewStore } from '@/stores/user-preview-store'

import { Wallet } from '..'
import type { UserWalletData } from '../types'

const target: UserWalletData = {
  id: 2,
  username: 'target',
  group: 'default',
  quota: 1_000_000,
  used_quota: 500_000,
  request_count: 3,
  aff_count: 1,
  aff_quota: 100_000,
  aff_history_quota: 100_000,
}
const plan = {
  id: 7,
  title: 'Starter Plan',
  price_amount: 10,
  currency: 'USD',
  duration_unit: 'month',
  duration_value: 1,
  quota_reset_period: 'monthly',
  enabled: true,
  sort_order: 1,
  total_amount: 500_000,
  max_purchase_per_user: 0,
}
const subscription = {
  id: 12,
  plan_id: 7,
  user_id: 2,
  status: 'active',
  start_time: 1_700_000_000,
  end_time: 2_000_000_000,
  amount_total: 500_000,
  amount_used: 100_000,
}

beforeEach(() => {
  localStorage.clear()
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
  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    let data: unknown = []
    if (url === '/api/user/self') data = target
    if (url === '/api/status') data = { price: 1 }
    if (url === '/api/user/aff') data = 'target-invite'
    if (url === '/api/referral-campaign/self') {
      data = { campaign: null, events: [] }
    }
    if (url === '/api/subscription/plans') data = [{ plan }]
    if (url === '/api/subscription/self') {
      data = {
        billing_preference: 'wallet_first',
        subscriptions: [{ subscription }],
        all_subscriptions: [{ subscription }],
      }
    }
    if (url === '/api/user/topup/info') {
      data = {
        enable_online_topup: true,
        min_topup: 1,
        pay_methods: [{ name: 'Alipay', type: 'alipay' }],
      }
    }
    if (url === '/api/user/manual-wechat/info') data = { enabled: true }
    return { data: { success: true, data } }
  })
  vi.spyOn(api, 'post').mockResolvedValue({ data: { success: true, data: 1 } })
  vi.spyOn(api, 'put').mockResolvedValue({ data: { success: true } })
  vi.spyOn(api, 'delete').mockResolvedValue({ data: { success: true } })
})

afterEach(() => {
  vi.restoreAllMocks()
  useUserPreviewStore.getState().clear()
  useAuthStore.getState().auth.reset()
})

test('preview reads balances and subscriptions without payment information or write controls', async () => {
  useUserPreviewStore.getState().start('preview-token', 2_000_000_000, target)
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const view = render(
    <QueryClientProvider client={client}>
      <Wallet initialShowHistory />
    </QueryClientProvider>
  )

  expect(await screen.findByText('Starter Plan')).toBeVisible()
  expect(await screen.findByDisplayValue(/target-invite/)).toBeVisible()
  expect(screen.getByText('Current Balance')).toBeVisible()
  expect(screen.getByText('Starter Plan · Subscription #12')).toBeVisible()
  expect(screen.queryByText('Add Funds')).not.toBeInTheDocument()
  expect(
    screen.queryByRole('button', { name: 'Redeem' })
  ).not.toBeInTheDocument()
  expect(
    screen.queryByRole('button', { name: 'Transfer to Balance' })
  ).not.toBeInTheDocument()
  expect(
    screen.queryByRole('button', { name: 'Save order' })
  ).not.toBeInTheDocument()
  expect(
    screen.queryByRole('button', { name: 'Restore default' })
  ).not.toBeInTheDocument()
  expect(
    screen.queryByRole('button', { name: 'Subscribe Now' })
  ).not.toBeInTheDocument()
  expect(screen.getByRole('combobox')).toBeDisabled()
  expect(view.container.querySelector('[draggable="true"]')).toBeNull()
  const requestedPaths = vi.mocked(api.get).mock.calls.map(([url]) => url)
  expect(requestedPaths).not.toContain('/api/user/topup/info')
  expect(requestedPaths).not.toContain('/api/user/manual-wechat/info')
  expect(requestedPaths.some((url) => url.includes('/topup/self'))).toBe(false)
  expect(api.post).not.toHaveBeenCalled()
  expect(api.put).not.toHaveBeenCalled()
  expect(api.delete).not.toHaveBeenCalled()
  expect(useAuthStore.getState().auth.accessToken).toBe('actor-access')
  expect(useAuthStore.getState().auth.session?.sid).toBe('actor-session')
  client.clear()
})

test('the ordinary wallet retains payment information and subscription write controls', async () => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const view = render(
    <QueryClientProvider client={client}>
      <Wallet />
    </QueryClientProvider>
  )

  expect(await screen.findByText('Starter Plan')).toBeVisible()
  expect(await screen.findByText('Add Funds')).toBeVisible()
  expect(screen.getByRole('button', { name: 'Subscribe Now' })).toBeEnabled()
  expect(screen.getByRole('button', { name: 'Restore default' })).toBeEnabled()
  expect(
    screen.getByRole('button', { name: 'Transfer to Balance' })
  ).toBeEnabled()
  expect(screen.getByRole('combobox')).toBeEnabled()
  expect(view.container.querySelector('[draggable="true"]')).not.toBeNull()
  await waitFor(() =>
    expect(api.get).toHaveBeenCalledWith('/api/user/topup/info')
  )
  expect(api.get).toHaveBeenCalledWith('/api/user/manual-wechat/info')
  fireEvent.click(screen.getByRole('button', { name: 'Restore default' }))
  await waitFor(() =>
    expect(api.delete).toHaveBeenCalledWith(
      '/api/subscription/self/consume-priority'
    )
  )
  client.clear()
})
