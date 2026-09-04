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
import { Link } from '@tanstack/react-router'
import dayjs from 'dayjs'
import { Bell, Eye, Gift, Timer, Trophy, Users } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'

import {
  claimLotteryResult,
  getClaimableLotteryResultsForSelf,
  getLotteryNotificationsPageForSelf,
  getLotteryPlansForSelf,
} from '../api'
import { LotteryIcon } from './lottery-icon'

export function LotteryDashboardPanel() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const plansQuery = useQuery({
    queryKey: ['lottery', 'self'],
    queryFn: getLotteryPlansForSelf,
    refetchInterval: 30_000,
  })
  const resultsQuery = useQuery({
    queryKey: ['lottery', 'results', 'pending'],
    queryFn: getClaimableLotteryResultsForSelf,
    refetchInterval: 30_000,
  })
  const notificationsQuery = useQuery({
    queryKey: ['lottery', 'notifications', 'page', 'unread'],
    queryFn: () => getLotteryNotificationsPageForSelf(undefined, 20, true),
    refetchInterval: 30_000,
  })
  const claimMutation = useMutation({
    mutationFn: ({ resultId }: { resultId: number }) =>
      claimLotteryResult(resultId),
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to claim lottery reward'))
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
    onError: () => toast.error(t('Failed to claim lottery reward')),
  })
  const plans = plansQuery.data?.success ? plansQuery.data.data : []
  const pendingClaims = resultsQuery.data?.success ? resultsQuery.data.data : []
  const unreadTotal = notificationsQuery.data?.success
    ? notificationsQuery.data.data.unread_total
    : 0
  const activePlans = plans.filter((plan) => {
    const participantCount = plan.participant_count ?? 0
    return (
      plan.status === 'open' &&
      !plan.joined &&
      participantCount < plan.max_participants
    )
  })
  if (plans.length === 0 && pendingClaims.length === 0 && unreadTotal === 0) {
    return null
  }

  return (
    <section className='bg-card rounded-lg border p-4'>
      <div className='mb-3 flex items-center justify-between gap-3'>
        <div className='flex items-center gap-2'>
          <Gift className='size-4' aria-hidden='true' />
          <h2 className='text-sm font-semibold'>{t('Lotteries')}</h2>
        </div>
        <Button size='sm' variant='outline' render={<Link to='/lotteries' />}>
          {t('View all')}
        </Button>
      </div>
      {(activePlans.length > 0 || unreadTotal > 0) && (
        <div className='mb-3 grid gap-2 sm:grid-cols-2'>
          {activePlans.length > 0 && (
            <Link
              to='/lotteries'
              className='border-primary/30 bg-primary/5 hover:bg-primary/10 flex items-start gap-2 rounded-md border px-3 py-2 text-left transition-colors'
            >
              <Gift
                className='text-primary mt-0.5 size-4 shrink-0'
                aria-hidden='true'
              />
              <span className='min-w-0'>
                <span className='block text-sm font-medium'>
                  {t('New lottery activities available')}
                </span>
                <span className='text-muted-foreground block text-xs'>
                  {t('{{count}} activities are open for participation', {
                    count: activePlans.length,
                  })}
                </span>
              </span>
            </Link>
          )}
          {unreadTotal > 0 && (
            <Link
              to='/lotteries'
              className='border-success/30 bg-success/5 hover:bg-success/10 flex items-start gap-2 rounded-md border px-3 py-2 text-left transition-colors'
            >
              <Bell
                className='text-success mt-0.5 size-4 shrink-0'
                aria-hidden='true'
              />
              <span className='min-w-0'>
                <span className='block text-sm font-medium'>
                  {t('Lottery result available')}
                </span>
                <span className='text-muted-foreground block text-xs'>
                  {t('{{count}} lottery result notifications', {
                    count: unreadTotal,
                  })}
                </span>
              </span>
            </Link>
          )}
        </div>
      )}
      <div className='grid gap-2'>
        {plans.slice(0, 3).map((plan) => (
          <Link
            key={plan.id}
            to='/lotteries'
            search={{ plan: plan.id }}
            className='bg-muted/35 flex items-center gap-3 rounded-md px-3 py-2'
          >
            <LotteryIcon src={plan.icon} size='sm' />
            <div className='min-w-0 flex-1'>
              <div className='truncate text-sm font-medium'>{plan.title}</div>
              <div className='text-muted-foreground mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs'>
                <span className='flex items-center gap-1'>
                  <Timer className='size-3' aria-hidden='true' />
                  {dayjs.unix(plan.draw_time).format('MM-DD HH:mm')}
                </span>
                <span className='flex items-center gap-1'>
                  <Users className='size-3' aria-hidden='true' />
                  {plan.participant_count ?? 0}
                </span>
                <span className='flex items-center gap-1'>
                  <Trophy className='size-3' aria-hidden='true' />
                  {plan.winner_count ?? 0}
                </span>
              </div>
            </div>
            <Eye
              className='text-muted-foreground size-4 shrink-0'
              aria-hidden='true'
            />
          </Link>
        ))}
      </div>
      {pendingClaims.length > 0 && (
        <div className='border-border/70 mt-3 border-t pt-3'>
          <div className='mb-2 flex items-center gap-2 text-sm font-medium'>
            <Trophy className='text-success size-4' aria-hidden='true' />
            {t('Rewards waiting for claim')}
          </div>
          <div className='grid gap-2'>
            {pendingClaims.slice(0, 3).map((result) => (
              <div
                key={result.id}
                className='bg-muted/35 flex items-center justify-between gap-3 rounded-md px-3 py-2'
              >
                <span className='truncate text-sm'>
                  {t('Lottery reward')} #{result.id}
                </span>
                <Button
                  size='sm'
                  disabled={claimMutation.isPending}
                  onClick={() =>
                    claimMutation.mutate({
                      resultId: result.id,
                    })
                  }
                >
                  {t('Claim lottery reward')}
                </Button>
              </div>
            ))}
          </div>
        </div>
      )}
    </section>
  )
}
