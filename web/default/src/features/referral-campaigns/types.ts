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
  invitee_reward?: RewardSnapshot
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
  qualified_quota: number
  qualified_at: number
  review_decision?: string
  reviewed_by?: number
  reviewed_at?: number
  review_remark?: string
  reward?: RewardSnapshot
  invitee_reward?: RewardSnapshot
}

export interface ReferralCampaignSelfView {
  campaign: ReferralCampaign | null
  events: Pick<
    ReferralCampaignEvent,
    | 'id'
    | 'campaign_id'
    | 'invitee_username'
    | 'status'
    | 'registered_at'
    | 'activation_deadline'
    | 'activated_at'
    | 'review_decision'
    | 'reward_status'
    | 'reward'
    | 'invitee_reward'
  >[]
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
  inviter_reward_usd?: string
  invitee_reward_usd?: string
  reward?: {
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

export interface ReferralReviewUser {
  id: number
  username: string
  created_at: number
  github_id: string
  github_created_at: number
  github_age_exempt: boolean
}

export interface ReferralEventReview {
  event: ReferralCampaignEvent
  inviter: ReferralReviewUser
  invitee: ReferralReviewUser
  has_consumption: boolean
  can_approve: boolean
  usage: {
    requests: number
    consumed_quota: number
    logs_available: boolean
    period_start: number
    period_end: number
    recent_calls: {
      id: number
      created_at: number
      request_id: string
      model_name: string
      quota: number
      prompt_tokens: number
      completion_tokens: number
      successful: boolean
      qualifying: boolean
      public_pool: boolean
    }[]
  }
  related_events: ReferralCampaignEvent[]
  reward_grants: {
    id: number
    recipient_user_id: number
    status: string
    quota: number
    failure_reason: string
  }[]
}
