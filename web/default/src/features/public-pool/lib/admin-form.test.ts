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

import type { PublicPoolSite } from '../types'
import {
  publicPoolSiteSchema,
  publicPoolSiteToForm,
  toPublicPoolSitePayload,
} from './admin-form'

const rewardDefaults = {
  reward_type: 'none' as const,
  reward_amount: '1',
  reward_unit: 'usd' as const,
  subscription_plan_id: 0,
}

describe('public pool site validation', () => {
  test('edits historical rewards using saved USD or quota and preserves unchanged snapshots', () => {
    const site: PublicPoolSite = {
      id: 1,
      name: 'Saved',
      url: 'https://example.com',
      description: '',
      status: 'enabled',
      sort_order: 0,
      created_at: 1,
      updated_at: 1,
      reward: {
        type: 'quota',
        amount: '1',
        unit: 'usd',
        quota: 500000,
        quota_per_unit: '500000',
      },
    }
    const values = publicPoolSiteToForm(site)
    assert.equal(values.reward_amount, '1')
    assert.equal(values.reward_unit, 'usd')
    assert.equal(
      toPublicPoolSitePayload(
        { ...values, name: 'Renamed', reward_amount: '01.00' },
        site
      ).preserve_reward,
      true
    )
    assert.equal(
      toPublicPoolSitePayload({ ...values, reward_amount: '2' }, site)
        .preserve_reward,
      false
    )
    const legacy = {
      ...site,
      reward: {
        ...site.reward,
        type: 'quota' as const,
        unit: 'cny' as const,
        amount: '7.3',
      },
    }
    const legacyValues = publicPoolSiteToForm(legacy)
    assert.equal(legacyValues.reward_amount, '500000')
    assert.equal(legacyValues.reward_unit, 'quota')
    assert.equal(
      toPublicPoolSitePayload(legacyValues, legacy).preserve_reward,
      true
    )
    const noReward = { ...site, reward: undefined }
    assert.equal(
      toPublicPoolSitePayload(publicPoolSiteToForm(noReward), noReward)
        .preserve_reward,
      true
    )
  })

  test('rejects quota overflow while allowing a subscription with invalid hidden amount', () => {
    const values = {
      name: 'Boundary',
      url: 'https://example.com',
      description: '',
      status: 'enabled' as const,
      sort_order: 0,
      ...rewardDefaults,
      reward_type: 'quota' as const,
      reward_unit: 'quota' as const,
      reward_amount: '2147483648',
    }
    assert.equal(publicPoolSiteSchema.safeParse(values).success, false)
    assert.equal(
      publicPoolSiteSchema.safeParse({
        ...values,
        reward_type: 'subscription',
        reward_amount: '',
        subscription_plan_id: 3,
      }).success,
      true
    )
  })

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

  test('does not accept CNY in new site forms', () => {
    const result = publicPoolSiteSchema.safeParse({
      name: 'Community AI',
      url: 'https://example.com/register',
      description: '',
      status: 'enabled',
      sort_order: 0,
      reward_type: 'quota',
      reward_amount: '73',
      reward_unit: 'cny',
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
