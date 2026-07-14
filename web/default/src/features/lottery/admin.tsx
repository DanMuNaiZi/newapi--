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
import { useState } from 'react'
import { useFieldArray, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

import {
  createLotteryPlan,
  drawLotteryPlan,
  getAdminLotteryPlans,
  getLotteryParticipants,
  updateLotteryParticipant,
} from './api'
import type { LotteryPlanCreatePayload } from './types'

const prizeSchema = z.object({
  name: z.string().min(1),
  quantity: z.number().int().min(1),
  reward_type: z.enum(['quota', 'subscription']),
  quota: z.number().int().min(0),
  subscription_plan_id: z.number().int().min(0),
  fulfillment_mode: z.enum(['auto', 'self_claim', 'redemption_code']),
  claim_expire_seconds: z.number().int().min(0),
})

const formSchema = z.object({
  title: z.string().min(1),
  description: z.string(),
  eligibility_mode: z.enum(['all', 'groups', 'users']),
  allow_list: z.string(),
  max_participants: z.number().int().min(1),
  registration_start: z.string().min(1),
  draw_time: z.string().min(1),
  prizes: z.array(prizeSchema).min(1),
})

type LotteryAdminForm = z.infer<typeof formSchema>

const DEFAULT_PRIZE: LotteryAdminForm['prizes'][number] = {
  name: '',
  quantity: 1,
  reward_type: 'quota',
  quota: 100,
  subscription_plan_id: 0,
  fulfillment_mode: 'auto',
  claim_expire_seconds: 604800,
}

function unixFromLocal(value: string): number {
  return Math.floor(new Date(value).getTime() / 1000)
}

function splitAllowList(value: string): string[] {
  return value
    .split(/[\s,]+/)
    .map((item) => item.trim())
    .filter(Boolean)
}

export function LotteryAdmin() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [selectedPlanId, setSelectedPlanId] = useState<number | null>(null)
  const form = useForm<LotteryAdminForm>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      title: '',
      description: '',
      eligibility_mode: 'all',
      allow_list: '',
      max_participants: 10,
      registration_start: '',
      draw_time: '',
      prizes: [DEFAULT_PRIZE],
    },
  })
  const prizes = useFieldArray({ control: form.control, name: 'prizes' })
  const plansQuery = useQuery({
    queryKey: ['lottery', 'admin', 'plans'],
    queryFn: getAdminLotteryPlans,
  })
  const participantsQuery = useQuery({
    queryKey: ['lottery', 'admin', 'participants', selectedPlanId],
    queryFn: () => getLotteryParticipants(selectedPlanId ?? 0),
    enabled: Boolean(selectedPlanId),
  })
  const createMutation = useMutation({
    mutationFn: createLotteryPlan,
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to create lottery plan'))
        return
      }
      toast.success(t('Lottery plan created'))
      form.reset()
      await queryClient.invalidateQueries({
        queryKey: ['lottery', 'admin', 'plans'],
      })
    },
  })
  const participantMutation = useMutation({
    mutationFn: updateLotteryParticipant,
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to update lottery participant'))
        return
      }
      await queryClient.invalidateQueries({
        queryKey: ['lottery', 'admin', 'participants', selectedPlanId],
      })
    },
  })
  const drawMutation = useMutation({
    mutationFn: (planId: number) =>
      drawLotteryPlan(planId, 'admin manual draw'),
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(result.message || t('Failed to draw lottery'))
        return
      }
      toast.success(t('Lottery draw completed'))
      await queryClient.invalidateQueries({
        queryKey: ['lottery', 'admin', 'plans'],
      })
    },
  })

  const submit = (values: LotteryAdminForm): void => {
    const allowList = splitAllowList(values.allow_list)
    const payload: LotteryPlanCreatePayload = {
      title: values.title,
      description: values.description,
      status: 'scheduled',
      eligibility_mode: values.eligibility_mode,
      max_participants: values.max_participants,
      registration_start_time: unixFromLocal(values.registration_start),
      draw_time: unixFromLocal(values.draw_time),
      user_ids:
        values.eligibility_mode === 'users'
          ? allowList.map((item) => Number(item)).filter(Number.isInteger)
          : [],
      groups: values.eligibility_mode === 'groups' ? allowList : [],
      prizes: values.prizes,
    }
    createMutation.mutate(payload)
  }

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>
        {t('Lottery Management')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='grid gap-6'>
          <form
            className='grid gap-4 rounded-lg border p-4'
            onSubmit={form.handleSubmit(submit)}
          >
            <div className='grid gap-3 md:grid-cols-2'>
              <Input
                placeholder={t('Lottery title')}
                {...form.register('title')}
              />
              <Input
                type='number'
                min={1}
                {...form.register('max_participants', { valueAsNumber: true })}
              />
              <Input
                type='datetime-local'
                {...form.register('registration_start')}
              />
              <Input type='datetime-local' {...form.register('draw_time')} />
              <select
                className='h-9 rounded-md border px-3 text-sm'
                {...form.register('eligibility_mode')}
              >
                <option value='all'>{t('All users')}</option>
                <option value='groups'>{t('Groups')}</option>
                <option value='users'>{t('User allow list')}</option>
              </select>
              <Input
                placeholder={t('Groups or user IDs')}
                {...form.register('allow_list')}
              />
            </div>
            <textarea
              className='min-h-20 rounded-md border p-3 text-sm'
              placeholder={t('Lottery description')}
              {...form.register('description')}
            />
            <div className='grid gap-3'>
              {prizes.fields.map((field, index) => (
                <div
                  key={field.id}
                  className='grid gap-2 rounded-md border p-3 md:grid-cols-4'
                >
                  <Input
                    placeholder={t('Prize name')}
                    {...form.register(`prizes.${index}.name`)}
                  />
                  <Input
                    type='number'
                    min={1}
                    {...form.register(`prizes.${index}.quantity`, {
                      valueAsNumber: true,
                    })}
                  />
                  <select
                    className='h-9 rounded-md border px-3 text-sm'
                    {...form.register(`prizes.${index}.reward_type`)}
                  >
                    <option value='quota'>{t('Quota')}</option>
                    <option value='subscription'>{t('Subscription')}</option>
                  </select>
                  <Input
                    type='number'
                    min={0}
                    {...form.register(`prizes.${index}.quota`, {
                      valueAsNumber: true,
                    })}
                  />
                  <Input
                    type='number'
                    min={0}
                    placeholder={t('Subscription plan ID')}
                    {...form.register(`prizes.${index}.subscription_plan_id`, {
                      valueAsNumber: true,
                    })}
                  />
                  <select
                    className='h-9 rounded-md border px-3 text-sm'
                    {...form.register(`prizes.${index}.fulfillment_mode`)}
                  >
                    <option value='auto'>{t('Automatic')}</option>
                    <option value='self_claim'>{t('Self claim')}</option>
                    <option value='redemption_code'>
                      {t('Redemption code')}
                    </option>
                  </select>
                  <Input
                    type='number'
                    min={0}
                    {...form.register(`prizes.${index}.claim_expire_seconds`, {
                      valueAsNumber: true,
                    })}
                  />
                  <Button
                    type='button'
                    variant='outline'
                    onClick={() => prizes.remove(index)}
                    disabled={prizes.fields.length === 1}
                  >
                    {t('Remove')}
                  </Button>
                </div>
              ))}
              <div className='flex flex-wrap gap-2'>
                <Button
                  type='button'
                  variant='outline'
                  onClick={() => prizes.append(DEFAULT_PRIZE)}
                >
                  {t('Add prize')}
                </Button>
                <Button type='submit' disabled={createMutation.isPending}>
                  {t('Create lottery plan')}
                </Button>
              </div>
            </div>
          </form>

          <div className='grid gap-3'>
            {(plansQuery.data?.success ? plansQuery.data.data : []).map(
              (plan) => (
                <section
                  key={plan.id}
                  className='flex flex-wrap items-center justify-between gap-3 rounded-lg border p-4'
                >
                  <div>
                    <div className='font-medium'>{plan.title}</div>
                    <div className='text-muted-foreground text-xs'>
                      {plan.status}
                    </div>
                  </div>
                  <div className='flex gap-2'>
                    <Button
                      size='sm'
                      variant='outline'
                      onClick={() => setSelectedPlanId(plan.id)}
                    >
                      {t('Participants')}
                    </Button>
                    <Button
                      size='sm'
                      variant='outline'
                      disabled={
                        drawMutation.isPending || plan.status !== 'open'
                      }
                      onClick={() => drawMutation.mutate(plan.id)}
                    >
                      {t('Draw now')}
                    </Button>
                  </div>
                </section>
              )
            )}
          </div>

          {selectedPlanId && (
            <div className='grid gap-2 rounded-lg border p-4'>
              {(participantsQuery.data?.success
                ? participantsQuery.data.data
                : []
              ).map((participant) => (
                <div
                  key={participant.id}
                  className='flex flex-wrap items-center gap-2 border-b py-2 last:border-b-0'
                >
                  <span className='min-w-40 text-sm'>
                    {participant.display_name || participant.username} #
                    {participant.user_id}
                  </span>
                  <Input
                    className='w-24'
                    type='number'
                    defaultValue={participant.weight}
                    onBlur={(event) =>
                      participantMutation.mutate({
                        planId: selectedPlanId,
                        userId: participant.user_id,
                        weight: Number(event.target.value),
                      })
                    }
                  />
                  <Input
                    className='w-28'
                    type='number'
                    placeholder={t('Preset prize ID')}
                    defaultValue={participant.preset_prize_id || ''}
                    onBlur={(event) =>
                      participantMutation.mutate({
                        planId: selectedPlanId,
                        userId: participant.user_id,
                        presetPrizeId: Number(event.target.value) || 0,
                      })
                    }
                  />
                </div>
              ))}
            </div>
          )}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
