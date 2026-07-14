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

export type LotteryPlanStatus =
  | 'draft'
  | 'scheduled'
  | 'open'
  | 'drawing'
  | 'finished'
  | 'cancelled'

export interface LotteryPlan {
  id: number
  title: string
  description: string
  status: LotteryPlanStatus
  eligibility_mode: 'all' | 'groups' | 'users'
  max_participants: number
  registration_start_time: number
  draw_time: number
}

export interface LotteryResult {
  id: number
  plan_id: number
  prize_id: number
  reward_type: 'quota' | 'subscription'
  quota: number
  fulfillment_mode: 'auto' | 'self_claim' | 'redemption_code'
  fulfillment_status: string
  claim_expires_at: number
}

export interface LotteryPrize {
  id: number
  plan_id: number
  name: string
  quantity: number
  reward_type: 'quota' | 'subscription'
  quota: number
  subscription_plan_id: number
  fulfillment_mode: 'auto' | 'self_claim' | 'redemption_code'
  claim_expire_seconds: number
}

export interface LotteryParticipant {
  id: number
  plan_id: number
  user_id: number
  username: string
  display_name: string
  user_group: string
  weight: number
  preset_prize_id: number
  status: 'joined' | 'left'
}

export interface LotteryPlanCreatePayload {
  title: string
  description: string
  status: 'draft' | 'scheduled' | 'open'
  eligibility_mode: 'all' | 'groups' | 'users'
  max_participants: number
  registration_start_time: number
  draw_time: number
  user_ids: number[]
  groups: string[]
  prizes: Array<{
    name: string
    quantity: number
    reward_type: 'quota' | 'subscription'
    quota: number
    subscription_plan_id: number
    fulfillment_mode: 'auto' | 'self_claim' | 'redemption_code'
    claim_expire_seconds: number
  }>
}

export interface ApiResponse<T = undefined> {
  success: boolean
  message: string
  data: T
}
