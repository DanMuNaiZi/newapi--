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
import type {
  ColumnDef,
  ColumnFiltersState,
  PaginationState,
} from '@tanstack/react-table'
import { Plus } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { DataTablePage, useDataTable } from '@/components/data-table'
import { StatusBadge, type StatusVariant } from '@/components/status-badge'
import { TableId } from '@/components/table-id'
import { Button } from '@/components/ui/button'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'

import {
  filterLotteryPlansByView,
  matchesLotteryPlanSearch,
  type LotteryPlanListView,
} from '../lib/admin-list'
import { getLotteryPlanStatusLabel } from '../lib/status'
import type { LotteryPlan } from '../types'
import type { LotteryDetailsTab } from './lottery-plan-details-drawer'
import { LotteryPlanRowActions } from './lottery-plan-row-actions'

type LotteryPlansTableProps = {
  cancelPending: boolean
  drawPending: boolean
  isFetching: boolean
  isLoading: boolean
  plans: LotteryPlan[]
  onCancel: (plan: LotteryPlan) => void
  onCreate: () => void
  onDraw: (plan: LotteryPlan) => void
  onEdit: (plan: LotteryPlan) => void
  onView: (plan: LotteryPlan, tab: LotteryDetailsTab) => void
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

export function LotteryPlansTable(props: LotteryPlansTableProps) {
  const { t } = useTranslation()
  const [view, setView] = useState<LotteryPlanListView>('all')
  const [globalFilter, setGlobalFilter] = useState('')
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([])
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })

  const currentCount = filterLotteryPlansByView(props.plans, 'current').length
  const historyCount = filterLotteryPlansByView(props.plans, 'history').length
  const visiblePlans = useMemo(
    () =>
      [...filterLotteryPlansByView(props.plans, view)].sort(
        (left, right) => right.id - left.id
      ),
    [props.plans, view]
  )

  const columns = useMemo<ColumnDef<LotteryPlan>[]>(
    () => [
      {
        accessorKey: 'id',
        header: t('ID'),
        meta: { mobileHidden: true },
        cell: ({ row }) => <TableId value={row.original.id} />,
        size: 80,
      },
      {
        accessorKey: 'title',
        header: t('Lottery title'),
        meta: { mobileTitle: true },
        cell: ({ row }) => (
          <div className='max-w-[420px] min-w-[220px]'>
            <div className='truncate font-medium'>{row.original.title}</div>
            {row.original.description && (
              <div className='text-muted-foreground mt-0.5 line-clamp-1 text-xs'>
                {row.original.description}
              </div>
            )}
          </div>
        ),
        size: 320,
      },
      {
        accessorKey: 'status',
        header: t('Status'),
        meta: { mobileBadge: true },
        cell: ({ row }) => (
          <StatusBadge
            label={getLotteryPlanStatusLabel(t, row.original.status)}
            variant={getStatusVariant(row.original.status)}
            copyable={false}
            className='-ml-1.5'
          />
        ),
        size: 120,
      },
      {
        id: 'schedule',
        header: t('Schedule'),
        cell: ({ row }) => (
          <div className='text-xs sm:min-w-[210px]'>
            <div>
              <span className='text-muted-foreground'>
                {t('Registration')}:
              </span>{' '}
              {formatLotteryDate(row.original.registration_start_time)}
            </div>
            <div className='mt-1'>
              <span className='text-muted-foreground'>{t('Draw time')}:</span>{' '}
              {formatLotteryDate(row.original.draw_time)}
            </div>
          </div>
        ),
        size: 250,
      },
      {
        accessorKey: 'max_participants',
        header: t('Maximum participants'),
        meta: { mobileHidden: true },
        cell: ({ row }) => row.original.max_participants,
        size: 130,
      },
      {
        accessorKey: 'eligibility_mode',
        header: t('Eligible participants'),
        meta: { mobileHidden: true },
        cell: ({ row }) => {
          if (row.original.eligibility_mode === 'groups') {
            return t('Selected groups')
          }
          if (row.original.eligibility_mode === 'users') {
            return t('Selected users')
          }
          return t('All users')
        },
        size: 150,
      },
      {
        id: 'actions',
        header: t('Actions'),
        cell: ({ row }) => (
          <LotteryPlanRowActions
            plan={row.original}
            cancelPending={props.cancelPending}
            drawPending={props.drawPending}
            onCancel={props.onCancel}
            onDraw={props.onDraw}
            onEdit={props.onEdit}
            onView={props.onView}
          />
        ),
        meta: { pinned: 'right' as const },
        size: 130,
      },
    ],
    [
      props.cancelPending,
      props.drawPending,
      props.onCancel,
      props.onDraw,
      props.onEdit,
      props.onView,
      t,
    ]
  )

  const { table } = useDataTable({
    data: visiblePlans,
    columns,
    columnFilters,
    onColumnFiltersChange: setColumnFilters,
    globalFilter,
    onGlobalFilterChange: setGlobalFilter,
    globalFilterFn: (row, _columnId, value) =>
      matchesLotteryPlanSearch(row.original, value),
    pagination,
    onPaginationChange: setPagination,
  })

  const changeView = (value: string): void => {
    setView(value as LotteryPlanListView)
    setPagination((current) => ({ ...current, pageIndex: 0 }))
  }

  return (
    <div className='flex h-full min-h-0 flex-col gap-3'>
      <Tabs value={view} onValueChange={changeView} className='shrink-0'>
        <TabsList>
          <TabsTrigger value='all'>
            {t('All plans ({{count}})', { count: props.plans.length })}
          </TabsTrigger>
          <TabsTrigger value='current'>
            {t('Current ({{count}})', { count: currentCount })}
          </TabsTrigger>
          <TabsTrigger value='history'>
            {t('History ({{count}})', { count: historyCount })}
          </TabsTrigger>
        </TabsList>
      </Tabs>
      <div className='min-h-0 flex-1'>
        <DataTablePage
          table={table}
          columns={columns}
          isLoading={props.isLoading}
          isFetching={props.isFetching}
          emptyTitle={t('No lottery plans found')}
          emptyDescription={t(
            'No lottery plans match the selected view or search.'
          )}
          emptyAction={
            <Button size='sm' onClick={props.onCreate}>
              <Plus />
              {t('Create lottery plan')}
            </Button>
          }
          skeletonKeyPrefix='lottery-plans-skeleton'
          applyHeaderSize
          toolbarProps={{
            searchPlaceholder: t('Search lotteries by title or ID...'),
          }}
        />
      </div>
    </div>
  )
}
