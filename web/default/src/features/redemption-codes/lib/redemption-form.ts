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
import type { TFunction } from 'i18next'
import { z } from 'zod'

import { parseQuotaFromDollars, quotaUnitsToDollars } from '@/lib/format'

import {
  REDEMPTION_VALIDATION,
  getRedemptionFormErrorMessages,
} from '../constants'
import type { Redemption, RedemptionFormData } from '../types'

// ============================================================================
// Form Schema (use getRedemptionFormSchema(t) in components for i18n messages)
// ============================================================================

export function getRedemptionFormSchema(t: TFunction) {
  const msg = getRedemptionFormErrorMessages(t)
  return z
    .object({
      name: z
        .string()
        .min(REDEMPTION_VALIDATION.NAME_MIN_LENGTH, msg.NAME_LENGTH_INVALID)
        .max(REDEMPTION_VALIDATION.NAME_MAX_LENGTH, msg.NAME_LENGTH_INVALID),
      reward_type: z.enum(['quota', 'subscription']),
      quota_dollars: z.number().min(0, t('Quota must be a positive number')),
      subscription_plan_id: z.number().int().min(0),
      batch: z.string().max(64),
      source_ref: z.string().max(128),
      remark: z.string().max(255),
      expired_time: z.date().optional(),
      count: z
        .number()
        .min(REDEMPTION_VALIDATION.COUNT_MIN, msg.COUNT_INVALID)
        .max(REDEMPTION_VALIDATION.COUNT_MAX, msg.COUNT_INVALID)
        .optional(),
    })
    .superRefine((values, context) => {
      if (values.reward_type === 'quota' && values.quota_dollars <= 0) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          message: t('Quota must be a positive number'),
          path: ['quota_dollars'],
        })
      }
      if (
        values.reward_type === 'subscription' &&
        values.subscription_plan_id <= 0
      ) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          message: t('Select a subscription plan'),
          path: ['subscription_plan_id'],
        })
      }
    })
}

export type RedemptionFormValues = {
  name: string
  reward_type: 'quota' | 'subscription'
  quota_dollars: number
  subscription_plan_id: number
  batch: string
  source_ref: string
  remark: string
  expired_time?: Date
  count?: number
}

// ============================================================================
// Form Defaults
// ============================================================================

export const REDEMPTION_FORM_DEFAULT_VALUES: RedemptionFormValues = {
  name: '',
  reward_type: 'quota',
  quota_dollars: 10,
  subscription_plan_id: 0,
  batch: '',
  source_ref: '',
  remark: '',
  expired_time: undefined,
  count: 1,
}

// ============================================================================
// Form Data Transformation
// ============================================================================

/**
 * Transform form data to API payload
 */
export function transformFormDataToPayload(
  data: RedemptionFormValues,
): RedemptionFormData {
  return {
    name: data.name,
    quota:
      data.reward_type === 'quota'
        ? parseQuotaFromDollars(data.quota_dollars)
        : 0,
    reward_type: data.reward_type,
    subscription_plan_id:
      data.reward_type === 'subscription' ? data.subscription_plan_id : 0,
    batch: data.batch.trim(),
    source_ref: data.source_ref.trim(),
    remark: data.remark.trim(),
    expired_time: data.expired_time
      ? Math.floor(data.expired_time.getTime() / 1000)
      : 0,
    count: data.count || 1,
  }
}

/**
 * Transform redemption data to form defaults
 */
export function transformRedemptionToFormDefaults(
  redemption: Redemption,
): RedemptionFormValues {
  return {
    name: redemption.name,
    reward_type: redemption.reward_type,
    quota_dollars: quotaUnitsToDollars(redemption.quota),
    subscription_plan_id: redemption.subscription_plan_id,
    batch: redemption.batch,
    source_ref: redemption.source_ref,
    remark: redemption.remark,
    expired_time:
      redemption.expired_time > 0
        ? new Date(redemption.expired_time * 1000)
        : undefined,
    count: 1,
  }
}
