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

import dayjs from 'dayjs'
import { z } from 'zod'

import type { ReferralCampaign, ReferralCampaignPayload } from '../types'

const invalidCampaignMessage = 'Please enter a valid campaign and time range'

export const referralCampaignFormSchema = z
  .object({
    title: z.string().trim().min(1, invalidCampaignMessage).max(128),
    description: z.string().trim().max(4000),
    enabled: z.boolean(),
    start: z.string().min(1, invalidCampaignMessage),
    end: z.string().min(1, invalidCampaignMessage),
    activation_hours: z
      .number()
      .finite()
      .min(1 / 60, invalidCampaignMessage)
      .max(365 * 24, invalidCampaignMessage),
    max_rewards_per_inviter: z
      .number()
      .int()
      .min(0, invalidCampaignMessage)
      .max(2_147_483_647, invalidCampaignMessage),
    total_reward_limit: z
      .number()
      .int()
      .min(0, invalidCampaignMessage)
      .max(2_147_483_647, invalidCampaignMessage),
    reward_type: z.enum(['quota', 'subscription']),
    reward_amount: z.string(),
    reward_unit: z.enum(['usd', 'cny', 'quota']),
    subscription_plan_id: z.number().int().min(0),
  })
  .superRefine((values, context) => {
    const startTime = dayjs(values.start)
    const endTime = dayjs(values.end)
    if (
      !startTime.isValid() ||
      !endTime.isValid() ||
      !endTime.isAfter(startTime)
    ) {
      context.addIssue({
        code: 'custom',
        path: ['end'],
        message: invalidCampaignMessage,
      })
    }

    if (values.reward_type === 'subscription') {
      if (values.subscription_plan_id <= 0) {
        context.addIssue({
          code: 'custom',
          path: ['subscription_plan_id'],
          message: 'Please select a subscription plan',
        })
      }
      return
    }

    const amount = values.reward_amount.trim()
    if (!/^\d+(?:\.\d+)?$/.test(amount) || Number(amount) <= 0) {
      context.addIssue({
        code: 'custom',
        path: ['reward_amount'],
        message: invalidCampaignMessage,
      })
      return
    }
    if (values.reward_unit === 'quota' && !/^\d+$/.test(amount)) {
      context.addIssue({
        code: 'custom',
        path: ['reward_amount'],
        message: invalidCampaignMessage,
      })
    }
  })

export type ReferralCampaignFormValues = z.infer<
  typeof referralCampaignFormSchema
>

export function emptyReferralCampaignForm(): ReferralCampaignFormValues {
  return {
    title: '',
    description: '',
    enabled: true,
    start: dayjs().add(5, 'minute').format('YYYY-MM-DDTHH:mm'),
    end: dayjs().add(7, 'day').format('YYYY-MM-DDTHH:mm'),
    activation_hours: 24,
    max_rewards_per_inviter: 0,
    total_reward_limit: 0,
    reward_type: 'quota',
    reward_amount: '1',
    reward_unit: 'usd',
    subscription_plan_id: 0,
  }
}

export function referralCampaignToForm(
  campaign: ReferralCampaign
): ReferralCampaignFormValues {
  return {
    title: campaign.title,
    description: campaign.description,
    enabled: campaign.enabled,
    start: dayjs.unix(campaign.start_time).format('YYYY-MM-DDTHH:mm'),
    end: dayjs.unix(campaign.end_time).format('YYYY-MM-DDTHH:mm'),
    activation_hours: campaign.activation_window_seconds / 3600,
    max_rewards_per_inviter: campaign.max_rewards_per_inviter,
    total_reward_limit: campaign.total_reward_limit,
    reward_type: campaign.reward?.type ?? 'quota',
    reward_amount: campaign.reward?.amount ?? '1',
    reward_unit: campaign.reward?.unit ?? 'usd',
    subscription_plan_id: campaign.reward?.subscription_plan_id ?? 0,
  }
}

export function buildReferralCampaignPayload(
  values: ReferralCampaignFormValues
): ReferralCampaignPayload {
  const form = referralCampaignFormSchema.parse(values)
  return {
    title: form.title.trim(),
    description: form.description.trim(),
    enabled: form.enabled,
    start_time: dayjs(form.start).unix(),
    end_time: dayjs(form.end).unix(),
    activation_window_seconds: Math.round(form.activation_hours * 3600),
    max_rewards_per_inviter: form.max_rewards_per_inviter,
    total_reward_limit: form.total_reward_limit,
    reward: {
      type: form.reward_type,
      amount: form.reward_type === 'quota' ? form.reward_amount.trim() : '',
      unit: form.reward_type === 'quota' ? form.reward_unit : 'quota',
      subscription_plan_id:
        form.reward_type === 'subscription' ? form.subscription_plan_id : 0,
    },
  }
}
