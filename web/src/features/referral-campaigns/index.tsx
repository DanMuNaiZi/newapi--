/* Copyright (C) 2023-2026 QuantumNous */
import { useQuery } from '@tanstack/react-query'
import dayjs from 'dayjs'
import { Gift, Users } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Spinner } from '@/components/ui/spinner'
import {
  getServerErrorMessage,
  requireServerSuccess,
} from '@/lib/server-error-message'

import { getReferralCampaignSelf } from './api'

export function ReferralCampaigns() {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: ['referral-campaign', 'self'],
    queryFn: async () => requireServerSuccess(await getReferralCampaignSelf()),
  })
  const view = query.data?.data
  const campaign = view?.campaign
  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Referral Program')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='grid gap-4'>
          {query.isLoading && <Spinner className='size-4' />}
          {query.isError && (
            <ErrorState
              description={getServerErrorMessage(query.error)}
              onRetry={() => void query.refetch()}
            />
          )}
          {!query.isLoading && !query.isError && !campaign && (
            <div className='text-muted-foreground rounded-lg border border-dashed p-8 text-center text-sm'>
              {t('No referral campaign is currently active')}
            </div>
          )}
          {campaign && (
            <section className='bg-card space-y-3 rounded-lg border p-5'>
              <div className='flex items-center gap-2'>
                <Gift className='size-5' />
                <h2 className='font-semibold'>{campaign.title}</h2>
                <Badge>{campaign.enabled ? t('Enabled') : t('Disabled')}</Badge>
              </div>
              <p className='text-muted-foreground text-sm'>
                {campaign.description}
              </p>
              <div className='text-muted-foreground flex flex-wrap gap-4 text-xs'>
                <span>
                  {t('Ends')}:{' '}
                  {dayjs.unix(campaign.end_time).format('YYYY-MM-DD HH:mm')}
                </span>
                <span>
                  <Users className='mr-1 inline size-3.5' />
                  {t('{{count}} referrals', {
                    count: view?.events.length ?? 0,
                  })}
                </span>
              </div>
            </section>
          )}
          {view?.events && view.events.length > 0 && (
            <section className='bg-card space-y-3 rounded-lg border p-5'>
              <h2 className='font-semibold'>{t('Referral progress')}</h2>
              <div className='grid gap-2'>
                {view.events.map((event) => (
                  <div
                    key={event.id}
                    className='flex flex-wrap items-center justify-between gap-2 border-b py-2 text-sm last:border-0'
                  >
                    <span>{event.invitee_username || t('Invitee')}</span>
                    <span className='text-muted-foreground'>
                      {t(event.status)}
                    </span>
                  </div>
                ))}
              </div>
            </section>
          )}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
