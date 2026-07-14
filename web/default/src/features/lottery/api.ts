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
  LotteryParticipant,
  LotteryPlan,
  LotteryPlanCreatePayload,
  LotteryPrize,
  LotteryResult,
} from './types'

export async function getLotteryPlansForSelf(): Promise<
  ApiResponse<LotteryPlan[]>
> {
  const response = await api.get('/api/lottery/self')
  return response.data
}

export async function joinLotteryPlan(id: number): Promise<ApiResponse> {
  const response = await api.post(`/api/lottery/${id}/join`)
  return response.data
}

export async function leaveLotteryPlan(id: number): Promise<ApiResponse> {
  const response = await api.post(`/api/lottery/${id}/leave`)
  return response.data
}

export async function claimLotteryResult(id: number): Promise<ApiResponse> {
  const response = await api.post(`/api/lottery/results/${id}/claim`)
  return response.data
}

export async function getLotteryResultsForSelf(): Promise<
  ApiResponse<LotteryResult[]>
> {
  const response = await api.get('/api/lottery/results/self')
  return response.data
}

export async function getAdminLotteryPlans(): Promise<
  ApiResponse<LotteryPlan[]>
> {
  const response = await api.get('/api/lottery/admin/plans')
  return response.data
}

export async function createLotteryPlan(
  payload: LotteryPlanCreatePayload
): Promise<ApiResponse<LotteryPlan>> {
  const response = await api.post('/api/lottery/admin/plans', payload)
  return response.data
}

export async function getLotteryPrizes(
  planId: number
): Promise<ApiResponse<LotteryPrize[]>> {
  const response = await api.get(`/api/lottery/admin/plans/${planId}/prizes`)
  return response.data
}

export async function getLotteryParticipants(
  planId: number
): Promise<ApiResponse<LotteryParticipant[]>> {
  const response = await api.get(
    `/api/lottery/admin/plans/${planId}/participants`
  )
  return response.data
}

export async function updateLotteryParticipant(payload: {
  planId: number
  userId: number
  weight?: number
  presetPrizeId?: number
}): Promise<ApiResponse> {
  const response = await api.put(
    `/api/lottery/admin/plans/${payload.planId}/participants`,
    {
      user_id: payload.userId,
      weight: payload.weight,
      preset_prize_id: payload.presetPrizeId,
    }
  )
  return response.data
}

export async function drawLotteryPlan(
  planId: number,
  reason: string
): Promise<ApiResponse> {
  const response = await api.post(`/api/lottery/admin/plans/${planId}/draw`, {
    reason,
  })
  return response.data
}
