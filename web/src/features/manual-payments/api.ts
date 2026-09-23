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
  AdminManualWechatSubscriptionOrder,
  AdminManualWechatTopUpOrder,
  ApiResponse,
  ManualWechatOrder,
  ManualWechatPaymentInfo,
  ManualWechatPaymentSettings,
  PaginatedResponse,
} from './types'

export async function getManualWechatPaymentInfo(): Promise<
  ApiResponse<ManualWechatPaymentInfo>
> {
  const response = await api.get('/api/user/manual-wechat/info')
  return response.data
}

export async function createManualWechatTopUp(
  amount: number
): Promise<ApiResponse<ManualWechatOrder>> {
  const response = await api.post('/api/user/manual-wechat/pay', { amount })
  return response.data
}

export async function createManualWechatSubscription(
  planId: number
): Promise<ApiResponse<ManualWechatOrder>> {
  const response = await api.post('/api/subscription/manual-wechat/pay', {
    plan_id: planId,
  })
  return response.data
}

export async function getManualWechatPaymentSettings(): Promise<
  ApiResponse<ManualWechatPaymentSettings>
> {
  const response = await api.get('/api/option/manual-wechat-payment')
  return response.data
}

export async function updateManualWechatPaymentSettings(
  settings: ManualWechatPaymentSettings
): Promise<ApiResponse<ManualWechatPaymentSettings>> {
  const response = await api.put('/api/option/manual-wechat-payment', settings)
  return response.data
}

export interface ManualPaymentListParams {
  page: number
  pageSize: number
  status?: string
  userId?: number
  planId?: number
  keyword?: string
}

function manualPaymentSearchParams(
  params: ManualPaymentListParams
): URLSearchParams {
  const search = new URLSearchParams({
    page: String(params.page),
    page_size: String(params.pageSize),
    payment_provider: 'manual_wechat',
  })
  if (params.status) search.set('status', params.status)
  if (params.userId) search.set('user_id', String(params.userId))
  if (params.planId) search.set('plan_id', String(params.planId))
  if (params.keyword?.trim()) search.set('keyword', params.keyword.trim())
  return search
}

export async function getManualWechatTopUpOrders(
  params: ManualPaymentListParams
): Promise<ApiResponse<PaginatedResponse<AdminManualWechatTopUpOrder>>> {
  const search = manualPaymentSearchParams(params)
  const response = await api.get(`/api/user/topup?${search.toString()}`)
  return response.data
}

export async function getManualWechatSubscriptionOrders(
  params: ManualPaymentListParams
): Promise<ApiResponse<PaginatedResponse<AdminManualWechatSubscriptionOrder>>> {
  const search = manualPaymentSearchParams(params)
  const response = await api.get(
    `/api/subscription/admin/orders?${search.toString()}`
  )
  return response.data
}

export async function completeManualWechatTopUp(
  tradeNo: string
): Promise<ApiResponse> {
  const response = await api.post('/api/user/topup/manual-wechat/complete', {
    trade_no: tradeNo,
  })
  return response.data
}

export async function expireManualWechatTopUp(
  tradeNo: string
): Promise<ApiResponse> {
  const response = await api.post('/api/user/topup/manual-wechat/expire', {
    trade_no: tradeNo,
  })
  return response.data
}

export async function completeManualWechatSubscription(
  tradeNo: string
): Promise<ApiResponse> {
  const response = await api.post(
    `/api/subscription/admin/orders/${encodeURIComponent(tradeNo)}/complete`
  )
  return response.data
}

export async function expireManualWechatSubscription(
  tradeNo: string
): Promise<ApiResponse> {
  const response = await api.post(
    `/api/subscription/admin/orders/${encodeURIComponent(tradeNo)}/expire`
  )
  return response.data
}
