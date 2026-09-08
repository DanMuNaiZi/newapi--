export type PublicPoolSiteStatus = 'enabled' | 'disabled'
export type PublicPoolContributionStatus = 'pending' | 'approved' | 'rejected'

export interface PublicPoolSite {
  id: number
  name: string
  url: string
  description: string
  status: PublicPoolSiteStatus
  sort_order: number
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
}

export interface PublicPoolReviewPayload {
  status: Exclude<PublicPoolContributionStatus, 'pending'>
  review_note: string
}

export interface ApiResponse<T> {
  success: boolean
  message?: string
  data?: T
}
