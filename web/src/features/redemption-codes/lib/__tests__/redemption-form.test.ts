/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/
import type { TFunction } from 'i18next'
import { describe, expect, test } from 'vitest'

import {
  getRedemptionFormSchema,
  transformFormDataToPayload,
  type RedemptionFormValues,
} from '../redemption-form'

const t = ((key: string) => key) as TFunction

const baseValues: RedemptionFormValues = {
  name: 'campaign',
  reward_type: 'quota',
  quota_dollars: 12.5,
  subscription_plan_id: 0,
  batch: ' batch-1 ',
  source_ref: ' admin ',
  remark: ' note ',
  expired_time: undefined,
  count: 2,
}

describe('redemption form rewards', () => {
  test('quota rewards retain quota and clear subscription plan payload', () => {
    const result = transformFormDataToPayload(baseValues)

    expect(result.reward_type).toBe('quota')
    expect(result.quota).toBeGreaterThan(0)
    expect(result.subscription_plan_id).toBe(0)
    expect(result.batch).toBe('batch-1')
  })

  test('subscription rewards require a plan and omit quota payload', () => {
    const values = { ...baseValues, reward_type: 'subscription' as const }
    const invalid = getRedemptionFormSchema(t).safeParse(values)
    expect(invalid.success).toBe(false)

    const valid = transformFormDataToPayload({
      ...values,
      subscription_plan_id: 7,
    })
    expect(valid.reward_type).toBe('subscription')
    expect(valid.subscription_plan_id).toBe(7)
    expect(valid.quota).toBe(0)
  })
})
