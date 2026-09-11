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
    title: z
      .string()
      .trim()
      .min(1, 'Campaign title is required')
      .max(128, 'Campaign title cannot exceed 128 characters'),
    description: z
      .string()
      .trim()
      .max(4000, 'Description cannot exceed 4000 characters'),
    enabled: z.boolean(),
    start: z.string().min(1, invalidCampaignMessage),
    end: z.string().min(1, invalidCampaignMessage),
    activation_hours: z
      .number({
        error: 'Activation window must be between 1 minute and 365 days',
      })
      .finite('Activation window must be between 1 minute and 365 days')
      .min(1 / 60, 'Activation window must be between 1 minute and 365 days')
      .max(365 * 24, 'Activation window must be between 1 minute and 365 days'),
    max_rewards_per_inviter: z
      .number({
        error: 'Reward count must be an integer between 0 and 2147483647',
      })
      .int('Reward count must be an integer between 0 and 2147483647')
      .min(0, 'Reward count must be an integer between 0 and 2147483647')
      .max(
        2_147_483_647,
        'Reward count must be an integer between 0 and 2147483647'
      ),
    total_reward_limit: z
      .number({
        error: 'Reward count must be an integer between 0 and 2147483647',
      })
      .int('Reward count must be an integer between 0 and 2147483647')
      .min(0, 'Reward count must be an integer between 0 and 2147483647')
      .max(
        2_147_483_647,
        'Reward count must be an integer between 0 and 2147483647'
      ),
    replace_rewards: z.boolean(),
    inviter_reward_usd: z
      .string()
      .trim()
      .regex(
        /^(0|[1-9][0-9]{0,9})(\.[0-9]{1,12})?$/,
        'Enter a non-negative USD amount with at most 12 decimal places'
      ),
    invitee_reward_usd: z
      .string()
      .trim()
      .regex(
        /^(0|[1-9][0-9]{0,9})(\.[0-9]{1,12})?$/,
        'Enter a non-negative USD amount with at most 12 decimal places'
      ),
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
    replace_rewards: true,
    inviter_reward_usd: '1',
    invitee_reward_usd: '0',
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
    replace_rewards: false,
    inviter_reward_usd:
      campaign.reward?.unit === 'usd' ? campaign.reward.amount || '0' : '0',
    invitee_reward_usd:
      campaign.invitee_reward?.unit === 'usd'
        ? campaign.invitee_reward.amount || '0'
        : '0',
  }
}

export function buildReferralCampaignPayload(
  values: ReferralCampaignFormValues,
  campaign?: ReferralCampaign | null
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
    ...(campaign && !form.replace_rewards
      ? { preserve_reward: true }
      : {
          inviter_reward_usd: form.inviter_reward_usd,
          invitee_reward_usd: form.invitee_reward_usd,
        }),
  }
}
