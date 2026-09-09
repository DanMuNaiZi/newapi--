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

import { publicPoolSiteSchema, toPublicPoolSitePayload } from './admin-form'

const rewardDefaults = {
  reward_type: 'none' as const,
  reward_amount: '1',
  reward_unit: 'usd' as const,
  subscription_plan_id: 0,
}

describe('public pool site validation', () => {
  test('accepts a credential-free HTTPS URL', () => {
    const result = publicPoolSiteSchema.safeParse({
      name: 'Community AI',
      url: 'https://example.com/register',
      description: 'Free community quota',
      status: 'enabled',
      sort_order: 10,
      ...rewardDefaults,
    })

    assert.equal(result.success, true)
  })

  test('rejects URLs containing embedded credentials', () => {
    const result = publicPoolSiteSchema.safeParse({
      name: 'Unsafe site',
      url: 'https://user:secret@example.com/register',
      description: '',
      status: 'enabled',
      sort_order: 0,
      ...rewardDefaults,
    })

    assert.equal(result.success, false)
  })

  test('rejects sort values outside the database integer range', () => {
    const result = publicPoolSiteSchema.safeParse({
      name: 'Community AI',
      url: 'https://example.com/register',
      description: '',
      status: 'enabled',
      sort_order: 2147483648,
      ...rewardDefaults,
    })

    assert.equal(result.success, false)
  })

  test('rejects fractional raw quota rewards', () => {
    const result = publicPoolSiteSchema.safeParse({
      name: 'Community AI',
      url: 'https://example.com/register',
      description: '',
      status: 'enabled',
      sort_order: 0,
      reward_type: 'quota',
      reward_amount: '1.5',
      reward_unit: 'quota',
      subscription_plan_id: 0,
    })

    assert.equal(result.success, false)
  })

  test('serializes quota and subscription rewards with explicit meanings', () => {
    const quota = toPublicPoolSitePayload({
      name: 'Community AI',
      url: 'https://example.com/register',
      description: '',
      status: 'enabled',
      sort_order: 0,
      reward_type: 'quota',
      reward_amount: '10',
      reward_unit: 'usd',
      subscription_plan_id: 0,
    })
    assert.deepEqual(quota.reward, {
      type: 'quota',
      amount: '10',
      unit: 'usd',
      subscription_plan_id: 0,
    })

    const subscription = toPublicPoolSitePayload({
      name: 'Community AI',
      url: 'https://example.com/register',
      description: '',
      status: 'enabled',
      sort_order: 0,
      reward_type: 'subscription',
      reward_amount: '',
      reward_unit: 'quota',
      subscription_plan_id: 3,
    })
    assert.equal(subscription.reward?.type, 'subscription')
    assert.equal(subscription.reward?.subscription_plan_id, 3)
  })
})
