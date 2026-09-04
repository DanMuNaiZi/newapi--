import assert from 'node:assert/strict'
import { test } from 'node:test'

import { Route } from './index'

test('parses lottery plan IDs from URL search strings', () => {
  const validateSearch = Route.options.validateSearch as unknown as {
    parse: (input: unknown) => { plan?: number }
  }

  assert.deepEqual(validateSearch.parse({ plan: '20' }), { plan: 20 })
})
