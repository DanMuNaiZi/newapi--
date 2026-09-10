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
  infiniteQueryOptions,
  queryOptions,
  type QueryClient,
} from '@tanstack/react-query'

import {
  getClaimableLotteryResultsForSelf,
  getLotteryNotificationsPageForSelf,
  getLotteryPlansForSelf,
  getLotteryResultsPageForSelf,
} from '../api'
import type {
  ApiResponse,
  LotteryNotificationPage,
  LotteryPlan,
  LotteryResultPage,
  LotterySelfResult,
} from '../types'
import { lotteryQueryKeys } from './query-keys'

export interface LotteryQueryRequests {
  getPlans: () => Promise<ApiResponse<LotteryPlan[]>>
  getPendingResults: () => Promise<ApiResponse<LotterySelfResult[]>>
  getResultsPage: (
    cursor?: string,
    limit?: number
  ) => Promise<ApiResponse<LotteryResultPage>>
  getNotificationsPage: (
    cursor?: string,
    limit?: number,
    unreadOnly?: boolean
  ) => Promise<ApiResponse<LotteryNotificationPage>>
}

const defaultLotteryQueryRequests: LotteryQueryRequests = {
  getPlans: getLotteryPlansForSelf,
  getPendingResults: getClaimableLotteryResultsForSelf,
  getResultsPage: getLotteryResultsPageForSelf,
  getNotificationsPage: getLotteryNotificationsPageForSelf,
}

const lotteryQueryRefetchInterval = 30_000

export function lotteryPlansQueryOptions(
  userId: number,
  requests: LotteryQueryRequests = defaultLotteryQueryRequests
) {
  return queryOptions({
    queryKey: lotteryQueryKeys.plans(userId),
    queryFn: requests.getPlans,
    enabled: userId > 0,
    refetchInterval: lotteryQueryRefetchInterval,
  })
}

export function lotteryPendingResultsQueryOptions(
  userId: number,
  requests: LotteryQueryRequests = defaultLotteryQueryRequests
) {
  return queryOptions({
    queryKey: lotteryQueryKeys.pendingResults(userId),
    queryFn: requests.getPendingResults,
    enabled: userId > 0,
    refetchInterval: lotteryQueryRefetchInterval,
  })
}

export function lotteryResultsInfiniteQueryOptions(
  userId: number,
  requests: LotteryQueryRequests = defaultLotteryQueryRequests
) {
  return infiniteQueryOptions({
    queryKey: lotteryQueryKeys.results(userId),
    initialPageParam: '',
    queryFn: ({ pageParam }) => requests.getResultsPage(pageParam),
    getNextPageParam: (lastPage) => {
      if (!lastPage.success || !lastPage.data.has_more) return undefined
      return lastPage.data.next_cursor
    },
    enabled: userId > 0,
  })
}

export function lotteryNotificationsSummaryQueryOptions(
  userId: number,
  requests: LotteryQueryRequests = defaultLotteryQueryRequests
) {
  return queryOptions({
    queryKey: lotteryQueryKeys.notificationsSummary(userId),
    queryFn: () => requests.getNotificationsPage(undefined, 20, true),
    enabled: userId > 0,
    refetchInterval: lotteryQueryRefetchInterval,
  })
}

export function lotteryNotificationsInfiniteQueryOptions(
  userId: number,
  requests: LotteryQueryRequests = defaultLotteryQueryRequests
) {
  return infiniteQueryOptions({
    queryKey: lotteryQueryKeys.notificationsPage(userId),
    initialPageParam: '',
    queryFn: ({ pageParam }) =>
      requests.getNotificationsPage(pageParam, 20, true),
    getNextPageParam: (lastPage) => {
      if (!lastPage.success || !lastPage.data.has_more) return undefined
      return lastPage.data.next_cursor
    },
    enabled: userId > 0,
  })
}

export async function refreshLotteryClaimQueries(
  queryClient: QueryClient,
  userId: number
): Promise<void> {
  await Promise.all([
    queryClient.invalidateQueries({
      queryKey: lotteryQueryKeys.results(userId),
    }),
    queryClient.invalidateQueries({
      queryKey: lotteryQueryKeys.pendingResults(userId),
    }),
    queryClient.invalidateQueries({
      queryKey: lotteryQueryKeys.notificationsSummary(userId),
    }),
    queryClient.invalidateQueries({
      queryKey: lotteryQueryKeys.notificationsPage(userId),
    }),
  ])
}

export async function resetLotteryNotificationQueries(
  queryClient: QueryClient,
  userId: number
): Promise<void> {
  await Promise.all([
    queryClient.resetQueries({
      queryKey: lotteryQueryKeys.notificationsSummary(userId),
    }),
    queryClient.resetQueries({
      queryKey: lotteryQueryKeys.notificationsPage(userId),
    }),
  ])
}
