import {
  ADMIN_PERMISSION_ACTIONS,
  ADMIN_PERMISSION_RESOURCES,
  hasPermission,
} from '@/lib/admin-permissions'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

export function useReportAccess() {
  const user = useAuthStore((state) => state.auth.user)
  const canViewGlobalReports =
    user?.role === ROLE.SUPER_ADMIN ||
    user?.role === ROLE.ADMIN ||
    hasPermission(
      user,
      ADMIN_PERMISSION_RESOURCES.REPORT,
      ADMIN_PERMISSION_ACTIONS.READ
    )

  return { canViewGlobalReports }
}
