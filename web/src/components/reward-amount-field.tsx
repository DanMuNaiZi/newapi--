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
import type { UseFormRegisterReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'
import {
  getRewardConversionError,
  getRewardEquivalent,
  type RewardUnit,
} from '@/lib/reward-amount'

type RewardAmountFieldProps = {
  id: string
  amount: string
  unit: RewardUnit
  amountField: UseFormRegisterReturn
  unitField: UseFormRegisterReturn
  quotaPerUnit?: number
  usdExchangeRate?: number
  error?: string
}

export function RewardAmountField(props: RewardAmountFieldProps) {
  const { t, i18n } = useTranslation()
  const conversionError = getRewardConversionError(
    props.amount,
    props.unit,
    props.quotaPerUnit,
    props.usdExchangeRate
  )
  const equivalent = getRewardEquivalent(
    props.amount,
    props.unit,
    props.quotaPerUnit ?? 0,
    props.usdExchangeRate
  )
  const error = props.error || conversionError

  return (
    <Field data-invalid={Boolean(error)}>
      <div className='grid gap-3 sm:grid-cols-[minmax(0,1fr)_170px]'>
        <div className='grid min-w-0 gap-2'>
          <FieldLabel htmlFor={props.id}>{t('Reward amount')}</FieldLabel>
          <Input
            id={props.id}
            inputMode={props.unit === 'quota' ? 'numeric' : 'decimal'}
            autoComplete='off'
            aria-invalid={Boolean(error)}
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
            <NativeSelectOption value='cny'>
              {t('Chinese yuan (CNY)')}
            </NativeSelectOption>
            <NativeSelectOption value='quota'>
              {t('Platform quota')} (quota)
            </NativeSelectOption>
          </NativeSelect>
        </div>
      </div>
      <FieldDescription>
        {equivalent
          ? t('Equivalent: ${{usd}} · {{quota}} platform quota', {
              usd: equivalent.usd,
              quota: formatNumber(
                equivalent.quota,
                toIntlLocale(i18n.resolvedLanguage ?? i18n.language)
              ),
            })
          : t('Enter a valid amount to preview conversion.')}
      </FieldDescription>
      <FieldError>{error ? t(error) : null}</FieldError>
    </Field>
  )
}
