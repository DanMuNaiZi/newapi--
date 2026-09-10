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
import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import { AxiosError } from 'axios'
import dayjs from 'dayjs'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  sideDrawerContentClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Spinner } from '@/components/ui/spinner'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useAuthStore } from '@/stores/auth-store'

import {
  getLotteryParticipantsPageForSelf,
  getLotteryPlanForSelf,
  getLotteryPlanResultsPageForSelf,
} from '../api'
import { lotteryQueryKeys } from '../lib/query-keys'
import { getLotteryPlanStatusLabel } from '../lib/status'
import { LotteryIcon } from './lottery-icon'

type LotteryUserDetailsDrawerProps = {
  open: boolean
  planId: number | null
  onOpenChange: (open: boolean) => void
}

type DetailsTab = 'overview' | 'participants' | 'results'

function errorRequestId(error: unknown): string | undefined {
  if (!(error instanceof AxiosError)) return undefined
  const data = error.response?.data as { request_id?: string } | undefined
  return data?.request_id
}

function QueryFailure(props: {
  error: unknown
  onRetry: () => void
  label: string
}) {
  const { t } = useTranslation()
  const requestId = errorRequestId(props.error)
  return (
    <div className='flex flex-col items-center gap-3 py-12 text-center text-sm'>
      <p className='text-muted-foreground'>{props.label}</p>
      {requestId && (
        <p className='text-muted-foreground font-mono text-xs'>
          {t('Request ID')}: {requestId}
        </p>
      )}
      <Button size='sm' variant='outline' onClick={props.onRetry}>
        {t('Retry')}
      </Button>
    </div>
  )
}

export function LotteryUserDetailsDrawer(props: LotteryUserDetailsDrawerProps) {
  const { t } = useTranslation()
  const planId = props.planId ?? 0
  const userId = useAuthStore((state) => state.auth.user?.id ?? 0)
  const [activeTab, setActiveTab] = useState<DetailsTab>('overview')

  const planQuery = useQuery({
    queryKey: lotteryQueryKeys.userPlan(userId, planId),
    queryFn: () => getLotteryPlanForSelf(planId),
    enabled: props.open && planId > 0 && userId > 0,
    refetchInterval: props.open ? 30_000 : false,
    meta: { errorMode: 'local' },
  })
  const participantsQuery = useInfiniteQuery({
    queryKey: lotteryQueryKeys.userParticipants(userId, planId),
    initialPageParam: '',
    queryFn: ({ pageParam }) =>
      getLotteryParticipantsPageForSelf(planId, pageParam),
    getNextPageParam: (lastPage) => {
      if (!lastPage.success || !lastPage.data.has_more) return undefined
      return lastPage.data.next_cursor
    },
    enabled:
      props.open && planId > 0 && userId > 0 && activeTab === 'participants',
    meta: { errorMode: 'local' },
  })
  const resultsQuery = useInfiniteQuery({
    queryKey: lotteryQueryKeys.userResults(userId, planId),
    initialPageParam: '',
    queryFn: ({ pageParam }) =>
      getLotteryPlanResultsPageForSelf(planId, pageParam),
    getNextPageParam: (lastPage) => {
      if (!lastPage.success || !lastPage.data.has_more) return undefined
      return lastPage.data.next_cursor
    },
    enabled: props.open && planId > 0 && userId > 0 && activeTab === 'results',
    meta: { errorMode: 'local' },
  })

  const plan = planQuery.data?.success ? planQuery.data.data : null
  const participants =
    participantsQuery.data?.pages.flatMap((page) =>
      page.success ? page.data.items : []
    ) ?? []
  const results =
    resultsQuery.data?.pages.flatMap((page) =>
      page.success ? page.data.items : []
    ) ?? []
  const participantsFailed =
    participantsQuery.isError ||
    participantsQuery.data?.pages.some((page) => !page.success) === true
  const resultsFailed =
    resultsQuery.isError ||
    resultsQuery.data?.pages.some((page) => !page.success) === true

  return (
    <Sheet open={props.open} onOpenChange={props.onOpenChange}>
      <SheetContent className={sideDrawerContentClassName('sm:max-w-[720px]')}>
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <div className='flex items-start gap-3 pr-8'>
            <LotteryIcon src={plan?.icon} size='md' />
            <div className='min-w-0 flex-1'>
              <div className='flex flex-wrap items-center gap-2'>
                <SheetTitle>{plan?.title ?? t('Lottery details')}</SheetTitle>
                {plan && (
                  <StatusBadge
                    label={getLotteryPlanStatusLabel(t, plan.status)}
                    variant={plan.status === 'open' ? 'success' : 'neutral'}
                    copyable={false}
                  />
                )}
              </div>
              <SheetDescription className='mt-1 line-clamp-2'>
                {plan?.description || t('No description')}
              </SheetDescription>
            </div>
          </div>
        </SheetHeader>

        {planQuery.isLoading && (
          <div className='flex flex-1 items-center justify-center'>
            <Spinner />
          </div>
        )}
        {(planQuery.isError || planQuery.data?.success === false) && (
          <QueryFailure
            error={planQuery.error}
            label={t('Failed to load lottery details')}
            onRetry={() => void planQuery.refetch()}
          />
        )}

        {plan && (
          <Tabs
            value={activeTab}
            onValueChange={(value) => setActiveTab(value as DetailsTab)}
            className='min-h-0 flex-1 gap-0'
          >
            <div className='border-border/70 shrink-0 overflow-x-auto border-b px-4 sm:px-6'>
              <TabsList variant='line'>
                <TabsTrigger value='overview'>{t('Overview')}</TabsTrigger>
                <TabsTrigger value='participants'>
                  {t('Participants')} ({plan.participant_count ?? 0})
                </TabsTrigger>
                <TabsTrigger value='results'>
                  {t('Lottery results')} ({plan.winner_count ?? 0})
                </TabsTrigger>
              </TabsList>
            </div>

            <TabsContent
              value='overview'
              className='min-h-0 overflow-y-auto px-4 py-5 sm:px-6'
            >
              <dl className='border-border/70 grid border-y text-sm sm:grid-cols-2'>
                <div className='border-border/70 px-1 py-3 sm:border-r sm:pr-4'>
                  <dt className='text-muted-foreground text-xs'>
                    {t('Registration start time')}
                  </dt>
                  <dd className='mt-1 font-medium'>
                    {dayjs
                      .unix(plan.registration_start_time)
                      .format('YYYY-MM-DD HH:mm')}
                  </dd>
                </div>
                <div className='border-border/70 px-1 py-3 sm:pl-4'>
                  <dt className='text-muted-foreground text-xs'>
                    {t('Draw time')}
                  </dt>
                  <dd className='mt-1 font-medium'>
                    {dayjs.unix(plan.draw_time).format('YYYY-MM-DD HH:mm')}
                  </dd>
                </div>
                <div className='border-border/70 border-t px-1 py-3 sm:border-r sm:pr-4'>
                  <dt className='text-muted-foreground text-xs'>
                    {t('Participants')}
                  </dt>
                  <dd className='mt-1 font-medium'>
                    {plan.participant_count ?? 0}
                  </dd>
                </div>
                <div className='border-border/70 border-t px-1 py-3 sm:pl-4'>
                  <dt className='text-muted-foreground text-xs'>
                    {t('Winner count')}
                  </dt>
                  <dd className='mt-1 font-medium'>{plan.winner_count ?? 0}</dd>
                </div>
              </dl>
            </TabsContent>

            <TabsContent
              value='participants'
              className='min-h-0 overflow-y-auto px-4 py-5 sm:px-6'
            >
              {participantsQuery.isLoading && (
                <div className='flex justify-center py-12'>
                  <Spinner />
                </div>
              )}
              {participantsFailed && (
                <QueryFailure
                  error={participantsQuery.error}
                  label={t('Failed to load lottery participants')}
                  onRetry={() => void participantsQuery.refetch()}
                />
              )}
              {!participantsQuery.isLoading &&
                !participantsFailed &&
                participants.length === 0 && (
                  <div className='text-muted-foreground py-12 text-center text-sm'>
                    {t('No participants yet')}
                  </div>
                )}
              {!participantsFailed && participants.length > 0 && (
                <div className='border-border/70 divide-border/70 divide-y border-y'>
                  {participants.map((participant) => (
                    <div
                      key={participant.id}
                      className='flex items-center justify-between gap-3 py-3'
                    >
                      <div className='min-w-0 truncate text-sm font-medium'>
                        {participant.is_self
                          ? t('Me')
                          : participant.display_name || participant.username}
                      </div>
                      <time className='text-muted-foreground shrink-0 text-xs'>
                        {dayjs
                          .unix(participant.joined_at)
                          .format('MM-DD HH:mm')}
                      </time>
                    </div>
                  ))}
                </div>
              )}
              {!participantsFailed && participantsQuery.hasNextPage && (
                <Button
                  size='sm'
                  variant='outline'
                  className='mt-4 w-full'
                  disabled={participantsQuery.isFetchingNextPage}
                  onClick={() => void participantsQuery.fetchNextPage()}
                >
                  {t('Load more')}
                </Button>
              )}
            </TabsContent>

            <TabsContent
              value='results'
              className='min-h-0 overflow-y-auto px-4 py-5 sm:px-6'
            >
              {resultsQuery.isLoading && (
                <div className='flex justify-center py-12'>
                  <Spinner />
                </div>
              )}
              {resultsFailed && (
                <QueryFailure
                  error={resultsQuery.error}
                  label={t('Failed to load lottery results')}
                  onRetry={() => void resultsQuery.refetch()}
                />
              )}
              {!resultsQuery.isLoading &&
                !resultsFailed &&
                results.length === 0 && (
                  <div className='text-muted-foreground py-12 text-center text-sm'>
                    {t('No lottery results yet')}
                  </div>
                )}
              {!resultsFailed && results.length > 0 && (
                <div className='border-border/70 divide-border/70 divide-y border-y'>
                  {results.map((result) => (
                    <div
                      key={result.id}
                      className='flex items-center justify-between gap-3 py-3'
                    >
                      <div className='min-w-0 truncate text-sm font-medium'>
                        {result.is_self
                          ? t('Me')
                          : result.display_name || result.username}
                      </div>
                      <div className='min-w-0 text-right'>
                        <div className='truncate text-sm font-medium'>
                          {result.prize_name}
                        </div>
                        <div className='text-muted-foreground text-xs'>
                          {dayjs.unix(result.created_at).format('MM-DD HH:mm')}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
              {!resultsFailed && resultsQuery.hasNextPage && (
                <Button
                  size='sm'
                  variant='outline'
                  className='mt-4 w-full'
                  disabled={resultsQuery.isFetchingNextPage}
                  onClick={() => void resultsQuery.fetchNextPage()}
                >
                  {t('Load more')}
                </Button>
              )}
            </TabsContent>
          </Tabs>
        )}
      </SheetContent>
    </Sheet>
  )
}
