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
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Field, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'

import type { LotteryParticipant, LotteryPrize } from '../types'

export type LotteryParticipantUpdate = {
  planId: number
  presetPrizeId?: number
  userId: number
  weight?: number
}

type LotteryParticipantEditorProps = {
  isPending: boolean
  participant: LotteryParticipant
  planId: number
  prizes: LotteryPrize[]
  onUpdate: (payload: LotteryParticipantUpdate) => Promise<boolean>
}

export function LotteryParticipantEditor(props: LotteryParticipantEditorProps) {
  const { t } = useTranslation()
  const [weight, setWeight] = useState(String(props.participant.weight))
  const [presetPrizeId, setPresetPrizeId] = useState(
    props.participant.preset_prize_id
  )
  const disabled = props.isPending || props.participant.status !== 'joined'

  useEffect(() => {
    setWeight(String(props.participant.weight))
    setPresetPrizeId(props.participant.preset_prize_id)
  }, [
    props.participant.id,
    props.participant.preset_prize_id,
    props.participant.weight,
  ])

  const saveWeight = async (): Promise<void> => {
    const nextWeight = Number(weight)
    if (!Number.isInteger(nextWeight) || nextWeight < 1) {
      toast.error(t('Lottery weight must be at least 1'))
      setWeight(String(props.participant.weight))
      return
    }
    if (nextWeight === props.participant.weight) return

    const success = await props.onUpdate({
      planId: props.planId,
      userId: props.participant.user_id,
      weight: nextWeight,
    })
    if (!success) setWeight(String(props.participant.weight))
  }

  const savePresetPrize = async (nextPresetPrizeId: number): Promise<void> => {
    setPresetPrizeId(nextPresetPrizeId)
    const success = await props.onUpdate({
      planId: props.planId,
      userId: props.participant.user_id,
      presetPrizeId: nextPresetPrizeId,
    })
    if (!success) setPresetPrizeId(props.participant.preset_prize_id)
  }

  return (
    <div className='grid gap-3 py-4 lg:grid-cols-[minmax(200px,1fr)_140px_minmax(210px,280px)] lg:items-end'>
      <div className='min-w-0'>
        <div className='truncate text-sm font-medium'>
          {props.participant.display_name || props.participant.username}
        </div>
        <div className='text-muted-foreground truncate text-xs'>
          @{props.participant.username} | #{props.participant.user_id} |{' '}
          {props.participant.user_group}
        </div>
      </div>
      <Field>
        <FieldLabel htmlFor={`participant-${props.participant.id}-weight`}>
          {t('Lottery weight')}
        </FieldLabel>
        <Input
          id={`participant-${props.participant.id}-weight`}
          type='number'
          min={1}
          value={weight}
          disabled={disabled}
          onChange={(event) => setWeight(event.target.value)}
          onBlur={() => void saveWeight()}
        />
      </Field>
      <Field>
        <FieldLabel htmlFor={`participant-${props.participant.id}-prize`}>
          {t('Preset prize')}
        </FieldLabel>
        <NativeSelect
          id={`participant-${props.participant.id}-prize`}
          className='w-full'
          value={String(presetPrizeId || 0)}
          disabled={disabled}
          onChange={(event) =>
            void savePresetPrize(Number(event.target.value) || 0)
          }
        >
          <NativeSelectOption value='0'>
            {t('No preset prize')}
          </NativeSelectOption>
          {props.prizes.map((prize) => (
            <NativeSelectOption key={prize.id} value={prize.id}>
              #{prize.id} {prize.name}
            </NativeSelectOption>
          ))}
        </NativeSelect>
      </Field>
    </div>
  )
}
