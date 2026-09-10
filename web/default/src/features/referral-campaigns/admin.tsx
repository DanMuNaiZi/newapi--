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
import dayjs from 'dayjs'
import { Pencil, Plus, RefreshCw } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  ADMIN_PERMISSION_ACTIONS,
  ADMIN_PERMISSION_RESOURCES,
  hasPermission,
} from '@/lib/admin-permissions'
import { formatPlatformQuota } from '@/lib/reward-amount'
import { useAuthStore } from '@/stores/auth-store'

import {
  getAdminReferralCampaigns,
  getReferralRewardPlans,
  getReferralCampaignEvents,
  retryReferralCampaignReward,
} from './api'
import { ReferralCampaignFormDrawer } from './components/referral-campaign-form-drawer'
import type { ReferralCampaign } from './types'

export function ReferralCampaignAdmin() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const currentUser = useAuthStore((state) => state.auth.user)
  const canWrite = hasPermission(
    currentUser,
    ADMIN_PERMISSION_RESOURCES.REFERRAL_CAMPAIGN,
    ADMIN_PERMISSION_ACTIONS.WRITE
  )
  const canOperate = hasPermission(
    currentUser,
    ADMIN_PERMISSION_RESOURCES.REFERRAL_CAMPAIGN,
    ADMIN_PERMISSION_ACTIONS.OPERATE
  )
  const [editing, setEditing] = useState<ReferralCampaign | null>(null)
  const [formOpen, setFormOpen] = useState(false)
  const [selectedCampaignId, setSelectedCampaignId] = useState<number | null>(
    null
  )
  const [eventPage, setEventPage] = useState(1)

  const campaignsQuery = useQuery({
    queryKey: ['referral-campaigns', 'admin'],
    queryFn: getAdminReferralCampaigns,
    meta: { errorMode: 'local' },
  })
  const plansQuery = useQuery({
    queryKey: ['referral-campaigns', 'reward-plans'],
    queryFn: getReferralRewardPlans,
    enabled: canWrite,
    meta: { errorMode: 'local' },
  })
  const eventsQuery = useQuery({
    queryKey: ['referral-campaign-events', selectedCampaignId, eventPage],
    queryFn: () => {
      if (selectedCampaignId == null) {
        throw new Error('A referral campaign must be selected')
      }
      return getReferralCampaignEvents(selectedCampaignId, eventPage, 20)
    },
    enabled: selectedCampaignId != null,
    meta: { errorMode: 'local' },
  })
  const plans = useMemo(
    () => plansQuery.data?.data ?? [],
    [plansQuery.data?.data]
  )

  const retryMutation = useMutation({
    mutationFn: retryReferralCampaignReward,
    onSuccess: async (response) => {
      if (!response.success) throw new Error(response.message)
      toast.success(t('Reward retry completed'))
      await eventsQuery.refetch()
      await queryClient.invalidateQueries({
        queryKey: ['referral-campaigns', 'admin'],
      })
    },
    onError: (error) => toast.error(error.message),
  })

  const campaigns = campaignsQuery.data?.data ?? []
  const eventData = eventsQuery.data?.data
  const pageCount = Math.max(1, Math.ceil((eventData?.total ?? 0) / 20))
  const campaignsFailed =
    campaignsQuery.isError || campaignsQuery.data?.success === false
  const eventsFailed =
    eventsQuery.isError || eventsQuery.data?.success === false

  const campaignRewardLabel = (campaign: ReferralCampaign) => {
    if (campaign.reward?.type === 'subscription') {
      return campaign.reward.subscription_plan_title || t('Subscription')
    }
    if (campaign.reward?.quota) {
      return formatPlatformQuota(campaign.reward.quota)
    }
    return '—'
  }

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>
          {t('Referral campaigns')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          {canWrite && (
            <Button
              size='sm'
              onClick={() => {
                setEditing(null)
                setFormOpen(true)
              }}
            >
              <Plus data-icon='inline-start' />
              {t('Create campaign')}
            </Button>
          )}
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <div className='space-y-4'>
            <Card>
              <CardHeader>
                <CardTitle>{t('Campaign history')}</CardTitle>
              </CardHeader>
              <CardContent className='divide-y'>
                {campaignsQuery.isLoading && <Skeleton className='h-32' />}
                {campaignsFailed && (
                  <div className='space-y-3 py-8 text-center'>
                    <p className='text-muted-foreground text-sm'>
                      {t('Request failed')}
                    </p>
                    <Button
                      size='sm'
                      variant='outline'
                      onClick={() => void campaignsQuery.refetch()}
                    >
                      {t('Retry')}
                    </Button>
                  </div>
                )}
                {!campaignsQuery.isLoading &&
                  !campaignsFailed &&
                  campaigns.map((campaign) => (
                    <article key={campaign.id} className='space-y-2 py-3'>
                      <div className='flex items-start justify-between gap-3'>
                        <div>
                          <div className='flex flex-wrap items-center gap-2'>
                            <span className='font-medium'>
                              {campaign.title}
                            </span>
                            <Badge
                              variant={
                                campaign.enabled ? 'default' : 'secondary'
                              }
                            >
                              {t(campaign.enabled ? 'Enabled' : 'Disabled')}
                            </Badge>
                          </div>
                          <p className='text-muted-foreground mt-1 text-xs'>
                            {dayjs
                              .unix(campaign.start_time)
                              .format('YYYY-MM-DD HH:mm')}
                            {' – '}
                            {dayjs
                              .unix(campaign.end_time)
                              .format('YYYY-MM-DD HH:mm')}
                          </p>
                          <p className='text-muted-foreground mt-1 text-xs'>
                            {t('Reward')}: {campaignRewardLabel(campaign)} ·{' '}
                            {t('Rewarded')}: {campaign.rewarded_count}
                          </p>
                        </div>
                        {canWrite && (
                          <Button
                            size='icon-sm'
                            variant='ghost'
                            onClick={() => {
                              setEditing(campaign)
                              setFormOpen(true)
                            }}
                            aria-label={t('Edit campaign')}
                          >
                            <Pencil />
                          </Button>
                        )}
                      </div>
                      <div className='grid grid-cols-4 gap-2 text-center text-xs'>
                        {[
                          [t('Registered'), campaign.registration_count],
                          [t('Activated'), campaign.activated_count],
                          [t('Rewarded'), campaign.rewarded_count],
                          [t('Failed'), campaign.failed_reward_count],
                        ].map(([label, value]) => (
                          <div
                            key={String(label)}
                            className='bg-muted rounded-md p-2'
                          >
                            <div className='font-semibold'>{value}</div>
                            <div className='text-muted-foreground'>{label}</div>
                          </div>
                        ))}
                      </div>
                      <Button
                        size='sm'
                        variant='outline'
                        onClick={() => {
                          setSelectedCampaignId(campaign.id)
                          setEventPage(1)
                        }}
                      >
                        {t('View invite records')}
                      </Button>
                    </article>
                  ))}
                {!campaignsQuery.isLoading && campaigns.length === 0 && (
                  <p className='text-muted-foreground py-8 text-center text-sm'>
                    {t('No referral campaigns')}
                  </p>
                )}
              </CardContent>
            </Card>

            {selectedCampaignId && (
              <Card>
                <CardHeader>
                  <CardTitle>{t('Invite records')}</CardTitle>
                </CardHeader>
                <CardContent>
                  {eventsQuery.isLoading && <Skeleton className='h-32' />}
                  {eventsFailed && (
                    <div className='space-y-3 py-8 text-center'>
                      <p className='text-muted-foreground text-sm'>
                        {t('Request failed')}
                      </p>
                      <Button
                        size='sm'
                        variant='outline'
                        onClick={() => void eventsQuery.refetch()}
                      >
                        {t('Retry')}
                      </Button>
                    </div>
                  )}
                  {!eventsQuery.isLoading && !eventsFailed && (
                    <div className='divide-y'>
                      {(eventData?.items ?? []).map((event) => {
                        let statusLabel = t('Pending')
                        if (event.status === 'rewarded') {
                          statusLabel = t('Rewarded')
                        } else if (event.status === 'reward_failed') {
                          statusLabel = t('Reward delivery failed')
                        } else if (event.status === 'expired') {
                          statusLabel = t('Expired')
                        } else if (event.status === 'limit_reached') {
                          statusLabel = t('Limit Reached')
                        }
                        return (
                          <article
                            key={event.id}
                            className='flex flex-wrap items-center justify-between gap-3 py-3'
                          >
                            <div>
                              <div className='text-sm font-medium'>
                                {event.inviter_username} →{' '}
                                {event.invitee_username}
                              </div>
                              <div className='text-muted-foreground text-xs'>
                                {dayjs
                                  .unix(event.registered_at)
                                  .format('YYYY-MM-DD HH:mm')}
                              </div>
                              {event.reward_failure_reason && (
                                <div className='text-destructive mt-1 text-xs'>
                                  {event.reward_failure_reason}
                                </div>
                              )}
                            </div>
                            <div className='flex items-center gap-2'>
                              <Badge variant='outline'>{statusLabel}</Badge>
                              {canOperate &&
                                (event.status === 'reward_failed' ||
                                  event.status === 'reward_pending') && (
                                  <Button
                                    size='sm'
                                    variant='outline'
                                    disabled={retryMutation.isPending}
                                    onClick={() =>
                                      retryMutation.mutate(event.id)
                                    }
                                  >
                                    <RefreshCw />
                                    {t('Retry reward')}
                                  </Button>
                                )}
                            </div>
                          </article>
                        )
                      })}
                    </div>
                  )}
                  <div className='mt-3 flex items-center justify-end gap-2'>
                    <Button
                      size='sm'
                      variant='outline'
                      disabled={eventPage <= 1}
                      onClick={() => setEventPage((page) => page - 1)}
                    >
                      {t('Previous')}
                    </Button>
                    <span className='text-muted-foreground text-xs'>
                      {eventPage}/{pageCount}
                    </span>
                    <Button
                      size='sm'
                      variant='outline'
                      disabled={eventPage >= pageCount}
                      onClick={() => setEventPage((page) => page + 1)}
                    >
                      {t('Next')}
                    </Button>
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>
      <ReferralCampaignFormDrawer
        open={formOpen}
        campaign={editing}
        plans={plans}
        plansLoading={plansQuery.isLoading}
        plansFailed={plansQuery.isError || plansQuery.data?.success === false}
        onRetryPlans={() => void plansQuery.refetch()}
        onOpenChange={(open) => {
          setFormOpen(open)
          if (!open) setEditing(null)
        }}
        onSaved={() => {
          void queryClient.invalidateQueries({
            queryKey: ['referral-campaigns', 'admin'],
          })
        }}
      />
    </>
  )
}
