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

import type { User } from '@/features/users/types'

import {
  formatLotteryUserLabel,
  isEligibleLotteryUser,
  mergeLotteryUsers,
} from './user-options'

function user(id: number, displayName: string): User {
  return {
    id,
    username: `user-${id}`,
    display_name: displayName,
    quota: 0,
    used_quota: 0,
    request_count: 0,
    group: id === 1 ? 'vip' : 'default',
    status: 1,
    role: 1,
  }
}

describe('lottery user options', () => {
  test('keeps selected users while merging search results', () => {
    const selected = user(1, 'Selected User')
    const updated = user(1, 'Updated Name')
    const result = user(2, 'Search Result')

    const merged = mergeLotteryUsers([selected], [updated, result])

    assert.deepEqual(
      merged.map((item) => item.id),
      [1, 2]
    )
    assert.equal(merged[0]?.display_name, 'Selected User')
  })

  test('formats searchable identity details', () => {
    assert.equal(
      formatLotteryUserLabel(user(1, 'Long')),
      'Long (user-1) | #1 | vip'
    )
  })

  test('excludes administrators, disabled users, and deleted users', () => {
    assert.equal(isEligibleLotteryUser(user(1, 'Enabled user')), true)
    assert.equal(
      isEligibleLotteryUser({ ...user(2, 'Admin'), role: 10 }),
      false
    )
    assert.equal(
      isEligibleLotteryUser({ ...user(3, 'Disabled'), status: 2 }),
      false
    )
    assert.equal(
      isEligibleLotteryUser({ ...user(4, 'Deleted'), DeletedAt: {} }),
      false
    )
  })
})
