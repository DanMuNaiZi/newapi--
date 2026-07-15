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
import { Pencil, Play, Users, XCircle } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Textarea } from '@/components/ui/textarea'

import {
  cancelLotteryPlan,
  drawLotteryPlan,
  getAdminLotteryPlans,
  getLotteryParticipants,
  getLotteryPrizes,
  updateLotteryParticipant,
  updateLotteryPlan,
} from './api'
import { LotteryPlanForm } from './components/lottery-plan-form'
import { localInputFromUnix, unixFromLocal } from './lib/admin-form'
import { getLotteryPlanStatusLabel } from './lib/status'
import type { LotteryPlan, LotteryPlanUpdatePayload } from './types'

function formatLotteryDate(value: number): string {
  return new Date(value * 1000).toLocaleString()
}

export function LotteryAdmin() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [selectedPlanId, setSelectedPlanId] = useState<number | null>(null)
  const [editingPlan, setEditingPlan] = useState<LotteryPlan | null>(null)
  const [editTitle, setEditTitle] = useState('')
  const [editDescription, setEditDescription] = useState('')
  const [editDrawTime, setEditDrawTime] = useState('')

  const plansQuery = useQuery({
    queryKey: ['lottery', 'admin', 'plans'],
    queryFn: getAdminLotteryPlans,
  })
  const participantsQuery = useQuery({
    queryKey: ['lottery', 'admin', 'participants', selectedPlanId],
    queryFn: () => getLotteryParticipants(selectedPlanId ?? 0),
    enabled: Boolean(selectedPlanId),
  })
  const selectedPlanPrizesQuery = useQuery({
    queryKey: ['lottery', 'admin', 'prizes', selectedPlanId],
    queryFn: () => getLotteryPrizes(selectedPlanId ?? 0),
    enabled: Boolean(selectedPlanId),
  })

  const participantMutation = useMutation({
    mutationFn: updateLotteryParticipant,
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(t('Failed to update lottery participant'))
        return
      }
      toast.success(t('Lottery participant updated'))
      await queryClient.invalidateQueries({
        queryKey: ['lottery', 'admin', 'participants', selectedPlanId],
      })
    },
  })
  const drawMutation = useMutation({
    mutationFn: ({ planId, reason }: { planId: number; reason: string }) =>
      drawLotteryPlan(planId, reason),
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(t('Failed to draw lottery'))
        return
      }
      toast.success(t('Lottery draw completed'))
      await queryClient.invalidateQueries({
        queryKey: ['lottery', 'admin', 'plans'],
      })
    },
  })
  const updatePlanMutation = useMutation({
    mutationFn: ({
      planId,
      payload,
    }: {
      planId: number
      payload: LotteryPlanUpdatePayload
    }) => updateLotteryPlan(planId, payload),
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(t('Failed to update lottery plan'))
        return
      }
      toast.success(t('Lottery plan updated'))
      setEditingPlan(null)
      await queryClient.invalidateQueries({
        queryKey: ['lottery', 'admin', 'plans'],
      })
    },
  })
  const cancelPlanMutation = useMutation({
    mutationFn: cancelLotteryPlan,
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(t('Failed to cancel lottery plan'))
        return
      }
      toast.success(t('Lottery plan cancelled'))
      setEditingPlan(null)
      await queryClient.invalidateQueries({
        queryKey: ['lottery', 'admin', 'plans'],
      })
    },
  })

  const plans = plansQuery.data?.success ? (plansQuery.data.data ?? []) : []
  const selectedPlan = plans.find((plan) => plan.id === selectedPlanId)
  const participants = participantsQuery.data?.success
    ? (participantsQuery.data.data ?? [])
    : []
  const selectedPlanPrizes = selectedPlanPrizesQuery.data?.success
    ? (selectedPlanPrizesQuery.data.data ?? [])
    : []

  const getEligibilityLabel = (
    mode: LotteryPlan['eligibility_mode']
  ): string => {
    switch (mode) {
      case 'all':
        return t('All users')
      case 'groups':
        return t('Selected groups')
      case 'users':
        return t('Selected users')
    }
  }

  const beginEditing = (plan: LotteryPlan): void => {
    setEditingPlan(plan)
    setEditTitle(plan.title)
    setEditDescription(plan.description)
    setEditDrawTime(localInputFromUnix(plan.draw_time))
  }

  const submitPlanUpdate = (): void => {
    if (!editingPlan) return
    const drawTime = unixFromLocal(editDrawTime)
    if (!Number.isFinite(drawTime) || drawTime < editingPlan.draw_time) {
      toast.error(t('Draw time must be later than the current draw time'))
      return
    }
    const payload: LotteryPlanUpdatePayload = {
      title: editTitle.trim(),
      description: editDescription.trim(),
    }
    if (drawTime > editingPlan.draw_time) {
      payload.draw_time = drawTime
    }
    updatePlanMutation.mutate({
      planId: editingPlan.id,
      payload,
    })
  }

  const requestManualDraw = (planId: number): void => {
    const reason = window.prompt(t('Enter the reason for the early draw'))
    if (!reason?.trim()) return
    if (!window.confirm(t('Confirm early draw? This cannot be undone.'))) return
    drawMutation.mutate({ planId, reason: reason.trim() })
  }

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>
        {t('Lottery Management')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='grid gap-8'>
          <LotteryPlanForm />

          <section className='grid gap-4'>
            <div>
              <h3 className='text-base font-semibold'>{t('Lottery plans')}</h3>
              <p className='text-muted-foreground mt-1 text-sm'>
                {t(
                  'Review schedules, participation rules, and management actions.'
                )}
              </p>
            </div>

            {plans.length === 0 ? (
              <div className='text-muted-foreground border-border border-y py-10 text-center text-sm'>
                {t('No lottery plans')}
              </div>
            ) : (
              <div className='border-border divide-border divide-y border-y'>
                {plans.map((plan) => (
                  <article
                    key={plan.id}
                    className='grid gap-4 py-4 xl:grid-cols-[minmax(0,1fr)_auto] xl:items-center'
                  >
                    <div className='min-w-0'>
                      <div className='flex flex-wrap items-center gap-2'>
                        <h4 className='truncate font-medium'>{plan.title}</h4>
                        <Badge variant='outline'>
                          {getLotteryPlanStatusLabel(t, plan.status)}
                        </Badge>
                      </div>
                      {plan.description && (
                        <p className='text-muted-foreground mt-1 line-clamp-2 text-sm'>
                          {plan.description}
                        </p>
                      )}
                      <dl className='text-muted-foreground mt-3 grid gap-x-6 gap-y-1 text-xs sm:grid-cols-2 lg:grid-cols-4'>
                        <div>
                          <dt className='font-medium'>
                            {t('Registration start time')}
                          </dt>
                          <dd>
                            {formatLotteryDate(plan.registration_start_time)}
                          </dd>
                        </div>
                        <div>
                          <dt className='font-medium'>{t('Draw time')}</dt>
                          <dd>{formatLotteryDate(plan.draw_time)}</dd>
                        </div>
                        <div>
                          <dt className='font-medium'>
                            {t('Maximum participants')}
                          </dt>
                          <dd>{plan.max_participants}</dd>
                        </div>
                        <div>
                          <dt className='font-medium'>
                            {t('Eligible participants')}
                          </dt>
                          <dd>{getEligibilityLabel(plan.eligibility_mode)}</dd>
                        </div>
                      </dl>
                    </div>
                    <div className='flex flex-wrap gap-2 xl:justify-end'>
                      <Button
                        size='sm'
                        variant='outline'
                        onClick={() => beginEditing(plan)}
                        disabled={
                          plan.status !== 'scheduled' && plan.status !== 'open'
                        }
                      >
                        <Pencil />
                        {t('Edit')}
                      </Button>
                      <Button
                        size='sm'
                        variant={
                          selectedPlanId === plan.id ? 'secondary' : 'outline'
                        }
                        onClick={() => setSelectedPlanId(plan.id)}
                      >
                        <Users />
                        {t('Participants')}
                      </Button>
                      <Button
                        size='sm'
                        variant='outline'
                        disabled={
                          drawMutation.isPending || plan.status !== 'open'
                        }
                        onClick={() => requestManualDraw(plan.id)}
                      >
                        <Play />
                        {t('Draw now')}
                      </Button>
                      <Button
                        size='sm'
                        variant='destructive'
                        disabled={
                          cancelPlanMutation.isPending ||
                          (plan.status !== 'scheduled' &&
                            plan.status !== 'open')
                        }
                        onClick={() => {
                          if (window.confirm(t('Cancel this lottery plan?'))) {
                            cancelPlanMutation.mutate(plan.id)
                          }
                        }}
                      >
                        <XCircle />
                        {t('Cancel')}
                      </Button>
                    </div>
                  </article>
                ))}
              </div>
            )}
          </section>

          {editingPlan && (
            <section className='border-border grid gap-4 border-t pt-6'>
              <div>
                <h3 className='text-base font-semibold'>
                  {t('Edit lottery plan')}
                </h3>
                <p className='text-muted-foreground mt-1 text-sm'>
                  {t(
                    'Published plans only allow text changes and a later draw time.'
                  )}
                </p>
              </div>
              <FieldGroup>
                <div className='grid grid-cols-1 gap-4 lg:grid-cols-2'>
                  <Field className='lg:col-span-2'>
                    <FieldLabel htmlFor='edit-lottery-title'>
                      {t('Lottery title')}
                    </FieldLabel>
                    <Input
                      id='edit-lottery-title'
                      value={editTitle}
                      onChange={(event) => setEditTitle(event.target.value)}
                    />
                  </Field>
                  <Field className='lg:col-span-2'>
                    <FieldLabel htmlFor='edit-lottery-description'>
                      {t('Lottery description')}
                    </FieldLabel>
                    <Textarea
                      id='edit-lottery-description'
                      value={editDescription}
                      onChange={(event) =>
                        setEditDescription(event.target.value)
                      }
                    />
                  </Field>
                  <Field>
                    <FieldLabel htmlFor='edit-lottery-draw-time'>
                      {t('Draw time')}
                    </FieldLabel>
                    <Input
                      id='edit-lottery-draw-time'
                      type='datetime-local'
                      value={editDrawTime}
                      onChange={(event) => setEditDrawTime(event.target.value)}
                    />
                    <FieldDescription>
                      {t('The draw time can only be postponed.')}
                    </FieldDescription>
                  </Field>
                </div>
              </FieldGroup>
              <div className='flex flex-wrap gap-2'>
                <Button
                  type='button'
                  onClick={submitPlanUpdate}
                  disabled={updatePlanMutation.isPending || !editTitle.trim()}
                >
                  {t('Save changes')}
                </Button>
                <Button
                  type='button'
                  variant='outline'
                  onClick={() => setEditingPlan(null)}
                >
                  {t('Close')}
                </Button>
              </div>
            </section>
          )}

          {selectedPlanId && (
            <section className='border-border grid gap-4 border-t pt-6'>
              <div>
                <h3 className='text-base font-semibold'>
                  {t('Participants for {{title}}', {
                    title: selectedPlan?.title ?? `#${selectedPlanId}`,
                  })}
                </h3>
                <p className='text-muted-foreground mt-1 text-sm'>
                  {t('Set each participant weight or assign a preset prize.')}
                </p>
              </div>

              {participants.length === 0 ? (
                <div className='text-muted-foreground border-border border-y py-8 text-center text-sm'>
                  {t('No participants yet')}
                </div>
              ) : (
                <div className='border-border divide-border divide-y border-y'>
                  {participants.map((participant) => (
                    <div
                      key={participant.id}
                      className='grid gap-3 py-4 lg:grid-cols-[minmax(220px,1fr)_160px_minmax(220px,320px)] lg:items-end'
                    >
                      <div className='min-w-0'>
                        <div className='truncate text-sm font-medium'>
                          {participant.display_name || participant.username}
                        </div>
                        <div className='text-muted-foreground truncate text-xs'>
                          @{participant.username} | #{participant.user_id} |{' '}
                          {participant.user_group}
                        </div>
                      </div>
                      <Field>
                        <FieldLabel
                          htmlFor={`participant-${participant.id}-weight`}
                        >
                          {t('Lottery weight')}
                        </FieldLabel>
                        <Input
                          id={`participant-${participant.id}-weight`}
                          type='number'
                          min={1}
                          defaultValue={participant.weight}
                          onBlur={(event) => {
                            const weight = Number(event.target.value)
                            if (!Number.isInteger(weight) || weight < 1) {
                              toast.error(
                                t('Lottery weight must be at least 1')
                              )
                              return
                            }
                            participantMutation.mutate({
                              planId: selectedPlanId,
                              userId: participant.user_id,
                              weight,
                            })
                          }}
                        />
                      </Field>
                      <Field>
                        <FieldLabel
                          htmlFor={`participant-${participant.id}-prize`}
                        >
                          {t('Preset prize')}
                        </FieldLabel>
                        <NativeSelect
                          id={`participant-${participant.id}-prize`}
                          className='w-full'
                          defaultValue={String(
                            participant.preset_prize_id || 0
                          )}
                          onChange={(event) =>
                            participantMutation.mutate({
                              planId: selectedPlanId,
                              userId: participant.user_id,
                              presetPrizeId: Number(event.target.value) || 0,
                            })
                          }
                        >
                          <NativeSelectOption value='0'>
                            {t('No preset prize')}
                          </NativeSelectOption>
                          {selectedPlanPrizes.map((prize) => (
                            <NativeSelectOption key={prize.id} value={prize.id}>
                              #{prize.id} {prize.name}
                            </NativeSelectOption>
                          ))}
                        </NativeSelect>
                      </Field>
                    </div>
                  ))}
                </div>
              )}
            </section>
          )}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
