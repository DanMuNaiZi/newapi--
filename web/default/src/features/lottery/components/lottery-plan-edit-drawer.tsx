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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
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
import { Textarea } from '@/components/ui/textarea'

import { updateLotteryPlan } from '../api'
import { localInputFromUnix, unixFromLocal } from '../lib/admin-form'
import type { LotteryPlan, LotteryPlanUpdatePayload } from '../types'

type LotteryPlanEditDrawerProps = {
  open: boolean
  plan: LotteryPlan | null
  onOpenChange: (open: boolean) => void
}

export function LotteryPlanEditDrawer(props: LotteryPlanEditDrawerProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [drawTime, setDrawTime] = useState('')

  useEffect(() => {
    if (!props.open || !props.plan) return
    setTitle(props.plan.title)
    setDescription(props.plan.description)
    setDrawTime(localInputFromUnix(props.plan.draw_time))
  }, [props.open, props.plan])

  const mutation = useMutation({
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
      props.onOpenChange(false)
      await queryClient.invalidateQueries({
        queryKey: ['lottery', 'admin', 'plans'],
      })
    },
  })

  const submit = (): void => {
    if (!props.plan) return
    const nextDrawTime = unixFromLocal(drawTime)
    if (!Number.isFinite(nextDrawTime) || nextDrawTime < props.plan.draw_time) {
      toast.error(t('Draw time must be later than the current draw time'))
      return
    }
    const payload: LotteryPlanUpdatePayload = {
      title: title.trim(),
      description: description.trim(),
    }
    if (nextDrawTime > props.plan.draw_time) {
      payload.draw_time = nextDrawTime
    }
    mutation.mutate({ planId: props.plan.id, payload })
  }

  return (
    <Sheet open={props.open} onOpenChange={props.onOpenChange}>
      <SheetContent className={sideDrawerContentClassName('sm:max-w-[620px]')}>
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>{t('Edit lottery plan')}</SheetTitle>
          <SheetDescription>
            {t(
              'Published plans only allow text changes and a later draw time.'
            )}
          </SheetDescription>
        </SheetHeader>
        <div className={sideDrawerFormClassName()}>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor='edit-lottery-title'>
                {t('Lottery title')}
              </FieldLabel>
              <Input
                id='edit-lottery-title'
                value={title}
                onChange={(event) => setTitle(event.target.value)}
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='edit-lottery-description'>
                {t('Lottery description')}
              </FieldLabel>
              <Textarea
                id='edit-lottery-description'
                value={description}
                onChange={(event) => setDescription(event.target.value)}
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='edit-lottery-draw-time'>
                {t('Draw time')}
              </FieldLabel>
              <Input
                id='edit-lottery-draw-time'
                type='datetime-local'
                value={drawTime}
                onChange={(event) => setDrawTime(event.target.value)}
              />
              <FieldDescription>
                {t('The draw time can only be postponed.')}
              </FieldDescription>
            </Field>
          </FieldGroup>
        </div>
        <SheetFooter className={sideDrawerFooterClassName()}>
          <Button variant='outline' onClick={() => props.onOpenChange(false)}>
            {t('Close')}
          </Button>
          <Button
            onClick={submit}
            disabled={mutation.isPending || !title.trim()}
          >
            {t('Save changes')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
