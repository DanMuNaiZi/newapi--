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
import { Plus, Trash2 } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useFieldArray, useForm, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { MultiSelect } from '@/components/multi-select'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Separator } from '@/components/ui/separator'
import { Spinner } from '@/components/ui/spinner'
import { Textarea } from '@/components/ui/textarea'
import { getAdminPlans } from '@/features/subscriptions/api'
import type { User } from '@/features/users/types'
import { cn } from '@/lib/utils'

import { createLotteryPlan, getLotteryAdminGroups } from '../api'
import {
  buildLotteryPlanPayload,
  createLotteryFormDefaults,
  DEFAULT_LOTTERY_PRIZE,
  LOTTERY_DATABASE_INT_MAX,
  lotteryAdminFormSchema,
  type LotteryAdminFormValues,
} from '../lib/admin-form'
import { LotteryUserMultiSelect } from './lottery-user-multi-select'

type LotteryPlanFormProps = {
  className?: string
  drawer?: boolean
  formId?: string
  onCancel?: () => void
  onCreated?: () => void
}

export function LotteryPlanForm(props: LotteryPlanFormProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [selectedUsers, setSelectedUsers] = useState<User[]>([])
  const form = useForm<LotteryAdminFormValues>({
    resolver: zodResolver(lotteryAdminFormSchema),
    defaultValues: createLotteryFormDefaults(),
  })
  const prizes = useFieldArray({ control: form.control, name: 'prizes' })
  const prizeValues = useWatch({ control: form.control, name: 'prizes' })
  const eligibilityMode = useWatch({
    control: form.control,
    name: 'eligibility_mode',
  })
  const selectedGroups = useWatch({
    control: form.control,
    name: 'selected_groups',
  })

  const groupsQuery = useQuery({
    queryKey: ['lottery', 'admin', 'groups'],
    queryFn: getLotteryAdminGroups,
  })
  const subscriptionPlansQuery = useQuery({
    queryKey: ['admin-subscription-plans'],
    queryFn: getAdminPlans,
  })
  const groupOptions = useMemo(
    () =>
      (groupsQuery.data?.success ? (groupsQuery.data.data ?? []) : []).map(
        (group) => ({ label: group, value: group })
      ),
    [groupsQuery.data]
  )

  const createMutation = useMutation({
    mutationFn: createLotteryPlan,
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(t('Failed to create lottery plan'))
        return
      }
      toast.success(t('Lottery plan created'))
      form.reset(createLotteryFormDefaults())
      setSelectedUsers([])
      await queryClient.invalidateQueries({
        queryKey: ['lottery', 'admin', 'plans'],
      })
      props.onCreated?.()
    },
  })

  const eligibilityField = form.register('eligibility_mode')
  const submit = (values: LotteryAdminFormValues): void => {
    createMutation.mutate(buildLotteryPlanPayload(values))
  }

  return (
    <form
      id={props.formId}
      className={cn(
        props.drawer
          ? 'flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto overscroll-contain px-4 py-4 sm:px-6 sm:py-5'
          : 'border-border grid gap-6 border-b pb-6',
        props.className
      )}
      onSubmit={form.handleSubmit(submit)}
    >
      <section className='grid gap-4'>
        <div>
          <h3 className='text-base font-semibold'>{t('Basic information')}</h3>
          <p className='text-muted-foreground mt-1 text-sm'>
            {t('Set the activity name, capacity, and schedule.')}
          </p>
        </div>
        <FieldGroup>
          <div className='grid grid-cols-1 gap-4 lg:grid-cols-2'>
            <Field data-invalid={Boolean(form.formState.errors.title)}>
              <FieldLabel htmlFor='lottery-title'>
                {t('Lottery title')}
              </FieldLabel>
              <Input
                id='lottery-title'
                placeholder={t('Example: July member lottery')}
                aria-invalid={Boolean(form.formState.errors.title)}
                {...form.register('title')}
              />
              <FieldError>
                {form.formState.errors.title?.message
                  ? t(form.formState.errors.title.message)
                  : null}
              </FieldError>
            </Field>

            <Field
              data-invalid={Boolean(form.formState.errors.max_participants)}
            >
              <FieldLabel htmlFor='lottery-max-participants'>
                {t('Maximum participants')}
              </FieldLabel>
              <Input
                id='lottery-max-participants'
                type='number'
                min={1}
                max={LOTTERY_DATABASE_INT_MAX}
                aria-invalid={Boolean(form.formState.errors.max_participants)}
                {...form.register('max_participants', { valueAsNumber: true })}
              />
              <FieldDescription>
                {t(
                  'The lottery starts immediately when this number is reached.'
                )}
              </FieldDescription>
              <FieldError>
                {form.formState.errors.max_participants?.message
                  ? t(form.formState.errors.max_participants.message)
                  : null}
              </FieldError>
            </Field>

            <Field
              data-invalid={Boolean(form.formState.errors.registration_start)}
            >
              <FieldLabel htmlFor='lottery-registration-start'>
                {t('Registration start time')}
              </FieldLabel>
              <Input
                id='lottery-registration-start'
                type='datetime-local'
                aria-invalid={Boolean(form.formState.errors.registration_start)}
                {...form.register('registration_start')}
              />
              <FieldDescription>
                {t('Eligible users can join after this time.')}
              </FieldDescription>
              <FieldError>
                {form.formState.errors.registration_start?.message
                  ? t(form.formState.errors.registration_start.message)
                  : null}
              </FieldError>
            </Field>

            <Field data-invalid={Boolean(form.formState.errors.draw_time)}>
              <FieldLabel htmlFor='lottery-draw-time'>
                {t('Draw time')}
              </FieldLabel>
              <Input
                id='lottery-draw-time'
                type='datetime-local'
                aria-invalid={Boolean(form.formState.errors.draw_time)}
                {...form.register('draw_time')}
              />
              <FieldDescription>
                {t(
                  'The draw runs at this time even if the activity is not full.'
                )}
              </FieldDescription>
              <FieldError>
                {form.formState.errors.draw_time?.message
                  ? t(form.formState.errors.draw_time.message)
                  : null}
              </FieldError>
            </Field>

            <Field className='lg:col-span-2'>
              <FieldLabel htmlFor='lottery-description'>
                {t('Lottery description')}
              </FieldLabel>
              <Textarea
                id='lottery-description'
                placeholder={t(
                  'Describe the activity rules or prize information'
                )}
                {...form.register('description')}
              />
            </Field>
          </div>
        </FieldGroup>
      </section>

      <Separator />

      <section className='grid gap-4'>
        <div>
          <h3 className='text-base font-semibold'>
            {t('Participation eligibility')}
          </h3>
          <p className='text-muted-foreground mt-1 text-sm'>
            {t('Choose who can see and join this lottery.')}
          </p>
        </div>
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor='lottery-eligibility-mode'>
              {t('Eligible participants')}
            </FieldLabel>
            <NativeSelect
              id='lottery-eligibility-mode'
              className='w-full'
              {...eligibilityField}
              onChange={(event) => {
                eligibilityField.onChange(event)
                form.setValue('selected_groups', [], { shouldDirty: true })
                form.setValue('selected_user_ids', [], { shouldDirty: true })
                form.clearErrors(['selected_groups', 'selected_user_ids'])
                setSelectedUsers([])
              }}
            >
              <NativeSelectOption value='all'>
                {t('All users')}
              </NativeSelectOption>
              <NativeSelectOption value='groups'>
                {t('Selected groups')}
              </NativeSelectOption>
              <NativeSelectOption value='users'>
                {t('Selected users')}
              </NativeSelectOption>
            </NativeSelect>
          </Field>

          {eligibilityMode === 'groups' && (
            <Field
              data-invalid={Boolean(form.formState.errors.selected_groups)}
            >
              <FieldLabel htmlFor='lottery-groups'>
                {t('Allowed groups')}
              </FieldLabel>
              <MultiSelect
                id='lottery-groups'
                options={groupOptions}
                selected={selectedGroups}
                onChange={(groups) =>
                  form.setValue('selected_groups', groups, {
                    shouldDirty: true,
                    shouldValidate: true,
                  })
                }
                placeholder={t('Search and select groups')}
                emptyText={t('No groups found')}
              />
              <FieldDescription>
                {t('Users in any selected group can participate.')}
              </FieldDescription>
              <FieldError>
                {form.formState.errors.selected_groups?.message
                  ? t(form.formState.errors.selected_groups.message)
                  : null}
              </FieldError>
            </Field>
          )}

          {eligibilityMode === 'users' && (
            <Field
              data-invalid={Boolean(form.formState.errors.selected_user_ids)}
            >
              <FieldLabel htmlFor='lottery-users'>
                {t('Allowed users')}
              </FieldLabel>
              <LotteryUserMultiSelect
                id='lottery-users'
                selectedUsers={selectedUsers}
                onChange={(users) => {
                  setSelectedUsers(users)
                  form.setValue(
                    'selected_user_ids',
                    users.map((user) => user.id),
                    { shouldDirty: true, shouldValidate: true }
                  )
                }}
                invalid={Boolean(form.formState.errors.selected_user_ids)}
              />
              <FieldDescription>
                {t('Search by username, display name, email, or user ID.')}
              </FieldDescription>
              <FieldError>
                {form.formState.errors.selected_user_ids?.message
                  ? t(form.formState.errors.selected_user_ids.message)
                  : null}
              </FieldError>
            </Field>
          )}
        </FieldGroup>
      </section>

      <Separator />

      <section className='grid gap-4'>
        <div>
          <h3 className='text-base font-semibold'>{t('Prize settings')}</h3>
          <p className='text-muted-foreground mt-1 text-sm'>
            {t('Configure each prize and how winners receive it.')}
          </p>
        </div>
        <div className='grid gap-4'>
          {prizes.fields.map((field, index) => {
            const rewardType = prizeValues[index]?.reward_type ?? 'quota'
            const fulfillmentMode =
              prizeValues[index]?.fulfillment_mode ?? 'auto'
            let fulfillmentDescription = t(
              'The reward is delivered immediately after the draw.'
            )
            if (fulfillmentMode === 'self_claim') {
              fulfillmentDescription = t(
                'The winner claims the reward from their dashboard.'
              )
            } else if (fulfillmentMode === 'redemption_code') {
              fulfillmentDescription = t(
                'The winner receives a code to redeem later.'
              )
            }
            const errors = form.formState.errors.prizes?.[index]
            return (
              <fieldset
                key={field.id}
                className='border-border grid gap-4 rounded-lg border p-4'
              >
                <div className='flex items-center justify-between gap-3'>
                  <legend className='text-sm font-semibold'>
                    {t('Prize {{number}}', { number: index + 1 })}
                  </legend>
                  <Button
                    type='button'
                    size='icon-sm'
                    variant='ghost'
                    onClick={() => prizes.remove(index)}
                    disabled={prizes.fields.length === 1}
                    aria-label={t('Remove prize')}
                    title={t('Remove prize')}
                  >
                    <Trash2 />
                  </Button>
                </div>
                <FieldGroup>
                  <div className='grid grid-cols-1 gap-4 lg:grid-cols-3'>
                    <Field data-invalid={Boolean(errors?.name)}>
                      <FieldLabel htmlFor={`lottery-prize-${index}-name`}>
                        {t('Prize name')}
                      </FieldLabel>
                      <Input
                        id={`lottery-prize-${index}-name`}
                        placeholder={t('Example: First prize')}
                        aria-invalid={Boolean(errors?.name)}
                        {...form.register(`prizes.${index}.name`)}
                      />
                      <FieldError>
                        {errors?.name?.message ? t(errors.name.message) : null}
                      </FieldError>
                    </Field>

                    <Field data-invalid={Boolean(errors?.quantity)}>
                      <FieldLabel htmlFor={`lottery-prize-${index}-quantity`}>
                        {t('Winner count')}
                      </FieldLabel>
                      <Input
                        id={`lottery-prize-${index}-quantity`}
                        type='number'
                        min={1}
                        max={LOTTERY_DATABASE_INT_MAX}
                        aria-invalid={Boolean(errors?.quantity)}
                        {...form.register(`prizes.${index}.quantity`, {
                          valueAsNumber: true,
                        })}
                      />
                      <FieldDescription>
                        {t('Number of winners for this prize.')}
                      </FieldDescription>
                      <FieldError>
                        {errors?.quantity?.message
                          ? t(errors.quantity.message)
                          : null}
                      </FieldError>
                    </Field>

                    <Field>
                      <FieldLabel
                        htmlFor={`lottery-prize-${index}-reward-type`}
                      >
                        {t('Reward type')}
                      </FieldLabel>
                      <NativeSelect
                        id={`lottery-prize-${index}-reward-type`}
                        className='w-full'
                        {...form.register(`prizes.${index}.reward_type`)}
                      >
                        <NativeSelectOption value='quota'>
                          {t('Quota')}
                        </NativeSelectOption>
                        <NativeSelectOption value='subscription'>
                          {t('Subscription')}
                        </NativeSelectOption>
                      </NativeSelect>
                    </Field>

                    {rewardType === 'quota' ? (
                      <Field data-invalid={Boolean(errors?.quota)}>
                        <FieldLabel htmlFor={`lottery-prize-${index}-quota`}>
                          {t('Reward quota')}
                        </FieldLabel>
                        <Input
                          id={`lottery-prize-${index}-quota`}
                          type='number'
                          min={1}
                          max={LOTTERY_DATABASE_INT_MAX}
                          aria-invalid={Boolean(errors?.quota)}
                          {...form.register(`prizes.${index}.quota`, {
                            valueAsNumber: true,
                          })}
                        />
                        <FieldDescription>
                          {t('Uses the same quota unit as the user balance.')}
                        </FieldDescription>
                        <FieldError>
                          {errors?.quota?.message
                            ? t(errors.quota.message)
                            : null}
                        </FieldError>
                      </Field>
                    ) : (
                      <Field
                        data-invalid={Boolean(errors?.subscription_plan_id)}
                      >
                        <FieldLabel
                          htmlFor={`lottery-prize-${index}-subscription`}
                        >
                          {t('Subscription plan')}
                        </FieldLabel>
                        <NativeSelect
                          id={`lottery-prize-${index}-subscription`}
                          className='w-full'
                          aria-invalid={Boolean(errors?.subscription_plan_id)}
                          {...form.register(
                            `prizes.${index}.subscription_plan_id`,
                            { valueAsNumber: true }
                          )}
                        >
                          <NativeSelectOption value='0'>
                            {t('Select a subscription plan')}
                          </NativeSelectOption>
                          {(subscriptionPlansQuery.data?.success
                            ? (subscriptionPlansQuery.data.data ?? [])
                            : []
                          ).map(({ plan }) => (
                            <NativeSelectOption key={plan.id} value={plan.id}>
                              #{plan.id} {plan.title}
                            </NativeSelectOption>
                          ))}
                        </NativeSelect>
                        <FieldError>
                          {errors?.subscription_plan_id?.message
                            ? t(errors.subscription_plan_id.message)
                            : null}
                        </FieldError>
                      </Field>
                    )}

                    <Field>
                      <FieldLabel
                        htmlFor={`lottery-prize-${index}-fulfillment`}
                      >
                        {t('Delivery method')}
                      </FieldLabel>
                      <NativeSelect
                        id={`lottery-prize-${index}-fulfillment`}
                        className='w-full'
                        {...form.register(`prizes.${index}.fulfillment_mode`)}
                      >
                        <NativeSelectOption value='auto'>
                          {t('Automatic delivery')}
                        </NativeSelectOption>
                        <NativeSelectOption value='self_claim'>
                          {t('Winner claims manually')}
                        </NativeSelectOption>
                        <NativeSelectOption value='redemption_code'>
                          {t('Generate a redemption code')}
                        </NativeSelectOption>
                      </NativeSelect>
                      <FieldDescription>
                        {fulfillmentDescription}
                      </FieldDescription>
                    </Field>

                    {fulfillmentMode === 'self_claim' && (
                      <Field data-invalid={Boolean(errors?.claim_expire_days)}>
                        <FieldLabel
                          htmlFor={`lottery-prize-${index}-claim-expiry`}
                        >
                          {t('Claim validity (days)')}
                        </FieldLabel>
                        <Input
                          id={`lottery-prize-${index}-claim-expiry`}
                          type='number'
                          min={0}
                          max={3650}
                          aria-invalid={Boolean(errors?.claim_expire_days)}
                          {...form.register(
                            `prizes.${index}.claim_expire_days`,
                            {
                              valueAsNumber: true,
                            }
                          )}
                        />
                        <FieldDescription>
                          {t('Enter 0 for no expiration.')}
                        </FieldDescription>
                        <FieldError>
                          {errors?.claim_expire_days?.message
                            ? t(errors.claim_expire_days.message)
                            : null}
                        </FieldError>
                      </Field>
                    )}
                  </div>
                </FieldGroup>
              </fieldset>
            )
          })}
        </div>
        <div
          className={cn(
            'flex flex-wrap items-center justify-between gap-3',
            props.drawer &&
              'bg-background sticky bottom-0 z-10 -mx-4 -mb-4 border-t px-4 py-3 sm:-mx-6 sm:-mb-5 sm:px-6 sm:py-4'
          )}
        >
          <Button
            type='button'
            variant='outline'
            onClick={() => prizes.append({ ...DEFAULT_LOTTERY_PRIZE })}
          >
            <Plus />
            {t('Add prize')}
          </Button>
          <div className='flex items-center gap-2'>
            {props.onCancel && (
              <Button type='button' variant='outline' onClick={props.onCancel}>
                {t('Close')}
              </Button>
            )}
            <Button type='submit' disabled={createMutation.isPending}>
              {createMutation.isPending && <Spinner />}
              {t('Create lottery plan')}
            </Button>
          </div>
        </div>
      </section>
    </form>
  )
}
