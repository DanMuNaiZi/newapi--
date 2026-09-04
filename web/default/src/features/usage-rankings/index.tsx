/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/
import { getRouteApi } from '@tanstack/react-router'
import { Crown, Medal, Users } from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatCompactNumber, formatNumber, formatQuota } from '@/lib/format'
import { cn } from '@/lib/utils'

import { useUsageRankings } from './hooks/use-usage-rankings'
import type {
  UsageRankingPeriod,
  UsageRankingRow,
  UsageRankingsSnapshot,
} from './types'

const route = getRouteApi('/_authenticated/usage-rankings/')
const PERIODS: { id: UsageRankingPeriod; labelKey: string }[] = [
  { id: 'today', labelKey: 'Today' },
  { id: 'week', labelKey: 'Week' },
  { id: 'month', labelKey: 'Month' },
  { id: 'year', labelKey: 'Year' },
]
const DISPLAY_COUNTS = [5, 10, 20, 50] as const

export function UsageRankings() {
  const { t } = useTranslation()
  const search = route.useSearch()
  const navigate = route.useNavigate()
  const period = PERIODS.some((item) => item.id === search.period)
    ? (search.period as UsageRankingPeriod)
    : 'week'
  const requestedLimit = search.limit ?? 10
  const limit = DISPLAY_COUNTS.includes(
    requestedLimit as (typeof DISPLAY_COUNTS)[number]
  )
    ? requestedLimit
    : 10
  const rankingsQuery = useUsageRankings(period, limit)
  const snapshot = rankingsQuery.data?.success
    ? rankingsQuery.data.data
    : undefined
  let rankingsContent: ReactNode
  if (rankingsQuery.isLoading) {
    rankingsContent = <UsageRankingsLoading />
  } else if (!snapshot) {
    rankingsContent = (
      <div className='border-border/70 border border-dashed px-6 py-12 text-center'>
        <h2 className='text-base font-semibold'>
          {t('Unable to load usage rankings')}
        </h2>
        <p className='text-muted-foreground mt-2 text-sm'>
          {rankingsQuery.data?.message ||
            t('Usage ranking data is unavailable right now.')}
        </p>
      </div>
    )
  } else {
    rankingsContent = (
      <>
        <UsageSummary snapshot={snapshot} />
        <RankHighlights topUser={snapshot.top_user} myRank={snapshot.my_rank} />
        <UsageLeaderboard rows={snapshot.users} />
      </>
    )
  }

  const updateSearch = (next: {
    period?: UsageRankingPeriod
    limit?: number
  }): void => {
    void navigate({
      search: (previous) => ({
        ...previous,
        ...next,
      }),
    })
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Usage Rankings')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <div className='flex items-center gap-2'>
          <span className='text-muted-foreground text-xs'>
            {t('Display count')}
          </span>
          <Select
            value={String(limit)}
            onValueChange={(value) => {
              if (value != null) updateSearch({ limit: Number(value) })
            }}
          >
            <SelectTrigger size='sm' aria-label={t('Display count')}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {DISPLAY_COUNTS.map((count) => (
                <SelectItem key={count} value={String(count)}>
                  {count}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='mx-auto flex w-full max-w-6xl flex-col gap-4 pb-2'>
          <section className='space-y-3'>
            <p className='text-muted-foreground text-sm'>
              {t('Compare usage by period')}
            </p>
            <div
              role='tablist'
              aria-label={t('Period')}
              className='border-border/60 flex items-center overflow-x-auto border-b'
            >
              {PERIODS.map((item) => (
                <button
                  key={item.id}
                  type='button'
                  role='tab'
                  aria-selected={period === item.id}
                  onClick={() => updateSearch({ period: item.id })}
                  className={cn(
                    'relative -mb-px shrink-0 rounded-sm px-3 py-2 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40',
                    period === item.id
                      ? 'text-foreground after:bg-foreground after:absolute after:inset-x-3 after:-bottom-px after:h-0.5'
                      : 'text-muted-foreground hover:text-foreground'
                  )}
                >
                  {t(item.labelKey)}
                </button>
              ))}
            </div>
          </section>

          {rankingsContent}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

function UsageSummary(props: { snapshot: UsageRankingsSnapshot }) {
  const { t } = useTranslation()
  if (!props.snapshot) return null
  const metrics = [
    {
      label: t('Total consumption'),
      value: formatQuota(props.snapshot.total_quota),
      detail: t('Across all ranked users'),
    },
    {
      label: t('Tokens'),
      value: formatCompactNumber(props.snapshot.total_tokens),
      detail: t('Prompt and completion tokens'),
    },
    {
      label: t('Requests'),
      value: formatCompactNumber(props.snapshot.total_requests),
      detail: t('Successful consume records'),
    },
    {
      label: t('Total users'),
      value: formatNumber(props.snapshot.total_users),
      detail: t('{{count}} users with usage in the period', {
        count: props.snapshot.total_users,
      }),
    },
  ]

  return (
    <section className='border-border/70 overflow-hidden rounded-lg border'>
      <div className='divide-border/70 grid grid-cols-2 divide-x sm:grid-cols-4'>
        {metrics.map((metric) => (
          <div key={metric.label} className='min-w-0 px-3 py-3 sm:px-4 sm:py-4'>
            <div className='text-muted-foreground truncate text-xs'>
              {metric.label}
            </div>
            <div className='mt-1 truncate text-lg font-semibold tabular-nums sm:text-xl'>
              {metric.value}
            </div>
            <div className='text-muted-foreground mt-1 truncate text-xs'>
              {metric.detail}
            </div>
          </div>
        ))}
      </div>
    </section>
  )
}

function RankHighlights(props: {
  topUser?: UsageRankingRow
  myRank?: UsageRankingRow
}) {
  const { t } = useTranslation()
  return (
    <section className='grid gap-4 lg:grid-cols-2'>
      <RankHighlight
        icon={<Crown className='size-4' aria-hidden='true' />}
        title={t('Top spender')}
        row={props.topUser}
        emptyLabel={t('No usage ranking yet')}
      />
      <RankHighlight
        icon={<Medal className='size-4' aria-hidden='true' />}
        title={t('My rank')}
        row={props.myRank}
        emptyLabel={t('No usage ranking yet')}
      />
    </section>
  )
}

function RankHighlight(props: {
  icon: ReactNode
  title: string
  row?: UsageRankingRow
  emptyLabel: string
}) {
  const { t } = useTranslation()
  return (
    <section className='border-border/70 rounded-lg border p-4'>
      <div className='text-muted-foreground flex items-center gap-2 text-sm font-medium'>
        {props.icon}
        {props.title}
      </div>
      {props.row ? (
        <div className='mt-4 flex items-end justify-between gap-3'>
          <div className='min-w-0'>
            <div className='truncate text-xl font-semibold'>
              {props.row.username}
            </div>
            <div className='text-muted-foreground mt-1 text-xs'>
              {t('Rank')} #{props.row.rank}
            </div>
          </div>
          <div className='text-right'>
            <div className='text-lg font-semibold tabular-nums'>
              {formatQuota(props.row.total_quota)}
            </div>
            <div className='text-muted-foreground mt-1 text-xs'>
              {formatCompactNumber(props.row.total_tokens)} {t('Tokens')} ·{' '}
              {formatCompactNumber(props.row.request_count)} {t('Requests')}
            </div>
          </div>
        </div>
      ) : (
        <p className='text-muted-foreground mt-4 text-sm'>{props.emptyLabel}</p>
      )}
    </section>
  )
}

function UsageLeaderboard(props: { rows: UsageRankingRow[] }) {
  const { t } = useTranslation()
  return (
    <section className='border-border/70 rounded-lg border'>
      <div className='flex items-center gap-2 border-b px-4 py-3'>
        <Users className='size-4' aria-hidden='true' />
        <h2 className='text-sm font-semibold'>{t('Leaderboard')}</h2>
      </div>
      {props.rows.length === 0 ? (
        <p className='text-muted-foreground px-4 py-10 text-center text-sm'>
          {t('No usage data for this period')}
        </p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className='w-16'>{t('Rank')}</TableHead>
              <TableHead>{t('Masked username')}</TableHead>
              <TableHead className='text-right'>{t('Consumption')}</TableHead>
              <TableHead className='text-right'>{t('Tokens')}</TableHead>
              <TableHead className='text-right'>{t('Requests')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {props.rows.map((row) => (
              <TableRow
                key={`${row.rank}-${row.username}`}
                className={row.is_self ? 'bg-primary/5' : undefined}
              >
                <TableCell className='font-medium tabular-nums'>
                  #{row.rank}
                </TableCell>
                <TableCell>
                  <span className='font-medium'>{row.username}</span>
                  {row.is_self && (
                    <span className='text-primary ml-2 text-xs'>
                      {t('You')}
                    </span>
                  )}
                </TableCell>
                <TableCell className='text-right font-medium tabular-nums'>
                  {formatQuota(row.total_quota)}
                </TableCell>
                <TableCell className='text-muted-foreground text-right tabular-nums'>
                  {formatCompactNumber(row.total_tokens)}
                </TableCell>
                <TableCell className='text-muted-foreground text-right tabular-nums'>
                  {formatCompactNumber(row.request_count)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </section>
  )
}

function UsageRankingsLoading() {
  return (
    <div className='space-y-4'>
      <Skeleton className='h-24 w-full rounded-lg' />
      <div className='grid gap-4 lg:grid-cols-2'>
        <Skeleton className='h-32 w-full rounded-lg' />
        <Skeleton className='h-32 w-full rounded-lg' />
      </div>
      <Skeleton className='h-80 w-full rounded-lg' />
    </div>
  )
}
