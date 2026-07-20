import {
  ADMIN_PERMISSION_ACTIONS,
  ADMIN_PERMISSION_RESOURCES,
  hasPermission,
} from '@/lib/admin-permissions'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

export function useUsageLogAccess() {
  const user = useAuthStore((state) => state.auth.user)
  const isRoot = user?.role === ROLE.SUPER_ADMIN
  const isAdministrator = user?.role === ROLE.ADMIN
  const canViewGlobalLogs =
    isRoot ||
    isAdministrator ||
    hasPermission(
      user,
      ADMIN_PERMISSION_RESOURCES.USAGE_LOG,
      ADMIN_PERMISSION_ACTIONS.READ
    )
  const canViewActualModel =
    isRoot ||
    isAdministrator ||
    hasPermission(user, ADMIN_PERMISSION_RESOURCES.USAGE_LOG, 'actual_model_view')

  return { canViewGlobalLogs, canViewActualModel }
}
