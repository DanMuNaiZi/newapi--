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

const campaign: ReferralCampaign = {
  id: 5,
  title: 'Existing campaign',
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

describe('referral campaign manual-review form', () => {
  test('new campaigns configure independent USD rewards, with 0 disabling a side', () => {
    const values = { ...emptyReferralCampaignForm(), title: 'September' }
    const payload = buildReferralCampaignPayload(values)
    assert.equal(payload.activation_window_seconds, 86400)
    assert.equal(payload.inviter_reward_usd, '1')
    assert.equal(payload.invitee_reward_usd, '0')
    assert.equal(payload.reward, undefined)
    assert.equal(
      buildReferralCampaignPayload({
        ...values,
        inviter_reward_usd: '0',
        invitee_reward_usd: '2.5',
      }).invitee_reward_usd,
      '2.5'
    )
  })

  test('metadata edits preserve both stored rewards despite changed system conversion', () => {
    const values = referralCampaignToForm(campaign)
    values.title = 'Renamed'
    const payload = buildReferralCampaignPayload(values, campaign)
    assert.equal(payload.preserve_reward, true)
    assert.equal(payload.inviter_reward_usd, undefined)
    assert.equal(payload.invitee_reward_usd, undefined)
  })

  test('replacing rewards is explicit and submits both USD amounts', () => {
    const values = {
      ...referralCampaignToForm(campaign),
      replace_rewards: true,
      inviter_reward_usd: '2',
      invitee_reward_usd: '1',
    }
    const payload = buildReferralCampaignPayload(values, campaign)
    assert.equal(payload.preserve_reward, undefined)
    assert.equal(payload.inviter_reward_usd, '2')
    assert.equal(payload.invitee_reward_usd, '1')
  })

  test('legacy CNY and subscription snapshots remain untouched until explicitly replaced', () => {
    for (const reward of [
      {
        type: 'quota' as const,
        amount: '73',
        unit: 'cny' as const,
        quota: 5000000,
      },
      {
        type: 'subscription' as const,
        subscription_plan_id: 3,
        subscription_plan_title: 'Old plan',
      },
    ]) {
      const legacy = { ...campaign, reward }
      const payload = buildReferralCampaignPayload(
        referralCampaignToForm(legacy),
        legacy
      )
      assert.equal(payload.preserve_reward, true)
      assert.equal(payload.reward, undefined)
    }
  })

  test('rejects invalid dates, fractional limits, negative and unbounded decimal input', () => {
    const values = { ...emptyReferralCampaignForm(), title: 'Validation' }
    for (const invalid of [
      { end: values.start },
      { max_rewards_per_inviter: 0.5 },
      { activation_hours: 0 },
      { total_reward_limit: 2147483648 },
      { inviter_reward_usd: '-1' },
      { invitee_reward_usd: '1e2147483647' },
      { invitee_reward_usd: 'NaN' },
      { inviter_reward_usd: '' },
    ]) {
      assert.equal(
        referralCampaignFormSchema.safeParse({ ...values, ...invalid }).success,
        false
      )
    }
  })
})
