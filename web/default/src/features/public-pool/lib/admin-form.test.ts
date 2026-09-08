import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { publicPoolSiteSchema } from './admin-form'

describe('public pool site validation', () => {
  test('accepts a credential-free HTTPS URL', () => {
    const result = publicPoolSiteSchema.safeParse({
      name: 'Community AI',
      url: 'https://example.com/register',
      description: 'Free community quota',
      status: 'enabled',
      sort_order: 10,
    })

    assert.equal(result.success, true)
  })

  test('rejects URLs containing embedded credentials', () => {
    const result = publicPoolSiteSchema.safeParse({
      name: 'Unsafe site',
      url: 'https://user:secret@example.com/register',
      description: '',
      status: 'enabled',
      sort_order: 0,
    })

    assert.equal(result.success, false)
  })

  test('rejects sort values outside the database integer range', () => {
    const result = publicPoolSiteSchema.safeParse({
      name: 'Community AI',
      url: 'https://example.com/register',
      description: '',
      status: 'enabled',
      sort_order: 2147483648,
    })

    assert.equal(result.success, false)
  })
})
