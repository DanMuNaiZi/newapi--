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
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const newReferralAndGitHubKeys = [
  '0 means no reward for this person.',
  'A successful non-public-pool call with quota consumption is required within {{hours}} hours after registration. Rewards wait for manual approval.',
  'Account created',
  'Activation deadline',
  'Actual payout: {{quota}} platform quota',
  'After manual approval: inviter ${{inviter}}, new user ${{invitee}}.',
  'Approve and issue rewards',
  'Approved',
  'Confirm GitHub identity',
  'Confirm and add exemption',
  'Consumed quota',
  'Eligible quota consumption confirmed',
  'Enter a non-negative USD amount with at most 12 decimal places',
  'Exemptions are matched by the stable numeric GitHub ID and only bypass the account-age requirement.',
  'Existing reward snapshots stay unchanged unless you replace them. Already registered users keep their original rewards.',
  'Existing reward snapshots will be preserved.',
  'Failed to add exemption',
  'Failed to remove exemption',
  'Failed to update remark',
  'GitHub account',
  'GitHub account age exemptions',
  'GitHub account created',
  'GitHub age exemption',
  'GitHub age exemption added',
  'GitHub age exemption removed',
  'GitHub registration is unavailable',
  'GitHub registration required',
  'GitHub username',
  'Inviter reward',
  'Inviter reward (USD)',
  'Logged quota',
  'Minimum GitHub account age (days)',
  'New GitHub accounts younger than this are rejected. Set 0 to disable the age limit.',
  'New accounts must register with GitHub.',
  'New accounts must register with a GitHub account created at least {{days}} days ago.',
  'New user reward',
  'New user reward (USD)',
  'No GitHub age exemptions',
  'No eligible quota consumption',
  'No usage records',
  'Numeric ID',
  'Pending manual review',
  'Platform quota consumption is not proof of a cash payment. Free pool calls and interrupted calls do not qualify.',
  'Please contact the administrator for access.',
  'Please contact the administrator to configure GitHub OAuth.',
  'Qualifying call',
  'Qualifying deduction',
  'Reason for exemption',
  'Recent 20 calls. Totals are logged amounts, not proof of settlement; verified consumption is shown separately.',
  'Recent 20 calls. Totals follow log retention; the first qualifying deduction is saved separately.',
  'Recent invitations by this inviter',
  'Referral review',
  'Registration time',
  'Registration → quota consumption → manual review → rewards',
  'Rejected',
  'Remark for {{username}}',
  'Remark updated',
  'Remove GitHub age exemption?',
  'Remove exemption for {{username}}',
  'Replace rewards for future registrations',
  'Resolve identity',
  'Review details',
  'Review remark',
  'Review saved',
  'Rewards after approval',
  'Rewards are issued only after actual consumption and manual approval.',
  'Share invite link → friend registers with GitHub → successful paid non-public-pool call → administrator reviews → both rewards are issued.',
  'This only removes the age exemption. It does not change existing users.',
  'Unable to verify GitHub account',
  'Usage logs are unavailable. Saved qualifying evidence is still valid.',
  'Usage since registration',
  'a successful non-public-pool call with actual quota consumption is required',
  'an approved referral cannot be rejected',
  'invalid referral review',
  'referral campaign reward limit reached',
  'referral event is no longer reviewable',
  'referral reward requires manual approval',
  '{{count}} campaign invites',
] as const

function placeholders(value: string) {
  return [...value.matchAll(/{{[^{}]+}}/g)].map(([match]) => match).sort()
}

test('new referral and GitHub keys have Traditional Chinese translations with matching placeholders', async () => {
  const [englishDocument, traditionalChineseDocument] = await Promise.all([
    readFile(new URL('./locales/en.json', import.meta.url), 'utf8'),
    readFile(new URL('./locales/zh-TW.json', import.meta.url), 'utf8'),
  ])
  const english = JSON.parse(englishDocument).translation as Record<
    string,
    string
  >
  const traditionalChinese = JSON.parse(traditionalChineseDocument)
    .translation as Record<string, string>

  assert.equal(newReferralAndGitHubKeys.length, 78)
  for (const key of newReferralAndGitHubKeys) {
    assert.equal(typeof english[key], 'string', `missing English key: ${key}`)
    assert.equal(
      typeof traditionalChinese[key],
      'string',
      `missing Traditional Chinese key: ${key}`
    )
    assert.notEqual(
      traditionalChinese[key],
      english[key],
      `untranslated Traditional Chinese key: ${key}`
    )
    assert.deepEqual(
      placeholders(traditionalChinese[key]),
      placeholders(english[key]),
      `placeholder mismatch: ${key}`
    )
  }
})
