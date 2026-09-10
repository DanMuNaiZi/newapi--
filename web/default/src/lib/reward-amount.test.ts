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

import { getRewardEquivalent } from './reward-amount'

describe('reward amount conversion', () => {
  test('converts USD and quota without a currency exchange rate', () => {
    assert.deepEqual(getRewardEquivalent('10', 'usd', 500_000), {
      usd: '10',
      quota: 5_000_000,
    })
    assert.deepEqual(getRewardEquivalent('5000000', 'quota', 500_000), {
      usd: '10',
      quota: 5_000_000,
    })
  })

  test('matches server Decimal rounding at a binary floating-point boundary', () => {
    assert.deepEqual(getRewardEquivalent('0.000249', 'usd', 500_000), {
      usd: '0.00025',
      quota: 125,
    })
    assert.equal(getRewardEquivalent('0.000001', 'usd', 500_000)?.quota, 1)
    assert.equal(
      getRewardEquivalent('0.00000099999999999999', 'usd', 500_000),
      null
    )
  })

  test('enforces the persisted quota range without overflow', () => {
    assert.equal(
      getRewardEquivalent('2147483647', 'quota', 500_000)?.quota,
      2147483647
    )
    assert.equal(getRewardEquivalent('2147483648', 'quota', 500_000), null)
    assert.equal(getRewardEquivalent('4294.967295', 'usd', 500_000), null)
    assert.equal(
      getRewardEquivalent('4294.967294', 'usd', 500_000)?.quota,
      2147483647
    )
  })

  test('rejects invalid amounts and runtime conversion configuration', () => {
    for (const amount of ['', '0', '-1', 'NaN', 'Infinity', '1x', '1.5']) {
      assert.equal(getRewardEquivalent(amount, 'quota', 500_000), null, amount)
    }
    for (const quotaPerUnit of [0, -1, Number.NaN, Infinity]) {
      assert.equal(getRewardEquivalent('1', 'usd', quotaPerUnit), null)
    }
  })

  test('rejects fractional raw quota and unsafe conversion results', () => {
    assert.equal(getRewardEquivalent(1.5, 'quota', 500_000), null)
    assert.equal(getRewardEquivalent(Number.MAX_VALUE, 'usd', 500_000), null)
  })
})
