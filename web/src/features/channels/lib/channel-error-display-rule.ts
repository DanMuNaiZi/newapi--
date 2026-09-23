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

import type { ChannelErrorRecord } from '../types'
import { MAX_UPSTREAM_ERROR_MESSAGE_LENGTH } from './channel-form'

export const channelErrorDisplayRuleSchema = z
  .object({
    display_mode: z.enum(['inherit', 'generic', 'original', 'custom']),
    display_status_code: z.number().int().optional(),
    display_message: z.string().optional(),
  })
  .superRefine((value, ctx) => {
    if (value.display_mode !== 'custom') return
    if (
      value.display_status_code === undefined ||
      value.display_status_code < 400 ||
      value.display_status_code > 599
    ) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['display_status_code'],
        message: 'Status code must be between 400 and 599',
      })
    }
    const message = value.display_message?.trim() || ''
    if (!message) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['display_message'],
        message: 'Client error message is required',
      })
    } else if ([...message].length > MAX_UPSTREAM_ERROR_MESSAGE_LENGTH) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['display_message'],
        message: `Client error message must be ${MAX_UPSTREAM_ERROR_MESSAGE_LENGTH} characters or fewer`,
      })
    }
  })

export type ChannelErrorDisplayRuleFormValues = z.infer<
  typeof channelErrorDisplayRuleSchema
>

export function channelErrorDisplayRuleDefaults(
  record?: ChannelErrorRecord | null
): ChannelErrorDisplayRuleFormValues {
  return {
    display_mode: record?.display_mode || 'inherit',
    display_status_code: record?.display_status_code || 503,
    display_message: record?.display_message || '',
  }
}
