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

export type PublicPoolSiteStatus = 'enabled' | 'disabled'
export type PublicPoolContributionStatus = 'pending' | 'approved' | 'rejected'
export type RewardGrantStatus = 'pending' | 'succeeded' | 'failed'

export interface RewardSubscriptionPlanOption {
  id: number
  title: string
}

export interface RewardSpec {
  type: 'quota' | 'subscription'
  amount?: string
  unit?: 'usd' | 'cny' | 'quota'
  quota?: number
  quota_per_unit?: string
  usd_exchange_rate?: string
  subscription_plan_id?: number
  subscription_plan_title?: string
}

export interface PublicPoolSite {
  id: number
  name: string
  url: string
  description: string
  status: PublicPoolSiteStatus
  sort_order: number
  reward?: RewardSpec
  created_at: number
  updated_at: number
}

export interface PublicPoolContribution {
  id: number
  user_id: number
  site_id: number
  description: string
  proof: string
  status: PublicPoolContributionStatus
  reviewer_id: number
  review_note: string
  reviewed_at: number
  created_at: number
  updated_at: number
  username?: string
  site_name?: string
  reward?: RewardSpec
  reward_grant_id?: number
  reward_status?: RewardGrantStatus
  reward_failure_reason?: string
}

export interface PublicPoolStatus {
  enabled: boolean
  available: boolean
  group_ratio: number
  reason?: string
  channel_count: number
}

export interface PublicPoolContributionPayload {
  site_id: number
  description: string
  proof: string
}

export interface PublicPoolSitePayload {
  name: string
  url: string
  description: string
  status: PublicPoolSiteStatus
  sort_order: number
  reward: {
    type: 'quota' | 'subscription'
    amount: string
    unit: 'usd' | 'cny' | 'quota'
    subscription_plan_id: number
  } | null
}

export interface PublicPoolReviewPayload {
  status: Exclude<PublicPoolContributionStatus, 'pending'>
  review_note: string
}

export interface PublicPoolContributionPage {
  items: PublicPoolContribution[]
  total: number
  page: number
  page_size: number
}

export interface ApiResponse<T> {
  success: boolean
  message?: string
  data?: T
}
