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
export type RewardType = 'quota' | 'subscription'
export type RewardUnit = 'usd' | 'quota'
/** Legacy snapshots may still contain CNY; forms never submit it. */
export type StoredRewardUnit = RewardUnit | 'cny'

export interface RewardSubscriptionPlanOption {
  id: number
  title: string
}

export interface RewardSnapshot {
  type: RewardType
  amount?: string
  unit?: StoredRewardUnit
  quota?: number
  subscription_plan_id?: number
  subscription_plan_title?: string
}

export interface ReferralCampaign {
  id: number
  title: string
  description: string
  enabled: boolean
  start_time: number
  end_time: number
  activation_window_seconds: number
  max_rewards_per_inviter: number
  total_reward_limit: number
  reward?: RewardSnapshot
  registration_count: number
  activated_count: number
  rewarded_count: number
  failed_reward_count: number
}

export interface ReferralCampaignEvent {
  id: number
  campaign_id: number
  inviter_user_id: number
  invitee_user_id: number
  inviter_username?: string
  invitee_username?: string
  status: string
  registered_at: number
  activation_deadline: number
  activated_at: number
  reward_status?: string
  reward_failure_reason?: string
}

export interface ReferralCampaignSelfView {
  campaign: ReferralCampaign | null
  events: ReferralCampaignEvent[]
}

export interface ReferralCampaignPayload {
  preserve_reward?: boolean
  title: string
  description: string
  enabled: boolean
  start_time: number
  end_time: number
  activation_window_seconds: number
  max_rewards_per_inviter: number
  total_reward_limit: number
  reward: {
    type: RewardType
    amount: string
    unit: RewardUnit
    subscription_plan_id: number
  }
}

export interface ApiResponse<T> {
  success: boolean
  message?: string
  data?: T
}

export interface ReferralCampaignEventPage {
  items: ReferralCampaignEvent[]
  total: number
  page: number
  page_size: number
}
