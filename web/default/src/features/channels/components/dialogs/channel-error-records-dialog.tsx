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
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery } from '@tanstack/react-query'
import {
  getCoreRowModel,
  useReactTable,
  type PaginationState,
} from '@tanstack/react-table'
import { Loader2, Pencil, Search, X } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  DataTablePagination,
  StaticDataTable,
  type StaticDataTableColumn,
} from '@/components/data-table'
import { Dialog } from '@/components/dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import {
  ADMIN_PERMISSION_ACTIONS,
  ADMIN_PERMISSION_RESOURCES,
  hasPermission,
} from '@/lib/admin-permissions'
import { formatTimestampToDate } from '@/lib/format'
import { useAuthStore } from '@/stores/auth-store'

import { getChannelErrors, updateChannelErrorRecordDisplay } from '../../api'
import {
  channelErrorDisplayRuleDefaults,
  channelErrorDisplayRuleSchema,
  type ChannelErrorDisplayRuleFormValues,
} from '../../lib'
import type {
  ChannelErrorDisplayMode,
  ChannelErrorRecord,
  GetChannelErrorsParams,
} from '../../types'
import { useChannels } from '../channels-provider'

type ChannelErrorRecordsDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

type FilterState = {
  statusCode: string
  requestModel: string
  keyword: string
}

const EMPTY_FILTERS: FilterState = {
  statusCode: '',
  requestModel: '',
  keyword: '',
}

const MODE_OPTIONS: ChannelErrorDisplayMode[] = [
  'inherit',
  'generic',
  'original',
  'custom',
]

export function ChannelErrorRecordsDialog({
  open,
  onOpenChange,
}: ChannelErrorRecordsDialogProps) {
  const { t } = useTranslation()
  const { currentRow } = useChannels()
  const currentUser = useAuthStore((state) => state.auth.user)
  const canEdit = hasPermission(
    currentUser,
    ADMIN_PERMISSION_RESOURCES.CHANNEL,
    ADMIN_PERMISSION_ACTIONS.WRITE
  )
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })
  const [draftFilters, setDraftFilters] = useState<FilterState>(EMPTY_FILTERS)
  const [filters, setFilters] = useState<FilterState>(EMPTY_FILTERS)
  const [editingRecord, setEditingRecord] = useState<ChannelErrorRecord | null>(
    null
  )
  const form = useForm<ChannelErrorDisplayRuleFormValues>({
    resolver: zodResolver(channelErrorDisplayRuleSchema),
    defaultValues: channelErrorDisplayRuleDefaults(),
  })
  const selectedMode = form.watch('display_mode')
  const channelId = currentRow?.id

  useEffect(() => {
    if (!open) return
    setPagination({ pageIndex: 0, pageSize: 10 })
    setDraftFilters(EMPTY_FILTERS)
    setFilters(EMPTY_FILTERS)
    setEditingRecord(null)
    form.reset(channelErrorDisplayRuleDefaults())
  }, [form, open, currentRow?.id])

  const queryParams = useMemo<GetChannelErrorsParams>(() => {
    const statusCode = Number(filters.statusCode)
    return {
      page: pagination.pageIndex + 1,
      page_size: pagination.pageSize,
      status_code:
        Number.isInteger(statusCode) && statusCode >= 400 && statusCode <= 599
          ? statusCode
          : undefined,
      request_model: filters.requestModel.trim() || undefined,
      keyword: filters.keyword.trim() || undefined,
    }
  }, [filters, pagination])

  const errorsQuery = useQuery({
    queryKey: ['channels', currentRow?.id, 'errors', queryParams],
    enabled: open && Boolean(channelId),
    queryFn: async () => {
      if (!channelId) {
        throw new Error(t('Failed to load error records'))
      }
      const response = await getChannelErrors(channelId, queryParams)
      if (!response.success) {
        throw new Error(response.message || t('Failed to load error records'))
      }
      return response.data
    },
  })

  const items = errorsQuery.data?.items || []
  const total = errorsQuery.data?.total || 0
  const paginationTable = useReactTable({
    data: items,
    columns: [],
    state: { pagination },
    onPaginationChange: setPagination,
    manualPagination: true,
    rowCount: total,
    getCoreRowModel: getCoreRowModel(),
  })

  const updateMutation = useMutation({
    mutationFn: async (values: ChannelErrorDisplayRuleFormValues) => {
      if (!currentRow || !editingRecord) return
      const response = await updateChannelErrorRecordDisplay(
        currentRow.id,
        editingRecord.id,
        {
          display_mode: values.display_mode,
          display_status_code:
            values.display_mode === 'custom'
              ? values.display_status_code
              : undefined,
          display_message:
            values.display_mode === 'custom'
              ? values.display_message?.trim()
              : undefined,
        }
      )
      if (!response.success) {
        throw new Error(response.message || t('Failed to update display rule'))
      }
    },
    onSuccess: async () => {
      toast.success(t('Display rule updated'))
      setEditingRecord(null)
      form.reset(channelErrorDisplayRuleDefaults())
      await errorsQuery.refetch()
    },
    onError: (error: unknown) => {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to update display rule')
      )
    },
  })

  const startEditing = (record: ChannelErrorRecord) => {
    setEditingRecord(record)
    form.reset(channelErrorDisplayRuleDefaults(record))
  }

  const applyFilters = () => {
    setPagination((value) => ({ ...value, pageIndex: 0 }))
    setFilters({ ...draftFilters })
  }

  const clearFilters = () => {
    setPagination((value) => ({ ...value, pageIndex: 0 }))
    setDraftFilters(EMPTY_FILTERS)
    setFilters(EMPTY_FILTERS)
  }

  const modeLabel = (mode: ChannelErrorDisplayMode) => {
    const labels: Record<ChannelErrorDisplayMode, string> = {
      inherit: t('Inherit channel default'),
      generic: t('Generic hidden error'),
      original: t('Sanitized original error'),
      custom: t('Custom response'),
    }
    return labels[mode]
  }

  const columns: StaticDataTableColumn<ChannelErrorRecord>[] = [
    {
      id: 'request',
      header: t('Request'),
      className: 'min-w-[190px]',
      cell: (record) => (
        <div className='space-y-1'>
          <div className='font-medium'>{record.request_model || '-'}</div>
          <div className='text-muted-foreground max-w-[260px] truncate font-mono text-xs'>
            {record.request_path || '-'}
          </div>
        </div>
      ),
    },
    {
      id: 'status',
      header: t('Status'),
      className: 'w-28',
      cell: (record) => (
        <div className='space-y-1'>
          <Badge variant='destructive'>{record.upstream_status_code}</Badge>
          {record.final_status_code !== record.upstream_status_code && (
            <div className='text-muted-foreground text-xs'>
              {t('Mapped')}: {record.final_status_code}
            </div>
          )}
        </div>
      ),
    },
    {
      id: 'classification',
      header: t('Error type / code'),
      className: 'min-w-[180px]',
      cell: (record) => (
        <div className='space-y-1 font-mono text-xs'>
          <div>{record.error_type || '-'}</div>
          <div className='text-muted-foreground'>
            {record.error_code || '-'}
          </div>
        </div>
      ),
    },
    {
      id: 'sample',
      header: t('Sanitized error sample'),
      className: 'min-w-[360px]',
      cellClassName: 'whitespace-normal',
      cell: (record) => (
        <div
          className='max-h-28 overflow-auto font-mono text-xs break-all whitespace-pre-wrap'
          title={record.sample_message}
        >
          {record.sample_message || '-'}
        </div>
      ),
    },
    {
      id: 'occurrences',
      header: t('Occurrences'),
      className: 'w-28 text-right',
      cellClassName: 'text-right tabular-nums',
      cell: (record) => record.occurrence_count.toLocaleString(),
    },
    {
      id: 'times',
      header: t('First / last seen'),
      className: 'min-w-[180px]',
      cell: (record) => (
        <div className='space-y-1 text-xs tabular-nums'>
          <div>{formatTimestampToDate(record.first_seen_time)}</div>
          <div className='text-muted-foreground'>
            {formatTimestampToDate(record.last_seen_time)}
          </div>
        </div>
      ),
    },
    {
      id: 'display',
      header: t('Display mode'),
      className: 'min-w-[180px]',
      cell: (record) => (
        <div className='flex items-center justify-between gap-2'>
          <Badge variant='outline'>{modeLabel(record.display_mode)}</Badge>
          {canEdit && (
            <Button
              type='button'
              variant='ghost'
              size='icon-sm'
              aria-label={t('Edit display rule')}
              onClick={() => startEditing(record)}
            >
              <Pencil className='size-4' />
            </Button>
          )}
        </div>
      ),
    },
  ]

  let tableContent
  if (errorsQuery.isLoading || errorsQuery.isFetching) {
    tableContent = (
      <div className='flex min-h-48 items-center justify-center'>
        <Loader2 className='text-muted-foreground size-8 animate-spin' />
      </div>
    )
  } else if (errorsQuery.isError) {
    tableContent = (
      <div className='text-destructive flex min-h-48 items-center justify-center px-4 text-center'>
        {errorsQuery.error instanceof Error
          ? errorsQuery.error.message
          : t('Failed to load error records')}
      </div>
    )
  } else {
    tableContent = (
      <StaticDataTable
        className='rounded-none border-0'
        tableClassName='min-w-[1350px]'
        data={items}
        columns={columns}
        getRowKey={(record) => record.id}
        emptyContent={t('No error records found')}
      />
    )
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Channel Error Records')}
      description={t(
        'Aggregated, sanitized upstream errors for channel "{{name}}".',
        { name: currentRow?.name || '-' }
      )}
      contentHeight='min(78vh, 820px)'
      contentClassName='sm:max-w-[min(96vw,1500px)]'
      bodyClassName='flex h-full min-h-0 flex-col gap-4'
    >
      <div className='flex h-full min-h-0 flex-col gap-4'>
        <div className='grid gap-2 md:grid-cols-[8rem_13rem_minmax(14rem,1fr)_auto]'>
          <Input
            type='number'
            min={400}
            max={599}
            value={draftFilters.statusCode}
            onChange={(event) =>
              setDraftFilters((value) => ({
                ...value,
                statusCode: event.target.value,
              }))
            }
            placeholder={t('Status code')}
          />
          <Input
            value={draftFilters.requestModel}
            onChange={(event) =>
              setDraftFilters((value) => ({
                ...value,
                requestModel: event.target.value,
              }))
            }
            placeholder={t('Request model')}
          />
          <Input
            value={draftFilters.keyword}
            onChange={(event) =>
              setDraftFilters((value) => ({
                ...value,
                keyword: event.target.value,
              }))
            }
            onKeyDown={(event) => {
              if (event.key === 'Enter') applyFilters()
            }}
            placeholder={t('Search error message, code, type, or path')}
          />
          <div className='flex gap-2'>
            <Button type='button' onClick={applyFilters}>
              <Search className='size-4' />
              {t('Search')}
            </Button>
            <Button type='button' variant='outline' onClick={clearFilters}>
              <X className='size-4' />
              {t('Clear')}
            </Button>
          </div>
        </div>

        {editingRecord && canEdit && (
          <Form {...form}>
            <form
              className='bg-muted/30 grid gap-4 rounded-lg border p-4 lg:grid-cols-[15rem_10rem_minmax(16rem,1fr)_auto] lg:items-end'
              onSubmit={form.handleSubmit((values) =>
                updateMutation.mutate(values)
              )}
            >
              <FormField
                control={form.control}
                name='display_mode'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Display mode')}</FormLabel>
                    <Select
                      value={field.value}
                      onValueChange={(value) =>
                        field.onChange(value as ChannelErrorDisplayMode)
                      }
                    >
                      <FormControl>
                        <SelectTrigger className='w-full'>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectGroup>
                          {MODE_OPTIONS.map((mode) => (
                            <SelectItem key={mode} value={mode}>
                              {modeLabel(mode)}
                            </SelectItem>
                          ))}
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='display_status_code'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Custom status code')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={400}
                        max={599}
                        disabled={selectedMode !== 'custom'}
                        value={field.value ?? 503}
                        onChange={(event) =>
                          field.onChange(Number(event.target.value))
                        }
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='display_message'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Custom client message')}</FormLabel>
                    <FormControl>
                      <Textarea
                        rows={2}
                        maxLength={500}
                        disabled={selectedMode !== 'custom'}
                        {...field}
                      />
                    </FormControl>
                    {selectedMode === 'original' && (
                      <FormDescription>
                        {t('The upstream error will still be sanitized.')}
                      </FormDescription>
                    )}
                    <FormMessage />
                  </FormItem>
                )}
              />
              <div className='flex gap-2'>
                <Button type='submit' disabled={updateMutation.isPending}>
                  {updateMutation.isPending && (
                    <Loader2 className='size-4 animate-spin' />
                  )}
                  {t('Save')}
                </Button>
                <Button
                  type='button'
                  variant='outline'
                  onClick={() => setEditingRecord(null)}
                >
                  {t('Cancel')}
                </Button>
              </div>
            </form>
          </Form>
        )}

        <div className='min-h-0 flex-1 overflow-auto rounded-md border'>
          {tableContent}
        </div>

        <div className='shrink-0'>
          <DataTablePagination table={paginationTable} />
        </div>
      </div>
    </Dialog>
  )
}
