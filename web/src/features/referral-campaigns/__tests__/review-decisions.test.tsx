/* Copyright (C) 2023-2026 QuantumNous */
import { QueryClientProvider, type QueryClient } from '@tanstack/react-query'
import { act, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createInstance, type i18n } from 'i18next'
import { I18nextProvider } from 'react-i18next'
import { Toaster, toast } from 'sonner'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import { api } from '@/lib/api'
import { createAppQueryClient } from '@/lib/query-client'
import { useAuthStore } from '@/stores/auth-store'

import { ReferralReviewDrawer } from '../components/referral-review-drawer'
import type { ReferralEventReview } from '../types'

let originalAdapter: typeof api.defaults.adapter
let client: QueryClient
let testI18n: i18n
let view: ReferralEventReview
let response: unknown
let requests: Array<{ method: string; url: string; data: unknown }>
let changes: string[]

function renderReview() {
  render(
    <I18nextProvider i18n={testI18n}>
      <QueryClientProvider client={client}>
        <ReferralReviewDrawer
          eventId={7}
          onClose={() => changes.push('close')}
          onChanged={() => {
            changes.push('changed')
          }}
        />
        <Toaster />
      </QueryClientProvider>
    </I18nextProvider>
  )
}

beforeEach(async () => {
  useAuthStore.getState().auth.setUser({
    id: 1,
    username: 'reviewer',
    role: 10,
    permissions: {
      admin_permissions: { referral_campaign: { read: true, operate: true } },
    },
  })
  testI18n = createInstance()
  await testI18n.init({
    lng: 'en',
    fallbackLng: 'en',
    resources: Object.fromEntries(
      ['en', 'zhCN', 'zhTW', 'fr', 'ru', 'ja', 'vi', 'bad_locale'].map(
        (language) => [language, { translation: { 'Known key': 'Known key' } }]
      )
    ),
    interpolation: { escapeValue: false },
  })
  changes = []
  requests = []
  view = {
    event: {
      id: 7,
      campaign_id: 4,
      inviter_user_id: 1,
      invitee_user_id: 2,
      status: 'pending_review',
      registered_at: 1_800_000_000,
      activation_deadline: 1_800_086_400,
      activated_at: 1_800_000_100,
      qualified_quota: 25,
      reward: {
        type: 'subscription',
        subscription_plan_id: 6,
        subscription_plan_title: 'Original annual plan',
      },
      invitee_reward: {
        type: 'quota',
        quota: 1000,
        amount: '0.002',
        unit: 'usd',
      },
    },
    inviter: {
      id: 1,
      username: 'inviter',
      created_at: 1_600_000_000,
      github_id: '100',
      github_created_at: 1_600_000_000,
      github_age_exempt: false,
    },
    invitee: {
      id: 2,
      username: 'newcomer',
      created_at: 1_800_000_000,
      github_id: '200',
      github_created_at: 1_600_000_000,
      github_age_exempt: true,
    },
    has_consumption: true,
    can_approve: true,
    usage: {
      requests: 1,
      consumed_quota: 25,
      logs_available: true,
      period_start: 1_800_000_000,
      period_end: 1_800_000_100,
      recent_calls: [
        {
          id: 3,
          created_at: 1_800_000_100,
          request_id: 'settled-call',
          model_name: 'gpt-test',
          quota: 25,
          prompt_tokens: 23456,
          completion_tokens: 34567,
          successful: true,
          qualifying: true,
          public_pool: false,
        },
      ],
    },
    related_events: [],
    reward_grants: [],
  }
  response = { success: false, message: 'Approval limit reached' }
  client = createAppQueryClient()
  client.setDefaultOptions({
    queries: { retry: false },
    mutations: { retry: false },
  })
  originalAdapter = api.defaults.adapter
  api.defaults.adapter = async (config) => {
    const url = config.url ?? ''
    requests.push({
      method: config.method ?? '',
      url,
      data: config.data ? JSON.parse(config.data) : undefined,
    })
    let data: unknown
    if (url === '/api/status') {
      data = {
        success: true,
        data: { quota_per_unit: 500_000, usd_exchange_rate: 7 },
      }
    } else if (config.method === 'get') data = { success: true, data: view }
    else data = response
    return { data, status: 200, statusText: 'OK', headers: {}, config }
  }
})

afterEach(() => {
  api.defaults.adapter = originalAdapter
  client.clear()
  useAuthStore.getState().auth.reset()
  toast.dismiss()
})

describe('referral review decisions', () => {
  it('keeps a failed approval confirmation and note without announcing a change', async () => {
    const user = userEvent.setup()
    renderReview()
    await user.type(
      await screen.findByRole('textbox', { name: 'Review remark' }),
      'Verified the saved evidence'
    )
    await user.click(
      screen.getByRole('button', { name: 'Approve and issue rewards' })
    )
    const confirm = await screen.findByRole('alertdialog', {
      name: 'Approve and issue rewards',
    })
    await user.click(within(confirm).getByRole('button', { name: 'Continue' }))
    expect(await screen.findByText('Approval limit reached')).toBeVisible()
    expect(confirm).toBeVisible()
    expect(changes).toEqual([])
    expect(
      screen.getByRole('textbox', { name: 'Review remark', hidden: true })
    ).toHaveValue('Verified the saved evidence')
    response = { success: true, data: { ...view.event, status: 'rewarded' } }
    view.event = {
      ...view.event,
      status: 'rewarded',
      review_decision: 'approved',
    }
    await user.click(within(confirm).getByRole('button', { name: 'Continue' }))
    await waitFor(() =>
      expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    )
    expect(changes).toEqual(['changed'])
    expect(requests.filter((request) => request.method === 'post')).toEqual([
      {
        method: 'post',
        url: '/api/referral-campaign/admin/events/7/review',
        data: { decision: 'approve', remark: 'Verified the saved evidence' },
      },
      {
        method: 'post',
        url: '/api/referral-campaign/admin/events/7/review',
        data: { decision: 'approve', remark: 'Verified the saved evidence' },
      },
    ])
  })

  it('allows rejection without qualifying consumption while keeping approval disabled', async () => {
    const user = userEvent.setup()
    view.has_consumption = false
    view.can_approve = false
    view.event.status = 'registered'
    view.event.qualified_quota = 0
    renderReview()
    const reject = await screen.findByRole('button', { name: 'Reject' })
    expect(
      screen.getByRole('button', { name: 'Approve and issue rewards' })
    ).toBeDisabled()
    await user.click(reject)
    const confirm = await screen.findByRole('alertdialog', { name: 'Reject' })
    response = { success: true }
    view.event = {
      ...view.event,
      status: 'rejected',
      review_decision: 'rejected',
    }
    await user.click(within(confirm).getByRole('button', { name: 'Continue' }))
    await waitFor(() => expect(changes).toEqual(['changed']))
    expect(requests.find((request) => request.method === 'post')?.data).toEqual(
      { decision: 'reject', remark: '' }
    )
  })

  it('shows the saved reward snapshot and evidence to read-only reviewers without decisions', async () => {
    useAuthStore.getState().auth.setUser({
      id: 1,
      username: 'reader',
      role: 10,
      permissions: {
        admin_permissions: { referral_campaign: { read: true } },
      },
    })
    renderReview()
    expect(await screen.findByText(/Original annual plan/)).toBeVisible()
    expect(
      screen.getByText('Eligible quota consumption confirmed')
    ).toBeVisible()
    expect(
      screen.queryByRole('button', { name: 'Approve and issue rewards' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Reject' })
    ).not.toBeInTheDocument()
  })

  it('refreshes token formatting for every UI language and an invalid locale fallback', async () => {
    renderReview()
    await screen.findByText(/Original annual plan/)
    for (const [language, expected] of [
      ['en', '23,456 / 34,567'],
      ['zhCN', '23,456 / 34,567'],
      ['zhTW', '23,456 / 34,567'],
      ['fr', '23 456 / 34 567'],
      ['ru', '23 456 / 34 567'],
      ['ja', '23,456 / 34,567'],
      ['vi', '23.456 / 34.567'],
      ['bad_locale', '23,456 / 34,567'],
    ]) {
      await act(async () => {
        await testI18n.changeLanguage(language)
      })
      expect(screen.getByRole('cell', { name: expected })).toBeVisible()
    }
  })
})
