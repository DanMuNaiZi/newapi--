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
import Decimal from 'decimal.js-light'

// Keep multiplication exact enough to decide the integer quota boundary.
// Truncating intermediate digits cannot round a value below .5 up to .5.
const RewardDecimal = Decimal.clone({
  precision: 80,
  rounding: Decimal.ROUND_DOWN,
})

export const MAX_REWARD_QUOTA = 2_147_483_647
export type RewardUnit = 'usd' | 'quota'

export interface RewardEquivalent {
  usd: string
  quota: number
}

/** Format the persisted platform quota without applying the user's currency display mode. */
export function formatPlatformQuota(quota: number | null | undefined): string {
  if (quota == null || !Number.isFinite(quota)) return '-'
  return Math.trunc(quota).toLocaleString()
}

/** Convert a form amount to the persisted quota and expose its USD value. */
export function getRewardEquivalent(
  amount: string | number,
  unit: RewardUnit,
  quotaPerUnit: number
): RewardEquivalent | null {
  const text = String(amount).trim()
  if (
    getRewardAmountError(text, unit) ||
    !isValidRewardQuotaPerUnit(quotaPerUnit)
  ) {
    return null
  }
  const parsed = new RewardDecimal(text)
  const rawQuota = unit === 'usd' ? parsed.times(String(quotaPerUnit)) : parsed
  const quota = rawQuota.toDecimalPlaces(0, Decimal.ROUND_HALF_UP)
  if (quota.lt(1) || quota.gt(MAX_REWARD_QUOTA)) {
    return null
  }
  return {
    usd: quota.div(String(quotaPerUnit)).toSignificantDigits(12).toString(),
    quota: quota.toNumber(),
  }
}

export function isValidRewardQuotaPerUnit(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value) && value > 0
}

/** Validate the input independently of configuration (also used by form schemas). */
export function getRewardAmountError(
  amount: string,
  unit: RewardUnit
): string | null {
  const text = amount.trim()
  if (!/^(?:\d+(?:\.\d*)?|\.\d+)$/.test(text)) {
    return 'Enter a positive decimal reward amount'
  }
  const parsed = new RewardDecimal(text)
  if (!parsed.gt(0)) return 'Reward amount must be positive'
  if (unit === 'quota') {
    if (!/^\d+$/.test(text)) return 'Raw reward quota must be an integer'
    if (parsed.gt(MAX_REWARD_QUOTA)) {
      return 'Reward quota must be between 1 and 2147483647'
    }
  }
  return null
}

/** Validate the quota actually issued, before a form can submit. */
export function getRewardConversionError(
  amount: string,
  unit: RewardUnit,
  quotaPerUnit: number | undefined
): string | null {
  const inputError = getRewardAmountError(amount, unit)
  if (inputError) return inputError
  if (!isValidRewardQuotaPerUnit(quotaPerUnit)) {
    return 'Invalid quota per USD configuration'
  }
  return getRewardEquivalent(amount, unit, quotaPerUnit)
    ? null
    : 'Reward quota must be between 1 and 2147483647'
}

export interface StoredReward {
  type: 'quota' | 'subscription'
  amount?: string
  unit?: string
  quota?: number
  subscription_plan_id?: number
}

export interface RewardFormFields {
  reward_type: 'none' | 'quota' | 'subscription'
  reward_amount: string
  reward_unit: RewardUnit
  subscription_plan_id: number
}

/** Old currency inputs never determine the saved payout during editing. */
export function rewardSnapshotToForm(
  reward?: StoredReward | null
): RewardFormFields {
  const unit = reward?.unit === 'usd' ? 'usd' : 'quota'
  let amount = '1'
  if (reward) {
    amount = unit === 'usd' ? (reward.amount ?? '') : String(reward.quota ?? '')
  }
  return {
    reward_type: reward?.type ?? 'none',
    reward_amount: amount,
    reward_unit: reward ? unit : 'usd',
    subscription_plan_id: reward?.subscription_plan_id ?? 0,
  }
}

/** Compare meaning, so whitespace/decimal formatting and unrelated edits preserve the snapshot. */
export function isRewardUnchanged(
  values: RewardFormFields,
  reward?: StoredReward | null
): boolean {
  const original = rewardSnapshotToForm(reward)
  if (values.reward_type !== original.reward_type) return false
  if (values.reward_type === 'none') return true
  if (values.reward_type === 'subscription') {
    return values.subscription_plan_id === original.subscription_plan_id
  }
  if (values.reward_unit !== original.reward_unit) return false
  if (
    getRewardAmountError(values.reward_amount, values.reward_unit) ||
    getRewardAmountError(original.reward_amount, original.reward_unit)
  ) {
    return false
  }
  return new RewardDecimal(values.reward_amount.trim()).eq(
    original.reward_amount.trim()
  )
}

export function normalizeRewardAmount(amount: string): string {
  return new RewardDecimal(amount.trim()).toFixed()
}
