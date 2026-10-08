/* Copyright (C) 2023-2026 QuantumNous */
import { QueryClientProvider, type QueryClient } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Toaster, toast } from 'sonner'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import { api } from '@/lib/api'
import { createAppQueryClient } from '@/lib/query-client'
import { useAuthStore } from '@/stores/auth-store'

import { PublicPoolAdmin } from '../admin'
import { PublicPool } from '../index'
import type { PublicPoolContribution, PublicPoolSite } from '../types'

const site: PublicPoolSite = {
  id: 3,
  name: 'Community endpoint',
  url: 'https://pool.example.com',
  description: '',
  status: 'enabled',
  sort_order: 0,
  created_at: 1,
  updated_at: 1,
  reward: { type: 'quota', amount: '2', unit: 'usd', quota: 1_000_000 },
}
const contribution: PublicPoolContribution = {
  id: 8,
  user_id: 2,
  site_id: 3,
  username: 'contributor',
  site_name: site.name,
  description: 'OpenAI compatible endpoint',
  proof: 'Verification evidence',
  status: 'pending',
  reviewer_id: 0,
  review_note: '',
  reviewed_at: 0,
  created_at: 1,
  updated_at: 1,
  reward: { type: 'quota', amount: '1', unit: 'usd', quota: 500_000 },
}
let originalAdapter: typeof api.defaults.adapter
let client: QueryClient
let reads: string[]
let writes: Array<{ url: string; data: unknown }>
let contributions: PublicPoolContribution[]
let sitesResponse: unknown
let submitResponse: unknown
let deleteResponse: unknown
let finishReview: ((value: unknown) => void) | undefined
let finishRetry: ((value: unknown) => void) | undefined
let failRetry: ((error: Error) => void) | undefined

function setPermissions(permissions: Record<string, boolean>) {
  useAuthStore.getState().auth.setUser({
    id: 1,
    username: 'reviewer',
    role: 10,
    permissions: { admin_permissions: { public_pool: permissions } },
  })
}

function renderPage(page = <PublicPoolAdmin />) {
  render(
    <QueryClientProvider client={client}>
      {page}
      <Toaster />
    </QueryClientProvider>
  )
}

beforeEach(() => {
  reads = []
  writes = []
  contributions = [contribution]
  sitesResponse = { success: true, data: [site] }
  submitResponse = { success: false, message: 'Contribution review is paused' }
  deleteResponse = {
    success: false,
    message: 'Site still has pending contributions',
  }
  finishReview = undefined
  finishRetry = undefined
  failRetry = undefined
  setPermissions({ read: true, write: true, operate: true })
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
      if (url.endsWith('/status')) {
        data = {
          success: true,
          data: {
            available: true,
            enabled: true,
            channel_count: 1,
            group_ratio: 0,
          },
        }
      } else if (url.endsWith('/sites')) data = sitesResponse
      else if (url.endsWith('/reward-plans')) data = { success: true, data: [] }
      else if (url.endsWith('/self')) data = { success: true, data: [] }
      else {
        data = {
          success: true,
          data: {
            items: contributions,
            total: contributions.length,
            page: 1,
            page_size: 100,
          },
        }
      }
    } else {
      writes.push({
        url,
        data: config.data ? JSON.parse(config.data) : undefined,
      })
      if (url.endsWith('/review')) {
        data = await new Promise((resolve) => {
          finishReview = resolve
        })
      } else if (url.endsWith('/retry-reward')) {
        data = await new Promise((resolve, reject) => {
          finishRetry = resolve
          failRetry = reject
        })
      } else if (config.method === 'delete') data = deleteResponse
      else data = submitResponse
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

describe('public pool permissions and review actions', () => {
  it('does not fetch protected data when read permission is absent', async () => {
    setPermissions({})
    renderPage()
    expect(
      await screen.findByText('You do not have permission to view this page')
    ).toBeVisible()
    expect(reads).toEqual([])
  })

  it('keeps read-only access free of site edits and reward operations', async () => {
    setPermissions({ read: true })
    renderPage()
    expect(await screen.findByText('contributor')).toBeVisible()
    expect(
      screen.queryByRole('button', { name: 'Save' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Approve' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Reject' })
    ).not.toBeInTheDocument()
    expect(reads.some((url) => url.endsWith('/reward-plans'))).toBe(false)
  })

  it('shows submitted evidence and the original reward snapshot to reviewers', async () => {
    renderPage()
    expect(await screen.findByText(contribution.proof)).toBeVisible()
    expect(screen.getByText(contribution.description)).toBeVisible()
    expect(screen.getByText('500,000')).toBeVisible()
  })

  it.each([
    [{ read: true, write: true }, true, false],
    [{ read: true, operate: true }, false, true],
  ])(
    'separates site-write and reward-operate access for %j',
    async (permissions, canWrite, canOperate) => {
      setPermissions(permissions)
      renderPage()
      await screen.findByText('contributor')
      expect(Boolean(screen.queryByRole('button', { name: 'Save' }))).toBe(
        canWrite
      )
      expect(Boolean(screen.queryByRole('button', { name: 'Approve' }))).toBe(
        canOperate
      )
    }
  )

  it('preserves an existing subscription reward when only site metadata is saved', async () => {
    const user = userEvent.setup()
    sitesResponse = {
      success: true,
      data: [
        {
          ...site,
          reward: {
            type: 'subscription',
            subscription_plan_id: 22,
            subscription_plan_title: 'Retired reward plan',
          },
        },
      ],
    }
    renderPage()
    await user.click(await screen.findByRole('button', { name: 'Edit' }))
    expect(screen.getByRole('combobox', { name: 'Reward type' })).toBeDisabled()
    expect(
      screen.getByRole('combobox', { name: 'Subscription plan' })
    ).toBeDisabled()
    await user.type(screen.getByRole('textbox', { name: 'Name' }), ' revised')
    submitResponse = { success: false, message: 'Site cannot be saved' }
    await user.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('Site cannot be saved')).toBeVisible()
    expect(screen.getByRole('textbox', { name: 'Name' })).toHaveValue(
      'Community endpoint revised'
    )
    expect(writes[0].data).toMatchObject({
      preserve_reward: true,
      name: 'Community endpoint revised',
    })
    submitResponse = { success: true, data: site }
    await user.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() =>
      expect(screen.getByRole('textbox', { name: 'Name' })).toHaveValue('')
    )
  })

  it('disables review decisions while approval is pending and retains the note after failure', async () => {
    const user = userEvent.setup()
    renderPage()
    await screen.findByText('contributor')
    await user.type(
      screen.getByRole('textbox', { name: 'Review note' }),
      'verified'
    )
    await user.click(screen.getByRole('button', { name: 'Approve' }))
    await waitFor(() => expect(finishReview).toBeDefined())
    expect(screen.getByRole('button', { name: 'Approve' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Reject' })).toBeDisabled()
    finishReview?.({ success: false, message: 'Approval is not available' })
    expect(await screen.findByText('Approval is not available')).toBeVisible()
    expect(screen.getByRole('textbox', { name: 'Review note' })).toHaveValue(
      'verified'
    )
    expect(writes[0]).toEqual({
      url: '/api/public-pool/admin/contributions/8/review',
      data: { status: 'approved', review_note: 'verified' },
    })
  })

  it('retains the shared delete confirmation after a business failure for retry', async () => {
    const user = userEvent.setup()
    renderPage()
    await screen.findByText('contributor')
    await user.click(screen.getByRole('button', { name: 'Delete' }))
    const dialog = await screen.findByRole('alertdialog', {
      name: 'Delete this site?',
    })
    await user.click(within(dialog).getByRole('button', { name: 'Delete' }))
    expect(
      await screen.findByText('Site still has pending contributions')
    ).toBeVisible()
    expect(dialog).toBeVisible()
    deleteResponse = { success: true }
    await user.click(within(dialog).getByRole('button', { name: 'Delete' }))
    await waitFor(() =>
      expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    )
  })

  it('blocks duplicate reward retry requests while pending and allows retry after network failure', async () => {
    const user = userEvent.setup()
    contributions = [
      {
        ...contribution,
        status: 'approved',
        reward_status: 'failed',
        reward_failure_reason: 'Grant service unavailable',
      },
    ]
    renderPage()
    await user.click(
      await screen.findByRole('button', { name: 'Retry reward' })
    )
    await waitFor(() => expect(failRetry).toBeDefined())
    expect(screen.getByRole('button', { name: 'Retry reward' })).toBeDisabled()
    failRetry?.(new Error('Reward service not reachable'))
    expect(
      await screen.findByText('Reward service not reachable')
    ).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Retry reward' }))
    contributions = [{ ...contributions[0], reward_status: 'succeeded' }]
    finishRetry?.({ success: true, data: contributions[0] })
    await waitFor(() =>
      expect(
        screen.queryByRole('button', { name: 'Retry reward' })
      ).not.toBeInTheDocument()
    )
    expect(writes.map((request) => request.url)).toEqual([
      '/api/public-pool/admin/contributions/8/retry-reward',
      '/api/public-pool/admin/contributions/8/retry-reward',
    ])
  })

  it('renders a retryable error instead of an empty list on a failed sites response', async () => {
    sitesResponse = { success: false, message: 'Public sites are unavailable' }
    renderPage()
    expect(await screen.findByRole('button', { name: 'Retry' })).toBeVisible()
    sitesResponse = { success: true, data: [site] }
    await userEvent.setup().click(screen.getByRole('button', { name: 'Retry' }))
    expect(await screen.findByText(site.url)).toBeVisible()
  })

  it('keeps the user contribution draft on failure and clears it only after success', async () => {
    const user = userEvent.setup()
    renderPage(<PublicPool />)
    const selector = await screen.findByRole('combobox', { name: 'Site' })
    await user.click(selector)
    await user.click(await screen.findByRole('option', { name: site.name }))
    await user.type(
      screen.getByRole('textbox', { name: 'Description' }),
      'Useful endpoint'
    )
    await user.click(screen.getByRole('button', { name: 'Submit for review' }))
    expect(
      await screen.findByText('Contribution review is paused')
    ).toBeVisible()
    expect(screen.getByRole('textbox', { name: 'Description' })).toHaveValue(
      'Useful endpoint'
    )
    submitResponse = { success: true, data: contribution }
    await user.click(screen.getByRole('button', { name: 'Submit for review' }))
    await waitFor(() =>
      expect(screen.getByRole('textbox', { name: 'Description' })).toHaveValue(
        ''
      )
    )
  })
})
