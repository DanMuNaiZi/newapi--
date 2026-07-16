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
import { useQuery } from '@tanstack/react-query'
import dayjs from 'dayjs'
import { useTranslation } from 'react-i18next'

import {
  sideDrawerContentClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { StatusBadge } from '@/components/status-badge'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Spinner } from '@/components/ui/spinner'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import {
  getLotteryParticipantsForSelf,
  getLotteryPlanResultsForSelf,
} from '../api'
import { getLotteryPlanStatusLabel } from '../lib/status'
import type { LotteryPlan } from '../types'
import { LotteryIcon } from './lottery-icon'

type LotteryUserDetailsDrawerProps = {
  open: boolean
  plan: LotteryPlan | null
  onOpenChange: (open: boolean) => void
}

export function LotteryUserDetailsDrawer(props: LotteryUserDetailsDrawerProps) {
  const { t } = useTranslation()
  const planId = props.plan?.id ?? 0
  const participantsQuery = useQuery({
    queryKey: ['lottery', 'self', 'participants', planId],
    queryFn: () => getLotteryParticipantsForSelf(planId),
    enabled: props.open && planId > 0,
    refetchInterval: props.open ? 30_000 : false,
  })
  const resultsQuery = useQuery({
    queryKey: ['lottery', 'self', 'plan-results', planId],
    queryFn: () => getLotteryPlanResultsForSelf(planId),
    enabled: props.open && planId > 0,
    refetchInterval: props.open ? 30_000 : false,
  })
  const participants = participantsQuery.data?.success
    ? (participantsQuery.data.data ?? [])
    : []
  const results = resultsQuery.data?.success
    ? (resultsQuery.data.data ?? [])
    : []

  return (
    <Sheet open={props.open} onOpenChange={props.onOpenChange}>
      <SheetContent className={sideDrawerContentClassName('sm:max-w-[720px]')}>
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
                    variant={
                      props.plan.status === 'open' ? 'success' : 'neutral'
                    }
                    copyable={false}
                  />
                )}
              </div>
              <SheetDescription className='mt-1 line-clamp-2'>
                {props.plan?.description || t('No description')}
              </SheetDescription>
            </div>
          </div>
        </SheetHeader>

        <Tabs defaultValue='overview' className='min-h-0 flex-1 gap-0'>
          <div className='border-border/70 shrink-0 overflow-x-auto border-b px-4 sm:px-6'>
            <TabsList variant='line'>
              <TabsTrigger value='overview'>{t('Overview')}</TabsTrigger>
              <TabsTrigger value='participants'>
                {t('Participants')} ({participants.length})
              </TabsTrigger>
              <TabsTrigger value='results'>
                {t('Lottery results')} ({results.length})
              </TabsTrigger>
            </TabsList>
          </div>

          <TabsContent
            value='overview'
            className='min-h-0 overflow-y-auto px-4 py-5 sm:px-6'
          >
            {props.plan && (
              <dl className='border-border/70 grid border-y text-sm sm:grid-cols-2'>
                <div className='border-border/70 px-1 py-3 sm:border-r sm:pr-4'>
                  <dt className='text-muted-foreground text-xs'>
                    {t('Registration start time')}
                  </dt>
                  <dd className='mt-1 font-medium'>
                    {dayjs
                      .unix(props.plan.registration_start_time)
                      .format('YYYY-MM-DD HH:mm')}
                  </dd>
                </div>
                <div className='border-border/70 px-1 py-3 sm:pl-4'>
                  <dt className='text-muted-foreground text-xs'>
                    {t('Draw time')}
                  </dt>
                  <dd className='mt-1 font-medium'>
                    {dayjs
                      .unix(props.plan.draw_time)
                      .format('YYYY-MM-DD HH:mm')}
                  </dd>
                </div>
                <div className='border-border/70 border-t px-1 py-3 sm:border-r sm:pr-4'>
                  <dt className='text-muted-foreground text-xs'>
                    {t('Participants')}
                  </dt>
                  <dd className='mt-1 font-medium'>{participants.length}</dd>
                </div>
                <div className='border-border/70 border-t px-1 py-3 sm:pl-4'>
                  <dt className='text-muted-foreground text-xs'>
                    {t('Winner count')}
                  </dt>
                  <dd className='mt-1 font-medium'>{results.length}</dd>
                </div>
              </dl>
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
                  <div
                    key={`${participant.username}-${participant.joined_at}`}
                    className='flex items-center justify-between gap-3 py-3'
                  >
                    <div className='min-w-0'>
                      <div className='truncate text-sm font-medium'>
                        {participant.display_name || participant.username}
                      </div>
                      {participant.display_name && (
                        <div className='text-muted-foreground truncate text-xs'>
                          @{participant.username}
                        </div>
                      )}
                    </div>
                    <time className='text-muted-foreground shrink-0 text-xs'>
                      {dayjs.unix(participant.joined_at).format('MM-DD HH:mm')}
                    </time>
                  </div>
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
                {results.map((result) => (
                  <div
                    key={`${result.username}-${result.created_at}`}
                    className='flex items-center justify-between gap-3 py-3'
                  >
                    <div className='min-w-0'>
                      <div className='truncate text-sm font-medium'>
                        {result.display_name || result.username}
                      </div>
                      {result.display_name && (
                        <div className='text-muted-foreground truncate text-xs'>
                          @{result.username}
                        </div>
                      )}
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
          </TabsContent>
        </Tabs>
      </SheetContent>
    </Sheet>
  )
}
