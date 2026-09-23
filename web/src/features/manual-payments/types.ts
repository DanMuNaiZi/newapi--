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
export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}

export interface ManualWechatPaymentInfo {
  enabled: boolean
  display_name: string
  expires_at: number
}

export interface ManualWechatPaymentSettings {
  enabled: boolean
  qr_code_image: string
  expires_at: number
  instructions: string
}

export type ManualWechatOrderStatus = 'pending' | 'success' | 'expired'

export type ManualWechatOrderPurpose =
  | { topup_amount: number }
  | { plan_id: number; plan_title: string }

export interface ManualWechatOrder {
  kind: 'topup' | 'subscription'
  trade_no: string
  user_id: number
  amount_cny: number
  currency: 'CNY'
  status: ManualWechatOrderStatus
  created_at: number
  reused: boolean
  purpose: ManualWechatOrderPurpose
  qr_code_image: string
  qr_code_expires_at: number
  instructions: string
}

export interface AdminManualWechatTopUpOrder {
  id: number
  user_id: number
  username: string
  amount: number
  money: number
  trade_no: string
  payment_method: string
  payment_provider: string
  status: ManualWechatOrderStatus
  create_time: number
  complete_time: number
}

export interface AdminManualWechatSubscriptionOrder {
  id: number
  user_id: number
  username: string
  plan_id: number
  plan_title: string
  money: number
  trade_no: string
  payment_method: string
  payment_provider: string
  status: ManualWechatOrderStatus
  create_time: number
  complete_time: number
}

export interface PaginatedResponse<T> {
  page: number
  page_size: number
  total: number
  items: T[]
}
