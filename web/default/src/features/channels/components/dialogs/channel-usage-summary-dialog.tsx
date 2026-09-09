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
import { AlertTriangle, BarChart3, CheckCircle2 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import dayjs from '@/lib/dayjs'
import { formatCompactNumber, formatNumber, formatQuota } from '@/lib/format'

import { getChannelUsageSummary } from '../../api'
import type { ChannelUsagePeriod, ChannelUsageSummary } from '../../types'
import { useChannels } from '../channels-provider'

type ChannelUsageSummaryDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

const PERIODS: Array<{ value: ChannelUsagePeriod; label: string }> = [
  { value: 'day', label: 'Last 24 hours' },
  { value: 'week', label: 'Last 7 days' },
  { value: 'month', label: 'Last 30 days' },
]

function UsageSummaryContent({ summary }: { summary: ChannelUsageSummary }) {
  const { t } = useTranslation()
  const totalTokens = summary.prompt_tokens + summary.completion_tokens
  const successRate =
    summary.request_count > 0
      ? (summary.success_count / summary.request_count) * 100
      : 0
  const maxQuota = Math.max(
    1,
    ...summary.trend.flatMap((point) => [
      point.resource_quota,
      point.charged_quota,
    ])
  )

  const cards = [
    { label: t('Resource quota'), value: formatQuota(summary.resource_quota) },
    {
      label: t('User charged quota'),
      value: formatQuota(summary.charged_quota),
    },
    { label: t('Total tokens'), value: formatCompactNumber(totalTokens) },
    {
      label: t('Success rate'),
      value: `${formatNumber(successRate)}%`,
    },
  ]

  return (
    <div className='space-y-4'>
      {!summary.resource_coverage_complete && (
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
        {cards.map((card) => (
          <Card key={card.label} size='sm'>
            <CardHeader>
              <CardTitle className='text-muted-foreground text-xs font-medium'>
                {card.label}
              </CardTitle>
            </CardHeader>
            <CardContent className='text-xl font-semibold tabular-nums'>
              {card.value}
            </CardContent>
          </Card>
        ))}
      </div>

      <div className='rounded-xl border p-3 sm:p-4'>
        <div className='mb-3 flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between'>
          <div className='flex items-center gap-2 font-medium'>
            <BarChart3 className='size-4' />
            {t('Usage trend')}
          </div>
          <div className='text-muted-foreground text-xs'>
            {t(
              '{{success}} successful · {{failure}} failed · {{requests}} requests',
              {
                success: formatNumber(summary.success_count),
                failure: formatNumber(summary.failure_count),
                requests: formatNumber(summary.request_count),
              }
            )}
          </div>
        </div>

        {summary.request_count === 0 ? (
          <Empty className='min-h-44 border-0'>
            <EmptyHeader>
              <EmptyTitle>{t('No usage in this period')}</EmptyTitle>
              <EmptyDescription>
                {t('Choose another time range or check again later.')}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <div className='space-y-2'>
            {summary.trend.map((point) => {
              const label =
                summary.period === 'day'
                  ? dayjs.unix(point.start_time).format('MM-DD HH:mm')
                  : dayjs.unix(point.start_time).format('MM-DD')
              return (
                <div
                  key={point.start_time}
                  className='grid grid-cols-[5.25rem_1fr_auto] items-center gap-2 text-xs'
                >
                  <span className='text-muted-foreground tabular-nums'>
                    {label}
                  </span>
                  <div className='space-y-1'>
                    <div className='bg-muted h-1.5 overflow-hidden rounded-full'>
                      <div
                        className='bg-primary h-full rounded-full'
                        style={{
                          width: `${Math.max(0, (point.resource_quota / maxQuota) * 100)}%`,
                        }}
                      />
                    </div>
                    <div className='bg-muted h-1.5 overflow-hidden rounded-full'>
                      <div
                        className='bg-secondary-foreground/50 h-full rounded-full'
                        style={{
                          width: `${Math.max(0, (point.charged_quota / maxQuota) * 100)}%`,
                        }}
                      />
                    </div>
                  </div>
                  <span className='w-7 text-end tabular-nums'>
                    {point.request_count}
                  </span>
                </div>
              )
            })}
          </div>
        )}
      </div>

      <div className='text-muted-foreground flex flex-wrap items-center gap-x-4 gap-y-1 text-xs'>
        <span className='inline-flex items-center gap-1.5'>
          <span className='bg-primary size-2 rounded-full' />
          {t('Resource quota')}
        </span>
        <span className='inline-flex items-center gap-1.5'>
          <span className='bg-secondary-foreground/50 size-2 rounded-full' />
          {t('User charged quota')}
        </span>
        {summary.earliest_available_time > 0 && (
          <span className='ml-auto'>
            {t('Earliest retained log: {{time}}', {
              time: dayjs
                .unix(summary.earliest_available_time)
                .format('YYYY-MM-DD HH:mm'),
            })}
          </span>
        )}
      </div>
    </div>
  )
}

export function ChannelUsageSummaryDialog({
  open,
  onOpenChange,
}: ChannelUsageSummaryDialogProps) {
  const { t } = useTranslation()
  const { currentRow } = useChannels()
  const [period, setPeriod] = useState<ChannelUsagePeriod>('month')
  const channelId = currentRow?.id
  const query = useQuery({
    queryKey: ['channels', 'usage-summary', channelId, period],
    enabled: open && channelId != null,
    queryFn: async () => {
      if (channelId == null) {
        throw new Error(t('Failed to fetch usage report'))
      }
      const response = await getChannelUsageSummary(channelId, period)
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
      contentHeight='min(36rem, calc(100vh - 12rem))'
      contentClassName='sm:max-w-2xl'
      bodyClassName='space-y-4'
      footer={
        <Button variant='outline' onClick={() => onOpenChange(false)}>
          {t('Close')}
        </Button>
      }
    >
      <ToggleGroup
        value={[period]}
        onValueChange={(value) => {
          const nextPeriod = value.find((item) => item !== period)
          if (
            nextPeriod === 'day' ||
            nextPeriod === 'week' ||
            nextPeriod === 'month'
          ) {
            setPeriod(nextPeriod)
          }
        }}
        variant='outline'
        size='sm'
        spacing={2}
        className='grid w-full grid-cols-3'
        aria-label={t('Usage report period')}
      >
        {PERIODS.map((item) => (
          <ToggleGroupItem
            key={item.value}
            value={item.value}
            className='w-full'
          >
            {t(item.label)}
          </ToggleGroupItem>
        ))}
      </ToggleGroup>

      {query.isLoading && (
        <div className='space-y-3'>
          <div className='grid gap-3 sm:grid-cols-2'>
            {Array.from({ length: 4 }, (_, index) => (
              <Skeleton key={index} className='h-24 rounded-xl' />
            ))}
          </div>
          <Skeleton className='h-72 rounded-xl' />
        </div>
      )}

      {query.isError && (
        <Alert variant='destructive'>
          <AlertTriangle />
          <AlertTitle>{t('Failed to fetch usage report')}</AlertTitle>
          <AlertDescription className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
            <span>{query.error.message}</span>
            <Button size='sm' variant='outline' onClick={() => query.refetch()}>
              {t('Retry')}
            </Button>
          </AlertDescription>
        </Alert>
      )}

      {query.data && <UsageSummaryContent summary={query.data} />}

      {query.data?.resource_coverage_complete &&
        query.data.request_count > 0 && (
          <div className='text-muted-foreground flex items-center gap-1.5 text-xs'>
            <CheckCircle2 className='text-success size-3.5' />
            {t('Resource quota coverage is complete for this period.')}
          </div>
        )}
    </Dialog>
  )
}
