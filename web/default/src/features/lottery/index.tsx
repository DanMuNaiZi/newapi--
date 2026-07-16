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
import dayjs from 'dayjs'
import {
  CalendarClock,
  Eye,
  Gift,
  LogIn,
  LogOut,
  Trophy,
  Users,
} from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'

import {
  claimLotteryResult,
  getLotteryPlansForSelf,
  getLotteryResultsForSelf,
  joinLotteryPlan,
  leaveLotteryPlan,
} from './api'
import { LotteryIcon } from './components/lottery-icon'
import { LotteryUserDetailsDrawer } from './components/lottery-user-details-drawer'
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
  const queryClient = useQueryClient()
  const [selectedPlan, setSelectedPlan] = useState<LotteryPlan | null>(null)
  const plansQuery = useQuery({
    queryKey: LOTTERY_QUERY_KEY,
    queryFn: getLotteryPlansForSelf,
    refetchInterval: 30_000,
  })
  const resultsQuery = useQuery({
    queryKey: ['lottery', 'results'],
    queryFn: getLotteryResultsForSelf,
    refetchInterval: 30_000,
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
      await queryClient.invalidateQueries({ queryKey: ['lottery', 'results'] })
    },
  })
  const plans = plansQuery.data?.success ? plansQuery.data.data : []

  const handleJoin = (plan: LotteryPlan): void => {
    joinMutation.mutate(plan.id)
  }

  const handleLeave = (plan: LotteryPlan): void => {
    leaveMutation.mutate(plan.id)
  }

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>{t('Lotteries')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='grid gap-3'>
          {plans.map((plan) => {
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
                    onClick={() => setSelectedPlan(plan)}
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
          {plansQuery.isFetched && plans.length === 0 && (
            <div className='text-muted-foreground border border-dashed p-8 text-center text-sm'>
              {t('No lottery plans are available')}
            </div>
          )}

          {(resultsQuery.data?.success ? resultsQuery.data.data : []).map(
            (result) => {
              const canClaim =
                result.fulfillment_mode === 'self_claim' &&
                result.fulfillment_status !== 'fulfilled'
              return (
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
                  {canClaim ? (
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
                        getLotteryRewardStatusLabel(
                          t,
                          result.fulfillment_status
                        )}
                    </span>
                  )}
                </section>
              )
            }
          )}
        </div>
      </SectionPageLayout.Content>
      <LotteryUserDetailsDrawer
        open={selectedPlan !== null}
        plan={selectedPlan}
        onOpenChange={(open) => {
          if (!open) setSelectedPlan(null)
        }}
      />
    </SectionPageLayout>
  )
}
