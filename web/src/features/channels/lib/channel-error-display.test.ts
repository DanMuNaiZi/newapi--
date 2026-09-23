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

import { describe, test } from 'vitest'

import { channelSchema } from '../types'
import {
  CHANNEL_FORM_DEFAULT_VALUES,
  channelFormSchema,
  transformChannelToFormDefaults,
  transformFormDataToCreatePayload,
  transformFormDataToUpdatePayload,
} from './channel-form'

const DEFAULT_MESSAGE = '上游服务暂时不可用，请稍后重试'

function buildChannel(settings: Record<string, unknown> = {}) {
  return channelSchema.parse({
    id: 7,
    type: 1,
    key: '',
    status: 1,
    name: 'OpenAI',
    created_time: 0,
    test_time: 0,
    response_time: 0,
    balance_updated_time: 0,
    settings: JSON.stringify(settings),
  })
}

describe('channel upstream error display form', () => {
  test('uses safe defaults when the channel has no display setting', () => {
    const defaults = transformChannelToFormDefaults(buildChannel())

    assert.equal(defaults.upstream_error_show_details, false)
    assert.equal(defaults.upstream_error_status_code, 503)
    assert.equal(defaults.upstream_error_message, DEFAULT_MESSAGE)
  })

  test('parses an existing display setting', () => {
    const defaults = transformChannelToFormDefaults(
      buildChannel({
        upstream_error_display: {
          show_details: true,
          status_code: 502,
          message: 'Provider unavailable',
        },
      })
    )

    assert.equal(defaults.upstream_error_show_details, true)
    assert.equal(defaults.upstream_error_status_code, 502)
    assert.equal(defaults.upstream_error_message, 'Provider unavailable')
  })

  test('serializes the setting without dropping unrelated channel settings', () => {
    const values = {
      ...CHANNEL_FORM_DEFAULT_VALUES,
      name: 'OpenAI',
      key: 'sk-test',
      models: 'gpt-4o',
      settings: JSON.stringify({ existing_option: 'keep-me' }),
      upstream_error_show_details: true,
      upstream_error_status_code: 502,
      upstream_error_message: 'Provider unavailable',
    }

    const createPayload = transformFormDataToCreatePayload(values)
    const updatePayload = transformFormDataToUpdatePayload(values, 7)
    const createSettings = JSON.parse(String(createPayload.channel.settings))
    const updateSettings = JSON.parse(String(updatePayload.settings))

    for (const settings of [createSettings, updateSettings]) {
      assert.equal(settings.existing_option, 'keep-me')
      assert.deepEqual(settings.upstream_error_display, {
        show_details: true,
        status_code: 502,
        message: 'Provider unavailable',
      })
    }
  })

  test('validates the generic status code and message', () => {
    const baseValues = {
      ...CHANNEL_FORM_DEFAULT_VALUES,
      name: 'OpenAI',
      key: 'sk-test',
      models: 'gpt-4o',
    }

    assert.equal(
      channelFormSchema.safeParse({
        ...baseValues,
        upstream_error_status_code: 399,
      }).success,
      false
    )
    assert.equal(
      channelFormSchema.safeParse({
        ...baseValues,
        upstream_error_message: '',
      }).success,
      false
    )
    assert.equal(
      channelFormSchema.safeParse({
        ...baseValues,
        upstream_error_message: 'x'.repeat(501),
      }).success,
      false
    )
  })
})
