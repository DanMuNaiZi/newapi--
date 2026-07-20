export function reorderSubscriptionIds(
  subscriptionIds: number[],
  activeId: number,
  overId: number
): number[] {
  const activeIndex = subscriptionIds.indexOf(activeId)
  const overIndex = subscriptionIds.indexOf(overId)
  if (activeIndex < 0 || overIndex < 0 || activeIndex === overIndex) {
    return subscriptionIds
  }
  const next = [...subscriptionIds]
  const [movedId] = next.splice(activeIndex, 1)
  next.splice(overIndex, 0, movedId)
  return next
}
