import { api } from '@/lib/api'

import type {
  ApiResponse,
  PublicPoolContribution,
  PublicPoolContributionPayload,
  PublicPoolReviewPayload,
  PublicPoolSite,
  PublicPoolSitePayload,
  PublicPoolStatus,
} from './types'

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

export async function getAdminPublicPoolContributions(): Promise<
  ApiResponse<PublicPoolContribution[]>
> {
  const response = await api.get('/api/public-pool/admin/contributions')
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
