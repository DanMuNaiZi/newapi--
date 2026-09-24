/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

export type UsageRankingPeriod = 'today' | 'week' | 'month' | 'year'

export type UsageRankingRow = {
  rank: number
  username: string
  total_quota: number
  total_tokens: number
  request_count: number
  is_self: boolean
}

export type UsageRankingsSnapshot = {
  period: UsageRankingPeriod
  identity_visible: boolean
  display_count: number
  total_users: number
  total_quota: number
  total_tokens: number
  total_requests: number
  top_user?: UsageRankingRow
  my_rank?: UsageRankingRow
  users: UsageRankingRow[]
}

export type UsageRankingsResponse = {
  success: boolean
  message?: string
  data?: UsageRankingsSnapshot
}
