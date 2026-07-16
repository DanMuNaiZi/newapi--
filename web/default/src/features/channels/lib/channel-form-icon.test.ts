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
  CHANNEL_FORM_DEFAULT_VALUES,
  channelFormSchema,
  transformFormDataToCreatePayload,
  transformFormDataToUpdatePayload,
} from './channel-form'

const FORM_VALUES = {
  ...CHANNEL_FORM_DEFAULT_VALUES,
  name: 'Custom icon channel',
  icon: 'https://cdn.example.com/channel.png',
  key: 'sk-test',
  models: 'gpt-4o',
}

describe('channel icon form', () => {
  test('includes the custom icon in create and update payloads', () => {
    const createPayload = transformFormDataToCreatePayload(FORM_VALUES)
    const updatePayload = transformFormDataToUpdatePayload(FORM_VALUES, 9)

    assert.equal(
      createPayload.channel.icon,
      'https://cdn.example.com/channel.png'
    )
    assert.equal(updatePayload.icon, 'https://cdn.example.com/channel.png')
  })

  test('allows clearing an existing icon', () => {
    const payload = transformFormDataToUpdatePayload(
      { ...FORM_VALUES, icon: '' },
      9
    )

    assert.equal(payload.icon, '')
  })

  test('rejects unsafe icon URLs', () => {
    const result = channelFormSchema.safeParse({
      ...FORM_VALUES,
      icon: 'data:image/svg+xml,test',
    })

    assert.equal(result.success, false)
  })
})
