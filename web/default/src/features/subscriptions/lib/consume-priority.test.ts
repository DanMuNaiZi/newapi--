import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { reorderSubscriptionIds } from './consume-priority'

describe('reorderSubscriptionIds', () => {
  test('moves the dragged subscription to the target position', () => {
    assert.deepEqual(reorderSubscriptionIds([11, 12, 13], 13, 11), [13, 11, 12])
  })

  test('places the dragged subscription after the target when dropped on its lower half', () => {
    assert.deepEqual(
      reorderSubscriptionIds([11, 12, 13], 11, 12, 'after'),
      [12, 11, 13]
    )
  })

  test('keeps the existing order when either subscription is absent', () => {
    assert.deepEqual(reorderSubscriptionIds([11, 12], 99, 11), [11, 12])
  })
})
