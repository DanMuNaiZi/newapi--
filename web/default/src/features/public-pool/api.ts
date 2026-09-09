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
  PublicPoolContributionPayload,
  PublicPoolReviewPayload,
  RewardSubscriptionPlanOption,
  PublicPoolSite,
  PublicPoolSitePayload,
  PublicPoolStatus,
} from './types'

export async function getAdminPublicPoolRewardPlans(): Promise<
  ApiResponse<RewardSubscriptionPlanOption[]>
> {
  const response = await api.get('/api/public-pool/admin/reward-plans')
  return response.data
}

export async function getPublicPoolStatus(): Promise<
  ApiResponse<PublicPoolStatus>
> {
  const response = await api.get('/api/public-pool/status')
  return response.data
}

export async function getPublicPoolSites(): Promise<
  ApiResponse<PublicPoolSite[]>
> {
  const response = await api.get('/api/public-pool/sites')
  return response.data
}

export async function getPublicPoolContributions(): Promise<
  ApiResponse<PublicPoolContribution[]>
> {
  const response = await api.get('/api/public-pool/contributions/self')
  return response.data
}

export async function createPublicPoolContribution(
  payload: PublicPoolContributionPayload
): Promise<ApiResponse<PublicPoolContribution>> {
  const response = await api.post('/api/public-pool/contributions', payload)
  return response.data
}

export async function getAdminPublicPoolSites(): Promise<
  ApiResponse<PublicPoolSite[]>
> {
  const response = await api.get('/api/public-pool/admin/sites')
  return response.data
}

export async function createAdminPublicPoolSite(
  payload: PublicPoolSitePayload
): Promise<ApiResponse<PublicPoolSite>> {
  const response = await api.post('/api/public-pool/admin/sites', payload)
  return response.data
}

export async function updateAdminPublicPoolSite(
  id: number,
  payload: PublicPoolSitePayload
): Promise<ApiResponse<PublicPoolSite>> {
  const response = await api.put(`/api/public-pool/admin/sites/${id}`, payload)
  return response.data
}

export async function deleteAdminPublicPoolSite(
  id: number
): Promise<ApiResponse<null>> {
  const response = await api.delete(`/api/public-pool/admin/sites/${id}`)
  return response.data
}

export async function getAdminPublicPoolContributions(params: {
  search?: string
  status?: string
  page?: number
  page_size?: number
}): Promise<ApiResponse<PublicPoolContributionPage>> {
  const response = await api.get('/api/public-pool/admin/contributions', {
    params,
  })
  return response.data
}

export async function reviewAdminPublicPoolContribution(
  id: number,
  payload: PublicPoolReviewPayload
): Promise<ApiResponse<PublicPoolContribution>> {
  const response = await api.post(
    `/api/public-pool/admin/contributions/${id}/review`,
    payload
  )
  return response.data
}

export async function retryAdminPublicPoolContributionReward(
  id: number
): Promise<ApiResponse<PublicPoolContribution>> {
  const response = await api.post(
    `/api/public-pool/admin/contributions/${id}/retry-reward`
  )
  return response.data
}
