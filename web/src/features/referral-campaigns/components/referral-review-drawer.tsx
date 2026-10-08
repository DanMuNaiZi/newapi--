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
import { useMutation, useQuery } from '@tanstack/react-query'
import dayjs from 'dayjs'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { StaticDataTable } from '@/components/data-table'
import {
  sideDrawerContentClassName,
  sideDrawerFooterClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Field, FieldLabel } from '@/components/ui/field'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { Textarea } from '@/components/ui/textarea'
import { useRewardQuotaConfig } from '@/hooks/use-reward-quota-config'
import { toIntlLocale } from '@/i18n/languages'
import {
  ADMIN_PERMISSION_ACTIONS,
  ADMIN_PERMISSION_RESOURCES,
  hasPermission,
} from '@/lib/admin-permissions'
import { formatNumber } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import {
  getRewardEquivalent,
  isValidRewardQuotaPerUnit,
} from '@/lib/reward-amount'
import {
  createServerError,
  requireServerSuccess,
} from '@/lib/server-error-message'
import { useAuthStore } from '@/stores/auth-store'

import { getReferralEventReview, reviewReferralEvent } from '../api'

type Props = {
  eventId: number | null
  onClose: () => void
  onChanged: () => void | Promise<void>
}

export function ReferralReviewDrawer({ eventId, onClose, onChanged }: Props) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const user = useAuthStore((state) => state.auth.user)
  const canRead = hasPermission(
    user,
    ADMIN_PERMISSION_RESOURCES.REFERRAL_CAMPAIGN,
    ADMIN_PERMISSION_ACTIONS.READ
  )
  const canOperate = hasPermission(
    user,
    ADMIN_PERMISSION_RESOURCES.REFERRAL_CAMPAIGN,
    ADMIN_PERMISSION_ACTIONS.OPERATE
  )
  const [remark, setRemark] = useState('')
  const [confirmDecision, setConfirmDecision] = useState<
    'approve' | 'reject' | null
  >(null)
  useEffect(() => {
    setRemark('')
    setConfirmDecision(null)
  }, [eventId])
  const quotaConfig = useRewardQuotaConfig(canRead && eventId != null)
  const query = useQuery({
    queryKey: ['referral-event-review', user?.id, eventId],
    queryFn: async () => {
      if (eventId == null) throw new Error(t('Request failed'))
      const response = requireServerSuccess(
        await getReferralEventReview(eventId)
      )
      if (!response.data) throw createServerError(response, t('Request failed'))
      return response.data
    },
    enabled: canRead && eventId != null,
  })
  const mutation = useMutation({
    mutationFn: async (decision: 'approve' | 'reject') => {
      if (eventId == null) throw new Error(t('Request failed'))
      if (!canDecide || (decision === 'approve' && !canApprove)) {
        throw new Error(t('Request failed'))
      }
      return requireServerSuccess(
        await reviewReferralEvent(eventId, decision, remark.trim())
      )
    },
    onSuccess: async () => {
      toast.success(t('Review saved'))
      setConfirmDecision(null)
      await onChanged()
      await query.refetch()
    },
    onError: (error) => handleServerError(error, t('Request failed')),
  })
  const view = query.data
  const canDecide =
    canOperate &&
    canRead &&
    Boolean(view) &&
    !query.isError &&
    !view?.event.review_decision &&
    view?.event.status !== 'rewarded' &&
    view?.event.status !== 'expired' &&
    view?.event.status !== 'rejected'
  const canApprove = canDecide && Boolean(view?.can_approve)
  const date = (value: number) =>
    value > 0 ? dayjs.unix(value).format('YYYY-MM-DD HH:mm') : t('Unknown')
  const quotaText = (value: number) => formatNumber(value, locale)
  const usdText = (value: number) => {
    if (!Number.isFinite(value) || value < 0) return '—'
    if (!isValidRewardQuotaPerUnit(quotaConfig.quotaPerUnit)) return '—'
    const equivalent = getRewardEquivalent(
      String(value),
      'quota',
      quotaConfig.quotaPerUnit,
      quotaConfig.usdExchangeRate
    )
    return equivalent ? `$${equivalent.usd}` : '—'
  }

  return (
    <>
      <Sheet
        open={canRead && eventId != null}
        onOpenChange={(open) => {
          if (!open && !mutation.isPending) onClose()
        }}
      >
        <SheetContent
          className={sideDrawerContentClassName('sm:max-w-[760px]')}
        >
          <SheetHeader className={sideDrawerHeaderClassName()}>
            <SheetTitle>{t('Referral review')}</SheetTitle>
            <SheetDescription>
              {t('Registration → quota consumption → manual review → rewards')}
            </SheetDescription>
          </SheetHeader>
          <div className={sideDrawerFormClassName()}>
            {query.isLoading && <Skeleton className='h-40' />}
            {query.isError && (
              <Alert variant='destructive'>
                <AlertTitle>{t('Request failed')}</AlertTitle>
                <AlertDescription>
                  <p>
                    {query.error instanceof Error
                      ? query.error.message
                      : t('Request failed')}
                  </p>
                  <Button
                    variant='outline'
                    onClick={() => void query.refetch()}
                  >
                    {t('Retry')}
                  </Button>
                </AlertDescription>
              </Alert>
            )}
            {view && (
              <div className='flex min-w-0 flex-col gap-6'>
                <section className='flex flex-col gap-2'>
                  <h3 className='font-medium'>
                    {view.inviter.username} → {view.invitee.username}
                  </h3>
                  <dl className='grid grid-cols-1 gap-3 text-sm sm:grid-cols-2'>
                    <div>
                      <dt className='text-muted-foreground'>
                        {t('Registration time')}
                      </dt>
                      <dd>{date(view.event.registered_at)}</dd>
                    </div>
                    <div>
                      <dt className='text-muted-foreground'>
                        {t('Activation deadline')}
                      </dt>
                      <dd>{date(view.event.activation_deadline)}</dd>
                    </div>
                    <div>
                      <dt className='text-muted-foreground'>
                        {t('GitHub account created')}
                      </dt>
                      <dd>{date(view.invitee.github_created_at)}</dd>
                    </div>
                    <div>
                      <dt className='text-muted-foreground'>
                        {t('GitHub age exemption')}
                      </dt>
                      <dd>
                        {t(view.invitee.github_age_exempt ? 'Yes' : 'No')} ·{' '}
                        {view.invitee.github_id || t('Unknown')}
                      </dd>
                    </div>
                  </dl>
                </section>
                <Alert>
                  <AlertTitle>
                    {t(
                      view.has_consumption
                        ? 'Eligible quota consumption confirmed'
                        : 'No eligible quota consumption'
                    )}
                  </AlertTitle>
                  <AlertDescription>
                    {t('Qualifying deduction')}:{' '}
                    {quotaText(view.event.qualified_quota)} (
                    {usdText(view.event.qualified_quota)})
                    <p>
                      {t(
                        'Platform quota consumption is not proof of a cash payment. Free pool calls and interrupted calls do not qualify.'
                      )}
                    </p>
                  </AlertDescription>
                </Alert>
                <section className='flex flex-col gap-2 text-sm'>
                  <h3 className='font-medium'>{t('Rewards after approval')}</h3>
                  <p>
                    {t('Inviter reward')}:{' '}
                    {view.event.reward?.type === 'subscription'
                      ? view.event.reward.subscription_plan_title
                      : `${quotaText(view.event.reward?.quota ?? 0)} (${usdText(view.event.reward?.quota ?? 0)})`}
                  </p>
                  <p>
                    {t('New user reward')}:{' '}
                    {quotaText(view.event.invitee_reward?.quota ?? 0)} (
                    {usdText(view.event.invitee_reward?.quota ?? 0)})
                  </p>
                  {(view.reward_grants ?? []).map((grant) => (
                    <div
                      key={grant.id}
                      className='flex flex-wrap items-center gap-2'
                    >
                      <span>#{grant.recipient_user_id}</span>
                      <Badge variant='outline'>{t(grant.status)}</Badge>
                      {grant.failure_reason && (
                        <span className='text-destructive break-all'>
                          {grant.failure_reason}
                        </span>
                      )}
                    </div>
                  ))}
                </section>
                <section className='flex min-w-0 flex-col gap-2 text-sm'>
                  <h3 className='font-medium'>
                    {t('Usage since registration')}
                  </h3>
                  <p>
                    {t('Requests')}: {formatNumber(view.usage.requests, locale)}{' '}
                    · {t('Logged quota')}:{' '}
                    {quotaText(view.usage.consumed_quota)} (
                    {usdText(view.usage.consumed_quota)})
                  </p>
                  {!view.usage.logs_available && (
                    <p role='alert'>
                      {t(
                        'Usage logs are unavailable. Saved qualifying evidence is still valid.'
                      )}
                    </p>
                  )}
                  <StaticDataTable
                    data={view.usage.recent_calls}
                    getRowKey={(call) => call.id}
                    emptyContent={t('No usage records')}
                    columns={[
                      {
                        id: 'time',
                        header: t('Time'),
                        cell: (call) => date(call.created_at),
                      },
                      {
                        id: 'model',
                        header: t('Model'),
                        cell: (call) => call.model_name || '—',
                      },
                      {
                        id: 'tokens',
                        header: t('Tokens'),
                        cell: (call) =>
                          `${formatNumber(call.prompt_tokens, locale)} / ${formatNumber(call.completion_tokens, locale)}`,
                      },
                      {
                        id: 'quota',
                        header: t('Logged quota'),
                        cell: (call) =>
                          `${quotaText(call.quota)} (${usdText(call.quota)})`,
                      },
                      {
                        id: 'status',
                        header: t('Status'),
                        cell: (call) => {
                          if (call.public_pool) return t('Public Pool')
                          return t(
                            call.qualifying
                              ? 'Qualifying call'
                              : 'No eligible quota consumption'
                          )
                        },
                      },
                    ]}
                  />
                </section>
                <section className='flex flex-col gap-2 text-sm'>
                  <h3 className='font-medium'>
                    {t('Recent invitations by this inviter')}
                  </h3>
                  {(view.related_events ?? []).map((event) => (
                    <div
                      key={event.id}
                      className='flex flex-wrap justify-between gap-2'
                    >
                      <span>
                        {event.invitee_username || `#${event.invitee_user_id}`}
                      </span>
                      <span>{date(event.registered_at)}</span>
                    </div>
                  ))}
                </section>
                {view.event.review_decision && (
                  <p className='text-sm'>
                    {t(
                      view.event.review_decision === 'approved'
                        ? 'Approved'
                        : 'Rejected'
                    )}{' '}
                    · #{view.event.reviewed_by} ·{' '}
                    {date(view.event.reviewed_at ?? 0)}
                    <br />
                    {view.event.review_remark}
                  </p>
                )}
                {canDecide && (
                  <Field>
                    <FieldLabel htmlFor='referral-review-remark'>
                      {t('Review remark')}
                    </FieldLabel>
                    <Textarea
                      id='referral-review-remark'
                      value={remark}
                      maxLength={2000}
                      onChange={(event) => setRemark(event.target.value)}
                    />
                  </Field>
                )}
              </div>
            )}
          </div>
          <SheetFooter className={sideDrawerFooterClassName()}>
            <Button
              variant='outline'
              disabled={mutation.isPending}
              onClick={onClose}
            >
              {t('Close')}
            </Button>
            {view && canDecide && (
              <>
                <Button
                  variant='destructive'
                  disabled={mutation.isPending}
                  onClick={() => setConfirmDecision('reject')}
                >
                  {t('Reject')}
                </Button>
                <Button
                  disabled={mutation.isPending || !canApprove}
                  onClick={() => setConfirmDecision('approve')}
                >
                  {mutation.isPending && <Spinner />}
                  {t('Approve and issue rewards')}
                </Button>
              </>
            )}
          </SheetFooter>
        </SheetContent>
      </Sheet>
      <ConfirmDialog
        open={confirmDecision != null}
        onOpenChange={(open) => {
          if (!open && !mutation.isPending) setConfirmDecision(null)
        }}
        title={t(
          confirmDecision === 'approve' ? 'Approve and issue rewards' : 'Reject'
        )}
        desc={
          <>
            <p>
              {view?.inviter.username} → {view?.invitee.username}
            </p>
            <p>
              {t(
                'Rewards are issued only after actual consumption and manual approval.'
              )}
            </p>
            {remark && <p>{remark}</p>}
          </>
        }
        destructive={confirmDecision === 'reject'}
        isLoading={mutation.isPending}
        disabled={!canDecide || (confirmDecision === 'approve' && !canApprove)}
        handleConfirm={() => {
          if (confirmDecision) mutation.mutate(confirmDecision)
        }}
      />
    </>
  )
}
