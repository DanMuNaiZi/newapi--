import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { reorderSubscriptionIds } from './consume-priority'

describe('reorderSubscriptionIds', () => {
  test('moves the dragged subscription to the target position', () => {
    assert.deepEqual(reorderSubscriptionIds([11, 12, 13], 13, 11), [13, 11, 12])
  })

  test('keeps the existing order when either subscription is absent', () => {
    assert.deepEqual(reorderSubscriptionIds([11, 12], 99, 11), [11, 12])
  })
})
