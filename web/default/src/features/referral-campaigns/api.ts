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

import { api } from '@/lib/api'

import type {
  ApiResponse,
  ReferralCampaign,
  ReferralCampaignEvent,
  ReferralCampaignEventPage,
  ReferralCampaignPayload,
  ReferralCampaignSelfView,
  RewardSubscriptionPlanOption,
} from './types'

export async function getReferralRewardPlans(): Promise<
  ApiResponse<RewardSubscriptionPlanOption[]>
> {
  const response = await api.get('/api/referral-campaign/admin/reward-plans')
  return response.data
}

export async function getReferralCampaignSelf(): Promise<
  ApiResponse<ReferralCampaignSelfView>
> {
  const response = await api.get('/api/referral-campaign/self')
  return response.data
}

export async function getAdminReferralCampaigns(): Promise<
  ApiResponse<ReferralCampaign[]>
> {
  const response = await api.get('/api/referral-campaign/admin/campaigns')
  return response.data
}

export async function createReferralCampaign(
  payload: ReferralCampaignPayload
): Promise<ApiResponse<ReferralCampaign>> {
  const response = await api.post(
    '/api/referral-campaign/admin/campaigns',
    payload
  )
  return response.data
}

export async function updateReferralCampaign(
  id: number,
  payload: ReferralCampaignPayload
): Promise<ApiResponse<ReferralCampaign>> {
  const response = await api.put(
    `/api/referral-campaign/admin/campaigns/${id}`,
    payload
  )
  return response.data
}

export async function getReferralCampaignEvents(
  id: number,
  page: number,
  pageSize: number
): Promise<ApiResponse<ReferralCampaignEventPage>> {
  const response = await api.get(
    `/api/referral-campaign/admin/campaigns/${id}/events`,
    { params: { page, page_size: pageSize } }
  )
  return response.data
}

export async function retryReferralCampaignReward(
  eventId: number
): Promise<ApiResponse<ReferralCampaignEvent>> {
  const response = await api.post(
    `/api/referral-campaign/admin/events/${eventId}/retry-reward`
  )
  return response.data
}
