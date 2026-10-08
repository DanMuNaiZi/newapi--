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
  PublicPoolContribution,
  PublicPoolContributionPage,
  PublicPoolSite,
  PublicPoolSitePayload,
  PublicPoolStatus,
  RewardSubscriptionPlanOption,
} from './types'

export function getPublicPoolStatus(): Promise<ApiResponse<PublicPoolStatus>> {
  return api.get('/api/public-pool/status').then((response) => response.data)
}

export function getPublicPoolSites(): Promise<ApiResponse<PublicPoolSite[]>> {
  return api.get('/api/public-pool/sites').then((response) => response.data)
}

export function getPublicPoolContributions(): Promise<
  ApiResponse<PublicPoolContribution[]>
> {
  return api
    .get('/api/public-pool/contributions/self')
    .then((response) => response.data)
}

export function createPublicPoolContribution(payload: {
  site_id: number
  description: string
  proof: string
}): Promise<ApiResponse<PublicPoolContribution>> {
  return api
    .post('/api/public-pool/contributions', payload)
    .then((response) => response.data)
}

export function getAdminPublicPoolSites(): Promise<
  ApiResponse<PublicPoolSite[]>
> {
  return api
    .get('/api/public-pool/admin/sites')
    .then((response) => response.data)
}

export function getAdminPublicPoolRewardPlans(): Promise<
  ApiResponse<RewardSubscriptionPlanOption[]>
> {
  return api
    .get('/api/public-pool/admin/reward-plans')
    .then((response) => response.data)
}

export function createAdminPublicPoolSite(
  payload: PublicPoolSitePayload
): Promise<ApiResponse<PublicPoolSite>> {
  return api
    .post('/api/public-pool/admin/sites', payload)
    .then((response) => response.data)
}

export function updateAdminPublicPoolSite(
  id: number,
  payload: PublicPoolSitePayload
): Promise<ApiResponse<PublicPoolSite>> {
  return api
    .put(`/api/public-pool/admin/sites/${id}`, payload)
    .then((response) => response.data)
}

export function deleteAdminPublicPoolSite(
  id: number
): Promise<ApiResponse<null>> {
  return api
    .delete(`/api/public-pool/admin/sites/${id}`)
    .then((response) => response.data)
}

export function getAdminPublicPoolContributions(params: {
  search?: string
  status?: string
  page?: number
  page_size?: number
}): Promise<ApiResponse<PublicPoolContributionPage>> {
  return api
    .get('/api/public-pool/admin/contributions', { params })
    .then((response) => response.data)
}

export function reviewAdminPublicPoolContribution(
  id: number,
  payload: { status: 'approved' | 'rejected'; review_note: string }
): Promise<ApiResponse<PublicPoolContribution>> {
  return api
    .post(`/api/public-pool/admin/contributions/${id}/review`, payload)
    .then((response) => response.data)
}

export function retryAdminPublicPoolContributionReward(
  id: number
): Promise<ApiResponse<PublicPoolContribution>> {
  return api
    .post(`/api/public-pool/admin/contributions/${id}/retry-reward`)
    .then((response) => response.data)
}
