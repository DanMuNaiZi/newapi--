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
import type { UseQueryResult } from '@tanstack/react-query'
import type { UseFormRegisterReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Spinner } from '@/components/ui/spinner'
import {
  getRewardConversionError,
  getRewardEquivalent,
  type RewardUnit,
} from '@/lib/reward-amount'

interface RewardAmountFieldProps {
  id: string
  amount: string
  unit: RewardUnit
  amountField: UseFormRegisterReturn
  unitField: UseFormRegisterReturn
  config: UseQueryResult<number, Error>
  error?: string
  preservedQuota?: number
  className?: string
}

export function RewardAmountField(props: RewardAmountFieldProps) {
  const { t } = useTranslation()
  const preserved = props.preservedQuota != null
  const error =
    props.error ||
    (!preserved && props.config.isSuccess
      ? getRewardConversionError(props.amount, props.unit, props.config.data)
      : null)
  const equivalent = props.config.isSuccess
    ? getRewardEquivalent(
        preserved ? String(props.preservedQuota) : props.amount,
        preserved ? 'quota' : props.unit,
        props.config.data
      )
    : null

  let description = t('Enter a valid amount to preview conversion.')
  if (preserved) {
    description = t('Keeping the saved reward of {{quota}} platform quota.', {
      quota: props.preservedQuota?.toLocaleString(),
    })
  } else if (equivalent) {
    description = t('Equivalent: ${{usd}} · {{quota}} platform quota', {
      usd: equivalent.usd,
      quota: equivalent.quota.toLocaleString(),
    })
  }

  return (
    <Field className={props.className} data-invalid={Boolean(error)}>
      <div className='grid gap-3 sm:grid-cols-[minmax(0,1fr)_170px]'>
        <div className='grid min-w-0 gap-2'>
          <FieldLabel htmlFor={props.id}>{t('Reward amount')}</FieldLabel>
          <Input
            id={props.id}
            inputMode={props.unit === 'quota' ? 'numeric' : 'decimal'}
            autoComplete='off'
            aria-invalid={Boolean(error)}
            aria-describedby={`${props.id}-description${error ? ` ${props.id}-error` : ''}`}
            {...props.amountField}
          />
        </div>
        <div className='grid min-w-0 gap-2'>
          <FieldLabel htmlFor={`${props.id}-unit`}>
            {t('Reward unit')}
          </FieldLabel>
          <NativeSelect id={`${props.id}-unit`} {...props.unitField}>
            <NativeSelectOption value='usd'>
              {t('US dollars (USD)')}
            </NativeSelectOption>
            <NativeSelectOption value='quota'>
              {t('Platform quota')} (quota)
            </NativeSelectOption>
          </NativeSelect>
        </div>
      </div>
      <FieldDescription id={`${props.id}-description`} aria-live='polite'>
        {description}
        {preserved && props.unit === 'quota' && equivalent && (
          <> {t('Equivalent USD value: ${{usd}}', { usd: equivalent.usd })}</>
        )}
      </FieldDescription>
      {(props.config.isPending || props.config.isFetching) && !preserved && (
        <div
          className='text-muted-foreground flex items-center gap-2 text-sm'
          role='status'
        >
          <Spinner />
          {t('Loading reward configuration')}
        </div>
      )}
      {props.config.isError && (
        <div className='flex flex-wrap items-center gap-2 text-sm' role='alert'>
          <span className='text-destructive'>
            {t(props.config.error.message)}
          </span>
          <Button
            type='button'
            size='sm'
            variant='outline'
            onClick={() => void props.config.refetch()}
          >
            {t('Retry')}
          </Button>
        </div>
      )}
      <FieldError id={`${props.id}-error`}>
        {error ? t(error) : null}
      </FieldError>
    </Field>
  )
}
