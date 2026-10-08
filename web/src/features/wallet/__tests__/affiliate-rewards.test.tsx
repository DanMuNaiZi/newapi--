/* Copyright (C) 2023-2026 QuantumNous */
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'

import type { ReferralCampaignSelfView } from '@/features/referral-campaigns/types'
import { api } from '@/lib/api'
import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
} from '@/stores/system-config-store'

import { AffiliateRewardsCard } from '../components/affiliate-rewards-card'
import type { UserWalletData } from '../types'

const user: UserWalletData = {
  id: 2,
  username: 'target',
  group: 'default',
  quota: 0,
  used_quota: 0,
  request_count: 0,
  aff_count: 2,
  aff_quota: 100_000,
  aff_history_quota: 100_000,
}
const view: ReferralCampaignSelfView = {
  campaign: {
    id: 3,
    title: 'Referral Spring',
    description: 'Try the gateway together.',
    enabled: true,
    start_time: 1_700_000_000,
    end_time: 2_000_000_000,
    activation_window_seconds: 3600,
    max_rewards_per_inviter: 0,
    total_reward_limit: 0,
    registration_count: 2,
    activated_count: 2,
    rewarded_count: 1,
    failed_reward_count: 0,
    reward: { type: 'quota', quota: 500_000 },
    invitee_reward: { type: 'quota', quota: 250_000 },
  },
  events: [
    {
      id: 5,
      campaign_id: 3,
      invitee_username: 'waiting-user',
      status: 'pending_review',
      registered_at: 1,
      activation_deadline: 2,
      activated_at: 1,
    },
    {
      id: 6,
      campaign_id: 3,
      invitee_username: 'rewarded-user',
      status: 'rewarded',
      registered_at: 1,
      activation_deadline: 2,
      activated_at: 1,
    },
  ],
}

beforeEach(() => {
  localStorage.clear()
  useSystemConfigStore
    .getState()
    .setConfig({ currency: { ...DEFAULT_CURRENCY_CONFIG } })
})
afterEach(() => vi.restoreAllMocks())

test('an active campaign explains both rewards, consumption and manual review', async () => {
  const get = vi
    .spyOn(api, 'get')
    .mockResolvedValue({ data: { success: true, data: view } })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <AffiliateRewardsCard
        user={user}
        affiliateLink='https://example.com/register?aff=target'
        onTransfer={vi.fn()}
      />
    </QueryClientProvider>
  )

  expect(await screen.findByText('Referral Spring')).toBeVisible()
  expect(
    screen.getByText('Inviter reward: $1 · New user reward: $0.5')
  ).toBeVisible()
  expect(
    screen.getAllByText(
      'Rewards are issued only after actual consumption and manual approval.'
    ).length
  ).toBeGreaterThan(0)
  expect(screen.getByText('Registered: 2')).toBeVisible()
  expect(screen.getByText('Activated: 2')).toBeVisible()
  expect(screen.getByText('Rewarded: 1')).toBeVisible()
  expect(screen.getByText('waiting-user: Pending manual review')).toBeVisible()
  expect(screen.getByText('rewarded-user: Rewarded')).toBeVisible()
  await waitFor(() =>
    expect(get).toHaveBeenCalledWith('/api/referral-campaign/self')
  )
  client.clear()
})

test('subscription reward snapshots display the promised plans for both recipients', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: {
        ...view,
        campaign: {
          ...view.campaign,
          reward: {
            type: 'subscription',
            subscription_plan_title: 'Inviter Plan',
          },
          invitee_reward: {
            type: 'subscription',
            subscription_plan_title: 'Welcome Plan',
          },
        },
      },
    },
  })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <AffiliateRewardsCard
        user={user}
        affiliateLink='https://example.com/register?aff=target'
        onTransfer={vi.fn()}
      />
    </QueryClientProvider>
  )

  expect(
    await screen.findByText(
      'Inviter reward: Inviter Plan · New user reward: Welcome Plan'
    )
  ).toBeVisible()
  client.clear()
})

test('no active campaign preserves the standard referral link and earned rewards', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: { campaign: null, events: [] } },
  })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <AffiliateRewardsCard
        user={user}
        affiliateLink='https://example.com/register?aff=target'
        onTransfer={vi.fn()}
      />
    </QueryClientProvider>
  )

  await waitFor(() => expect(client.isFetching()).toBe(0))
  expect(
    screen.getByRole('button', { name: 'Copy referral link' })
  ).toBeVisible()
  expect(
    screen.getByRole('button', { name: 'Transfer to Balance' })
  ).toBeEnabled()
  expect(
    screen.getByText(
      'Earn rewards when users join through your referral link. Transfer accumulated rewards to your balance anytime.'
    )
  ).toBeVisible()
  expect(screen.queryByText(/Inviter reward:/)).not.toBeInTheDocument()
  client.clear()
})

test('a campaign read failure can be retried without hiding the ordinary referral link', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: false, message: 'Campaign unavailable' },
  })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <AffiliateRewardsCard
        user={user}
        affiliateLink='https://example.com/register?aff=target'
        onTransfer={vi.fn()}
      />
    </QueryClientProvider>
  )

  expect(await screen.findByText('Campaign unavailable')).toBeVisible()
  expect(screen.getByRole('button', { name: 'Retry' })).toBeVisible()
  expect(
    screen.getByRole('button', { name: 'Copy referral link' })
  ).toBeVisible()
  vi.mocked(api.get).mockResolvedValue({ data: { success: true, data: view } })
  fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
  expect(await screen.findByText('Referral Spring')).toBeVisible()
  expect(screen.queryByText('Campaign unavailable')).not.toBeInTheDocument()
  client.clear()
})
