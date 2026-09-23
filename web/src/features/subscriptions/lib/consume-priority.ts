export type SubscriptionDropPosition = 'before' | 'after'

export function reorderSubscriptionIds(
  subscriptionIds: number[],
  activeId: number,
  overId: number,
  position: SubscriptionDropPosition = 'before'
): number[] {
  const activeIndex = subscriptionIds.indexOf(activeId)
  const overIndex = subscriptionIds.indexOf(overId)
  if (activeIndex < 0 || overIndex < 0 || activeIndex === overIndex) {
    return subscriptionIds
  }
  const next = [...subscriptionIds]
  const [movedId] = next.splice(activeIndex, 1)
  const targetIndex = next.indexOf(overId)
  next.splice(targetIndex + (position === 'after' ? 1 : 0), 0, movedId)
  return next
}
