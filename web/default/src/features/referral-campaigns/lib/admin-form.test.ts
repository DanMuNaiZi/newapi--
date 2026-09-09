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
  buildReferralCampaignPayload,
  emptyReferralCampaignForm,
  referralCampaignFormSchema,
} from './admin-form'

describe('referral campaign form', () => {
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
})
