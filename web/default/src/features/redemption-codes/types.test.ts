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
import { describe, test } from 'node:test'

import { redemptionSchema } from './types'

describe('redemptionSchema', () => {
  test('treats legacy empty reward types as quota rewards', () => {
    const redemption = redemptionSchema.parse({
      id: 1,
      user_id: 1,
      name: 'Legacy code',
      key: '0123456789abcdef0123456789abcdef',
      status: 1,
      quota: 500000,
      reward_type: '',
      subscription_plan_id: 0,
      batch: '',
      source_ref: '',
      remark: '',
      created_time: 1,
      redeemed_time: 0,
      expired_time: 0,
      used_user_id: 0,
    })

    assert.equal(redemption.reward_type, 'quota')
  })
})
