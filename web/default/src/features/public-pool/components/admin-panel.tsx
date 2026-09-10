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
import {
  Check,
  Pencil,
  Plus,
  RefreshCw,
  ShieldCheck,
  Trash2,
  X,
} from 'lucide-react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { RewardAmountField } from '@/components/reward-amount-field'
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
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { useRewardQuotaConfig } from '@/hooks/use-reward-quota-config'
import {
  ADMIN_PERMISSION_ACTIONS,
  ADMIN_PERMISSION_RESOURCES,
  hasPermission,
} from '@/lib/admin-permissions'
import dayjs from '@/lib/dayjs'
import {
  formatPlatformQuota,
  getRewardEquivalent,
  getRewardConversionError,
  isRewardUnchanged,
} from '@/lib/reward-amount'
import { useAuthStore } from '@/stores/auth-store'

import {
  createAdminPublicPoolSite,
  deleteAdminPublicPoolSite,
  getAdminPublicPoolContributions,
  getAdminPublicPoolRewardPlans,
  getAdminPublicPoolSites,
  getPublicPoolStatus,
  retryAdminPublicPoolContributionReward,
  reviewAdminPublicPoolContribution,
  updateAdminPublicPoolSite,
} from '../api'
import {
  publicPoolSiteSchema,
  publicPoolSiteToForm,
  toPublicPoolSitePayload,
  type PublicPoolSiteFormValues,
} from '../lib/admin-form'
import type { PublicPoolContributionStatus, PublicPoolSite } from '../types'

const PAGE_SIZE = 10
const EMPTY_SITE: PublicPoolSiteFormValues = {
  name: '',
  url: '',
  description: '',
  status: 'enabled',
  sort_order: 0,
  reward_type: 'quota',
  reward_amount: '1',
  reward_unit: 'usd',
  subscription_plan_id: 0,
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
  const [siteSearch, setSiteSearch] = useState('')
  const [siteStatusFilter, setSiteStatusFilter] = useState('all')
  const [sitePage, setSitePage] = useState(1)
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState('all')
  const [reviewPage, setReviewPage] = useState(1)
  const form = useForm<PublicPoolSiteFormValues>({
    resolver: zodResolver(publicPoolSiteSchema),
    defaultValues: EMPTY_SITE,
  })
  const rewardType = form.watch('reward_type')
  const rewardUnit = form.watch('reward_unit')
  const rewardAmount = form.watch('reward_amount')
  const subscriptionPlanId = form.watch('subscription_plan_id')
  const quotaConfig = useRewardQuotaConfig(canWrite && rewardType === 'quota')
  const preserveReward = Boolean(
    editingSite &&
    isRewardUnchanged(
      {
        reward_type: rewardType,
        reward_amount: rewardAmount,
        reward_unit: rewardUnit,
        subscription_plan_id: subscriptionPlanId,
      },
      editingSite.reward
    )
  )
  const preservedQuota =
    preserveReward && rewardType === 'quota'
      ? editingSite?.reward?.quota
      : undefined

  const sitesQuery = useQuery({
    queryKey: ['public-pool', 'admin', 'sites'],
    queryFn: getAdminPublicPoolSites,
    enabled: canRead,
    meta: { errorMode: 'local' },
  })
  const contributionsQuery = useQuery({
    queryKey: [
      'public-pool',
      'admin',
      'contributions',
      search,
      statusFilter,
      reviewPage,
    ],
    queryFn: () =>
      getAdminPublicPoolContributions({
        search: search.trim() || undefined,
        status: statusFilter === 'all' ? undefined : statusFilter,
        page: reviewPage,
        page_size: PAGE_SIZE,
      }),
    enabled: canRead,
    meta: { errorMode: 'local' },
  })
  const statusQuery = useQuery({
    queryKey: ['public-pool', 'admin', 'status'],
    queryFn: getPublicPoolStatus,
    enabled: canRead,
    meta: { errorMode: 'local' },
  })
  const plansQuery = useQuery({
    queryKey: ['public-pool', 'admin', 'reward-plans'],
    queryFn: getAdminPublicPoolRewardPlans,
    enabled: canWrite,
    meta: { errorMode: 'local' },
  })

  const invalidate = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['public-pool'] }),
      queryClient.invalidateQueries({ queryKey: ['subscription', 'self'] }),
    ])
  }
  const saveSiteMutation = useMutation({
    mutationFn: async (values: PublicPoolSiteFormValues) => {
      const payload = toPublicPoolSitePayload(values, editingSite)
      return editingSite
        ? updateAdminPublicPoolSite(editingSite.id, payload)
        : createAdminPublicPoolSite(payload)
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
      await invalidate()
    },
  })
  const deleteSiteMutation = useMutation({
    mutationFn: deleteAdminPublicPoolSite,
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Failed to delete public site'))
        return
      }
      setSiteToDelete(null)
      toast.success(t('Public site deleted'))
      await invalidate()
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
      if (
        response.data?.status === 'approved' &&
        response.data.reward_status !== 'succeeded'
      ) {
        toast.error(t('Reward delivery failed'))
      } else {
        toast.success(t('Contribution review saved'))
      }
      await invalidate()
    },
  })
  const retryMutation = useMutation({
    mutationFn: retryAdminPublicPoolContributionReward,
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Failed to retry reward'))
        return
      }
      if (response.data?.reward_status === 'succeeded') {
        toast.success(t('Reward delivered'))
      } else {
        toast.error(t('Reward delivery failed'))
      }
      await invalidate()
    },
  })

  const sites = sitesQuery.data?.success ? (sitesQuery.data.data ?? []) : []
  const siteKeyword = siteSearch.trim().toLocaleLowerCase()
  const filteredSites = sites.filter((site) => {
    const matchesSearch =
      siteKeyword === '' ||
      site.name.toLocaleLowerCase().includes(siteKeyword) ||
      site.url.toLocaleLowerCase().includes(siteKeyword)
    const matchesStatus =
      siteStatusFilter === 'all' || site.status === siteStatusFilter
    return matchesSearch && matchesStatus
  })
  const sitePageCount = Math.max(1, Math.ceil(filteredSites.length / PAGE_SIZE))
  const visibleSitePage = Math.min(sitePage, sitePageCount)
  const visibleSites = filteredSites.slice(
    (visibleSitePage - 1) * PAGE_SIZE,
    visibleSitePage * PAGE_SIZE
  )
  const contributionPage = contributionsQuery.data?.success
    ? contributionsQuery.data.data
    : undefined
  const plans = plansQuery.data?.data ?? []
  const pageItems = contributionPage?.items ?? []
  const pageCount = Math.max(
    1,
    Math.ceil((contributionPage?.total ?? 0) / PAGE_SIZE)
  )
  const sitesFailed = sitesQuery.isError || sitesQuery.data?.success === false
  const contributionsFailed =
    contributionsQuery.isError || contributionsQuery.data?.success === false

  if (!canRead) return null

  const editSite = (site: PublicPoolSite) => {
    setEditingSite(site)
    form.reset(publicPoolSiteToForm(site))
  }

  const submitSite = (values: PublicPoolSiteFormValues) => {
    if (
      values.reward_type === 'quota' &&
      !(editingSite && isRewardUnchanged(values, editingSite.reward))
    ) {
      const error = quotaConfig.isError
        ? quotaConfig.error.message
        : getRewardConversionError(
            values.reward_amount,
            values.reward_unit,
            quotaConfig.data
          )
      if (error || quotaConfig.isFetching) {
        form.setError(
          'reward_amount',
          {
            type: 'validate',
            message: error || 'Loading reward configuration',
          },
          { shouldFocus: true }
        )
        return
      }
    }
    saveSiteMutation.mutate(values)
  }

  const rewardLabel = (site: PublicPoolSite) => {
    if (!site.reward) return t('No reward configured')
    if (site.reward.type === 'subscription') {
      return site.reward.subscription_plan_title || t('Subscription reward')
    }
    const quota = Number(site.reward.quota ?? 0)
    const siteQuotaPerUnit = Number(site.reward.quota_per_unit)
    const equivalent =
      site.reward.unit === 'usd' || site.reward.unit === 'quota'
        ? getRewardEquivalent(String(quota), 'quota', siteQuotaPerUnit)
        : null
    if (equivalent) {
      return t('${{usd}} · {{quota}} platform quota', {
        usd: equivalent.usd,
        quota: formatPlatformQuota(equivalent.quota),
      })
    }
    return t('{{quota}} platform quota', {
      quota: formatPlatformQuota(Number.isFinite(quota) ? quota : null),
    })
  }

  let siteSubmitIcon = <Plus data-icon='inline-start' />
  if (saveSiteMutation.isPending) {
    siteSubmitIcon = <Spinner data-icon='inline-start' />
  } else if (editingSite) {
    siteSubmitIcon = <Check data-icon='inline-start' />
  }

  const runtimeStatus = statusQuery.data?.data
  let runtimeStatusContent = (
    <div className='space-y-3 text-center'>
      <p className='text-muted-foreground text-sm'>{t('Request failed')}</p>
      <Button
        size='sm'
        variant='outline'
        onClick={() => void statusQuery.refetch()}
      >
        {t('Retry')}
      </Button>
    </div>
  )
  if (statusQuery.isLoading) {
    runtimeStatusContent = <Skeleton className='h-24' />
  } else if (runtimeStatus) {
    runtimeStatusContent = (
      <dl className='grid gap-4 sm:grid-cols-3'>
        <div>
          <dt className='text-muted-foreground text-xs'>{t('Available')}</dt>
          <dd className='mt-1 font-medium'>
            {runtimeStatus.available ? t('Yes') : t('No')}
          </dd>
        </div>
        <div>
          <dt className='text-muted-foreground text-xs'>
            {t('Channel count')}
          </dt>
          <dd className='mt-1 font-medium'>{runtimeStatus.channel_count}</dd>
        </div>
        <div>
          <dt className='text-muted-foreground text-xs'>{t('Group ratio')}</dt>
          <dd className='mt-1 font-medium'>{runtimeStatus.group_ratio}</dd>
        </div>
      </dl>
    )
  }

  return (
    <section className='space-y-4'>
      <div className='flex items-start gap-3'>
        <ShieldCheck className='text-primary mt-1 size-5' aria-hidden='true' />
        <div>
          <h2 className='text-lg font-semibold'>
            {t('Public pool management')}
          </h2>
          <p className='text-muted-foreground text-sm'>
            {t('Maintain public sites, rewards, reviews, and runtime status.')}
          </p>
        </div>
      </div>

      <Tabs defaultValue='sites'>
        <TabsList>
          <TabsTrigger value='sites'>{t('Site management')}</TabsTrigger>
          <TabsTrigger value='reviews'>{t('Contribution reviews')}</TabsTrigger>
          <TabsTrigger value='status'>{t('Runtime status')}</TabsTrigger>
        </TabsList>

        <TabsContent value='sites' className='mt-4 grid gap-6 xl:grid-cols-2'>
          {canWrite && (
            <Card>
              <CardHeader>
                <CardTitle>
                  {t(editingSite ? 'Edit public site' : 'Add public site')}
                </CardTitle>
                <CardDescription>
                  {t(
                    'The configured reward is frozen when a user submits proof.'
                  )}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <form onSubmit={form.handleSubmit(submitSite)}>
                  <FieldGroup>
                    <Field data-invalid={Boolean(form.formState.errors.name)}>
                      <FieldLabel htmlFor='public-pool-admin-name'>
                        {t('Site name')}
                      </FieldLabel>
                      <Input
                        id='public-pool-admin-name'
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
                        {...form.register('url')}
                      />
                      <FieldError>
                        {form.formState.errors.url?.message
                          ? t(form.formState.errors.url.message)
                          : null}
                      </FieldError>
                    </Field>
                    <Field>
                      <FieldLabel htmlFor='public-pool-admin-description'>
                        {t('Description')}
                      </FieldLabel>
                      <Textarea
                        id='public-pool-admin-description'
                        {...form.register('description')}
                      />
                      <FieldError>
                        {form.formState.errors.description?.message
                          ? t(form.formState.errors.description.message)
                          : null}
                      </FieldError>
                    </Field>
                    <div className='grid gap-3 sm:grid-cols-2'>
                      <Field>
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
                    <Field>
                      <FieldLabel htmlFor='public-pool-reward-type'>
                        {t('Fixed reward')}
                      </FieldLabel>
                      <NativeSelect
                        id='public-pool-reward-type'
                        {...form.register('reward_type')}
                      >
                        <NativeSelectOption value='none'>
                          {t('No reward')}
                        </NativeSelectOption>
                        <NativeSelectOption value='quota'>
                          {t('Main account quota')}
                        </NativeSelectOption>
                        <NativeSelectOption value='subscription'>
                          {t('Subscription')}
                        </NativeSelectOption>
                      </NativeSelect>
                    </Field>
                    {rewardType === 'quota' && (
                      <RewardAmountField
                        id='public-pool-reward'
                        amount={rewardAmount}
                        unit={rewardUnit}
                        amountField={form.register('reward_amount')}
                        unitField={form.register('reward_unit')}
                        config={quotaConfig}
                        preservedQuota={preservedQuota}
                        error={form.formState.errors.reward_amount?.message}
                      />
                    )}
                    {rewardType === 'subscription' && (
                      <Field
                        data-invalid={Boolean(
                          form.formState.errors.subscription_plan_id
                        )}
                      >
                        <FieldLabel htmlFor='public-pool-reward-plan'>
                          {t('Subscription plan')}
                        </FieldLabel>
                        <NativeSelect
                          id='public-pool-reward-plan'
                          {...form.register('subscription_plan_id', {
                            valueAsNumber: true,
                          })}
                        >
                          <NativeSelectOption value='0'>
                            {t('Select a subscription plan')}
                          </NativeSelectOption>
                          {plans.map((plan) => (
                            <NativeSelectOption key={plan.id} value={plan.id}>
                              {plan.title}
                            </NativeSelectOption>
                          ))}
                        </NativeSelect>
                        {plansQuery.isLoading && (
                          <p role='status'>
                            <Spinner />
                            {t('Loading subscription plans')}
                          </p>
                        )}
                        {(plansQuery.isError ||
                          plansQuery.data?.success === false) && (
                          <div role='alert'>
                            <p>{t('Failed to load subscription plans')}</p>
                            <Button
                              type='button'
                              variant='outline'
                              onClick={() => void plansQuery.refetch()}
                            >
                              {t('Retry')}
                            </Button>
                          </div>
                        )}
                        {plansQuery.isSuccess &&
                          plansQuery.data?.success &&
                          plans.length === 0 && (
                            <FieldDescription>
                              {t('No subscription plans available')}
                            </FieldDescription>
                          )}
                        <FieldError>
                          {form.formState.errors.subscription_plan_id?.message
                            ? t(
                                form.formState.errors.subscription_plan_id
                                  .message
                              )
                            : null}
                        </FieldError>
                      </Field>
                    )}
                    <div className='flex flex-wrap gap-2'>
                      <Button
                        type='submit'
                        disabled={
                          saveSiteMutation.isPending ||
                          (rewardType === 'quota' &&
                            !preserveReward &&
                            (!quotaConfig.isSuccess || quotaConfig.isFetching))
                        }
                      >
                        {siteSubmitIcon}
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
            <CardContent className='space-y-4'>
              <div className='grid gap-3 sm:grid-cols-2'>
                <Input
                  value={siteSearch}
                  onChange={(event) => {
                    setSiteSearch(event.target.value)
                    setSitePage(1)
                  }}
                  placeholder={t('Search')}
                />
                <NativeSelect
                  value={siteStatusFilter}
                  onChange={(event) => {
                    setSiteStatusFilter(event.target.value)
                    setSitePage(1)
                  }}
                >
                  <NativeSelectOption value='all'>
                    {t('All statuses')}
                  </NativeSelectOption>
                  <NativeSelectOption value='enabled'>
                    {t('enabled')}
                  </NativeSelectOption>
                  <NativeSelectOption value='disabled'>
                    {t('disabled')}
                  </NativeSelectOption>
                </NativeSelect>
              </div>
              {sitesQuery.isLoading && (
                <div className='space-y-3'>
                  <Skeleton className='h-24' />
                  <Skeleton className='h-24' />
                </div>
              )}
              {sitesFailed && (
                <div className='space-y-3 py-8 text-center'>
                  <p className='text-muted-foreground text-sm'>
                    {t('Request failed')}
                  </p>
                  <Button
                    size='sm'
                    variant='outline'
                    onClick={() => void sitesQuery.refetch()}
                  >
                    {t('Retry')}
                  </Button>
                </div>
              )}
              {!sitesQuery.isLoading && !sitesFailed && (
                <div className='divide-y'>
                  {visibleSites.map((site) => (
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
                          <p className='mt-1 text-xs'>{rewardLabel(site)}</p>
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
                  {filteredSites.length === 0 && (
                    <p className='text-muted-foreground py-8 text-center text-sm'>
                      {t('No public sites configured')}
                    </p>
                  )}
                </div>
              )}
              {sitePageCount > 1 && (
                <div className='flex items-center justify-between gap-3'>
                  <Button
                    size='sm'
                    variant='outline'
                    disabled={visibleSitePage <= 1}
                    onClick={() => setSitePage(visibleSitePage - 1)}
                  >
                    {t('Previous')}
                  </Button>
                  <span className='text-muted-foreground text-xs'>
                    {t('Page {{current}} of {{total}}', {
                      current: visibleSitePage,
                      total: sitePageCount,
                    })}
                  </span>
                  <Button
                    size='sm'
                    variant='outline'
                    disabled={visibleSitePage >= sitePageCount}
                    onClick={() => setSitePage(visibleSitePage + 1)}
                  >
                    {t('Next')}
                  </Button>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value='reviews' className='mt-4'>
          <Card>
            <CardHeader>
              <CardTitle>{t('Contribution reviews')}</CardTitle>
              <CardDescription>
                {t(
                  'Approving a contribution delivers its frozen reward exactly once.'
                )}
              </CardDescription>
            </CardHeader>
            <CardContent className='space-y-4'>
              <div className='grid gap-3 sm:grid-cols-2'>
                <Input
                  value={search}
                  onChange={(event) => {
                    setSearch(event.target.value)
                    setReviewPage(1)
                  }}
                  placeholder={t('Search user or site')}
                />
                <NativeSelect
                  value={statusFilter}
                  onChange={(event) => {
                    setStatusFilter(event.target.value)
                    setReviewPage(1)
                  }}
                >
                  <NativeSelectOption value='all'>
                    {t('All statuses')}
                  </NativeSelectOption>
                  <NativeSelectOption value='pending'>
                    {t('pending')}
                  </NativeSelectOption>
                  <NativeSelectOption value='approved'>
                    {t('approved')}
                  </NativeSelectOption>
                  <NativeSelectOption value='rejected'>
                    {t('rejected')}
                  </NativeSelectOption>
                </NativeSelect>
              </div>
              {contributionsQuery.isLoading && (
                <div className='space-y-3'>
                  <Skeleton className='h-32' />
                  <Skeleton className='h-32' />
                </div>
              )}
              {contributionsFailed && (
                <div className='space-y-3 py-8 text-center'>
                  <p className='text-muted-foreground text-sm'>
                    {t('Request failed')}
                  </p>
                  <Button
                    size='sm'
                    variant='outline'
                    onClick={() => void contributionsQuery.refetch()}
                  >
                    {t('Retry')}
                  </Button>
                </div>
              )}
              {!contributionsQuery.isLoading && !contributionsFailed && (
                <div className='divide-y'>
                  {pageItems.map((item) => {
                    let rewardStatusLabel = t('Pending')
                    if (item.reward_status === 'succeeded') {
                      rewardStatusLabel = t('Rewarded')
                    } else if (item.reward_status === 'failed') {
                      rewardStatusLabel = t('Failed')
                    }
                    return (
                      <article
                        key={item.id}
                        className='space-y-3 py-4 first:pt-0'
                      >
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
                          <div className='flex items-center gap-2'>
                            {item.status === 'approved' && item.reward && (
                              <Badge
                                variant={
                                  item.reward_status === 'failed'
                                    ? 'destructive'
                                    : 'outline'
                                }
                              >
                                {rewardStatusLabel}
                              </Badge>
                            )}
                            <Badge
                              variant={contributionStatusVariant(item.status)}
                            >
                              {t(item.status)}
                            </Badge>
                          </div>
                        </div>
                        <div className='bg-muted/40 space-y-1 rounded-lg p-3 text-sm'>
                          {item.description && <p>{item.description}</p>}
                          {item.proof && (
                            <p className='text-muted-foreground'>
                              {item.proof}
                            </p>
                          )}
                        </div>
                        <div className='text-muted-foreground text-xs'>
                          {dayjs
                            .unix(item.created_at)
                            .format('YYYY-MM-DD HH:mm')}
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
                            />
                            <div className='flex flex-wrap gap-2'>
                              <Button
                                size='sm'
                                disabled={
                                  reviewMutation.isPending || !item.reward
                                }
                                onClick={() =>
                                  reviewMutation.mutate({
                                    id: item.id,
                                    status: 'approved',
                                    review_note: reviewNotes[item.id] ?? '',
                                  })
                                }
                              >
                                <Check data-icon='inline-start' />
                                {t('Approve and reward')}
                              </Button>
                              <Button
                                size='sm'
                                variant='destructive'
                                disabled={reviewMutation.isPending}
                                onClick={() =>
                                  reviewMutation.mutate({
                                    id: item.id,
                                    status: 'rejected',
                                    review_note: reviewNotes[item.id] ?? '',
                                  })
                                }
                              >
                                <X data-icon='inline-start' />
                                {t('Reject')}
                              </Button>
                            </div>
                          </div>
                        )}
                        {item.status === 'approved' &&
                          item.reward &&
                          item.reward_status !== 'succeeded' &&
                          canOperate && (
                            <div className='space-y-2'>
                              <p className='text-destructive text-sm'>
                                {item.reward_failure_reason ||
                                  t('Reward delivery failed')}
                              </p>
                              <Button
                                size='sm'
                                variant='outline'
                                disabled={retryMutation.isPending}
                                onClick={() => retryMutation.mutate(item.id)}
                              >
                                <RefreshCw data-icon='inline-start' />
                                {t('Retry reward')}
                              </Button>
                            </div>
                          )}
                      </article>
                    )
                  })}
                  {pageItems.length === 0 && (
                    <p className='text-muted-foreground py-8 text-center text-sm'>
                      {t('No contribution applications')}
                    </p>
                  )}
                </div>
              )}
              <div className='flex items-center justify-end gap-2'>
                <Button
                  size='sm'
                  variant='outline'
                  disabled={reviewPage <= 1}
                  onClick={() => setReviewPage((page) => Math.max(1, page - 1))}
                >
                  {t('Previous')}
                </Button>
                <span className='text-muted-foreground text-xs'>
                  {reviewPage}/{pageCount}
                </span>
                <Button
                  size='sm'
                  variant='outline'
                  disabled={reviewPage >= pageCount}
                  onClick={() =>
                    setReviewPage((page) => Math.min(pageCount, page + 1))
                  }
                >
                  {t('Next')}
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value='status' className='mt-4'>
          <Card>
            <CardHeader>
              <CardTitle>{t('Runtime status')}</CardTitle>
              <CardDescription>
                {t('Public pool availability and zero-cost group validation.')}
              </CardDescription>
            </CardHeader>
            <CardContent>{runtimeStatusContent}</CardContent>
          </Card>
        </TabsContent>
      </Tabs>

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
