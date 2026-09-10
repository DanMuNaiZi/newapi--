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
import assert from 'node:assert/strict'
import { test } from 'node:test'

import { QueryClient } from '@tanstack/react-query'
import { AxiosError } from 'axios'

import { api } from '@/lib/api'

import {
  fetchRewardQuotaPerUnit,
  rewardQuotaConfigOptions,
} from './use-reward-quota-config'

test('loads the raw status rate without currency defaults and retries local failures', async () => {
  const originalAdapter = api.defaults.adapter
  const client = new QueryClient()
  let body: unknown = {
    success: true,
    data: { quota_per_unit: 500000, usd_exchange_rate: 0 },
  }
  let fail = false
  let requests = 0
  api.defaults.adapter = async (config) => {
    assert.equal(config.url, '/api/status')
    assert.equal(config.skipBusinessError, true)
    assert.equal(config.skipErrorHandler, true)
    requests++
    if (fail) throw new AxiosError('HTTP failure', 'ERR_NETWORK', config)
    return { status: 200, statusText: 'OK', headers: {}, config, data: body }
  }
  try {
    assert.equal(await client.fetchQuery(rewardQuotaConfigOptions), 500000)
    for (const value of [
      undefined,
      null,
      '500000',
      'bad',
      0,
      -1,
      Number.NaN,
      Number.POSITIVE_INFINITY,
    ]) {
      body = { success: true, data: { quota_per_unit: value } }
      await assert.rejects(
        fetchRewardQuotaPerUnit(),
        /Invalid quota per USD configuration/
      )
    }
    for (const invalidBody of [
      null,
      {},
      { success: false },
      { success: true },
    ]) {
      body = invalidBody
      await assert.rejects(
        fetchRewardQuotaPerUnit(),
        /Failed to load reward configuration|Invalid quota per USD configuration/
      )
    }
    fail = true
    const before = requests
    await assert.rejects(
      client.fetchQuery(rewardQuotaConfigOptions),
      /Failed to load reward configuration/
    )
    assert.equal(requests, before + 1)
    assert.equal(
      client.getQueryState(rewardQuotaConfigOptions.queryKey)?.status,
      'error'
    )
    fail = false
    body = { success: true, data: { quota_per_unit: 600000 } }
    assert.equal(await client.fetchQuery(rewardQuotaConfigOptions), 600000)
    assert.equal(
      client.getQueryState(rewardQuotaConfigOptions.queryKey)?.status,
      'success'
    )
  } finally {
    api.defaults.adapter = originalAdapter
    client.clear()
  }
})
