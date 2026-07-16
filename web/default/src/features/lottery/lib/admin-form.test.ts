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

import {
  buildLotteryPlanPayload,
  createLotteryFormDefaults,
  lotteryAdminFormSchema,
  type LotteryAdminFormValues,
} from './admin-form'

const BASE_FORM: LotteryAdminFormValues = {
  title: 'July lottery',
  icon: 'https://cdn.example.com/lottery.png',
  description: 'Monthly campaign',
  eligibility_mode: 'all',
  selected_groups: [],
  selected_user_ids: [],
  max_participants: 100,
  registration_start: '2026-07-15T10:00',
  draw_time: '2026-07-15T12:00',
  prizes: [
    {
      name: 'Quota prize',
      quantity: 2,
      reward_type: 'quota',
      quota: 100,
      subscription_plan_id: 9,
      fulfillment_mode: 'self_claim',
      claim_expire_days: 7,
    },
  ],
}

describe('buildLotteryPlanPayload', () => {
  test('provides a usable default registration and draw schedule', () => {
    const defaults = createLotteryFormDefaults(new Date('2026-07-15T08:00:00'))

    const registration = new Date(defaults.registration_start).getTime()
    const draw = new Date(defaults.draw_time).getTime()
    assert.ok(registration > new Date('2026-07-15T08:00:00').getTime())
    assert.ok(draw > registration)
  })

  test('uses a translatable validation key for an empty numeric field', () => {
    const result = lotteryAdminFormSchema.safeParse({
      ...BASE_FORM,
      max_participants: Number.NaN,
    })

    assert.equal(result.success, false)
    if (result.success) return
    assert.equal(
      result.error.issues.find((issue) => issue.path[0] === 'max_participants')
        ?.message,
      'Maximum participants must be at least 1'
    )
  })

  test('rejects values that exceed cross-database integer columns', () => {
    const result = lotteryAdminFormSchema.safeParse({
      ...BASE_FORM,
      max_participants: 2_147_483_648,
      prizes: [
        {
          ...BASE_FORM.prizes[0],
          quantity: 2_147_483_648,
          quota: 2_147_483_648,
        },
      ],
    })

    assert.equal(result.success, false)
    if (result.success) return
    assert.deepEqual(
      result.error.issues.map((issue) => issue.message).sort(),
      [
        'Maximum participants cannot exceed 2147483647',
        'Prize quantity cannot exceed 2147483647',
        'Quota reward cannot exceed 2147483647',
      ].sort()
    )
  })

  test('keeps only selected groups for group eligibility', () => {
    const payload = buildLotteryPlanPayload({
      ...BASE_FORM,
      eligibility_mode: 'groups',
      selected_groups: ['vip', 'partner'],
      selected_user_ids: [12, 13],
    })

    assert.deepEqual(payload.groups, ['vip', 'partner'])
    assert.deepEqual(payload.user_ids, [])
  })

  test('includes a valid lottery icon URL in the payload', () => {
    const payload = buildLotteryPlanPayload(BASE_FORM)

    assert.equal(payload.icon, 'https://cdn.example.com/lottery.png')
  })

  test('rejects unsafe lottery icon URLs', () => {
    const result = lotteryAdminFormSchema.safeParse({
      ...BASE_FORM,
      icon: 'javascript:alert(1)',
    })

    assert.equal(result.success, false)
    if (result.success) return
    assert.equal(
      result.error.issues.find((issue) => issue.path[0] === 'icon')?.message,
      'Icon URL must start with http:// or https://'
    )
  })

  test('keeps only selected users for user eligibility', () => {
    const payload = buildLotteryPlanPayload({
      ...BASE_FORM,
      eligibility_mode: 'users',
      selected_groups: ['vip'],
      selected_user_ids: [12, 13],
    })

    assert.deepEqual(payload.groups, [])
    assert.deepEqual(payload.user_ids, [12, 13])
  })

  test('normalizes quota prizes and converts claim days to seconds', () => {
    const payload = buildLotteryPlanPayload(BASE_FORM)

    assert.deepEqual(payload.prizes[0], {
      name: 'Quota prize',
      quantity: 2,
      reward_type: 'quota',
      quota: 100,
      subscription_plan_id: 0,
      fulfillment_mode: 'self_claim',
      claim_expire_seconds: 604800,
    })
  })

  test('normalizes subscription prizes without quota', () => {
    const payload = buildLotteryPlanPayload({
      ...BASE_FORM,
      prizes: [
        {
          name: 'Subscription prize',
          quantity: 1,
          reward_type: 'subscription',
          quota: 500,
          subscription_plan_id: 4,
          fulfillment_mode: 'redemption_code',
          claim_expire_days: 0,
        },
      ],
    })

    assert.equal(payload.prizes[0]?.quota, 0)
    assert.equal(payload.prizes[0]?.subscription_plan_id, 4)
    assert.equal(payload.prizes[0]?.claim_expire_seconds, 0)
  })
})
