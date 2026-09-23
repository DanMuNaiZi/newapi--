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
import type { RedemptionResponse } from '../types'

export type RedemptionOutcome =
  | { type: 'quota'; quota: number }
  | {
      type: 'subscription'
      subscriptionPlanId: number
      subscriptionId: number
    }

export function parseRedemptionOutcome(
  response: RedemptionResponse
): RedemptionOutcome | null {
  if (!response.success) return null
  if (
    response.reward_type === 'subscription' ||
    (response.reward_type === undefined && response.data === 0)
  ) {
    return {
      type: 'subscription',
      subscriptionPlanId: response.subscription_plan_id || 0,
      subscriptionId: response.subscription_id || 0,
    }
  }
  return { type: 'quota', quota: response.data || 0 }
}
