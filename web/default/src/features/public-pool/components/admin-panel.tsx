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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Check, Pencil, Plus, ShieldCheck, Trash2, X } from 'lucide-react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { Textarea } from '@/components/ui/textarea'
import {
  ADMIN_PERMISSION_ACTIONS,
  ADMIN_PERMISSION_RESOURCES,
  hasPermission,
} from '@/lib/admin-permissions'
import dayjs from '@/lib/dayjs'
import { useAuthStore } from '@/stores/auth-store'

import {
  createAdminPublicPoolSite,
  deleteAdminPublicPoolSite,
  getAdminPublicPoolContributions,
  getAdminPublicPoolSites,
  reviewAdminPublicPoolContribution,
  updateAdminPublicPoolSite,
} from '../api'
import {
  publicPoolSiteSchema,
  type PublicPoolSiteFormValues,
} from '../lib/admin-form'
import type { PublicPoolContributionStatus, PublicPoolSite } from '../types'

const EMPTY_SITE: PublicPoolSiteFormValues = {
  name: '',
  url: '',
  description: '',
  status: 'enabled',
  sort_order: 0,
}

function contributionStatusVariant(
  status: PublicPoolContributionStatus
): 'default' | 'secondary' | 'destructive' | 'outline' {
  if (status === 'approved') return 'default'
  if (status === 'rejected') return 'destructive'
  return 'secondary'
}

export function PublicPoolAdminPanel() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const currentUser = useAuthStore((state) => state.auth.user)
  const canRead = hasPermission(
    currentUser,
    ADMIN_PERMISSION_RESOURCES.PUBLIC_POOL,
    ADMIN_PERMISSION_ACTIONS.READ
  )
  const canWrite = hasPermission(
    currentUser,
    ADMIN_PERMISSION_RESOURCES.PUBLIC_POOL,
    ADMIN_PERMISSION_ACTIONS.WRITE
  )
  const canOperate = hasPermission(
    currentUser,
    ADMIN_PERMISSION_RESOURCES.PUBLIC_POOL,
    ADMIN_PERMISSION_ACTIONS.OPERATE
  )
  const [editingSite, setEditingSite] = useState<PublicPoolSite | null>(null)
  const [siteToDelete, setSiteToDelete] = useState<PublicPoolSite | null>(null)
  const [reviewNotes, setReviewNotes] = useState<Record<number, string>>({})
  const form = useForm<PublicPoolSiteFormValues>({
    resolver: zodResolver(publicPoolSiteSchema),
    defaultValues: EMPTY_SITE,
  })

  const sitesQuery = useQuery({
    queryKey: ['public-pool', 'admin', 'sites'],
    queryFn: getAdminPublicPoolSites,
    enabled: canRead,
  })
  const contributionsQuery = useQuery({
    queryKey: ['public-pool', 'admin', 'contributions'],
    queryFn: getAdminPublicPoolContributions,
    enabled: canRead,
  })
  const saveSiteMutation = useMutation({
    mutationFn: async (values: PublicPoolSiteFormValues) => {
      if (editingSite) {
        return updateAdminPublicPoolSite(editingSite.id, values)
      }
      return createAdminPublicPoolSite(values)
    },
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Failed to save public site'))
        return
      }
      toast.success(
        t(editingSite ? 'Public site updated' : 'Public site created')
      )
      setEditingSite(null)
      form.reset(EMPTY_SITE)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['public-pool', 'sites'] }),
        queryClient.invalidateQueries({
          queryKey: ['public-pool', 'admin', 'sites'],
        }),
      ])
    },
  })
  const deleteSiteMutation = useMutation({
    mutationFn: deleteAdminPublicPoolSite,
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Failed to delete public site'))
        return
      }
      toast.success(t('Public site deleted'))
      setSiteToDelete(null)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['public-pool', 'sites'] }),
        queryClient.invalidateQueries({
          queryKey: ['public-pool', 'admin', 'sites'],
        }),
      ])
    },
  })
  const reviewMutation = useMutation({
    mutationFn: (input: {
      id: number
      status: 'approved' | 'rejected'
      review_note: string
    }) =>
      reviewAdminPublicPoolContribution(input.id, {
        status: input.status,
        review_note: input.review_note,
      }),
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Failed to review contribution'))
        return
      }
      setReviewNotes((current) => {
        const next = { ...current }
        delete next[response.data?.id ?? 0]
        return next
      })
      toast.success(t('Contribution review saved'))
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: ['public-pool', 'admin', 'contributions'],
        }),
        queryClient.invalidateQueries({
          queryKey: ['public-pool', 'contributions', 'self'],
        }),
      ])
    },
  })

  if (!canRead) return null

  const sites = sitesQuery.data?.success ? (sitesQuery.data.data ?? []) : []
  const contributions = contributionsQuery.data?.success
    ? (contributionsQuery.data.data ?? [])
    : []

  const editSite = (site: PublicPoolSite) => {
    setEditingSite(site)
    form.reset({
      name: site.name,
      url: site.url,
      description: site.description,
      status: site.status,
      sort_order: site.sort_order,
    })
  }

  return (
    <section className='space-y-4 border-t pt-6'>
      <div>
        <div className='flex items-center gap-2'>
          <ShieldCheck className='text-primary size-5' aria-hidden='true' />
          <h2 className='text-lg font-semibold'>
            {t('Public pool management')}
          </h2>
        </div>
        <p className='text-muted-foreground mt-1 text-sm'>
          {t('Maintain public sites and review user contributions.')}
        </p>
      </div>

      <div className='grid gap-6 xl:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]'>
        <div className='space-y-4'>
          {canWrite && (
            <Card>
              <CardHeader>
                <CardTitle>
                  {t(editingSite ? 'Edit public site' : 'Add public site')}
                </CardTitle>
                <CardDescription>
                  {t('Only credential-free HTTP and HTTPS links are accepted.')}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <form
                  onSubmit={form.handleSubmit((values) =>
                    saveSiteMutation.mutate(values)
                  )}
                >
                  <FieldGroup>
                    <Field data-invalid={Boolean(form.formState.errors.name)}>
                      <FieldLabel htmlFor='public-pool-admin-name'>
                        {t('Site name')}
                      </FieldLabel>
                      <Input
                        id='public-pool-admin-name'
                        aria-invalid={Boolean(form.formState.errors.name)}
                        {...form.register('name')}
                      />
                      <FieldError>
                        {form.formState.errors.name?.message
                          ? t(form.formState.errors.name.message)
                          : null}
                      </FieldError>
                    </Field>
                    <Field data-invalid={Boolean(form.formState.errors.url)}>
                      <FieldLabel htmlFor='public-pool-admin-url'>
                        {t('Site URL')}
                      </FieldLabel>
                      <Input
                        id='public-pool-admin-url'
                        type='url'
                        aria-invalid={Boolean(form.formState.errors.url)}
                        {...form.register('url')}
                      />
                      <FieldError>
                        {form.formState.errors.url?.message
                          ? t(form.formState.errors.url.message)
                          : null}
                      </FieldError>
                    </Field>
                    <Field
                      data-invalid={Boolean(form.formState.errors.description)}
                    >
                      <FieldLabel htmlFor='public-pool-admin-description'>
                        {t('Description')}
                      </FieldLabel>
                      <Textarea
                        id='public-pool-admin-description'
                        aria-invalid={Boolean(
                          form.formState.errors.description
                        )}
                        {...form.register('description')}
                      />
                      <FieldError>
                        {form.formState.errors.description?.message
                          ? t(form.formState.errors.description.message)
                          : null}
                      </FieldError>
                    </Field>
                    <div className='grid gap-3 sm:grid-cols-2'>
                      <Field
                        data-invalid={Boolean(form.formState.errors.status)}
                      >
                        <FieldLabel htmlFor='public-pool-admin-status'>
                          {t('Status')}
                        </FieldLabel>
                        <NativeSelect
                          id='public-pool-admin-status'
                          {...form.register('status')}
                        >
                          <NativeSelectOption value='enabled'>
                            {t('enabled')}
                          </NativeSelectOption>
                          <NativeSelectOption value='disabled'>
                            {t('disabled')}
                          </NativeSelectOption>
                        </NativeSelect>
                      </Field>
                      <Field>
                        <FieldLabel htmlFor='public-pool-admin-sort'>
                          {t('Sort order')}
                        </FieldLabel>
                        <Input
                          id='public-pool-admin-sort'
                          type='number'
                          aria-invalid={Boolean(
                            form.formState.errors.sort_order
                          )}
                          {...form.register('sort_order', {
                            valueAsNumber: true,
                          })}
                        />
                        <FieldError>
                          {form.formState.errors.sort_order?.message
                            ? t(form.formState.errors.sort_order.message)
                            : null}
                        </FieldError>
                      </Field>
                    </div>
                    <div className='flex flex-wrap gap-2'>
                      <Button
                        type='submit'
                        disabled={saveSiteMutation.isPending}
                      >
                        {saveSiteMutation.isPending && (
                          <Spinner data-icon='inline-start' />
                        )}
                        {!saveSiteMutation.isPending && editingSite && (
                          <Check data-icon='inline-start' />
                        )}
                        {!saveSiteMutation.isPending && !editingSite && (
                          <Plus data-icon='inline-start' />
                        )}
                        {t(editingSite ? 'Save changes' : 'Add public site')}
                      </Button>
                      {editingSite && (
                        <Button
                          type='button'
                          variant='outline'
                          onClick={() => {
                            setEditingSite(null)
                            form.reset(EMPTY_SITE)
                          }}
                        >
                          {t('Cancel')}
                        </Button>
                      )}
                    </div>
                  </FieldGroup>
                </form>
              </CardContent>
            </Card>
          )}

          <Card>
            <CardHeader>
              <CardTitle>{t('Managed public sites')}</CardTitle>
              <CardDescription>
                {t('{{count}} sites configured', { count: sites.length })}
              </CardDescription>
            </CardHeader>
            <CardContent>
              {sitesQuery.isLoading ? (
                <div className='space-y-3'>
                  <Skeleton className='h-20' />
                  <Skeleton className='h-20' />
                </div>
              ) : (
                <div className='divide-y'>
                  {sites.map((site) => (
                    <article
                      key={site.id}
                      className='space-y-2 py-3 first:pt-0'
                    >
                      <div className='flex items-start justify-between gap-3'>
                        <div className='min-w-0'>
                          <div className='flex flex-wrap items-center gap-2'>
                            <span className='font-medium'>{site.name}</span>
                            <Badge
                              variant={
                                site.status === 'enabled'
                                  ? 'default'
                                  : 'secondary'
                              }
                            >
                              {t(site.status)}
                            </Badge>
                          </div>
                          <p className='text-muted-foreground mt-1 truncate text-xs'>
                            {site.url}
                          </p>
                        </div>
                        {canWrite && (
                          <div className='flex shrink-0 gap-1'>
                            <Button
                              size='icon-sm'
                              variant='ghost'
                              onClick={() => editSite(site)}
                              aria-label={t('Edit public site')}
                            >
                              <Pencil />
                            </Button>
                            <Button
                              size='icon-sm'
                              variant='ghost'
                              className='text-destructive'
                              onClick={() => setSiteToDelete(site)}
                              aria-label={t('Delete public site')}
                            >
                              <Trash2 />
                            </Button>
                          </div>
                        )}
                      </div>
                    </article>
                  ))}
                  {sites.length === 0 && (
                    <p className='text-muted-foreground py-6 text-center text-sm'>
                      {t('No public sites configured')}
                    </p>
                  )}
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>{t('Contribution reviews')}</CardTitle>
            <CardDescription>
              {t(
                'Review registration and referral contributions without collecting credentials.'
              )}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {contributionsQuery.isLoading ? (
              <div className='space-y-3'>
                <Skeleton className='h-32' />
                <Skeleton className='h-32' />
              </div>
            ) : (
              <div className='divide-y'>
                {contributions.map((item) => (
                  <article key={item.id} className='space-y-3 py-4 first:pt-0'>
                    <div className='flex flex-wrap items-center justify-between gap-2'>
                      <div>
                        <span className='font-medium'>
                          {item.username ||
                            t('User #{{id}}', { id: item.user_id })}
                        </span>
                        <span className='text-muted-foreground ml-2 text-xs'>
                          {item.site_name ||
                            t('Site #{{id}}', { id: item.site_id })}
                        </span>
                      </div>
                      <Badge variant={contributionStatusVariant(item.status)}>
                        {t(item.status)}
                      </Badge>
                    </div>
                    <div className='bg-muted/40 space-y-1 rounded-lg p-3 text-sm'>
                      {item.description && <p>{item.description}</p>}
                      {item.proof && (
                        <p className='text-muted-foreground'>{item.proof}</p>
                      )}
                    </div>
                    <div className='text-muted-foreground text-xs'>
                      {dayjs.unix(item.created_at).format('YYYY-MM-DD HH:mm')}
                    </div>
                    {item.status === 'pending' && canOperate && (
                      <div className='space-y-2'>
                        <Textarea
                          value={reviewNotes[item.id] ?? ''}
                          onChange={(event) =>
                            setReviewNotes((current) => ({
                              ...current,
                              [item.id]: event.target.value,
                            }))
                          }
                          placeholder={t('Optional review note')}
                          aria-label={t('Review note')}
                        />
                        <div className='flex flex-wrap gap-2'>
                          <Button
                            size='sm'
                            onClick={() =>
                              reviewMutation.mutate({
                                id: item.id,
                                status: 'approved',
                                review_note: reviewNotes[item.id] ?? '',
                              })
                            }
                            disabled={reviewMutation.isPending}
                          >
                            <Check data-icon='inline-start' />
                            {t('Approve')}
                          </Button>
                          <Button
                            size='sm'
                            variant='destructive'
                            onClick={() =>
                              reviewMutation.mutate({
                                id: item.id,
                                status: 'rejected',
                                review_note: reviewNotes[item.id] ?? '',
                              })
                            }
                            disabled={reviewMutation.isPending}
                          >
                            <X data-icon='inline-start' />
                            {t('Reject')}
                          </Button>
                        </div>
                      </div>
                    )}
                    {(!canOperate || item.status !== 'pending') &&
                      item.review_note && (
                        <p className='text-sm'>
                          {t('Review note')}: {item.review_note}
                        </p>
                      )}
                  </article>
                ))}
                {contributions.length === 0 && (
                  <p className='text-muted-foreground py-8 text-center text-sm'>
                    {t('No contribution applications')}
                  </p>
                )}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      <ConfirmDialog
        open={siteToDelete != null}
        onOpenChange={(open) => !open && setSiteToDelete(null)}
        title={t('Delete public site')}
        desc={t(
          'Delete public site "{{name}}"? Sites with contribution records must be disabled instead.',
          { name: siteToDelete?.name ?? '' }
        )}
        confirmText={t('Delete')}
        destructive
        handleConfirm={() => {
          if (siteToDelete) deleteSiteMutation.mutate(siteToDelete.id)
        }}
      />
    </section>
  )
}
