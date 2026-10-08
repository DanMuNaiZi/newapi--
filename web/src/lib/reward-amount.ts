/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import Decimal from 'decimal.js-light'

const RewardDecimal = Decimal.clone({
  precision: 80,
  rounding: Decimal.ROUND_DOWN,
})

export const MAX_REWARD_QUOTA = 2_147_483_647
export type RewardUnit = 'usd' | 'cny' | 'quota'

export interface RewardEquivalent {
  usd: string
  quota: number
}

export function isValidRewardQuotaPerUnit(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value) && value > 0
}

export function isValidUsdExchangeRate(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value) && value > 0
}

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

export function getRewardEquivalent(
  amount: string | number,
  unit: RewardUnit,
  quotaPerUnit: number,
  usdExchangeRate = 1
): RewardEquivalent | null {
  const text = String(amount).trim()
  if (
    getRewardAmountError(text, unit) ||
    !isValidRewardQuotaPerUnit(quotaPerUnit) ||
    (unit === 'cny' && !isValidUsdExchangeRate(usdExchangeRate))
  ) {
    return null
  }
  const parsed = new RewardDecimal(text)
  let usd: Decimal
  let rawQuota: Decimal
  if (unit === 'usd') {
    usd = parsed
    rawQuota = parsed.times(String(quotaPerUnit))
  } else if (unit === 'cny') {
    usd = parsed.div(String(usdExchangeRate))
    rawQuota = usd.times(String(quotaPerUnit))
  } else {
    usd = parsed.div(String(quotaPerUnit))
    rawQuota = parsed
  }
  const quota = rawQuota.toDecimalPlaces(0, Decimal.ROUND_HALF_UP)
  if (quota.lt(1) || quota.gt(MAX_REWARD_QUOTA)) return null
  return {
    usd: quota.div(String(quotaPerUnit)).toSignificantDigits(12).toString(),
    quota: quota.toNumber(),
  }
}

export function getRewardConversionError(
  amount: string,
  unit: RewardUnit,
  quotaPerUnit: number | undefined,
  usdExchangeRate = 1
): string | null {
  const inputError = getRewardAmountError(amount, unit)
  if (inputError) return inputError
  if (
    !isValidRewardQuotaPerUnit(quotaPerUnit) ||
    (unit === 'cny' && !isValidUsdExchangeRate(usdExchangeRate))
  ) {
    return unit === 'cny'
      ? 'Invalid USD exchange rate configuration'
      : 'Invalid quota per USD configuration'
  }
  return getRewardEquivalent(amount, unit, quotaPerUnit, usdExchangeRate)
    ? null
    : 'Reward quota must be between 1 and 2147483647'
}

export function normalizeRewardAmount(amount: string): string {
  return new RewardDecimal(amount.trim()).toFixed()
}
