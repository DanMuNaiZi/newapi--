import { createFileRoute } from '@tanstack/react-router'

import { PublicPool } from '@/features/public-pool'

export const Route = createFileRoute('/_authenticated/public-pool/')({
  component: PublicPool,
})
