/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useQuery } from '@tanstack/react-query'

import { statusQueryOptions } from '@/lib/status-query'

export function useRewardQuotaConfig(enabled = true) {
  const query = useQuery({ ...statusQueryOptions, enabled })
  const status = query.data
  const quotaPerUnit =
    typeof status?.quota_per_unit === 'number'
      ? status.quota_per_unit
      : Number(status?.quota_per_unit)
  const usdExchangeRate =
    typeof status?.usd_exchange_rate === 'number'
      ? status.usd_exchange_rate
      : Number(status?.usd_exchange_rate)
  return {
    ...query,
    quotaPerUnit,
    usdExchangeRate,
  }
}
