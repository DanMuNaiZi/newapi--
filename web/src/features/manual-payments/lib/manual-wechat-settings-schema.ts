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
import { z } from 'zod'

export const MANUAL_WECHAT_QR_MAX_BYTES = 512 * 1024
export const MANUAL_WECHAT_INSTRUCTIONS_MAX_LENGTH = 500

const supportedImageTypes = new Set(['image/png', 'image/jpeg', 'image/webp'])

export function validateManualWechatQRCodeFile(file: File): string | null {
  if (!supportedImageTypes.has(file.type)) {
    return 'payment.manualWechat.settings.unsupportedFile'
  }
  if (file.size > MANUAL_WECHAT_QR_MAX_BYTES) {
    return 'payment.manualWechat.settings.fileTooLarge'
  }
  return null
}

export function createManualWechatSettingsSchema(nowSeconds: number) {
  return z
    .object({
      enabled: z.boolean(),
      qr_code_image: z.string(),
      expires_at: z.number().int().min(0),
      instructions: z
        .string()
        .max(
          MANUAL_WECHAT_INSTRUCTIONS_MAX_LENGTH,
          'payment.manualWechat.settings.instructionsTooLong'
        ),
    })
    .superRefine((value, context) => {
      if (value.enabled && !value.qr_code_image) {
        context.addIssue({
          code: 'custom',
          path: ['qr_code_image'],
          message: 'payment.manualWechat.settings.imageRequired',
        })
      }
      if (
        value.enabled &&
        value.expires_at > 0 &&
        value.expires_at <= nowSeconds
      ) {
        context.addIssue({
          code: 'custom',
          path: ['expires_at'],
          message: 'payment.manualWechat.settings.expiryInPast',
        })
      }
    })
}

export type ManualWechatSettingsFormValues = z.infer<
  ReturnType<typeof createManualWechatSettingsSchema>
>
