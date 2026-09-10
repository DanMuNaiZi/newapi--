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
import { describe, test } from 'node:test'

import {
  InfiniteQueryObserver,
  QueryClient,
  QueryObserver,
} from '@tanstack/react-query'

import type {
  ApiResponse,
  LotteryNotificationPage,
  LotteryPlan,
  LotteryResultPage,
  LotterySelfResult,
} from '../types'
import { lotteryQueryKeys } from './query-keys'
import {
  lotteryNotificationsInfiniteQueryOptions,
  lotteryNotificationsSummaryQueryOptions,
  lotteryPendingResultsQueryOptions,
  lotteryPlansQueryOptions,
  lotteryResultsInfiniteQueryOptions,
  refreshLotteryClaimQueries,
  resetLotteryNotificationQueries,
} from './query-options'

function createQueryClient(): QueryClient {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } })
}

function createPlan(id: number): LotteryPlan {
  return {
    id,
    title: `Lottery ${id}`,
    icon: '',
    description: '',
    status: 'open',
    eligibility_mode: 'all',
    max_participants: 10,
    registration_start_time: 1,
    draw_time: 2,
  }
}

function createResult(id: number): LotterySelfResult {
  return {
    id,
    plan_id: 1,
    user_id: 7,
    prize_id: 1,
    reward_type: 'quota',
    quota: 100,
    subscription_plan_id: 0,
    fulfillment_mode: 'self_claim',
    fulfillment_status: 'pending',
    claim_expires_at: 0,
    claimed_at: 0,
    redemption_code: '',
    created_at: 1,
    claim_status: 'pending',
    claimable: true,
  }
}

function createNotificationPage(
  unreadTotal: number,
  cursor?: string
): ApiResponse<LotteryNotificationPage> {
  return {
    success: true,
    message: '',
    data: {
      items: unreadTotal
        ? [
            {
              id: cursor ? 2 : 1,
              user_id: 7,
              plan_id: 1,
              type: 'lottery_result',
              content: '',
              read_at: 0,
              created_at: 1,
            },
          ]
        : [],
      has_more: unreadTotal > 1 && !cursor,
      next_cursor: unreadTotal > 1 && !cursor ? 'next-page' : undefined,
      unread_total: unreadTotal,
    },
  }
}

function createRequests(options?: { claimed?: boolean; unreadTotal?: number }) {
  let claimed = options?.claimed ?? false
  let unreadTotal = options?.unreadTotal ?? 1
  const calls = { pending: 0, results: 0, summary: 0, notifications: 0 }
  return {
    requests: {
      getPlans: async (): Promise<ApiResponse<LotteryPlan[]>> => ({
        success: true,
        message: '',
        data: [createPlan(1)],
      }),
      getPendingResults: async (): Promise<
        ApiResponse<LotterySelfResult[]>
      > => {
        calls.pending += 1
        return {
          success: true,
          message: '',
          data: claimed ? [] : [createResult(10)],
        }
      },
      getResultsPage: async (): Promise<ApiResponse<LotteryResultPage>> => {
        calls.results += 1
        return {
          success: true,
          message: '',
          data: { items: claimed ? [] : [createResult(10)], has_more: false },
        }
      },
      getNotificationsPage: async (
        cursor?: string
      ): Promise<ApiResponse<LotteryNotificationPage>> => {
        if (cursor) calls.notifications += 1
        else calls.summary += 1
        return createNotificationPage(unreadTotal, cursor)
      },
    },
    calls,
    claim: () => {
      claimed = true
      unreadTotal = 0
    },
    markNotificationsRead: () => {
      unreadTotal = 0
    },
  }
}

describe('lottery query configurations', () => {
  test('keeps overview and lottery page response shapes apart through navigation', async () => {
    const queryClient = createQueryClient()
    const { requests } = createRequests()
    const overviewPlans = new QueryObserver(
      queryClient,
      lotteryPlansQueryOptions(7, requests)
    )
    const overviewPending = new QueryObserver(
      queryClient,
      lotteryPendingResultsQueryOptions(7, requests)
    )
    const overviewNotifications = new QueryObserver(
      queryClient,
      lotteryNotificationsSummaryQueryOptions(7, requests)
    )
    const unsubscribeOverview = [
      overviewPlans.subscribe(() => undefined),
      overviewPending.subscribe(() => undefined),
      overviewNotifications.subscribe(() => undefined),
    ]
    await Promise.all([
      overviewPlans.refetch(),
      overviewPending.refetch(),
      overviewNotifications.refetch(),
    ])
    assert.deepEqual(overviewPlans.getCurrentResult().data?.data, [
      createPlan(1),
    ])
    assert.deepEqual(overviewPending.getCurrentResult().data?.data, [
      createResult(10),
    ])
    assert.equal(
      overviewNotifications.getCurrentResult().data?.data.unread_total,
      1
    )
    unsubscribeOverview.forEach((unsubscribe) => unsubscribe())

    const lotteryPlans = new QueryObserver(
      queryClient,
      lotteryPlansQueryOptions(7, requests)
    )
    const lotteryResults = new InfiniteQueryObserver(
      queryClient,
      lotteryResultsInfiniteQueryOptions(7, requests)
    )
    const lotteryNotifications = new InfiniteQueryObserver(
      queryClient,
      lotteryNotificationsInfiniteQueryOptions(7, requests)
    )
    const unsubscribeLottery = [
      lotteryPlans.subscribe(() => undefined),
      lotteryResults.subscribe(() => undefined),
      lotteryNotifications.subscribe(() => undefined),
    ]
    await Promise.all([
      lotteryPlans.refetch(),
      lotteryResults.refetch(),
      lotteryNotifications.refetch(),
    ])
    assert.deepEqual(lotteryPlans.getCurrentResult().data?.data, [
      createPlan(1),
    ])
    assert.deepEqual(
      lotteryResults.getCurrentResult().data?.pages[0].data.items,
      [createResult(10)]
    )
    assert.equal(
      lotteryNotifications.getCurrentResult().data?.pages[0].data.unread_total,
      1
    )
    unsubscribeLottery.forEach((unsubscribe) => unsubscribe())

    const overviewAgain = new QueryObserver(
      queryClient,
      lotteryNotificationsSummaryQueryOptions(7, requests)
    )
    const unsubscribeOverviewAgain = overviewAgain.subscribe(() => undefined)
    await overviewAgain.refetch()
    assert.equal(overviewAgain.getCurrentResult().data?.data.unread_total, 1)
    unsubscribeOverviewAgain()
  })

  test('keeps cached lottery data isolated when the active user changes', async () => {
    const queryClient = createQueryClient()
    let activeUserId = 7
    const { requests } = createRequests()
    requests.getPlans = async (): Promise<ApiResponse<LotteryPlan[]>> => ({
      success: true,
      message: '',
      data: [createPlan(activeUserId)],
    })
    requests.getPendingResults = async (): Promise<
      ApiResponse<LotterySelfResult[]>
    > => ({
      success: true,
      message: '',
      data: [{ ...createResult(activeUserId), user_id: activeUserId }],
    })
    requests.getResultsPage = async (): Promise<
      ApiResponse<LotteryResultPage>
    > => ({
      success: true,
      message: '',
      data: {
        items: [{ ...createResult(activeUserId), user_id: activeUserId }],
        has_more: false,
      },
    })
    requests.getNotificationsPage = async (): Promise<
      ApiResponse<LotteryNotificationPage>
    > => ({
      success: true,
      message: '',
      data: {
        items: [
          {
            id: activeUserId,
            user_id: activeUserId,
            plan_id: 1,
            type: 'lottery_result',
            content: '',
            read_at: 0,
            created_at: 1,
          },
        ],
        has_more: false,
        unread_total: activeUserId,
      },
    })
    await Promise.all([
      queryClient.fetchQuery(lotteryPlansQueryOptions(7, requests)),
      queryClient.fetchQuery(lotteryPendingResultsQueryOptions(7, requests)),
      queryClient.fetchQuery(
        lotteryNotificationsSummaryQueryOptions(7, requests)
      ),
      queryClient.fetchInfiniteQuery(
        lotteryResultsInfiniteQueryOptions(7, requests)
      ),
      queryClient.fetchInfiniteQuery(
        lotteryNotificationsInfiniteQueryOptions(7, requests)
      ),
    ])
    activeUserId = 8
    await Promise.all([
      queryClient.fetchQuery(lotteryPlansQueryOptions(8, requests)),
      queryClient.fetchQuery(lotteryPendingResultsQueryOptions(8, requests)),
      queryClient.fetchQuery(
        lotteryNotificationsSummaryQueryOptions(8, requests)
      ),
      queryClient.fetchInfiniteQuery(
        lotteryResultsInfiniteQueryOptions(8, requests)
      ),
      queryClient.fetchInfiniteQuery(
        lotteryNotificationsInfiniteQueryOptions(8, requests)
      ),
    ])
    assert.equal(
      queryClient.getQueryData<ApiResponse<LotteryPlan[]>>(
        lotteryQueryKeys.plans(7)
      )?.data[0].id,
      7
    )
    assert.equal(
      queryClient.getQueryData<ApiResponse<LotteryPlan[]>>(
        lotteryQueryKeys.plans(8)
      )?.data[0].id,
      8
    )
    assert.equal(
      queryClient.getQueryData<ApiResponse<LotterySelfResult[]>>(
        lotteryQueryKeys.pendingResults(7)
      )?.data[0].user_id,
      7
    )
    assert.equal(
      queryClient.getQueryData<ApiResponse<LotterySelfResult[]>>(
        lotteryQueryKeys.pendingResults(8)
      )?.data[0].user_id,
      8
    )
    assert.equal(
      queryClient.getQueryData<{
        pages: ApiResponse<LotteryResultPage>[]
      }>(lotteryQueryKeys.results(7))?.pages[0].data.items[0].user_id,
      7
    )
    assert.equal(
      queryClient.getQueryData<{
        pages: ApiResponse<LotteryResultPage>[]
      }>(lotteryQueryKeys.results(8))?.pages[0].data.items[0].user_id,
      8
    )
    assert.equal(
      queryClient.getQueryData<ApiResponse<LotteryNotificationPage>>(
        lotteryQueryKeys.notificationsSummary(7)
      )?.data.unread_total,
      7
    )
    assert.equal(
      queryClient.getQueryData<ApiResponse<LotteryNotificationPage>>(
        lotteryQueryKeys.notificationsSummary(8)
      )?.data.unread_total,
      8
    )
    assert.equal(
      queryClient.getQueryData<{
        pages: ApiResponse<LotteryNotificationPage>[]
      }>(lotteryQueryKeys.notificationsPage(7))?.pages[0].data.items[0].user_id,
      7
    )
    assert.equal(
      queryClient.getQueryData<{
        pages: ApiResponse<LotteryNotificationPage>[]
      }>(lotteryQueryKeys.notificationsPage(8))?.pages[0].data.items[0].user_id,
      8
    )
  })

  test('refreshes every claim-dependent query for the current user', async () => {
    const queryClient = createQueryClient()
    const state = createRequests()
    const pending = new QueryObserver(
      queryClient,
      lotteryPendingResultsQueryOptions(7, state.requests)
    )
    const results = new InfiniteQueryObserver(
      queryClient,
      lotteryResultsInfiniteQueryOptions(7, state.requests)
    )
    const summary = new QueryObserver(
      queryClient,
      lotteryNotificationsSummaryQueryOptions(7, state.requests)
    )
    const notifications = new InfiniteQueryObserver(
      queryClient,
      lotteryNotificationsInfiniteQueryOptions(7, state.requests)
    )
    const unsubscribers = [
      pending.subscribe(() => undefined),
      results.subscribe(() => undefined),
      summary.subscribe(() => undefined),
      notifications.subscribe(() => undefined),
    ]
    await Promise.all([
      pending.refetch(),
      results.refetch(),
      summary.refetch(),
      notifications.refetch(),
    ])
    const otherUserPending: ApiResponse<LotterySelfResult[]> = {
      success: true,
      message: '',
      data: [{ ...createResult(88), user_id: 8 }],
    }
    queryClient.setQueryData(
      lotteryQueryKeys.pendingResults(8),
      otherUserPending
    )
    const callsBeforeClaim = { ...state.calls }
    state.claim()
    await refreshLotteryClaimQueries(queryClient, 7)
    assert.deepEqual(pending.getCurrentResult().data?.data, [])
    assert.deepEqual(results.getCurrentResult().data?.pages[0].data.items, [])
    assert.equal(summary.getCurrentResult().data?.data.unread_total, 0)
    assert.equal(
      notifications.getCurrentResult().data?.pages[0].data.unread_total,
      0
    )
    assert.ok(state.calls.pending > callsBeforeClaim.pending)
    assert.ok(state.calls.results > callsBeforeClaim.results)
    assert.ok(state.calls.summary > callsBeforeClaim.summary)
    assert.deepEqual(
      queryClient.getQueryData<ApiResponse<LotterySelfResult[]>>(
        lotteryQueryKeys.pendingResults(8)
      ),
      otherUserPending
    )
    unsubscribers.forEach((unsubscribe) => unsubscribe())
  })

  test('resets summary and paginated unread notifications after marking them read', async () => {
    const queryClient = createQueryClient()
    const state = createRequests({ unreadTotal: 2 })
    const summary = new QueryObserver(
      queryClient,
      lotteryNotificationsSummaryQueryOptions(7, state.requests)
    )
    const notifications = new InfiniteQueryObserver(
      queryClient,
      lotteryNotificationsInfiniteQueryOptions(7, state.requests)
    )
    const unsubscribers = [
      summary.subscribe(() => undefined),
      notifications.subscribe(() => undefined),
    ]
    await Promise.all([summary.refetch(), notifications.refetch()])
    await notifications.fetchNextPage()
    assert.equal(notifications.getCurrentResult().data?.pages.length, 2)
    state.markNotificationsRead()
    await resetLotteryNotificationQueries(queryClient, 7)
    assert.equal(summary.getCurrentResult().data?.data.unread_total, 0)
    assert.deepEqual(notifications.getCurrentResult().data?.pages, [
      createNotificationPage(0),
    ])
    assert.equal(notifications.getCurrentResult().hasNextPage, false)
    unsubscribers.forEach((unsubscribe) => unsubscribe())
  })
})
