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
import { z } from 'zod'

import {
  getRewardAmountError,
  isRewardUnchanged,
  normalizeRewardAmount,
  rewardSnapshotToForm,
} from '@/lib/reward-amount'

import type { PublicPoolSite, PublicPoolSitePayload } from '../types'

function isCredentialFreeHttpUrl(value: string): boolean {
  try {
    const url = new URL(value)
    return (
      (url.protocol === 'http:' || url.protocol === 'https:') &&
      url.hostname.length > 0 &&
      url.username.length === 0 &&
      url.password.length === 0
    )
  } catch {
    return false
  }
}

export const publicPoolSiteSchema = z
  .object({
    name: z
      .string()
      .trim()
      .min(1, 'Site name is required')
      .max(128, 'Site name cannot exceed 128 characters'),
    url: z
      .string()
      .trim()
      .max(1024, 'URL cannot exceed 1024 characters')
      .refine(
        isCredentialFreeHttpUrl,
        'Enter an HTTP or HTTPS URL without credentials'
      ),
    description: z.string().trim().max(4000, 'Description is too long'),
    status: z.enum(['enabled', 'disabled'], { error: 'Select a site status' }),
    sort_order: z
      .number({ error: 'Sort order must be an integer' })
      .int('Sort order must be an integer')
      .min(-2147483648, 'Sort order is outside the supported range')
      .max(2147483647, 'Sort order is outside the supported range'),
    reward_type: z.enum(['none', 'quota', 'subscription'], {
      error: 'Select a reward type',
    }),
    reward_amount: z.string().trim(),
    reward_unit: z.enum(['usd', 'quota'], {
      error: 'Select USD or platform quota',
    }),
    subscription_plan_id: z
      .number({ error: 'Please select a subscription plan' })
      .int('Please select a subscription plan')
      .min(0, 'Please select a subscription plan'),
  })
  .superRefine((value, context) => {
    if (value.reward_type === 'quota') {
      const amountError = getRewardAmountError(
        value.reward_amount,
        value.reward_unit
      )
      if (amountError) {
        context.addIssue({
          code: 'custom',
          path: ['reward_amount'],
          message: amountError,
        })
      }
    }
    if (
      value.reward_type === 'subscription' &&
      value.subscription_plan_id <= 0
    ) {
      context.addIssue({
        code: 'custom',
        path: ['subscription_plan_id'],
        message: 'Select a subscription plan',
      })
    }
  })

export type PublicPoolSiteFormValues = z.infer<typeof publicPoolSiteSchema>

export function publicPoolSiteToForm(
  site: PublicPoolSite
): PublicPoolSiteFormValues {
  return {
    name: site.name,
    url: site.url,
    description: site.description,
    status: site.status,
    sort_order: site.sort_order,
    ...rewardSnapshotToForm(site.reward),
  }
}

export function toPublicPoolSitePayload(
  values: PublicPoolSiteFormValues,
  site?: PublicPoolSite | null
): PublicPoolSitePayload {
  const base = {
    name: values.name,
    url: values.url,
    description: values.description,
    status: values.status,
    sort_order: values.sort_order,
    ...(site
      ? { preserve_reward: isRewardUnchanged(values, site.reward) }
      : {}),
  }
  if (values.reward_type === 'none') {
    return { ...base, reward: null }
  }
  if (values.reward_type === 'subscription') {
    return {
      ...base,
      reward: {
        type: 'subscription' as const,
        amount: '',
        unit: 'quota' as const,
        subscription_plan_id: values.subscription_plan_id,
      },
    }
  }
  return {
    ...base,
    reward: {
      type: 'quota' as const,
      amount: normalizeRewardAmount(values.reward_amount),
      unit: values.reward_unit,
      subscription_plan_id: 0,
    },
  }
}
