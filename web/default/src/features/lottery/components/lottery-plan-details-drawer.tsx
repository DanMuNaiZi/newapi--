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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  sideDrawerContentClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { StatusBadge, type StatusVariant } from '@/components/status-badge'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Spinner } from '@/components/ui/spinner'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { formatQuota } from '@/lib/format'

import {
  getAdminLotteryResults,
  getLotteryParticipants,
  getLotteryPrizes,
  updateLotteryParticipant,
} from '../api'
import {
  getLotteryPlanStatusLabel,
  getLotteryRewardStatusLabel,
} from '../lib/status'
import type { LotteryPlan } from '../types'
import { LotteryIcon } from './lottery-icon'
import {
  LotteryParticipantEditor,
  type LotteryParticipantUpdate,
} from './lottery-participant-editor'

export type LotteryDetailsTab =
  | 'overview'
  | 'prizes'
  | 'participants'
  | 'results'

type LotteryPlanDetailsDrawerProps = {
  initialTab: LotteryDetailsTab
  open: boolean
  plan: LotteryPlan | null
  onOpenChange: (open: boolean) => void
}

function formatLotteryDate(value: number): string {
  return new Date(value * 1000).toLocaleString()
}

function getStatusVariant(status: LotteryPlan['status']): StatusVariant {
  switch (status) {
    case 'open':
      return 'success'
    case 'scheduled':
      return 'info'
    case 'drawing':
      return 'warning'
    case 'cancelled':
      return 'danger'
    default:
      return 'neutral'
  }
}

export function LotteryPlanDetailsDrawer(props: LotteryPlanDetailsDrawerProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [tab, setTab] = useState<LotteryDetailsTab>(props.initialTab)
  const planId = props.plan?.id ?? 0

  useEffect(() => {
    if (props.open) setTab(props.initialTab)
  }, [props.initialTab, props.open])

  const prizesQuery = useQuery({
    queryKey: ['lottery', 'admin', 'prizes', planId],
    queryFn: () => getLotteryPrizes(planId),
    enabled: props.open && planId > 0,
  })
  const participantsQuery = useQuery({
    queryKey: ['lottery', 'admin', 'participants', planId],
    queryFn: () => getLotteryParticipants(planId),
    enabled: props.open && planId > 0,
  })
  const resultsQuery = useQuery({
    queryKey: ['lottery', 'admin', 'results', planId],
    queryFn: () => getAdminLotteryResults(planId),
    enabled: props.open && planId > 0 && tab === 'results',
  })
  const participantMutation = useMutation({
    mutationFn: updateLotteryParticipant,
  })

  const updateParticipant = async (
    payload: LotteryParticipantUpdate
  ): Promise<boolean> => {
    try {
      const result = await participantMutation.mutateAsync(payload)
      if (!result.success) {
        toast.error(t('Failed to update lottery participant'))
      } else {
        toast.success(t('Lottery participant updated'))
      }
      await queryClient.invalidateQueries({
        queryKey: ['lottery', 'admin', 'participants', planId],
      })
      return result.success
    } catch {
      toast.error(t('Failed to update lottery participant'))
      await queryClient.invalidateQueries({
        queryKey: ['lottery', 'admin', 'participants', planId],
      })
      return false
    }
  }

  const prizes = prizesQuery.data?.success ? (prizesQuery.data.data ?? []) : []
  const participants = participantsQuery.data?.success
    ? (participantsQuery.data.data ?? [])
    : []
  const results = resultsQuery.data?.success
    ? (resultsQuery.data.data ?? [])
    : []

  const eligibilityLabel = (() => {
    switch (props.plan?.eligibility_mode) {
      case 'groups':
        return t('Selected groups')
      case 'users':
        return t('Selected users')
      default:
        return t('All users')
    }
  })()

  return (
    <Sheet open={props.open} onOpenChange={props.onOpenChange}>
      <SheetContent className={sideDrawerContentClassName('sm:max-w-[820px]')}>
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <div className='flex items-start gap-3 pr-8'>
            <LotteryIcon src={props.plan?.icon} size='md' />
            <div className='min-w-0 flex-1'>
              <div className='flex flex-wrap items-center gap-2'>
                <SheetTitle>
                  {props.plan?.title ?? t('Lottery details')}
                </SheetTitle>
                {props.plan && (
                  <StatusBadge
                    label={getLotteryPlanStatusLabel(t, props.plan.status)}
                    variant={getStatusVariant(props.plan.status)}
                    copyable={false}
                  />
                )}
              </div>
              <SheetDescription className='mt-1'>
                {t('Review plan settings, prizes, and participants.')}
              </SheetDescription>
            </div>
          </div>
        </SheetHeader>

        <Tabs
          value={tab}
          onValueChange={(value) => setTab(value as LotteryDetailsTab)}
          className='min-h-0 flex-1 gap-0'
        >
          <div className='border-border/70 shrink-0 overflow-x-auto border-b px-4 sm:px-6'>
            <TabsList variant='line'>
              <TabsTrigger value='overview'>{t('Overview')}</TabsTrigger>
              <TabsTrigger value='prizes'>{t('Prizes')}</TabsTrigger>
              <TabsTrigger value='participants'>
                {t('Participants')}
              </TabsTrigger>
              <TabsTrigger value='results'>{t('Lottery results')}</TabsTrigger>
            </TabsList>
          </div>

          <TabsContent
            value='overview'
            className='min-h-0 overflow-y-auto px-4 py-5 sm:px-6'
          >
            {props.plan && (
              <div className='space-y-6'>
                <section>
                  <h3 className='text-sm font-semibold'>
                    {t('Basic information')}
                  </h3>
                  <dl className='border-border/70 mt-3 grid border-y text-sm sm:grid-cols-2'>
                    <div className='border-border/70 px-1 py-3 sm:border-r sm:pr-4'>
                      <dt className='text-muted-foreground text-xs'>
                        {t('Plan ID')}
                      </dt>
                      <dd className='mt-1 font-medium'>#{props.plan.id}</dd>
                    </div>
                    <div className='border-border/70 px-1 py-3 sm:pl-4'>
                      <dt className='text-muted-foreground text-xs'>
                        {t('Maximum participants')}
                      </dt>
                      <dd className='mt-1 font-medium'>
                        {props.plan.max_participants}
                      </dd>
                    </div>
                    <div className='border-border/70 border-t px-1 py-3 sm:border-r sm:pr-4'>
                      <dt className='text-muted-foreground text-xs'>
                        {t('Registration start time')}
                      </dt>
                      <dd className='mt-1 font-medium'>
                        {formatLotteryDate(props.plan.registration_start_time)}
                      </dd>
                    </div>
                    <div className='border-border/70 border-t px-1 py-3 sm:pl-4'>
                      <dt className='text-muted-foreground text-xs'>
                        {t('Draw time')}
                      </dt>
                      <dd className='mt-1 font-medium'>
                        {formatLotteryDate(props.plan.draw_time)}
                      </dd>
                    </div>
                    <div className='border-border/70 border-t px-1 py-3 sm:border-r sm:pr-4'>
                      <dt className='text-muted-foreground text-xs'>
                        {t('Eligible participants')}
                      </dt>
                      <dd className='mt-1 font-medium'>{eligibilityLabel}</dd>
                    </div>
                    <div className='border-border/70 border-t px-1 py-3 sm:pl-4'>
                      <dt className='text-muted-foreground text-xs'>
                        {t('Prize count')}
                      </dt>
                      <dd className='mt-1 font-medium'>{prizes.length}</dd>
                    </div>
                  </dl>
                </section>
                <section>
                  <h3 className='text-sm font-semibold'>
                    {t('Lottery description')}
                  </h3>
                  <p className='text-muted-foreground mt-2 text-sm leading-6 whitespace-pre-wrap'>
                    {props.plan.description || t('No description')}
                  </p>
                </section>
              </div>
            )}
          </TabsContent>

          <TabsContent
            value='prizes'
            className='min-h-0 overflow-y-auto px-4 py-5 sm:px-6'
          >
            {prizesQuery.isLoading && (
              <div className='flex justify-center py-12'>
                <Spinner />
              </div>
            )}
            {!prizesQuery.isLoading && prizes.length === 0 && (
              <div className='text-muted-foreground py-12 text-center text-sm'>
                {t('No prizes')}
              </div>
            )}
            {!prizesQuery.isLoading && prizes.length > 0 && (
              <div className='border-border/70 divide-border/70 divide-y border-y'>
                {prizes.map((prize) => {
                  let deliveryLabel = t('Generate a redemption code')
                  if (prize.fulfillment_mode === 'auto') {
                    deliveryLabel = t('Automatic delivery')
                  } else if (prize.fulfillment_mode === 'self_claim') {
                    deliveryLabel = t('Winner claims manually')
                  }
                  const rewardLabel =
                    prize.reward_type === 'subscription'
                      ? t('Subscription #{{id}}', {
                          id: prize.subscription_plan_id,
                        })
                      : formatQuota(prize.quota)

                  return (
                    <article
                      key={prize.id}
                      className='grid gap-3 py-4 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center'
                    >
                      <div className='min-w-0'>
                        <div className='font-medium'>{prize.name}</div>
                        <div className='text-muted-foreground mt-1 text-xs'>
                          {rewardLabel}
                        </div>
                      </div>
                      <div className='text-muted-foreground grid grid-cols-2 gap-x-6 gap-y-1 text-xs'>
                        <span>{t('Winner count')}</span>
                        <span className='text-foreground text-right font-medium'>
                          {prize.quantity}
                        </span>
                        <span>{t('Delivery method')}</span>
                        <span className='text-foreground text-right font-medium'>
                          {deliveryLabel}
                        </span>
                      </div>
                    </article>
                  )
                })}
              </div>
            )}
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
            {!participantsQuery.isLoading && participants.length === 0 && (
              <div className='text-muted-foreground py-12 text-center text-sm'>
                {t('No participants yet')}
              </div>
            )}
            {!participantsQuery.isLoading && participants.length > 0 && (
              <div className='border-border/70 divide-border/70 divide-y border-y'>
                {participants.map((participant) => (
                  <LotteryParticipantEditor
                    key={participant.id}
                    participant={participant}
                    planId={planId}
                    prizes={prizes}
                    isPending={participantMutation.isPending}
                    onUpdate={updateParticipant}
                  />
                ))}
              </div>
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
            {!resultsQuery.isLoading && results.length === 0 && (
              <div className='text-muted-foreground py-12 text-center text-sm'>
                {t('No lottery results yet')}
              </div>
            )}
            {!resultsQuery.isLoading && results.length > 0 && (
              <div className='border-border/70 divide-border/70 divide-y border-y'>
                {results.map((result) => {
                  const winnerName =
                    result.display_name ||
                    result.username ||
                    `#${result.user_id}`
                  const rewardLabel =
                    result.reward_type === 'subscription'
                      ? t('Subscription #{{id}}', {
                          id: result.subscription_plan_id,
                        })
                      : formatQuota(result.quota)

                  return (
                    <article
                      key={result.id}
                      className='grid gap-3 py-4 sm:grid-cols-[minmax(0,1fr)_minmax(180px,auto)] sm:items-center'
                    >
                      <div className='min-w-0'>
                        <div className='truncate font-medium'>{winnerName}</div>
                        <div className='text-muted-foreground mt-1 text-xs'>
                          {result.display_name && result.username
                            ? `@${result.username} · `
                            : ''}
                          {t('User ID')} #{result.user_id}
                        </div>
                      </div>
                      <div className='min-w-0 text-sm sm:text-right'>
                        <div className='font-medium'>
                          {result.prize_name || `#${result.prize_id}`}
                        </div>
                        <div className='text-muted-foreground mt-1 text-xs'>
                          {rewardLabel} ·{' '}
                          {getLotteryRewardStatusLabel(
                            t,
                            result.fulfillment_status
                          )}
                        </div>
                        {result.redemption_code && (
                          <div className='mt-1 truncate font-mono text-xs'>
                            {result.redemption_code}
                          </div>
                        )}
                      </div>
                    </article>
                  )
                })}
              </div>
            )}
          </TabsContent>
        </Tabs>
      </SheetContent>
    </Sheet>
  )
}
