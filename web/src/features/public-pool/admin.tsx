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
import { RotateCcw, Save, Trash2 } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { StaticDataTable } from '@/components/data-table'
import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Spinner } from '@/components/ui/spinner'
import { Textarea } from '@/components/ui/textarea'
import { toIntlLocale } from '@/i18n/languages'
import {
  ADMIN_PERMISSION_ACTIONS,
  ADMIN_PERMISSION_RESOURCES,
  hasPermission,
} from '@/lib/admin-permissions'
import { formatNumber } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import {
  getServerErrorMessage,
  requireServerSuccess,
} from '@/lib/server-error-message'
import { useAuthStore } from '@/stores/auth-store'

import {
  createAdminPublicPoolSite,
  deleteAdminPublicPoolSite,
  getAdminPublicPoolContributions,
  getAdminPublicPoolRewardPlans,
  getAdminPublicPoolSites,
  retryAdminPublicPoolContributionReward,
  reviewAdminPublicPoolContribution,
  updateAdminPublicPoolSite,
} from './api'
import type {
  PublicPoolSite,
  PublicPoolSitePayload,
  RewardSubscriptionPlanOption,
} from './types'

type SiteDraft = {
  id?: number
  name: string
  url: string
  description: string
  status: 'enabled' | 'disabled'
  sort_order: string
  rewardType: '' | 'quota' | 'subscription'
  rewardAmount: string
  rewardUnit: 'usd' | 'cny' | 'quota'
  planId: string
  replaceReward: boolean
}

const emptyDraft: SiteDraft = {
  name: '',
  url: '',
  description: '',
  status: 'enabled',
  sort_order: '0',
  rewardType: 'quota',
  rewardAmount: '1',
  rewardUnit: 'quota',
  planId: '',
  replaceReward: true,
}

const EMPTY_REWARD_PLANS: RewardSubscriptionPlanOption[] = []

function contributionStatusVariant(
  status: string
): 'default' | 'secondary' | 'destructive' {
  if (status === 'approved') return 'default'
  if (status === 'rejected') return 'destructive'
  return 'secondary'
}

function draftFromSite(site: PublicPoolSite): SiteDraft {
  return {
    id: site.id,
    name: site.name,
    url: site.url,
    description: site.description,
    status: site.status,
    sort_order: String(site.sort_order),
    rewardType: site.reward?.type ?? '',
    rewardAmount:
      site.reward?.amount ??
      (site.reward?.quota ? String(site.reward.quota) : ''),
    rewardUnit: site.reward?.unit ?? 'quota',
    planId: site.reward?.subscription_plan_id
      ? String(site.reward.subscription_plan_id)
      : '',
    replaceReward: false,
  }
}

function toPayload(draft: SiteDraft): PublicPoolSitePayload {
  const reward =
    draft.rewardType === ''
      ? null
      : {
          type: draft.rewardType,
          amount: draft.rewardType === 'subscription' ? '' : draft.rewardAmount,
          unit:
            draft.rewardType === 'subscription' ? 'quota' : draft.rewardUnit,
          subscription_plan_id:
            draft.rewardType === 'subscription' ? Number(draft.planId) : 0,
        }
  return {
    preserve_reward: Boolean(draft.id && !draft.replaceReward),
    name: draft.name.trim(),
    url: draft.url.trim(),
    description: draft.description.trim(),
    status: draft.status,
    sort_order: Number(draft.sort_order) || 0,
    reward,
  }
}

export function PublicPoolAdmin() {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const user = useAuthStore((state) => state.auth.user)
  const canRead = hasPermission(
    user,
    ADMIN_PERMISSION_RESOURCES.PUBLIC_POOL,
    ADMIN_PERMISSION_ACTIONS.READ
  )
  const canWrite = hasPermission(
    user,
    ADMIN_PERMISSION_RESOURCES.PUBLIC_POOL,
    ADMIN_PERMISSION_ACTIONS.WRITE
  )
  const canOperate = hasPermission(
    user,
    ADMIN_PERMISSION_RESOURCES.PUBLIC_POOL,
    ADMIN_PERMISSION_ACTIONS.OPERATE
  )
  const queryClient = useQueryClient()
  const [draft, setDraft] = useState<SiteDraft>(emptyDraft)
  const [deleteTarget, setDeleteTarget] = useState<PublicPoolSite | null>(null)
  const [reviewNote, setReviewNote] = useState<Record<number, string>>({})
  const sitesQuery = useQuery({
    queryKey: ['public-pool', 'admin', 'sites', user?.id],
    queryFn: async () => requireServerSuccess(await getAdminPublicPoolSites()),
    enabled: canRead,
  })
  const plansQuery = useQuery({
    queryKey: ['public-pool', 'admin', 'reward-plans', user?.id],
    queryFn: async () =>
      requireServerSuccess(await getAdminPublicPoolRewardPlans()),
    enabled: canRead && canWrite,
  })
  const contributionsQuery = useQuery({
    queryKey: ['public-pool', 'admin', 'contributions', user?.id],
    queryFn: async () =>
      requireServerSuccess(
        await getAdminPublicPoolContributions({ page: 1, page_size: 100 })
      ),
    enabled: canRead,
  })
  const refresh = async (): Promise<void> => {
    await Promise.all([
      queryClient.invalidateQueries({
        queryKey: ['public-pool', 'admin', 'sites'],
      }),
      queryClient.invalidateQueries({
        queryKey: ['public-pool', 'admin', 'contributions'],
      }),
    ])
  }
  const saveMutation = useMutation({
    mutationFn: async () =>
      requireServerSuccess(
        await (draft.id
          ? updateAdminPublicPoolSite(draft.id, toPayload(draft))
          : createAdminPublicPoolSite(toPayload(draft)))
      ),
    onSuccess: async () => {
      toast.success(t('Saved'))
      setDraft(emptyDraft)
      await refresh()
    },
    onError: (error) => handleServerError(error, t('Request failed')),
  })
  const deleteMutation = useMutation({
    mutationFn: async (id: number) =>
      requireServerSuccess(await deleteAdminPublicPoolSite(id)),
    onSuccess: async (_response, id) => {
      toast.success(t('Deleted'))
      setDeleteTarget(null)
      if (draft.id === id) setDraft(emptyDraft)
      await refresh()
    },
    onError: (error) => handleServerError(error, t('Request failed')),
  })
  const reviewMutation = useMutation({
    mutationFn: async ({
      id,
      status,
    }: {
      id: number
      status: 'approved' | 'rejected'
    }) =>
      requireServerSuccess(
        await reviewAdminPublicPoolContribution(id, {
          status,
          review_note: reviewNote[id]?.trim() || '',
        })
      ),
    onSuccess: async () => {
      toast.success(t('Review saved'))
      await refresh()
    },
    onError: (error) => handleServerError(error, t('Request failed')),
  })
  const retryMutation = useMutation({
    mutationFn: async (id: number) =>
      requireServerSuccess(await retryAdminPublicPoolContributionReward(id)),
    onSuccess: async () => {
      toast.success(t('Reward retry submitted'))
      await refresh()
    },
    onError: (error) => handleServerError(error, t('Request failed')),
  })
  const sites = sitesQuery.data?.data ?? []
  const contributions = contributionsQuery.data?.data?.items ?? []
  const plans: RewardSubscriptionPlanOption[] =
    plansQuery.data?.data ?? EMPTY_REWARD_PLANS
  const selectedPlanAvailable = useMemo(
    () =>
      draft.rewardType !== 'subscription' ||
      !draft.replaceReward ||
      plans.some((plan) => String(plan.id) === draft.planId),
    [draft.planId, draft.rewardType, draft.replaceReward, plans]
  )

  if (!canRead) {
    return (
      <ErrorState title={t('You do not have permission to view this page')} />
    )
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Public Pool Management')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='grid gap-4'>
          {canWrite && (
            <section className='bg-card space-y-4 rounded-lg border p-5'>
              <div className='flex items-center justify-between gap-3'>
                <div>
                  <h2 className='font-semibold'>
                    {draft.id ? t('Edit public site') : t('Add public site')}
                  </h2>
                  <p className='text-muted-foreground text-sm'>
                    {t(
                      'Rewards are captured when the contribution is submitted.'
                    )}
                  </p>
                </div>
                {draft.id && (
                  <Button
                    variant='ghost'
                    size='sm'
                    onClick={() => setDraft(emptyDraft)}
                  >
                    {t('Cancel')}
                  </Button>
                )}
              </div>
              <div className='grid gap-3 md:grid-cols-2'>
                <div className='space-y-1.5'>
                  <Label htmlFor='pool-name'>{t('Name')}</Label>
                  <Input
                    id='pool-name'
                    value={draft.name}
                    onChange={(event) =>
                      setDraft({ ...draft, name: event.target.value })
                    }
                  />
                </div>
                <div className='space-y-1.5'>
                  <Label htmlFor='pool-url'>{t('URL')}</Label>
                  <Input
                    id='pool-url'
                    value={draft.url}
                    onChange={(event) =>
                      setDraft({ ...draft, url: event.target.value })
                    }
                  />
                </div>
                <div className='space-y-1.5 md:col-span-2'>
                  <Label htmlFor='pool-description'>{t('Description')}</Label>
                  <Textarea
                    id='pool-description'
                    value={draft.description}
                    onChange={(event) =>
                      setDraft({ ...draft, description: event.target.value })
                    }
                  />
                </div>
                <div className='space-y-1.5'>
                  <Label htmlFor='pool-status'>{t('Status')}</Label>
                  <Select
                    items={[
                      { value: 'enabled', label: t('enabled') },
                      { value: 'disabled', label: t('disabled') },
                    ]}
                    value={draft.status}
                    onValueChange={(value) =>
                      setDraft({
                        ...draft,
                        status: value as SiteDraft['status'],
                      })
                    }
                  >
                    <SelectTrigger id='pool-status' className='w-full'>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value='enabled'>{t('enabled')}</SelectItem>
                      <SelectItem value='disabled'>{t('disabled')}</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className='space-y-1.5'>
                  <Label htmlFor='pool-sort'>{t('Sort order')}</Label>
                  <Input
                    id='pool-sort'
                    type='number'
                    value={draft.sort_order}
                    onChange={(event) =>
                      setDraft({ ...draft, sort_order: event.target.value })
                    }
                  />
                </div>
                <div className='space-y-1.5'>
                  <Label htmlFor='pool-reward-type'>{t('Reward type')}</Label>
                  <Select
                    disabled={Boolean(draft.id && !draft.replaceReward)}
                    items={[
                      { value: '', label: t('No reward') },
                      { value: 'quota', label: t('Quota') },
                      { value: 'subscription', label: t('Subscription') },
                    ]}
                    value={draft.rewardType}
                    onValueChange={(value) =>
                      setDraft({
                        ...draft,
                        rewardType: value as SiteDraft['rewardType'],
                      })
                    }
                  >
                    <SelectTrigger id='pool-reward-type' className='w-full'>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value=''>{t('No reward')}</SelectItem>
                      <SelectItem value='quota'>{t('Quota')}</SelectItem>
                      <SelectItem value='subscription'>
                        {t('Subscription')}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                {draft.id && (
                  <div className='flex items-center gap-2 md:col-span-2'>
                    <Checkbox
                      id='pool-replace-reward'
                      checked={draft.replaceReward}
                      onCheckedChange={(checked) =>
                        setDraft({ ...draft, replaceReward: checked })
                      }
                    />
                    <Label htmlFor='pool-replace-reward'>
                      {t('Replace rewards for future contributions')}
                    </Label>
                  </div>
                )}
                {draft.rewardType === 'quota' && (
                  <>
                    <div className='space-y-1.5'>
                      <Label htmlFor='pool-reward-amount'>
                        {t('Reward amount')}
                      </Label>
                      <Input
                        id='pool-reward-amount'
                        disabled={Boolean(draft.id && !draft.replaceReward)}
                        value={draft.rewardAmount}
                        onChange={(event) =>
                          setDraft({
                            ...draft,
                            rewardAmount: event.target.value,
                          })
                        }
                      />
                    </div>
                    <div className='space-y-1.5'>
                      <Label htmlFor='pool-reward-unit'>
                        {t('Reward unit')}
                      </Label>
                      <Select
                        disabled={Boolean(draft.id && !draft.replaceReward)}
                        items={[
                          { value: 'quota', label: t('Quota') },
                          { value: 'usd', label: 'USD' },
                          { value: 'cny', label: 'CNY' },
                        ]}
                        value={draft.rewardUnit}
                        onValueChange={(value) =>
                          setDraft({
                            ...draft,
                            rewardUnit: value as SiteDraft['rewardUnit'],
                          })
                        }
                      >
                        <SelectTrigger id='pool-reward-unit' className='w-full'>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value='quota'>{t('Quota')}</SelectItem>
                          <SelectItem value='usd'>USD</SelectItem>
                          <SelectItem value='cny'>CNY</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </>
                )}
                {draft.rewardType === 'subscription' && (
                  <div className='space-y-1.5'>
                    <Label htmlFor='pool-reward-plan'>
                      {t('Subscription plan')}
                    </Label>
                    <Select
                      disabled={
                        Boolean(draft.id && !draft.replaceReward) ||
                        plansQuery.isPending
                      }
                      items={plans.map((plan) => ({
                        value: String(plan.id),
                        label: plan.title,
                      }))}
                      value={draft.planId}
                      onValueChange={(value) =>
                        setDraft({ ...draft, planId: value ?? '' })
                      }
                    >
                      <SelectTrigger id='pool-reward-plan' className='w-full'>
                        <SelectValue placeholder={t('Select a plan')} />
                      </SelectTrigger>
                      <SelectContent>
                        {plans.map((plan) => (
                          <SelectItem key={plan.id} value={String(plan.id)}>
                            {plan.title}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                )}
              </div>
              <Button
                disabled={
                  saveMutation.isPending ||
                  !draft.name.trim() ||
                  !draft.url.trim() ||
                  !selectedPlanAvailable
                }
                onClick={() => saveMutation.mutate()}
              >
                <Save data-icon='inline-start' />
                {t('Save')}
              </Button>
            </section>
          )}

          <section className='bg-card space-y-3 rounded-lg border p-5'>
            <div className='flex items-center justify-between'>
              <h2 className='font-semibold'>{t('Public sites')}</h2>
              {sitesQuery.isLoading && <Spinner className='size-4' />}
            </div>
            {sitesQuery.isError && (
              <ErrorState
                description={getServerErrorMessage(sitesQuery.error)}
                onRetry={() => void sitesQuery.refetch()}
              />
            )}
            {!sitesQuery.isError &&
              !sitesQuery.isPending &&
              sites.length === 0 && (
                <p className='text-muted-foreground text-sm'>
                  {t('No public pool sites are available')}
                </p>
              )}
            {!sitesQuery.isError && sites.length > 0 && (
              <div className='grid gap-2'>
                {sites.map((site) => (
                  <div
                    key={site.id}
                    className='flex flex-wrap items-center justify-between gap-3 border-b py-2 text-sm last:border-0'
                  >
                    <div className='min-w-0'>
                      <p className='font-medium'>{site.name}</p>
                      <p className='text-muted-foreground truncate'>
                        {site.url}
                      </p>
                    </div>
                    <div className='flex items-center gap-2'>
                      <Badge
                        variant={
                          site.status === 'enabled' ? 'default' : 'secondary'
                        }
                      >
                        {t(site.status)}
                      </Badge>
                      {canWrite && (
                        <>
                          <Button
                            size='icon-sm'
                            variant='ghost'
                            aria-label={t('Edit')}
                            onClick={() => setDraft(draftFromSite(site))}
                          >
                            <Save />
                          </Button>
                          <Button
                            size='icon-sm'
                            variant='ghost'
                            aria-label={t('Delete')}
                            disabled={deleteMutation.isPending}
                            onClick={() => setDeleteTarget(site)}
                          >
                            <Trash2 />
                          </Button>
                        </>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </section>

          <section className='bg-card space-y-3 rounded-lg border p-5'>
            <h2 className='font-semibold'>{t('Contribution reviews')}</h2>
            {contributionsQuery.isError ? (
              <ErrorState
                description={getServerErrorMessage(contributionsQuery.error)}
                onRetry={() => void contributionsQuery.refetch()}
              />
            ) : (
              <StaticDataTable
                data={contributions}
                getRowKey={(item) => item.id}
                emptyContent={
                  contributionsQuery.isPending ? (
                    <Spinner aria-label={t('Loading')} />
                  ) : (
                    t('No contributions yet')
                  )
                }
                columns={[
                  {
                    id: 'user',
                    header: t('User'),
                    cell: (item) => item.username || item.user_id,
                  },
                  {
                    id: 'site',
                    header: t('Site'),
                    cell: (item) => item.site_name || item.site_id,
                  },
                  {
                    id: 'evidence',
                    header: t('Contribution evidence'),
                    cellClassName: 'min-w-56 max-w-96 whitespace-normal',
                    cell: (item) => (
                      <div className='space-y-1 break-words'>
                        {item.description && <p>{item.description}</p>}
                        {item.proof && (
                          <p className='text-muted-foreground'>{item.proof}</p>
                        )}
                      </div>
                    ),
                  },
                  {
                    id: 'reward',
                    header: t('Reward snapshot'),
                    cell: (item) => {
                      if (!item.reward) return t('No reward')
                      if (item.reward.type === 'subscription') {
                        return (
                          item.reward.subscription_plan_title ||
                          t('Subscription')
                        )
                      }
                      return (
                        <div>
                          <span>
                            {formatNumber(item.reward.quota ?? 0, locale)}
                          </span>{' '}
                          {t('Quota')}
                        </div>
                      )
                    },
                  },
                  {
                    id: 'status',
                    header: t('Status'),
                    cell: (item) => (
                      <div className='space-y-1'>
                        <Badge variant={contributionStatusVariant(item.status)}>
                          {t(item.status)}
                        </Badge>
                        {item.reward_status && (
                          <p className='text-muted-foreground text-xs'>
                            {t(item.reward_status)}
                          </p>
                        )}
                        {item.reward_failure_reason && (
                          <p className='text-destructive max-w-64 break-words whitespace-normal'>
                            {item.reward_failure_reason}
                          </p>
                        )}
                      </div>
                    ),
                  },
                  {
                    id: 'note',
                    header: t('Review note'),
                    cellClassName: 'min-w-48',
                    cell: (item) =>
                      canOperate && item.status === 'pending' ? (
                        <Input
                          aria-label={t('Review note')}
                          value={reviewNote[item.id] ?? ''}
                          disabled={reviewMutation.isPending}
                          onChange={(event) =>
                            setReviewNote({
                              ...reviewNote,
                              [item.id]: event.target.value,
                            })
                          }
                          placeholder={t('Optional note')}
                        />
                      ) : (
                        <span>{item.review_note || '—'}</span>
                      ),
                  },
                  {
                    id: 'actions',
                    header: t('Actions'),
                    cell: (item) => (
                      <div className='flex gap-1'>
                        {canOperate && item.status === 'pending' && (
                          <>
                            <Button
                              size='sm'
                              disabled={
                                reviewMutation.isPending ||
                                retryMutation.isPending
                              }
                              onClick={() =>
                                reviewMutation.mutate({
                                  id: item.id,
                                  status: 'approved',
                                })
                              }
                            >
                              {t('Approve')}
                            </Button>
                            <Button
                              size='sm'
                              variant='outline'
                              disabled={
                                reviewMutation.isPending ||
                                retryMutation.isPending
                              }
                              onClick={() =>
                                reviewMutation.mutate({
                                  id: item.id,
                                  status: 'rejected',
                                })
                              }
                            >
                              {t('Reject')}
                            </Button>
                          </>
                        )}
                        {canOperate &&
                          item.status === 'approved' &&
                          item.reward_status === 'failed' && (
                            <Button
                              size='sm'
                              variant='outline'
                              disabled={
                                retryMutation.isPending ||
                                reviewMutation.isPending
                              }
                              onClick={() => retryMutation.mutate(item.id)}
                            >
                              <RotateCcw data-icon='inline-start' />
                              {t('Retry reward')}
                            </Button>
                          )}
                      </div>
                    ),
                  },
                ]}
              />
            )}
          </section>
        </div>
        <ConfirmDialog
          open={canWrite && deleteTarget != null}
          onOpenChange={(open) => {
            if (!open && !deleteMutation.isPending) setDeleteTarget(null)
          }}
          title={t('Delete this site?')}
          desc={deleteTarget?.name ?? ''}
          confirmText={t('Delete')}
          destructive
          isLoading={deleteMutation.isPending}
          disabled={!canWrite}
          handleConfirm={() => {
            if (canWrite && deleteTarget && !deleteMutation.isPending) {
              deleteMutation.mutate(deleteTarget.id)
            }
          }}
        />
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
