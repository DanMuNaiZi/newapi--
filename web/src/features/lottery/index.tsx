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
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import { getRouteApi } from '@tanstack/react-router'
import dayjs from 'dayjs'
import {
  Bell,
  CalendarClock,
  Eye,
  Gift,
  LogIn,
  LogOut,
  Trophy,
  Users,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'

import {
  claimLotteryResult,
  getLotteryNotificationsPageForSelf,
  getLotteryPlansForSelf,
  getLotteryResultsPageForSelf,
  joinLotteryPlan,
  leaveLotteryPlan,
  markLotteryNotificationsRead,
} from './api'
import { LotteryIcon } from './components/lottery-icon'
import { LotteryUserDetailsDrawer } from './components/lottery-user-details-drawer'
import { mergeLotteryPages } from './lib/pagination'
import {
  getLotteryPlanStatusLabel,
  getLotteryRewardStatusLabel,
} from './lib/status'
import type { LotteryPlan } from './types'

const LOTTERY_QUERY_KEY = ['lottery', 'self'] as const

function planTime(timestamp: number): string {
  return dayjs.unix(timestamp).format('YYYY-MM-DD HH:mm')
}

export function Lotteries() {
  const { t } = useTranslation()
  const route = getRouteApi('/_authenticated/lotteries/')
  const search = route.useSearch()
  const navigate = route.useNavigate()
  const queryClient = useQueryClient()
  const plansQuery = useQuery({
    queryKey: LOTTERY_QUERY_KEY,
    queryFn: getLotteryPlansForSelf,
    refetchInterval: 30_000,
  })
  const resultsQuery = useInfiniteQuery({
    queryKey: ['lottery', 'results', 'page'],
    initialPageParam: '',
    queryFn: ({ pageParam }) => getLotteryResultsPageForSelf(pageParam),
    getNextPageParam: (lastPage) => {
      if (!lastPage.success || !lastPage.data.has_more) return undefined
      return lastPage.data.next_cursor
    },
  })
  const notificationsQuery = useInfiniteQuery({
    queryKey: ['lottery', 'notifications', 'page', 'unread'],
    initialPageParam: '',
    queryFn: ({ pageParam }) =>
      getLotteryNotificationsPageForSelf(pageParam, 20, true),
    getNextPageParam: (lastPage) => {
      if (!lastPage.success || !lastPage.data.has_more) return undefined
      return lastPage.data.next_cursor
    },
  })
  const invalidatePlans = async (): Promise<void> => {
    await queryClient.invalidateQueries({ queryKey: LOTTERY_QUERY_KEY })
  }
  const joinMutation = useMutation({
    mutationFn: joinLotteryPlan,
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(t('Failed to join lottery'))
        return
      }
      toast.success(t('Joined lottery'))
      await invalidatePlans()
    },
  })
  const leaveMutation = useMutation({
    mutationFn: leaveLotteryPlan,
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(t('Failed to leave lottery'))
        return
      }
      toast.success(t('Left lottery'))
      await invalidatePlans()
    },
  })

  const claimMutation = useMutation({
    mutationFn: claimLotteryResult,
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(t('Failed to claim lottery reward'))
        return
      }
      toast.success(t('Lottery reward claimed'))
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['lottery', 'results'] }),
        queryClient.invalidateQueries({
          queryKey: ['lottery', 'notifications'],
        }),
      ])
    },
  })
  const markNotificationsReadMutation = useMutation({
    mutationFn: markLotteryNotificationsRead,
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(t('Request failed'))
        return
      }
      await queryClient.resetQueries({
        queryKey: ['lottery', 'notifications', 'page'],
      })
    },
    onError: () => toast.error(t('Request failed')),
  })
  const plans = plansQuery.data?.success ? plansQuery.data.data : []
  const results = mergeLotteryPages(
    resultsQuery.data?.pages.map((page) =>
      page.success ? page.data.items : []
    ) ?? []
  )
  const notifications = mergeLotteryPages(
    notificationsQuery.data?.pages.map((page) =>
      page.success ? page.data.items : []
    ) ?? []
  )
  const unreadTotal = notificationsQuery.data?.pages[0]?.success
    ? notificationsQuery.data.pages[0].data.unread_total
    : 0
  const plansFailed = plansQuery.isError || plansQuery.data?.success === false
  const resultsFailed =
    resultsQuery.isError ||
    resultsQuery.data?.pages.some((page) => !page.success) === true
  const notificationsFailed =
    notificationsQuery.isError ||
    notificationsQuery.data?.pages.some((page) => !page.success) === true
  const selectedPlan = search.plan
    ? (plans.find((plan) => plan.id === search.plan) ?? null)
    : null

  const selectPlan = (plan: LotteryPlan): void => {
    void navigate({
      search: (previous) => ({ ...previous, plan: plan.id }),
    })
  }

  const handleJoin = (plan: LotteryPlan): void => {
    joinMutation.mutate(plan.id)
  }

  const handleLeave = (plan: LotteryPlan): void => {
    leaveMutation.mutate(plan.id)
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Lotteries')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='grid gap-3'>
          {plansQuery.isLoading && (
            <div className='text-muted-foreground flex items-center justify-center gap-2 border border-dashed p-8 text-sm'>
              <Spinner className='size-4' />
              {t('Loading...')}
            </div>
          )}
          {plansFailed && (
            <div className='text-muted-foreground border border-dashed p-8 text-center text-sm'>
              {t('Request failed')}
            </div>
          )}
          {!plansQuery.isLoading &&
            !plansFailed &&
            plans.map((plan) => {
              const isOpen = plan.status === 'open'
              const pending = joinMutation.isPending || leaveMutation.isPending
              return (
                <section
                  key={plan.id}
                  className='bg-card flex flex-col gap-4 rounded-lg border p-4 sm:flex-row sm:items-center sm:justify-between'
                >
                  <div className='flex min-w-0 items-start gap-3'>
                    <LotteryIcon src={plan.icon} size='md' />
                    <div className='min-w-0 space-y-1.5'>
                      <div className='flex flex-wrap items-center gap-2'>
                        <h2 className='truncate text-base font-semibold'>
                          {plan.title}
                        </h2>
                        <span className='text-muted-foreground text-xs'>
                          {getLotteryPlanStatusLabel(t, plan.status)}
                        </span>
                      </div>
                      {plan.description && (
                        <p className='text-muted-foreground line-clamp-2 text-sm'>
                          {plan.description}
                        </p>
                      )}
                      <div className='text-muted-foreground flex flex-wrap items-center gap-x-4 gap-y-1 text-xs'>
                        <span className='flex items-center gap-1.5'>
                          <CalendarClock
                            className='size-3.5'
                            aria-hidden='true'
                          />
                          {t('Draw time')}: {planTime(plan.draw_time)}
                        </span>
                        <span className='flex items-center gap-1.5'>
                          <Users className='size-3.5' aria-hidden='true' />
                          {plan.participant_count ?? 0}/{plan.max_participants}
                        </span>
                        <span className='flex items-center gap-1.5'>
                          <Trophy className='size-3.5' aria-hidden='true' />
                          {plan.winner_count ?? 0}
                        </span>
                      </div>
                    </div>
                  </div>
                  <div className='flex shrink-0 flex-wrap gap-2'>
                    <Button
                      size='sm'
                      variant='outline'
                      onClick={() => selectPlan(plan)}
                    >
                      <Eye data-icon='inline-start' />
                      {t('View details')}
                    </Button>
                    {isOpen &&
                      (plan.joined ? (
                        <Button
                          size='sm'
                          variant='outline'
                          disabled={pending}
                          onClick={() => handleLeave(plan)}
                        >
                          <LogOut data-icon='inline-start' />
                          {t('Leave lottery')}
                        </Button>
                      ) : (
                        <Button
                          size='sm'
                          disabled={pending}
                          onClick={() => handleJoin(plan)}
                        >
                          <LogIn data-icon='inline-start' />
                          {t('Join lottery')}
                        </Button>
                      ))}
                  </div>
                </section>
              )
            })}
          {!plansQuery.isLoading && !plansFailed && plans.length === 0 && (
            <div className='text-muted-foreground border border-dashed p-8 text-center text-sm'>
              {t('No lottery plans are available')}
            </div>
          )}

          {notificationsQuery.isLoading && (
            <div className='text-muted-foreground flex items-center justify-center gap-2 text-sm'>
              <Spinner className='size-4' />
              {t('Loading...')}
            </div>
          )}
          {notificationsFailed && (
            <div className='text-muted-foreground border border-dashed p-4 text-center text-sm'>
              {t('Request failed')}
            </div>
          )}
          {!notificationsFailed && notifications.length > 0 && (
            <section className='bg-card rounded-lg border p-4'>
              <div className='mb-3 flex flex-wrap items-center justify-between gap-2'>
                <div className='flex items-center gap-2 text-sm font-semibold'>
                  <Bell className='size-4' aria-hidden='true' />
                  {t('Lottery notifications')}
                  <span className='text-muted-foreground text-xs font-normal'>
                    {t('{{count}} lottery result notifications', {
                      count: unreadTotal,
                    })}
                  </span>
                </div>
                <Button
                  size='sm'
                  variant='outline'
                  disabled={markNotificationsReadMutation.isPending}
                  onClick={() =>
                    markNotificationsReadMutation.mutate(
                      notifications.map((notification) => notification.id)
                    )
                  }
                >
                  {t('Mark displayed as read')}
                </Button>
              </div>
              <div className='grid gap-2'>
                {notifications.map((notification) => (
                  <div
                    key={notification.id}
                    className='bg-muted/35 flex items-center justify-between gap-3 rounded-md px-3 py-2'
                  >
                    <span className='min-w-0 truncate text-sm'>
                      {notification.type === 'lottery_result'
                        ? t('Lottery result available')
                        : notification.content}
                    </span>
                    <span className='text-muted-foreground shrink-0 text-xs'>
                      {planTime(notification.created_at)}
                    </span>
                  </div>
                ))}
              </div>
              {notificationsQuery.hasNextPage && (
                <Button
                  size='sm'
                  variant='outline'
                  className='mt-3 w-full'
                  disabled={notificationsQuery.isFetchingNextPage}
                  onClick={() => notificationsQuery.fetchNextPage()}
                >
                  {t('Load more')}
                </Button>
              )}
            </section>
          )}

          {resultsQuery.isLoading && (
            <div className='text-muted-foreground flex items-center justify-center gap-2 text-sm'>
              <Spinner className='size-4' />
              {t('Loading...')}
            </div>
          )}
          {resultsFailed && (
            <div className='text-muted-foreground border border-dashed p-4 text-center text-sm'>
              {t('Request failed')}
            </div>
          )}
          {!resultsQuery.isLoading &&
            !resultsFailed &&
            results.length === 0 && (
              <div className='text-muted-foreground border border-dashed p-8 text-center text-sm'>
                {t('No lottery results yet')}
              </div>
            )}
          {!resultsFailed &&
            results.map((result) => (
              <section
                key={result.id}
                className='bg-card flex items-center justify-between gap-3 rounded-lg border p-4'
              >
                <span className='flex min-w-0 items-center gap-2 text-sm'>
                  <Gift className='size-4 shrink-0' aria-hidden='true' />
                  <span className='truncate'>
                    {t('Lottery reward')} #{result.id}
                  </span>
                </span>
                {result.claimable ? (
                  <Button
                    size='sm'
                    disabled={claimMutation.isPending}
                    onClick={() => claimMutation.mutate(result.id)}
                  >
                    {t('Claim lottery reward')}
                  </Button>
                ) : (
                  <span className='text-muted-foreground max-w-52 truncate font-mono text-xs'>
                    {result.redemption_code ||
                      (result.claim_status === 'expired'
                        ? t('Claim expired')
                        : getLotteryRewardStatusLabel(
                            t,
                            result.fulfillment_status
                          ))}
                  </span>
                )}
              </section>
            ))}
          {!resultsFailed && resultsQuery.hasNextPage && (
            <Button
              size='sm'
              variant='outline'
              className='w-full'
              disabled={resultsQuery.isFetchingNextPage}
              onClick={() => resultsQuery.fetchNextPage()}
            >
              {t('Load more')}
            </Button>
          )}
        </div>
      </SectionPageLayout.Content>
      <LotteryUserDetailsDrawer
        open={selectedPlan !== null}
        plan={selectedPlan}
        onOpenChange={(open) => {
          if (!open) {
            void navigate({
              search: (previous) => ({ ...previous, plan: undefined }),
            })
          }
        }}
      />
    </SectionPageLayout>
  )
}
