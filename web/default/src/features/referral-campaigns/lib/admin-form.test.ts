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

import type { ReferralCampaign } from '../types'
import {
  buildReferralCampaignPayload,
  emptyReferralCampaignForm,
  referralCampaignFormSchema,
  referralCampaignToForm,
} from './admin-form'

describe('referral campaign form', () => {
  test('preserves the saved USD reward across metadata edits and reloads', () => {
    const campaign: ReferralCampaign = {
      id: 5,
      title: 'Historical USD',
      description: '',
      enabled: true,
      start_time: 1800000000,
      end_time: 1800086400,
      activation_window_seconds: 86400,
      max_rewards_per_inviter: 0,
      total_reward_limit: 0,
      registration_count: 0,
      activated_count: 0,
      rewarded_count: 0,
      failed_reward_count: 0,
      reward: { type: 'quota', amount: '1', unit: 'usd', quota: 500000 },
    }
    const values = referralCampaignToForm(campaign)
    values.title = 'Renamed'
    values.reward_amount = '1.00'
    assert.equal(
      buildReferralCampaignPayload(values, campaign).preserve_reward,
      true
    )
    assert.equal(values.reward_unit, 'usd')
    const reloaded = referralCampaignToForm({
      ...campaign,
      title: values.title,
    })
    assert.equal(reloaded.reward_amount, '1')
    assert.equal(
      buildReferralCampaignPayload(reloaded, campaign).preserve_reward,
      true
    )
    assert.equal(
      buildReferralCampaignPayload(
        { ...reloaded, reward_amount: '2' },
        campaign
      ).preserve_reward,
      false
    )
  })

  test('rejects quota overflow and ignores hidden subscription amounts', () => {
    const values = {
      ...emptyReferralCampaignForm(),
      title: 'Boundaries',
      reward_unit: 'quota' as const,
      reward_amount: '2147483648',
    }
    assert.equal(referralCampaignFormSchema.safeParse(values).success, false)
    const subscription = {
      ...values,
      reward_type: 'subscription' as const,
      reward_amount: '',
      subscription_plan_id: 3,
    }
    assert.equal(
      referralCampaignFormSchema.safeParse(subscription).success,
      true
    )
    assert.equal(buildReferralCampaignPayload(subscription).reward.amount, '')
  })

  test('builds a quota reward with the default 24 hour activation window', () => {
    const values = emptyReferralCampaignForm()
    values.title = 'September referral'

    const payload = buildReferralCampaignPayload(values)

    assert.equal(payload.activation_window_seconds, 24 * 60 * 60)
    assert.deepEqual(payload.reward, {
      type: 'quota',
      amount: '1',
      unit: 'usd',
      subscription_plan_id: 0,
    })
  })

  test('builds a subscription reward without an ambiguous amount', () => {
    const values = emptyReferralCampaignForm()
    values.title = 'Subscription referral'
    values.reward_type = 'subscription'
    values.subscription_plan_id = 3

    assert.deepEqual(buildReferralCampaignPayload(values).reward, {
      type: 'subscription',
      amount: '',
      unit: 'quota',
      subscription_plan_id: 3,
    })
  })

  test('rejects invalid time ranges and fractional raw quota', () => {
    const values = emptyReferralCampaignForm()
    values.title = 'Invalid referral'
    values.end = values.start
    values.reward_unit = 'quota'
    values.reward_amount = '1.5'

    assert.equal(referralCampaignFormSchema.safeParse(values).success, false)
  })

  test('does not accept CNY in new campaign forms', () => {
    const values = emptyReferralCampaignForm()
    const result = referralCampaignFormSchema.safeParse({
      ...values,
      reward_unit: 'cny',
    })

    assert.equal(result.success, false)
  })

  test('maps a legacy CNY snapshot to its persisted quota when editing', () => {
    const values = referralCampaignToForm({
      id: 1,
      title: 'Legacy campaign',
      description: '',
      enabled: true,
      start_time: 1_800_000_000,
      end_time: 1_800_086_400,
      activation_window_seconds: 86_400,
      max_rewards_per_inviter: 0,
      total_reward_limit: 0,
      reward: {
        type: 'quota',
        amount: '73',
        unit: 'cny',
        quota: 5_000_000,
      },
      registration_count: 0,
      activated_count: 0,
      rewarded_count: 0,
      failed_reward_count: 0,
    })

    assert.equal(values.reward_unit, 'quota')
    assert.equal(values.reward_amount, '5000000')
  })
})
