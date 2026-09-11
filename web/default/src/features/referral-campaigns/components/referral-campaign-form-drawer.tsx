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
import { useMutation } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  sideDrawerContentClassName,
  sideDrawerFooterClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Spinner } from '@/components/ui/spinner'
import { Textarea } from '@/components/ui/textarea'
import { useRewardQuotaConfig } from '@/hooks/use-reward-quota-config'
import {
  getRewardConversionError,
  getRewardEquivalent,
  formatPlatformQuota,
} from '@/lib/reward-amount'

import { createReferralCampaign, updateReferralCampaign } from '../api'
import {
  buildReferralCampaignPayload,
  emptyReferralCampaignForm,
  referralCampaignFormSchema,
  referralCampaignToForm,
  type ReferralCampaignFormValues,
} from '../lib/admin-form'
import type { ReferralCampaign } from '../types'

type ReferralCampaignFormDrawerProps = {
  open: boolean
  campaign: ReferralCampaign | null
  onOpenChange: (open: boolean) => void
  onSaved: () => void
}

export function ReferralCampaignFormDrawer(
  props: ReferralCampaignFormDrawerProps
) {
  const { t } = useTranslation()
  const form = useForm<ReferralCampaignFormValues>({
    resolver: zodResolver(referralCampaignFormSchema),
    defaultValues: emptyReferralCampaignForm(),
  })
  const replaceRewards = form.watch('replace_rewards')
  const inviterUSD = form.watch('inviter_reward_usd')
  const inviteeUSD = form.watch('invitee_reward_usd')
  const activationHours = form.watch('activation_hours')
  const preserveReward = Boolean(props.campaign && !replaceRewards)
  const needsQuotaConfig =
    !preserveReward && (Number(inviterUSD) > 0 || Number(inviteeUSD) > 0)
  const quotaConfig = useRewardQuotaConfig(props.open && needsQuotaConfig)

  useEffect(() => {
    if (!props.open) return
    form.reset(
      props.campaign
        ? referralCampaignToForm(props.campaign)
        : emptyReferralCampaignForm()
    )
  }, [form, props.campaign, props.open])

  const saveMutation = useMutation({
    mutationFn: (values: ReferralCampaignFormValues) => {
      const payload = buildReferralCampaignPayload(values, props.campaign)
      return props.campaign
        ? updateReferralCampaign(props.campaign.id, payload)
        : createReferralCampaign(payload)
    },
    onSuccess: (response) => {
      if (!response.success) {
        toast.error(response.message || t('Request failed'))
        return
      }
      toast.success(t(props.campaign ? 'Campaign updated' : 'Campaign created'))
      props.onSaved()
      props.onOpenChange(false)
    },
    onError: (error) =>
      toast.error(error instanceof Error ? error.message : t('Request failed')),
  })

  const timeZone = Intl.DateTimeFormat().resolvedOptions().timeZone

  const submit = (values: ReferralCampaignFormValues) => {
    if (!preserveReward) {
      for (const field of [
        'inviter_reward_usd',
        'invitee_reward_usd',
      ] as const) {
        if (Number(values[field]) === 0) continue
        const error = quotaConfig.isError
          ? quotaConfig.error.message
          : getRewardConversionError(values[field], 'usd', quotaConfig.data)
        if (error || quotaConfig.isFetching) {
          form.setError(
            field,
            {
              type: 'validate',
              message: error || 'Loading reward configuration',
            },
            { shouldFocus: true }
          )
          return
        }
      }
    }
    saveMutation.mutate(values)
  }

  return (
    <Sheet open={props.open} onOpenChange={props.onOpenChange}>
      <SheetContent className={sideDrawerContentClassName('sm:max-w-[680px]')}>
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>
            {t(props.campaign ? 'Edit campaign' : 'Create campaign')}
          </SheetTitle>
          <SheetDescription>
            {t(
              'Rewards are issued only after actual consumption and manual approval.'
            )}
          </SheetDescription>
        </SheetHeader>

        <div className={sideDrawerFormClassName()}>
          <div className='border-border/70 bg-muted/30 rounded-md border px-3 py-3 text-sm'>
            <p className='font-medium'>{t('How it works')}</p>
            <p className='text-muted-foreground mt-1 leading-6'>
              {t(
                'Share invite link → friend registers with GitHub → successful paid non-public-pool call → administrator reviews → both rewards are issued.'
              )}
            </p>
          </div>

          <form
            id='referral-campaign-form'
            className='grid gap-5'
            onSubmit={form.handleSubmit(submit, () =>
              toast.error(t('Please enter a valid campaign and time range'))
            )}
          >
            <FieldGroup>
              <Field data-invalid={Boolean(form.formState.errors.title)}>
                <FieldLabel htmlFor='referral-campaign-title'>
                  {t('Campaign title')}
                </FieldLabel>
                <Input
                  id='referral-campaign-title'
                  maxLength={128}
                  {...form.register('title')}
                />
                <FieldDescription>
                  {t('Required; up to 128 characters.')}
                </FieldDescription>
                <FieldError>
                  {form.formState.errors.title?.message
                    ? t(form.formState.errors.title.message)
                    : null}
                </FieldError>
              </Field>
              <Field data-invalid={Boolean(form.formState.errors.description)}>
                <FieldLabel htmlFor='referral-campaign-description'>
                  {t('Campaign description')}
                </FieldLabel>
                <Textarea
                  id='referral-campaign-description'
                  maxLength={4000}
                  {...form.register('description')}
                />
                <FieldDescription>
                  {t('Optional; up to 4000 characters.')}
                </FieldDescription>
                <FieldError>
                  {form.formState.errors.description?.message
                    ? t(form.formState.errors.description.message)
                    : null}
                </FieldError>
              </Field>

              <div className='grid gap-4 sm:grid-cols-2'>
                <Field data-invalid={Boolean(form.formState.errors.start)}>
                  <FieldLabel htmlFor='referral-campaign-start'>
                    {t('Start time')}
                  </FieldLabel>
                  <Input
                    id='referral-campaign-start'
                    type='datetime-local'
                    {...form.register('start')}
                  />
                  <FieldError>
                    {form.formState.errors.start?.message
                      ? t(form.formState.errors.start.message)
                      : null}
                  </FieldError>
                </Field>
                <Field data-invalid={Boolean(form.formState.errors.end)}>
                  <FieldLabel htmlFor='referral-campaign-end'>
                    {t('End time')}
                  </FieldLabel>
                  <Input
                    id='referral-campaign-end'
                    type='datetime-local'
                    {...form.register('end')}
                  />
                  <FieldError>
                    {form.formState.errors.end?.message
                      ? t(form.formState.errors.end.message)
                      : null}
                  </FieldError>
                </Field>
              </div>
              <FieldDescription>
                {t('Times use your browser time zone: {{timeZone}}.', {
                  timeZone,
                })}
              </FieldDescription>

              <div className='grid gap-4 sm:grid-cols-3'>
                <Field
                  data-invalid={Boolean(form.formState.errors.activation_hours)}
                >
                  <FieldLabel htmlFor='referral-campaign-activation-hours'>
                    {t('Activation window')}
                  </FieldLabel>
                  <Input
                    id='referral-campaign-activation-hours'
                    type='number'
                    min={1 / 60}
                    max={8760}
                    step='any'
                    {...form.register('activation_hours', {
                      valueAsNumber: true,
                    })}
                  />
                  <FieldDescription>
                    {t('Hours; from 1 minute to 365 days after registration.')}
                  </FieldDescription>
                  <FieldError>
                    {form.formState.errors.activation_hours?.message
                      ? t(form.formState.errors.activation_hours.message)
                      : null}
                  </FieldError>
                </Field>
                <Field
                  data-invalid={Boolean(
                    form.formState.errors.max_rewards_per_inviter
                  )}
                >
                  <FieldLabel htmlFor='referral-campaign-per-user-limit'>
                    {t('Per-inviter reward count limit')} ({t('times')})
                  </FieldLabel>
                  <Input
                    id='referral-campaign-per-user-limit'
                    type='number'
                    min={0}
                    max={2147483647}
                    step={1}
                    {...form.register('max_rewards_per_inviter', {
                      valueAsNumber: true,
                    })}
                  />
                  <FieldDescription>
                    {t('0 means unlimited rewards. Maximum: 2147483647 times.')}
                  </FieldDescription>
                  <FieldError>
                    {form.formState.errors.max_rewards_per_inviter?.message
                      ? t(form.formState.errors.max_rewards_per_inviter.message)
                      : null}
                  </FieldError>
                </Field>
                <Field
                  data-invalid={Boolean(
                    form.formState.errors.total_reward_limit
                  )}
                >
                  <FieldLabel htmlFor='referral-campaign-total-limit'>
                    {t('Total reward count limit')} ({t('times')})
                  </FieldLabel>
                  <Input
                    id='referral-campaign-total-limit'
                    type='number'
                    min={0}
                    max={2147483647}
                    step={1}
                    {...form.register('total_reward_limit', {
                      valueAsNumber: true,
                    })}
                  />
                  <FieldDescription>
                    {t('0 means unlimited rewards. Maximum: 2147483647 times.')}
                  </FieldDescription>
                  <FieldError>
                    {form.formState.errors.total_reward_limit?.message
                      ? t(form.formState.errors.total_reward_limit.message)
                      : null}
                  </FieldError>
                </Field>
              </div>

              {props.campaign && (
                <FieldGroup>
                  <FieldDescription>
                    {t(
                      'Existing reward snapshots stay unchanged unless you replace them. Already registered users keep their original rewards.'
                    )}
                  </FieldDescription>
                  <p className='text-sm'>
                    {t('Inviter reward')}:{' '}
                    {props.campaign.reward?.type === 'subscription'
                      ? props.campaign.reward.subscription_plan_title
                      : formatPlatformQuota(props.campaign.reward?.quota ?? 0)}
                    {' · '}
                    {t('New user reward')}:{' '}
                    {formatPlatformQuota(
                      props.campaign.invitee_reward?.quota ?? 0
                    )}
                  </p>
                  <Field orientation='horizontal'>
                    <input
                      id='referral-replace-rewards'
                      type='checkbox'
                      {...form.register('replace_rewards')}
                    />
                    <FieldLabel htmlFor='referral-replace-rewards'>
                      {t('Replace rewards for future registrations')}
                    </FieldLabel>
                  </Field>
                </FieldGroup>
              )}
              <div className='grid gap-4 sm:grid-cols-2'>
                {(
                  [
                    ['inviter_reward_usd', 'Inviter reward (USD)', inviterUSD],
                    ['invitee_reward_usd', 'New user reward (USD)', inviteeUSD],
                  ] as const
                ).map(([name, label, amount]) => {
                  const equivalent =
                    !preserveReward &&
                    Number(amount) > 0 &&
                    quotaConfig.isSuccess
                      ? getRewardEquivalent(amount, 'usd', quotaConfig.data)
                      : null
                  const error = form.formState.errors[name]?.message
                  return (
                    <Field
                      key={name}
                      data-invalid={Boolean(error)}
                      data-disabled={preserveReward}
                    >
                      <FieldLabel htmlFor={name}>{t(label)}</FieldLabel>
                      <Input
                        id={name}
                        inputMode='decimal'
                        maxLength={24}
                        disabled={preserveReward}
                        aria-invalid={Boolean(error)}
                        {...form.register(name)}
                      />
                      <FieldDescription>
                        {t('0 means no reward for this person.')}
                      </FieldDescription>
                      {equivalent && (
                        <FieldDescription>
                          {t('Actual payout: {{quota}} platform quota', {
                            quota: equivalent.quota.toLocaleString(),
                          })}
                        </FieldDescription>
                      )}
                      <FieldError>{error ? t(error) : null}</FieldError>
                    </Field>
                  )
                })}
              </div>
              {needsQuotaConfig && quotaConfig.isError && (
                <div role='alert'>
                  <p>{t('Failed to load reward configuration')}</p>
                  <Button
                    type='button'
                    variant='outline'
                    onClick={() => void quotaConfig.refetch()}
                  >
                    {t('Retry')}
                  </Button>
                </div>
              )}

              <Field orientation='horizontal'>
                <input
                  id='referral-campaign-enabled'
                  type='checkbox'
                  className='accent-primary size-4'
                  {...form.register('enabled')}
                />
                <FieldLabel htmlFor='referral-campaign-enabled'>
                  {t('Enabled')}
                </FieldLabel>
              </Field>

              <div className='border-border/70 bg-muted/20 rounded-md border px-3 py-3 text-sm'>
                <p className='font-medium'>{t('Rule summary')}</p>
                <p className='text-muted-foreground mt-1 leading-6'>
                  {preserveReward
                    ? t('Existing reward snapshots will be preserved.')
                    : t(
                        'After manual approval: inviter ${{inviter}}, new user ${{invitee}}.',
                        { inviter: inviterUSD, invitee: inviteeUSD }
                      )}
                </p>
                <p className='text-muted-foreground mt-1 leading-6'>
                  {t(
                    'A successful non-public-pool call with quota consumption is required within {{hours}} hours after registration. Rewards wait for manual approval.',
                    { hours: activationHours }
                  )}
                </p>
              </div>
            </FieldGroup>
          </form>
        </div>

        <SheetFooter className={sideDrawerFooterClassName()}>
          <Button
            type='button'
            variant='outline'
            onClick={() => props.onOpenChange(false)}
          >
            {t('Cancel')}
          </Button>
          <Button
            type='submit'
            form='referral-campaign-form'
            disabled={
              saveMutation.isPending ||
              (needsQuotaConfig &&
                (!quotaConfig.isSuccess || quotaConfig.isFetching))
            }
          >
            {saveMutation.isPending ? <Spinner /> : <Plus />}
            {t(props.campaign ? 'Save changes' : 'Create campaign')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
