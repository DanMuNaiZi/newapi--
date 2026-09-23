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
import { createFileRoute, redirect } from '@tanstack/react-router'

import { ManualPayments } from '@/features/manual-payments'
import {
  ADMIN_PERMISSION_ACTIONS,
  ADMIN_PERMISSION_RESOURCES,
  hasPermission,
} from '@/lib/admin-permissions'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

export const Route = createFileRoute('/_authenticated/manual-payments/')({
  beforeLoad: () => {
    const user = useAuthStore.getState().auth.user
    const isRoot = user?.role === ROLE.SUPER_ADMIN
    const canReadTopUps = hasPermission(
      user,
      ADMIN_PERMISSION_RESOURCES.USER,
      ADMIN_PERMISSION_ACTIONS.READ
    )
    const canReadSubscriptions = hasPermission(
      user,
      ADMIN_PERMISSION_RESOURCES.SUBSCRIPTION,
      ADMIN_PERMISSION_ACTIONS.READ
    )
    if (!isRoot && !canReadTopUps && !canReadSubscriptions) {
      throw redirect({ to: '/403' })
    }
  },
  component: ManualPayments,
})
