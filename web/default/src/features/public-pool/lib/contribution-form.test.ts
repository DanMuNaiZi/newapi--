import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { publicPoolContributionSchema } from './contribution-form'

describe('public pool contribution validation', () => {
  test('requires a site and at least one contribution detail', () => {
    const result = publicPoolContributionSchema.safeParse({
      site_id: 0,
      description: '',
      proof: '',
    })

    assert.equal(result.success, false)
    if (result.success) return
    assert.equal(result.error.issues.length >= 2, true)
  })

  test('accepts a bounded manual contribution without credentials', () => {
    const result = publicPoolContributionSchema.safeParse({
      site_id: 12,
      description: 'I registered through the invitation link.',
      proof: 'Account creation completed.',
    })

    assert.equal(result.success, true)
  })
})
