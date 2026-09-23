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
import { describe, expect, test } from 'vitest'

import {
  createManualWechatSettingsSchema,
  MANUAL_WECHAT_QR_MAX_BYTES,
  validateManualWechatQRCodeFile,
} from '../lib/manual-wechat-settings-schema'

describe('manual WeChat settings validation', () => {
  test('requires an image when the payment method is enabled', () => {
    const result = createManualWechatSettingsSchema(1_000).safeParse({
      enabled: true,
      qr_code_image: '',
      expires_at: 0,
      instructions: 'Add the order number.',
    })

    expect(result.success).toBe(false)
  })

  test('rejects an already expired QR code', () => {
    const result = createManualWechatSettingsSchema(1_000).safeParse({
      enabled: true,
      qr_code_image: 'data:image/png;base64,AA==',
      expires_at: 999,
      instructions: 'Add the order number.',
    })

    expect(result.success).toBe(false)
  })

  test('accepts supported image files and rejects oversized or unsupported files', () => {
    const valid = new File(['png'], 'qr.png', { type: 'image/png' })
    expect(validateManualWechatQRCodeFile(valid)).toBeNull()

    const oversized = new File(['x'], 'large.webp', { type: 'image/webp' })
    Object.defineProperty(oversized, 'size', {
      value: MANUAL_WECHAT_QR_MAX_BYTES + 1,
    })
    expect(validateManualWechatQRCodeFile(oversized)).toBe(
      'payment.manualWechat.settings.fileTooLarge'
    )

    const unsupported = new File(['gif'], 'qr.gif', { type: 'image/gif' })
    expect(validateManualWechatQRCodeFile(unsupported)).toBe(
      'payment.manualWechat.settings.unsupportedFile'
    )
  })
})
