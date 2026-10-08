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
import { FileSearch, Pencil, Plus, RotateCcw } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
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
  getAdminReferralCampaigns,
  getReferralCampaignEvents,
  retryReferralCampaignReward,
  reviewReferralEvent,
} from './api'
import { ReferralCampaignFormDrawer } from './components/referral-campaign-form-drawer'
import { ReferralReviewDrawer } from './components/referral-review-drawer'
import type { ReferralCampaign } from './types'

export function ReferralCampaignAdmin() {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const user = useAuthStore((state) => state.auth.user)
  const canRead = hasPermission(
    user,
    ADMIN_PERMISSION_RESOURCES.REFERRAL_CAMPAIGN,
    ADMIN_PERMISSION_ACTIONS.READ
  )
  const canWrite = hasPermission(
    user,
    ADMIN_PERMISSION_RESOURCES.REFERRAL_CAMPAIGN,
    ADMIN_PERMISSION_ACTIONS.WRITE
  )
  const canOperate = hasPermission(
    user,
    ADMIN_PERMISSION_RESOURCES.REFERRAL_CAMPAIGN,
    ADMIN_PERMISSION_ACTIONS.OPERATE
  )
  const queryClient = useQueryClient()
  const [selected, setSelected] = useState<ReferralCampaign | null>(null)
  const [editing, setEditing] = useState<ReferralCampaign | null>(null)
  const [formOpen, setFormOpen] = useState(false)
  const [reviewEventId, setReviewEventId] = useState<number | null>(null)
  const [notes, setNotes] = useState<Record<number, string>>({})
  const campaignsQuery = useQuery({
    queryKey: ['referral-campaign', 'admin', user?.id],
    queryFn: async () =>
      requireServerSuccess(await getAdminReferralCampaigns()),
    enabled: canRead,
  })
  const eventsQuery = useQuery({
    queryKey: ['referral-campaign', 'events', user?.id, selected?.id],
    queryFn: async () =>
      requireServerSuccess(await getReferralCampaignEvents(selected?.id ?? 0)),
    enabled: canRead && Boolean(selected),
  })
  const reviewMutation = useMutation({
    mutationFn: async ({
      id,
      decision,
    }: {
      id: number
      decision: 'approve' | 'reject'
    }) =>
      requireServerSuccess(
        await reviewReferralEvent(id, decision, notes[id]?.trim() ?? '')
      ),
    onSuccess: async () => {
      toast.success(t('Review saved'))
      await queryClient.invalidateQueries({ queryKey: ['referral-campaign'] })
    },
    onError: (error) => handleServerError(error, t('Request failed')),
  })
  const retryMutation = useMutation({
    mutationFn: async (id: number) =>
      requireServerSuccess(await retryReferralCampaignReward(id)),
    onSuccess: async () => {
      toast.success(t('Reward retry submitted'))
      await queryClient.invalidateQueries({ queryKey: ['referral-campaign'] })
    },
    onError: (error) => handleServerError(error, t('Request failed')),
  })
  const campaigns = campaignsQuery.data?.data ?? []
  const events = eventsQuery.data?.data?.items ?? []

  if (!canRead) {
    return (
      <ErrorState title={t('You do not have permission to view this page')} />
    )
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        <div className='flex flex-wrap items-center justify-between gap-3'>
          <span>{t('Referral Campaign Management')}</span>
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
        </div>
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='grid gap-4 lg:grid-cols-[minmax(16rem,24rem)_minmax(0,1fr)]'>
          <section className='bg-card space-y-2 rounded-lg border p-4'>
            <h2 className='font-semibold'>{t('Campaigns')}</h2>
            {campaignsQuery.isLoading && <Spinner className='size-4' />}
            {campaignsQuery.isError && (
              <ErrorState
                description={getServerErrorMessage(campaignsQuery.error)}
                onRetry={() => void campaignsQuery.refetch()}
              />
            )}
            {!campaignsQuery.isLoading &&
              !campaignsQuery.isError &&
              campaigns.length === 0 && (
                <p className='text-muted-foreground text-sm'>
                  {t('No referral campaigns')}
                </p>
              )}
            {campaigns.map((campaign) => (
              <div
                key={campaign.id}
                className={`rounded-md border p-3 ${selected?.id === campaign.id ? 'border-primary' : ''}`}
              >
                <Button
                  type='button'
                  variant='ghost'
                  className='h-auto w-full flex-col items-stretch gap-0 px-0 text-left whitespace-normal'
                  aria-pressed={selected?.id === campaign.id}
                  onClick={() => setSelected(campaign)}
                >
                  <div className='flex items-center justify-between gap-2'>
                    <span className='font-medium'>{campaign.title}</span>
                    <Badge variant={campaign.enabled ? 'default' : 'secondary'}>
                      {campaign.enabled ? t('Enabled') : t('Disabled')}
                    </Badge>
                  </div>
                  <p className='text-muted-foreground mt-1 text-xs'>
                    {t('{{count}} registrations', {
                      count: campaign.registration_count,
                    })}
                  </p>
                </Button>
                {canWrite && (
                  <Button
                    size='sm'
                    variant='ghost'
                    className='mt-2'
                    onClick={() => {
                      setEditing(campaign)
                      setFormOpen(true)
                    }}
                  >
                    <Pencil data-icon='inline-start' />
                    {t('Edit')}
                  </Button>
                )}
              </div>
            ))}
          </section>
          <section className='bg-card space-y-3 rounded-lg border p-4'>
            <h2 className='font-semibold'>{t('Referral events')}</h2>
            {!selected && (
              <p className='text-muted-foreground text-sm'>
                {t('Select a campaign to view events')}
              </p>
            )}
            {selected && eventsQuery.isLoading && (
              <Spinner className='size-4' />
            )}
            {selected && eventsQuery.isError && (
              <ErrorState
                description={getServerErrorMessage(eventsQuery.error)}
                onRetry={() => void eventsQuery.refetch()}
              />
            )}
            {selected &&
              !eventsQuery.isLoading &&
              !eventsQuery.isError &&
              events.length === 0 && (
                <p className='text-muted-foreground text-sm'>
                  {t('No referral events')}
                </p>
              )}
            {events.map((event) => {
              let statusVariant: 'default' | 'destructive' | 'secondary' =
                'secondary'
              if (event.status === 'rejected') {
                statusVariant = 'destructive'
              } else if (event.status === 'rewarded') {
                statusVariant = 'default'
              }
              return (
                <div
                  key={event.id}
                  className='grid gap-2 border-b py-3 last:border-0 md:grid-cols-[minmax(0,1fr)_auto]'
                >
                  <div>
                    <div className='flex flex-wrap items-center gap-2'>
                      <span className='font-medium'>
                        {event.inviter_username || event.inviter_user_id}
                      </span>
                      <span className='text-muted-foreground'>→</span>
                      <span>
                        {event.invitee_username || event.invitee_user_id}
                      </span>
                      <Badge variant={statusVariant}>{t(event.status)}</Badge>
                    </div>
                    <p className='text-muted-foreground text-xs'>
                      {t('Qualified quota')}:{' '}
                      {formatNumber(event.qualified_quota, locale)}
                    </p>
                    {canOperate && event.status === 'pending_review' && (
                      <div className='mt-2'>
                        <Label htmlFor={`review-${event.id}`}>
                          {t('Review note')}
                        </Label>
                        <Textarea
                          id={`review-${event.id}`}
                          maxLength={2000}
                          disabled={reviewMutation.isPending}
                          value={notes[event.id] ?? ''}
                          onChange={(eventInput) =>
                            setNotes({
                              ...notes,
                              [event.id]: eventInput.target.value,
                            })
                          }
                        />
                      </div>
                    )}
                  </div>
                  <div className='flex flex-wrap items-start gap-2'>
                    <Button
                      size='sm'
                      variant='ghost'
                      onClick={() => setReviewEventId(event.id)}
                    >
                      <FileSearch data-icon='inline-start' />
                      {t('Review details')}
                    </Button>
                    {canOperate && event.status === 'pending_review' && (
                      <>
                        <Button
                          size='sm'
                          disabled={
                            reviewMutation.isPending || retryMutation.isPending
                          }
                          onClick={() =>
                            reviewMutation.mutate({
                              id: event.id,
                              decision: 'approve',
                            })
                          }
                        >
                          {t('Approve')}
                        </Button>
                        <Button
                          size='sm'
                          variant='outline'
                          disabled={
                            reviewMutation.isPending || retryMutation.isPending
                          }
                          onClick={() =>
                            reviewMutation.mutate({
                              id: event.id,
                              decision: 'reject',
                            })
                          }
                        >
                          {t('Reject')}
                        </Button>
                      </>
                    )}
                    {canOperate && event.status === 'reward_failed' && (
                      <Button
                        size='sm'
                        variant='outline'
                        disabled={
                          retryMutation.isPending || reviewMutation.isPending
                        }
                        onClick={() => retryMutation.mutate(event.id)}
                      >
                        <RotateCcw data-icon='inline-start' />
                        {t('Retry reward')}
                      </Button>
                    )}
                  </div>
                </div>
              )
            })}
          </section>
        </div>
        <ReferralCampaignFormDrawer
          open={canWrite && formOpen}
          campaign={editing}
          onOpenChange={setFormOpen}
          onSaved={async () => {
            await queryClient.invalidateQueries({
              queryKey: ['referral-campaign', 'admin'],
            })
          }}
        />
        <ReferralReviewDrawer
          eventId={reviewEventId}
          onClose={() => setReviewEventId(null)}
          onChanged={async () => {
            await queryClient.invalidateQueries({
              queryKey: ['referral-campaign'],
            })
          }}
        />
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
