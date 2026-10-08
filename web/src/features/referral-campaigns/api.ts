/* Copyright (C) 2023-2026 QuantumNous */
import { api } from '@/lib/api'

import type {
  ApiResponse,
  ReferralCampaign,
  ReferralCampaignEvent,
  ReferralCampaignEventPage,
  ReferralCampaignPayload,
  ReferralCampaignSelfView,
  ReferralEventReview,
  RewardSubscriptionPlanOption,
} from './types'

export const getReferralCampaignSelf = (): Promise<
  ApiResponse<ReferralCampaignSelfView>
> => api.get('/api/referral-campaign/self').then((r) => r.data)
export const getReferralRewardPlans = (): Promise<
  ApiResponse<RewardSubscriptionPlanOption[]>
> => api.get('/api/referral-campaign/admin/reward-plans').then((r) => r.data)
export const getAdminReferralCampaigns = (): Promise<
  ApiResponse<ReferralCampaign[]>
> => api.get('/api/referral-campaign/admin/campaigns').then((r) => r.data)
export const createReferralCampaign = (
  payload: ReferralCampaignPayload
): Promise<ApiResponse<ReferralCampaign>> =>
  api
    .post('/api/referral-campaign/admin/campaigns', payload)
    .then((r) => r.data)
export const updateReferralCampaign = (
  id: number,
  payload: ReferralCampaignPayload
): Promise<ApiResponse<ReferralCampaign>> =>
  api
    .put(`/api/referral-campaign/admin/campaigns/${id}`, payload)
    .then((r) => r.data)
export const getReferralCampaignEvents = (
  id: number,
  page = 1,
  pageSize = 50
): Promise<ApiResponse<ReferralCampaignEventPage>> =>
  api
    .get(`/api/referral-campaign/admin/campaigns/${id}/events`, {
      params: { page, page_size: pageSize },
    })
    .then((r) => r.data)
export const retryReferralCampaignReward = (
  id: number
): Promise<ApiResponse<ReferralCampaignEvent>> =>
  api
    .post(`/api/referral-campaign/admin/events/${id}/retry-reward`)
    .then((r) => r.data)
export const getReferralEventReview = (
  id: number
): Promise<ApiResponse<ReferralEventReview>> =>
  api
    .get(`/api/referral-campaign/admin/events/${id}/review`)
    .then((r) => r.data)
export const reviewReferralEvent = (
  id: number,
  decision: 'approve' | 'reject',
  remark: string
): Promise<ApiResponse<ReferralCampaignEvent>> =>
  api
    .post(`/api/referral-campaign/admin/events/${id}/review`, {
      decision,
      remark,
    })
    .then((r) => r.data)
