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
import { Link } from '@tanstack/react-router'
import dayjs from 'dayjs'
import { Gift, Timer } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

import { getLotteryPlansForSelf } from '../api'

export function LotteryDashboardPanel() {
  const { t } = useTranslation()
  const plansQuery = useQuery({
    queryKey: ['lottery', 'self'],
    queryFn: getLotteryPlansForSelf,
    refetchInterval: 30_000,
  })
  const plans = plansQuery.data?.success ? plansQuery.data.data : []

  if (plans.length === 0) return null

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
      <div className='grid gap-2'>
        {plans.slice(0, 3).map((plan) => (
          <div
            key={plan.id}
            className='bg-muted/35 flex items-center justify-between gap-3 rounded-md px-3 py-2'
          >
            <span className='min-w-0 truncate text-sm font-medium'>
              {plan.title}
            </span>
            <span className='text-muted-foreground flex shrink-0 items-center gap-1 text-xs'>
              <Timer className='size-3' aria-hidden='true' />
              {dayjs.unix(plan.draw_time).format('MM-DD HH:mm')}
            </span>
          </div>
        ))}
      </div>
    </section>
  )
}
