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
import type {
  ColumnDef,
  ColumnFiltersState,
  PaginationState,
} from '@tanstack/react-table'
import { CheckCircle2, Ban } from 'lucide-react'
import * as React from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { CopyButton } from '@/components/copy-button'
import { DataTablePage, useDataTable } from '@/components/data-table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { toIntlLocale } from '@/i18n/languages'
import { formatCurrencyFromUSD } from '@/lib/currency'
import { handleServerError } from '@/lib/handle-server-error'
import { createServerError } from '@/lib/server-error-message'

import {
  completeManualWechatSubscription,
  completeManualWechatTopUp,
  expireManualWechatSubscription,
  expireManualWechatTopUp,
  getManualWechatSubscriptionOrders,
  getManualWechatTopUpOrders,
} from '../api'
import type {
  AdminManualWechatSubscriptionOrder,
  AdminManualWechatTopUpOrder,
  ApiResponse,
} from '../types'

type ManualPaymentOrder =
  | AdminManualWechatTopUpOrder
  | AdminManualWechatSubscriptionOrder

type ManualPaymentOrderKind = 'topup' | 'subscription'
type ManualPaymentAction = 'complete' | 'expire'

interface ManualPaymentOrdersTableProps {
  kind: ManualPaymentOrderKind
  canOperate: boolean
}

interface PendingAction {
  order: ManualPaymentOrder
  action: ManualPaymentAction
}

function statusBadge(status: string, label: string) {
  if (status === 'success') return <Badge>{label}</Badge>
  if (status === 'expired') return <Badge variant='secondary'>{label}</Badge>
  return <Badge variant='warning'>{label}</Badge>
}

function benefitLabel(order: ManualPaymentOrder): string {
  if ('plan_title' in order) return order.plan_title || `#${order.plan_id}`
  return formatCurrencyFromUSD(Number(order.amount), {
    digitsLarge: 2,
    digitsSmall: 2,
    abbreviate: false,
  })
}

export function ManualPaymentOrdersTable(props: ManualPaymentOrdersTableProps) {
  const { t, i18n } = useTranslation()
  const queryClient = useQueryClient()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const [pagination, setPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })
  const [globalFilter, setGlobalFilter] = React.useState('')
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>(
    []
  )
  const [pendingAction, setPendingAction] =
    React.useState<PendingAction | null>(null)
  const status = String(
    columnFilters.find((filter) => filter.id === 'status')?.value || ''
  )
  const queryKey = [
    'manual-wechat-orders',
    props.kind,
    pagination.pageIndex + 1,
    pagination.pageSize,
    globalFilter,
    status,
  ]

  const orderQuery = useQuery({
    queryKey,
    queryFn: async () => {
      const params = {
        page: pagination.pageIndex + 1,
        pageSize: pagination.pageSize,
        keyword: globalFilter,
        status,
      }
      const response =
        props.kind === 'topup'
          ? await getManualWechatTopUpOrders(params)
          : await getManualWechatSubscriptionOrders(params)
      if (!response.success || !response.data) {
        throw createServerError(response, t('admin.manualPayments.loadFailed'))
      }
      return {
        items: response.data.items as ManualPaymentOrder[],
        total: response.data.total,
      }
    },
    placeholderData: (previousData) => previousData,
  })

  const actionMutation = useMutation({
    mutationFn: async (action: PendingAction): Promise<ApiResponse> => {
      if (props.kind === 'topup') {
        return action.action === 'complete'
          ? completeManualWechatTopUp(action.order.trade_no)
          : expireManualWechatTopUp(action.order.trade_no)
      }
      return action.action === 'complete'
        ? completeManualWechatSubscription(action.order.trade_no)
        : expireManualWechatSubscription(action.order.trade_no)
    },
    onSuccess: (response, action) => {
      if (!response.success) {
        handleServerError(response, t('admin.manualPayments.actionFailed'))
        return
      }
      toast.success(
        t(
          action.action === 'complete'
            ? 'admin.manualPayments.completed'
            : 'admin.manualPayments.expired'
        )
      )
      setPendingAction(null)
      queryClient.invalidateQueries({
        queryKey: ['manual-wechat-orders', props.kind],
      })
    },
    onError: (error: Error) => {
      handleServerError(error, t('admin.manualPayments.actionFailed'))
    },
  })

  const columns = React.useMemo<ColumnDef<ManualPaymentOrder>[]>(
    () => [
      {
        accessorKey: 'trade_no',
        header: t('admin.manualPayments.orderNumber'),
        cell: ({ row }) => (
          <div className='flex min-w-40 items-center gap-1'>
            <span className='truncate font-mono text-xs'>
              {row.original.trade_no}
            </span>
            <CopyButton value={row.original.trade_no} />
          </div>
        ),
      },
      {
        id: 'user',
        header: t('admin.manualPayments.user'),
        cell: ({ row }) => (
          <span>
            {row.original.username || '-'} (#{row.original.user_id})
          </span>
        ),
      },
      {
        id: 'kind',
        header: t('admin.manualPayments.orderType'),
        cell: () =>
          t(
            props.kind === 'topup'
              ? 'admin.manualPayments.topupTab'
              : 'admin.manualPayments.subscriptionTab'
          ),
      },
      {
        accessorKey: 'money',
        header: t('admin.manualPayments.amount'),
        cell: ({ row }) => `¥${Number(row.original.money).toFixed(2)} CNY`,
      },
      {
        id: 'benefit',
        header: t('admin.manualPayments.benefit'),
        cell: ({ row }) => benefitLabel(row.original),
      },
      {
        accessorKey: 'create_time',
        header: t('admin.manualPayments.createdAt'),
        cell: ({ row }) =>
          new Date(row.original.create_time * 1000).toLocaleString(locale),
      },
      {
        accessorKey: 'complete_time',
        header: t('admin.manualPayments.completedAt'),
        cell: ({ row }) =>
          row.original.complete_time
            ? new Date(row.original.complete_time * 1000).toLocaleString(locale)
            : '-',
      },
      {
        accessorKey: 'status',
        header: t('admin.manualPayments.status'),
        cell: ({ row }) =>
          statusBadge(
            row.original.status,
            t(`admin.manualPayments.status.${row.original.status}`)
          ),
      },
      {
        accessorKey: 'payment_provider',
        header: t('admin.manualPayments.paymentChannel'),
        cell: () => t('payment.manualWechat.displayName'),
      },
      {
        id: 'actions',
        header: t('admin.manualPayments.actions'),
        cell: ({ row }) => {
          if (!props.canOperate || row.original.status !== 'pending') {
            return null
          }
          return (
            <div className='flex items-center justify-end gap-1'>
              <Button
                size='icon'
                variant='ghost'
                aria-label={t('admin.manualPayments.completeOrder')}
                onClick={() =>
                  setPendingAction({
                    order: row.original,
                    action: 'complete',
                  })
                }
              >
                <CheckCircle2 className='h-4 w-4' aria-hidden='true' />
              </Button>
              <Button
                size='icon'
                variant='ghost'
                aria-label={t('admin.manualPayments.expireOrder')}
                onClick={() =>
                  setPendingAction({
                    order: row.original,
                    action: 'expire',
                  })
                }
              >
                <Ban className='h-4 w-4' aria-hidden='true' />
              </Button>
            </div>
          )
        },
      },
    ],
    [locale, props.canOperate, props.kind, t]
  )

  const { table } = useDataTable({
    data: orderQuery.data?.items || [],
    columns,
    pagination,
    globalFilter,
    columnFilters,
    onPaginationChange: setPagination,
    onGlobalFilterChange: (value) => {
      setGlobalFilter(String(value))
      setPagination((current) => ({ ...current, pageIndex: 0 }))
    },
    onColumnFiltersChange: (value) => {
      setColumnFilters((current) =>
        typeof value === 'function' ? value(current) : value
      )
      setPagination((current) => ({ ...current, pageIndex: 0 }))
    },
    manualPagination: true,
    manualFiltering: true,
    totalCount: orderQuery.data?.total || 0,
  })

  const confirmationDescription = pendingAction ? (
    <div className='space-y-2 text-sm'>
      <p>
        {pendingAction.action === 'complete'
          ? t('admin.manualPayments.completeDescription')
          : t('admin.manualPayments.expireDescription')}
      </p>
      <dl className='grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 rounded-md border p-3'>
        <dt>{t('admin.manualPayments.user')}</dt>
        <dd className='text-right'>
          {pendingAction.order.username || '-'} (#{pendingAction.order.user_id})
        </dd>
        <dt>{t('admin.manualPayments.orderNumber')}</dt>
        <dd className='text-right font-mono text-xs'>
          {pendingAction.order.trade_no}
        </dd>
        <dt>{t('admin.manualPayments.amount')}</dt>
        <dd className='text-right font-semibold'>
          ¥{Number(pendingAction.order.money).toFixed(2)} CNY
        </dd>
        <dt>{t('admin.manualPayments.benefit')}</dt>
        <dd className='text-right'>{benefitLabel(pendingAction.order)}</dd>
      </dl>
    </div>
  ) : (
    ''
  )

  return (
    <>
      <DataTablePage
        table={table}
        columns={columns}
        isLoading={orderQuery.isLoading}
        isFetching={orderQuery.isFetching}
        emptyTitle={t('admin.manualPayments.emptyTitle')}
        emptyDescription={t('admin.manualPayments.emptyDescription')}
        skeletonKeyPrefix={`manual-payments-${props.kind}`}
        toolbarProps={{
          searchPlaceholder: t('admin.manualPayments.searchPlaceholder'),
          searchDebounceMs: 400,
          filters: [
            {
              columnId: 'status',
              title: t('admin.manualPayments.status'),
              singleSelect: true,
              options: [
                {
                  label: t('admin.manualPayments.status.pending'),
                  value: 'pending',
                },
                {
                  label: t('admin.manualPayments.status.success'),
                  value: 'success',
                },
                {
                  label: t('admin.manualPayments.status.expired'),
                  value: 'expired',
                },
              ],
            },
          ],
        }}
      />

      <ConfirmDialog
        open={pendingAction !== null}
        onOpenChange={(open) => !open && setPendingAction(null)}
        title={
          pendingAction?.action === 'complete'
            ? t('admin.manualPayments.completeOrder')
            : t('admin.manualPayments.expireOrder')
        }
        desc={confirmationDescription}
        destructive={pendingAction?.action === 'expire'}
        confirmText={
          pendingAction?.action === 'complete'
            ? t('admin.manualPayments.confirmReceived')
            : t('admin.manualPayments.confirmExpire')
        }
        isLoading={actionMutation.isPending}
        handleConfirm={() => {
          if (pendingAction) actionMutation.mutate(pendingAction)
        }}
      />
    </>
  )
}
