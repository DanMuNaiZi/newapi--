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
    name: z.string().trim().min(1, 'Site name is required').max(128),
    url: z
      .string()
      .trim()
      .max(1024)
      .refine(
        isCredentialFreeHttpUrl,
        'Enter an HTTP or HTTPS URL without credentials'
      ),
    description: z.string().trim().max(4000, 'Description is too long'),
    status: z.enum(['enabled', 'disabled']),
    sort_order: z
      .number()
      .int()
      .min(-2147483648, 'Sort order is outside the supported range')
      .max(2147483647, 'Sort order is outside the supported range'),
    reward_type: z.enum(['none', 'quota', 'subscription']),
    reward_amount: z.string().trim(),
    reward_unit: z.enum(['usd', 'cny', 'quota']),
    subscription_plan_id: z.number().int().min(0),
  })
  .superRefine((value, context) => {
    if (value.reward_type === 'quota') {
      const amount = value.reward_amount.trim()
      if (!/^\d+(?:\.\d+)?$/.test(amount) || Number(amount) <= 0) {
        context.addIssue({
          code: 'custom',
          path: ['reward_amount'],
          message: 'Reward amount must be positive',
        })
      } else if (value.reward_unit === 'quota' && !/^\d+$/.test(amount)) {
        context.addIssue({
          code: 'custom',
          path: ['reward_amount'],
          message: 'Raw reward quota must be an integer',
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

export function toPublicPoolSitePayload(values: PublicPoolSiteFormValues) {
  const base = {
    name: values.name,
    url: values.url,
    description: values.description,
    status: values.status,
    sort_order: values.sort_order,
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
      amount: values.reward_amount,
      unit: values.reward_unit,
      subscription_plan_id: 0,
    },
  }
}
