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
import dayjs from 'dayjs'
import { Share2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { getReferralCampaignSelf } from '@/features/referral-campaigns/api'
import { formatQuota } from '@/lib/format'

import type { UserWalletData } from '../types'

interface AffiliateRewardsCardProps {
  user: UserWalletData | null
  affiliateLink: string
  onTransfer: () => void
  complianceConfirmed?: boolean
  loading?: boolean
  readOnly?: boolean
}

export function AffiliateRewardsCard({
  user,
  affiliateLink,
  onTransfer,
  complianceConfirmed = true,
  loading,
  readOnly = false,
}: AffiliateRewardsCardProps) {
  const { t } = useTranslation()
  const campaignQuery = useQuery({
    queryKey: ['referral-campaign', 'self'],
    queryFn: getReferralCampaignSelf,
    meta: { errorMode: 'local' },
  })
  if (loading) {
    return (
      <Card data-card-hover='false' className='bg-muted/20 py-0'>
        <CardContent className='grid gap-4 p-3 sm:p-4 lg:grid-cols-[minmax(220px,1fr)_minmax(220px,0.72fr)_minmax(320px,1.15fr)] lg:items-center'>
          <div>
            <Skeleton className='h-5 w-32' />
            <Skeleton className='mt-2 h-4 w-48' />
          </div>
          <Skeleton className='h-14 rounded-lg' />
          <Skeleton className='h-10 rounded-lg' />
        </CardContent>
      </Card>
    )
  }

  const hasRewards = (user?.aff_quota ?? 0) > 0
  const campaignView = campaignQuery.data?.data
  const campaign = campaignView?.campaign
  const campaignEvents = campaignView?.events ?? []
  const progressEvents = campaign
    ? campaignEvents.filter((event) => event.campaign_id === campaign.id)
    : campaignEvents
  const activatedCount = progressEvents.filter((event) =>
    ['reward_pending', 'rewarded', 'reward_failed', 'limit_reached'].includes(
      event.status
    )
  ).length
  const rewardedCount = progressEvents.filter(
    (event) => event.status === 'rewarded'
  ).length
  let campaignReward = ''
  if (campaign?.reward?.type === 'subscription') {
    campaignReward =
      campaign.reward.subscription_plan_title || t('Subscription')
  } else if (campaign?.reward?.quota) {
    campaignReward = formatQuota(campaign.reward.quota)
  }

  return (
    <Card data-card-hover='false' className='bg-muted/20 py-0'>
      <CardContent className='grid gap-3 p-3 sm:gap-4 sm:p-4 lg:grid-cols-[minmax(200px,1fr)_minmax(180px,0.65fr)_minmax(280px,1fr)] lg:items-center'>
        <div className='flex min-w-0 items-center gap-2.5'>
          <div className='bg-background flex size-8 shrink-0 items-center justify-center rounded-lg border'>
            <Share2 className='text-muted-foreground size-4' />
          </div>
          <div className='min-w-0'>
            <h3 className='truncate text-sm font-semibold'>
              {t('Referral Program')}
            </h3>
            <p className='text-muted-foreground line-clamp-1 text-xs'>
              {t(
                'Earn rewards when users join through your referral link. Transfer accumulated rewards to your balance anytime.'
              )}
            </p>
          </div>
        </div>

        <div className='grid grid-cols-3 gap-1.5 text-center'>
          {[
            [t('Pending'), formatQuota(user?.aff_quota ?? 0)],
            [t('Total Earned'), formatQuota(user?.aff_history_quota ?? 0)],
            [t('Invites'), String(user?.aff_count ?? 0)],
          ].map(([label, value]) => (
            <div key={label}>
              <div className='text-muted-foreground truncate text-[10px] font-medium tracking-wider uppercase'>
                {label}
              </div>
              <div className='mt-0.5 truncate text-sm font-semibold tabular-nums'>
                {value}
              </div>
            </div>
          ))}
        </div>

        <div className='flex items-center gap-2'>
          <Input
            value={affiliateLink}
            readOnly
            className='border-muted bg-background/70 h-9 min-w-0 flex-1 font-mono text-xs'
          />
          <CopyButton
            value={affiliateLink}
            variant='outline'
            className='bg-background size-9 shrink-0'
            iconClassName='size-4'
            tooltip={t('Copy referral link')}
            aria-label={t('Copy referral link')}
          />
          {hasRewards && !readOnly && (
            <Button
              onClick={onTransfer}
              disabled={!complianceConfirmed}
              className='h-9 shrink-0 px-3'
              size='sm'
            >
              {t('Transfer to Balance')}
            </Button>
          )}
        </div>
        {!complianceConfirmed ? (
          <p className='text-muted-foreground text-xs lg:col-span-3'>
            {t(
              'Referral reward transfer is disabled until the administrator confirms compliance terms.'
            )}
          </p>
        ) : null}
        {campaign || campaignEvents.length > 0 ? (
          <div className='bg-background/70 rounded-lg border p-3 text-xs lg:col-span-3'>
            <div className='flex flex-wrap items-center justify-between gap-2'>
              <div>
                <span className='font-semibold'>
                  {campaign?.title || t('Referral Program')}
                </span>
                {campaign && (
                  <span className='text-muted-foreground ml-2'>
                    {t('Ends {{time}}', {
                      time: dayjs
                        .unix(campaign.end_time)
                        .format('YYYY-MM-DD HH:mm'),
                    })}
                  </span>
                )}
              </div>
              <span className='text-muted-foreground'>
                {t('{{count}} campaign invites', {
                  count: progressEvents.length,
                })}
              </span>
            </div>
            {campaign && (
              <p className='text-muted-foreground mt-1'>
                {campaign.description ||
                  t(
                    'Invited users qualify after their first successful non-public-pool API call.'
                  )}
              </p>
            )}
            <div className='text-muted-foreground mt-2 flex flex-wrap gap-x-4 gap-y-1'>
              {campaign && (
                <span>
                  {t('Reward')}: {campaignReward || '—'}
                </span>
              )}
              <span>
                {t('Registered')}: {progressEvents.length}
              </span>
              <span>
                {t('Activated')}: {activatedCount}
              </span>
              <span>
                {t('Rewarded')}: {rewardedCount}
              </span>
            </div>
            {campaignEvents.length > 0 ? (
              <div className='mt-2 flex flex-wrap gap-2'>
                {campaignEvents.slice(0, 5).map((event) => {
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
                    <span
                      key={event.id}
                      className='bg-muted rounded-md px-2 py-1'
                    >
                      {event.invitee_username || t('User')}: {statusLabel}
                    </span>
                  )
                })}
              </div>
            ) : null}
          </div>
        ) : null}
      </CardContent>
    </Card>
  )
}
