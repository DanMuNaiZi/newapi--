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

import type { LotteryPlan } from '../types'
import {
  filterLotteryPlansByView,
  matchesLotteryPlanSearch,
} from './admin-list'

function createPlan(
  id: number,
  status: LotteryPlan['status'],
  title = `Lottery ${id}`,
  description = ''
): LotteryPlan {
  return {
    id,
    title,
    icon: '',
    description,
    status,
    eligibility_mode: 'all',
    max_participants: 10,
    registration_start_time: 1,
    draw_time: 2,
  }
}

describe('lottery admin list', () => {
  const plans = [
    createPlan(1, 'draft'),
    createPlan(2, 'scheduled'),
    createPlan(3, 'open'),
    createPlan(4, 'drawing'),
    createPlan(5, 'finished'),
    createPlan(6, 'cancelled'),
  ]

  test('separates current plans from history', () => {
    assert.deepEqual(
      filterLotteryPlansByView(plans, 'current').map((plan) => plan.id),
      [1, 2, 3, 4]
    )
    assert.deepEqual(
      filterLotteryPlansByView(plans, 'history').map((plan) => plan.id),
      [5, 6]
    )
  })

  test('keeps all plans in the all view', () => {
    assert.deepEqual(filterLotteryPlansByView(plans, 'all'), plans)
  })

  test('searches by id, title, and description', () => {
    const plan = createPlan(42, 'open', 'July members', 'Premium user event')

    assert.equal(matchesLotteryPlanSearch(plan, '42'), true)
    assert.equal(matchesLotteryPlanSearch(plan, 'members'), true)
    assert.equal(matchesLotteryPlanSearch(plan, 'premium'), true)
    assert.equal(matchesLotteryPlanSearch(plan, 'missing'), false)
  })
})
