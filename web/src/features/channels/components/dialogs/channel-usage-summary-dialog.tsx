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
import { AlertTriangle, BarChart3 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import dayjs from '@/lib/dayjs'
import { formatCompactNumber, formatNumber, formatQuota } from '@/lib/format'

import { getChannelUsageSummary } from '../../api'
import type { ChannelUsagePeriod, ChannelUsageSummary } from '../../types'
import { useChannels } from '../channels-provider'

type Props = { open: boolean; onOpenChange: (open: boolean) => void }

const PERIODS: Array<{ value: ChannelUsagePeriod; label: string }> = [
  { value: 'day', label: 'Last 24 hours' },
  { value: 'week', label: 'Last 7 days' },
  { value: 'month', label: 'Last 30 days' },
]

function Summary({ value }: { value: ChannelUsageSummary }) {
  const { t } = useTranslation()
  const maxQuota = Math.max(
    1,
    ...value.trend.flatMap((point) => [
      point.resource_quota,
      point.charged_quota,
    ])
  )
  const successRate = value.request_count
    ? (value.success_count / value.request_count) * 100
    : 0
  const cards = [
    [t('Resource quota'), formatQuota(value.resource_quota)],
    [t('User charged quota'), formatQuota(value.charged_quota)],
    [
      t('Total tokens'),
      formatCompactNumber(value.prompt_tokens + value.completion_tokens),
    ],
    [t('Success rate'), `${formatNumber(successRate)}%`],
  ]
  return (
    <div className='space-y-4'>
      {!value.resource_coverage_complete && (
        <Alert>
          <AlertTriangle />
          <AlertTitle>{t('Resource quota coverage is incomplete')}</AlertTitle>
          <AlertDescription>
            {t(
              'Older logs do not contain resource quota. Request, token, and charged quota totals still include those logs.'
            )}
          </AlertDescription>
        </Alert>
      )}
      <div className='grid gap-3 sm:grid-cols-2'>
        {cards.map(([label, content]) => (
          <Card key={label} size='sm'>
            <CardHeader>
              <CardTitle className='text-muted-foreground text-xs font-medium'>
                {label}
              </CardTitle>
            </CardHeader>
            <CardContent className='text-xl font-semibold tabular-nums'>
              {content}
            </CardContent>
          </Card>
        ))}
      </div>
      <div className='rounded-xl border p-3'>
        <div className='mb-3 flex items-center gap-2 font-medium'>
          <BarChart3 className='size-4' />
          {t('Usage trend')}
        </div>
        <div className='space-y-2'>
          {value.trend.map((point) => (
            <div
              key={point.start_time}
              className='grid grid-cols-[6rem_1fr_auto] items-center gap-2 text-xs'
            >
              <span className='text-muted-foreground tabular-nums'>
                {dayjs
                  .unix(point.start_time)
                  .format(value.period === 'day' ? 'MM-DD HH:mm' : 'MM-DD')}
              </span>
              <div className='space-y-1'>
                <div className='bg-muted h-1.5 overflow-hidden rounded-full'>
                  <div
                    className='bg-primary h-full'
                    style={{
                      width: `${Math.max(0, (point.resource_quota / maxQuota) * 100)}%`,
                    }}
                  />
                </div>
                <div className='bg-muted h-1.5 overflow-hidden rounded-full'>
                  <div
                    className='bg-secondary-foreground/50 h-full'
                    style={{
                      width: `${Math.max(0, (point.charged_quota / maxQuota) * 100)}%`,
                    }}
                  />
                </div>
              </div>
              <span className='tabular-nums'>{point.request_count}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

export function ChannelUsageSummaryDialog({ open, onOpenChange }: Props) {
  const { t } = useTranslation()
  const { currentRow } = useChannels()
  const [period, setPeriod] = useState<ChannelUsagePeriod>('day')
  const query = useQuery({
    queryKey: ['channels', 'usage-summary', currentRow?.id, period],
    enabled: open && currentRow != null,
    queryFn: async () => {
      if (!currentRow) {
        throw new Error(t('No channel selected'))
      }
      const response = await getChannelUsageSummary(currentRow.id, period)
      if (!response.success || !response.data) {
        throw new Error(response.message || t('Failed to fetch usage report'))
      }
      return response.data
    },
  })
  if (!currentRow) return null
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Usage report')}
      description={t('Channel resource usage for {{name}}', {
        name: currentRow.name,
      })}
      contentClassName='sm:max-w-3xl'
      bodyClassName='space-y-4'
    >
      <ToggleGroup
        value={[period]}
        onValueChange={(values) => {
          const next = values[0]
          if (next === 'day' || next === 'week' || next === 'month') {
            setPeriod(next)
          }
        }}
        variant='outline'
        size='sm'
        className='grid w-full grid-cols-3'
        aria-label={t('Usage report period')}
      >
        {PERIODS.map((item) => (
          <ToggleGroupItem key={item.value} value={item.value}>
            {t(item.label)}
          </ToggleGroupItem>
        ))}
      </ToggleGroup>
      {query.isLoading && <Skeleton className='h-72' />}
      {query.isError && (
        <Alert variant='destructive'>
          <AlertTriangle />
          <AlertTitle>{t('Failed to fetch usage report')}</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
        </Alert>
      )}
      {query.data && <Summary value={query.data} />}
      <Button variant='outline' onClick={() => onOpenChange(false)}>
        {t('Close')}
      </Button>
    </Dialog>
  )
}
