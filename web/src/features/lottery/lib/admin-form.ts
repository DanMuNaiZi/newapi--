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

import type { LotteryPlanCreatePayload } from '../types'

export const LOTTERY_DATABASE_INT_MAX = 2_147_483_647

const prizeSchema = z
  .object({
    name: z.string().trim().min(1, 'Prize name is required'),
    quantity: z
      .number({ error: 'Prize quantity must be at least 1' })
      .int()
      .min(1, 'Prize quantity must be at least 1')
      .max(LOTTERY_DATABASE_INT_MAX, 'Prize quantity cannot exceed 2147483647'),
    reward_type: z.enum(['quota', 'subscription']),
    quota: z
      .number({ error: 'Quota reward must be greater than 0' })
      .int()
      .min(0)
      .max(LOTTERY_DATABASE_INT_MAX, 'Quota reward cannot exceed 2147483647'),
    subscription_plan_id: z
      .number({ error: 'Please select a subscription plan' })
      .int()
      .min(0),
    fulfillment_mode: z.enum(['auto', 'self_claim', 'redemption_code']),
    claim_expire_days: z
      .number({ error: 'Claim expiry cannot be negative' })
      .int()
      .min(0, 'Claim expiry cannot be negative')
      .max(3650, 'Claim expiry cannot exceed 3650 days'),
  })
  .superRefine((prize, context) => {
    if (prize.reward_type === 'quota' && prize.quota <= 0) {
      context.addIssue({
        code: 'custom',
        message: 'Quota reward must be greater than 0',
        path: ['quota'],
      })
    }
    if (
      prize.reward_type === 'subscription' &&
      prize.subscription_plan_id <= 0
    ) {
      context.addIssue({
        code: 'custom',
        message: 'Please select a subscription plan',
        path: ['subscription_plan_id'],
      })
    }
  })

export const lotteryAdminFormSchema = z
  .object({
    title: z.string().trim().min(1, 'Lottery title is required'),
    icon: z
      .string()
      .trim()
      .max(1024)
      .refine(
        (value) => !value || /^https?:\/\//i.test(value),
        'Icon URL must start with http:// or https://'
      ),
    description: z.string(),
    eligibility_mode: z.enum(['all', 'groups', 'users']),
    selected_groups: z.array(z.string()),
    selected_user_ids: z.array(z.number().int().positive()),
    max_participants: z
      .number({ error: 'Maximum participants must be at least 1' })
      .int()
      .min(1, 'Maximum participants must be at least 1')
      .max(
        LOTTERY_DATABASE_INT_MAX,
        'Maximum participants cannot exceed 2147483647'
      ),
    registration_start: z
      .string()
      .min(1, 'Registration start time is required'),
    draw_time: z.string().min(1, 'Draw time is required'),
    prizes: z.array(prizeSchema).min(1, 'At least one prize is required'),
  })
  .superRefine((values, context) => {
    if (
      values.eligibility_mode === 'groups' &&
      values.selected_groups.length === 0
    ) {
      context.addIssue({
        code: 'custom',
        message: 'Please select at least one group',
        path: ['selected_groups'],
      })
    }
    if (
      values.eligibility_mode === 'users' &&
      values.selected_user_ids.length === 0
    ) {
      context.addIssue({
        code: 'custom',
        message: 'Please select at least one user',
        path: ['selected_user_ids'],
      })
    }

    const registrationStart = new Date(values.registration_start).getTime()
    const drawTime = new Date(values.draw_time).getTime()
    if (
      Number.isFinite(registrationStart) &&
      Number.isFinite(drawTime) &&
      drawTime <= registrationStart
    ) {
      context.addIssue({
        code: 'custom',
        message: 'Draw time must be later than registration start time',
        path: ['draw_time'],
      })
    }
  })

export type LotteryAdminFormValues = z.infer<typeof lotteryAdminFormSchema>

export const DEFAULT_LOTTERY_PRIZE: LotteryAdminFormValues['prizes'][number] = {
  name: '',
  quantity: 1,
  reward_type: 'quota',
  quota: 100,
  subscription_plan_id: 0,
  fulfillment_mode: 'auto',
  claim_expire_days: 7,
}

export const DEFAULT_LOTTERY_FORM: LotteryAdminFormValues = {
  title: '',
  icon: '',
  description: '',
  eligibility_mode: 'all',
  selected_groups: [],
  selected_user_ids: [],
  max_participants: 10,
  registration_start: '',
  draw_time: '',
  prizes: [DEFAULT_LOTTERY_PRIZE],
}

export function unixFromLocal(value: string): number {
  return Math.floor(new Date(value).getTime() / 1000)
}

export function localInputFromUnix(value: number): string {
  const date = new Date(value * 1000)
  date.setMinutes(date.getMinutes() - date.getTimezoneOffset())
  return date.toISOString().slice(0, 16)
}

export function createLotteryFormDefaults(
  referenceDate = new Date()
): LotteryAdminFormValues {
  const registrationStart = new Date(referenceDate.getTime() + 5 * 60 * 1000)
  const drawTime = new Date(registrationStart.getTime() + 60 * 60 * 1000)
  return {
    ...DEFAULT_LOTTERY_FORM,
    registration_start: localInputFromUnix(
      Math.floor(registrationStart.getTime() / 1000)
    ),
    draw_time: localInputFromUnix(Math.floor(drawTime.getTime() / 1000)),
    prizes: [{ ...DEFAULT_LOTTERY_PRIZE }],
  }
}

export function buildLotteryPlanPayload(
  values: LotteryAdminFormValues
): LotteryPlanCreatePayload {
  return {
    title: values.title.trim(),
    icon: values.icon.trim(),
    description: values.description.trim(),
    status: 'scheduled',
    eligibility_mode: values.eligibility_mode,
    max_participants: values.max_participants,
    registration_start_time: unixFromLocal(values.registration_start),
    draw_time: unixFromLocal(values.draw_time),
    user_ids:
      values.eligibility_mode === 'users' ? values.selected_user_ids : [],
    groups: values.eligibility_mode === 'groups' ? values.selected_groups : [],
    prizes: values.prizes.map((prize) => ({
      name: prize.name.trim(),
      quantity: prize.quantity,
      reward_type: prize.reward_type,
      quota: prize.reward_type === 'quota' ? prize.quota : 0,
      subscription_plan_id:
        prize.reward_type === 'subscription' ? prize.subscription_plan_id : 0,
      fulfillment_mode: prize.fulfillment_mode,
      claim_expire_seconds: prize.claim_expire_days * 86400,
    })),
  }
}
