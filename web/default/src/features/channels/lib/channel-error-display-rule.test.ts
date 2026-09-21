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

import { channelErrorDisplayRuleSchema } from './channel-error-display-rule'

describe('channel error display rule validation', () => {
  test('accepts inherit, generic, and original without custom fields', () => {
    for (const display_mode of ['inherit', 'generic', 'original'] as const) {
      assert.equal(
        channelErrorDisplayRuleSchema.safeParse({ display_mode }).success,
        true
      )
    }
  })

  test('requires a valid status and message for custom mode', () => {
    assert.equal(
      channelErrorDisplayRuleSchema.safeParse({ display_mode: 'custom' })
        .success,
      false
    )
    assert.equal(
      channelErrorDisplayRuleSchema.safeParse({
        display_mode: 'custom',
        display_status_code: 399,
        display_message: 'Unavailable',
      }).success,
      false
    )
    assert.equal(
      channelErrorDisplayRuleSchema.safeParse({
        display_mode: 'custom',
        display_status_code: 503,
        display_message: 'x'.repeat(501),
      }).success,
      false
    )
    assert.equal(
      channelErrorDisplayRuleSchema.safeParse({
        display_mode: 'custom',
        display_status_code: 502,
        display_message: 'Provider unavailable',
      }).success,
      true
    )
  })
})
