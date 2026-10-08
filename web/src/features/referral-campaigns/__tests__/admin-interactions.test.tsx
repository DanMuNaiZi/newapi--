/* Copyright (C) 2023-2026 QuantumNous */
import { QueryClientProvider, type QueryClient } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Toaster, toast } from 'sonner'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import { api } from '@/lib/api'
import { createAppQueryClient } from '@/lib/query-client'
import { useAuthStore } from '@/stores/auth-store'

import { ReferralCampaignAdmin } from '../admin'
import { ReferralCampaigns } from '../index'
import type { ReferralCampaign, ReferralCampaignEvent } from '../types'

const campaign: ReferralCampaign = {
  id: 4,
  title: 'Autumn campaign',
  description: 'Reward useful referrals',
  enabled: true,
  start_time: 1_800_000_000,
  end_time: 1_800_604_800,
  activation_window_seconds: 86400,
  max_rewards_per_inviter: 0,
  total_reward_limit: 0,
  registration_count: 1,
  activated_count: 1,
  rewarded_count: 0,
  failed_reward_count: 0,
  reward: { type: 'quota', quota: 500_000, unit: 'usd', amount: '1' },
}
const event: ReferralCampaignEvent = {
  id: 7,
  campaign_id: 4,
  inviter_user_id: 1,
  invitee_user_id: 2,
  inviter_username: 'inviter',
  invitee_username: 'newcomer',
  status: 'pending_review',
  registered_at: 1_800_000_000,
  activation_deadline: 1_800_086_400,
  activated_at: 1_800_000_100,
  qualified_quota: 25,
}
let originalAdapter: typeof api.defaults.adapter
let client: QueryClient
let reads: string[]
let writes: Array<{ url: string; data: unknown }>
let listResponse: unknown
let saveResponse: unknown
let finishReview: ((value: unknown) => void) | undefined
let finishRetry: ((value: unknown) => void) | undefined
let events: ReferralCampaignEvent[]

function setPermissions(permissions: Record<string, boolean>) {
  useAuthStore.getState().auth.setUser({
    id: 1,
    username: 'admin',
    role: 10,
    permissions: { admin_permissions: { referral_campaign: permissions } },
  })
}

function renderPage(page = <ReferralCampaignAdmin />) {
  render(
    <QueryClientProvider client={client}>
      {page}
      <Toaster />
    </QueryClientProvider>
  )
}

beforeEach(() => {
  setPermissions({ read: true, write: true, operate: true })
  reads = []
  writes = []
  finishReview = undefined
  finishRetry = undefined
  events = [event]
  listResponse = { success: true, data: [campaign] }
  saveResponse = { success: false, message: 'Campaign dates overlap' }
  client = createAppQueryClient()
  client.setDefaultOptions({
    queries: { retry: false },
    mutations: { retry: false },
  })
  originalAdapter = api.defaults.adapter
  api.defaults.adapter = async (config) => {
    const url = config.url ?? ''
    let data: unknown
    if (config.method === 'get') {
      reads.push(url)
      if (url === '/api/status') {
        data = {
          success: true,
          data: { quota_per_unit: 500_000, usd_exchange_rate: 7 },
        }
      } else if (url.endsWith('/events')) {
        data = {
          success: true,
          data: { items: events, total: events.length, page: 1, page_size: 50 },
        }
      } else if (url.endsWith('/self')) data = listResponse
      else if (url.endsWith('/review')) {
        data = { success: false, message: 'Review evidence is unavailable' }
      } else data = listResponse
    } else {
      writes.push({ url, data: JSON.parse(config.data || '{}') })
      if (url.endsWith('/review')) {
        data = await new Promise((resolve) => {
          finishReview = resolve
        })
      } else if (url.endsWith('/retry-reward')) {
        data = await new Promise((resolve) => {
          finishRetry = resolve
        })
      } else data = saveResponse
    }
    return { data, status: 200, statusText: 'OK', headers: {}, config }
  }
})

afterEach(() => {
  api.defaults.adapter = originalAdapter
  client.clear()
  useAuthStore.getState().auth.reset()
  toast.dismiss()
})

describe('referral administration', () => {
  it('does not request campaign data without read access', async () => {
    setPermissions({})
    renderPage()
    expect(
      await screen.findByText('You do not have permission to view this page')
    ).toBeVisible()
    expect(reads).toEqual([])
  })

  it('opens the campaign form from its create action and retains a failed save', async () => {
    const user = userEvent.setup()
    renderPage()
    await screen.findByText(campaign.title)
    await user.click(screen.getByRole('button', { name: 'Create campaign' }))
    const dialog = await screen.findByRole('dialog', {
      name: 'Create campaign',
    })
    await user.type(
      within(dialog).getByRole('textbox', { name: 'Campaign title' }),
      'New campaign'
    )
    await user.click(
      within(dialog).getByRole('button', { name: 'Create campaign' })
    )
    expect(await screen.findByText('Campaign dates overlap')).toBeVisible()
    expect(
      within(dialog).getByRole('textbox', { name: 'Campaign title' })
    ).toHaveValue('New campaign')
    saveResponse = { success: true, data: campaign }
    await user.click(
      within(dialog).getByRole('button', { name: 'Create campaign' })
    )
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    )
  })

  it('preserves the existing reward snapshot when editing campaign metadata', async () => {
    const user = userEvent.setup()
    renderPage()
    await screen.findByText(campaign.title)
    await user.click(screen.getByRole('button', { name: 'Edit' }))
    const dialog = await screen.findByRole('dialog', { name: 'Edit campaign' })
    expect(
      within(dialog).getByRole('textbox', { name: 'Inviter reward (USD)' })
    ).toBeDisabled()
    await user.type(
      within(dialog).getByRole('textbox', { name: 'Campaign title' }),
      ' revised'
    )
    saveResponse = { success: true, data: campaign }
    await user.click(
      within(dialog).getByRole('button', { name: 'Save changes' })
    )
    await waitFor(() => expect(writes).toHaveLength(1))
    expect(writes[0].data).toMatchObject({
      title: 'Autumn campaign revised',
      preserve_reward: true,
    })
    expect(writes[0].data).not.toHaveProperty('inviter_reward_usd')
  })

  it('prevents submission of an invalid campaign title and exposes the field error', async () => {
    const user = userEvent.setup()
    renderPage()
    await user.click(screen.getByRole('button', { name: 'Create campaign' }))
    const dialog = await screen.findByRole('dialog', {
      name: 'Create campaign',
    })
    await user.click(
      within(dialog).getByRole('button', { name: 'Create campaign' })
    )
    expect(
      await within(dialog).findByText('Campaign title is required')
    ).toBeVisible()
    expect(
      within(dialog).getByRole('textbox', { name: 'Campaign title' })
    ).toHaveAttribute('aria-invalid', 'true')
    expect(writes).toEqual([])
  })

  it('allows read-only users to inspect review details while hiding edit and decision controls', async () => {
    const user = userEvent.setup()
    setPermissions({ read: true })
    renderPage()
    await user.click(
      await screen.findByRole('button', { name: /Autumn campaign/ })
    )
    await user.click(
      await screen.findByRole('button', { name: 'Review details' })
    )
    const dialog = await screen.findByRole('dialog', {
      name: 'Referral review',
    })
    expect(within(dialog).getByRole('button', { name: 'Retry' })).toBeVisible()
    expect(
      screen.queryByRole('button', { name: 'Create campaign' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Edit' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Approve' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Reject' })
    ).not.toBeInTheDocument()
  })

  it('prevents concurrent review decisions and keeps the note on API failure', async () => {
    const user = userEvent.setup()
    renderPage()
    await user.click(
      await screen.findByRole('button', { name: /Autumn campaign/ })
    )
    await user.type(
      await screen.findByRole('textbox', { name: 'Review note' }),
      'Checked consumption'
    )
    await user.click(screen.getByRole('button', { name: 'Reject' }))
    await waitFor(() => expect(finishReview).toBeDefined())
    expect(screen.getByRole('button', { name: 'Approve' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Reject' })).toBeDisabled()
    finishReview?.({ success: false, message: 'Review already changed' })
    expect(await screen.findByText('Review already changed')).toBeVisible()
    expect(screen.getByRole('textbox', { name: 'Review note' })).toHaveValue(
      'Checked consumption'
    )
    expect(writes[0].data).toEqual({
      decision: 'reject',
      remark: 'Checked consumption',
    })
  })

  it('offers retry for failed business responses instead of showing no campaign', async () => {
    listResponse = { success: false, message: 'Campaign service unavailable' }
    renderPage(<ReferralCampaigns />)
    expect(await screen.findByRole('button', { name: 'Retry' })).toBeVisible()
    expect(
      screen.queryByText('No referral campaign is currently active')
    ).not.toBeInTheDocument()
    listResponse = { success: true, data: { campaign, events: [] } }
    await userEvent.setup().click(screen.getByRole('button', { name: 'Retry' }))
    expect(await screen.findByText(campaign.title)).toBeVisible()
  })

  it('blocks concurrent reward retries and retains the operation after a business failure', async () => {
    const user = userEvent.setup()
    events = [
      {
        ...event,
        status: 'reward_failed',
        review_decision: 'approved',
        reward_status: 'failed',
      },
    ]
    renderPage()
    await user.click(
      await screen.findByRole('button', { name: /Autumn campaign/ })
    )
    await user.click(
      await screen.findByRole('button', { name: 'Retry reward' })
    )
    await waitFor(() => expect(finishRetry).toBeDefined())
    expect(screen.getByRole('button', { name: 'Retry reward' })).toBeDisabled()
    finishRetry?.({
      success: false,
      message: 'Reward recipient is unavailable',
    })
    expect(
      await screen.findByText('Reward recipient is unavailable')
    ).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Retry reward' }))
    events = [{ ...events[0], status: 'rewarded', reward_status: 'succeeded' }]
    finishRetry?.({ success: true, data: events[0] })
    await waitFor(() =>
      expect(
        screen.queryByRole('button', { name: 'Retry reward' })
      ).not.toBeInTheDocument()
    )
    expect(writes.map((request) => request.url)).toEqual([
      '/api/referral-campaign/admin/events/7/retry-reward',
      '/api/referral-campaign/admin/events/7/retry-reward',
    ])
  })
})
