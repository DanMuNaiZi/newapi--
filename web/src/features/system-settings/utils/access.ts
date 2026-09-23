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
import {
  ADMIN_PERMISSION_ACTIONS,
  ADMIN_PERMISSION_RESOURCES,
  hasPermission,
} from '@/lib/admin-permissions'
import type { AuthUser } from '@/stores/auth-store'

export type SystemSettingsAccess = {
  systemSettings: boolean
  payment: boolean
  oauth: boolean
}

export type SystemSettingsLandingPath =
  | '/system-settings/site'
  | '/system-settings/billing/payment'
  | '/system-settings/auth/oauth'

export function getSystemSettingsAccess(
  user: AuthUser | null | undefined
): SystemSettingsAccess {
  return {
    systemSettings: hasPermission(
      user,
      ADMIN_PERMISSION_RESOURCES.SYSTEM_SETTINGS,
      ADMIN_PERMISSION_ACTIONS.READ
    ),
    payment: hasPermission(
      user,
      ADMIN_PERMISSION_RESOURCES.PAYMENT,
      ADMIN_PERMISSION_ACTIONS.READ
    ),
    oauth: hasPermission(
      user,
      ADMIN_PERMISSION_RESOURCES.OAUTH,
      ADMIN_PERMISSION_ACTIONS.READ
    ),
  }
}

export function getSystemSettingsLandingPath(
  user: AuthUser | null | undefined
): SystemSettingsLandingPath | null {
  const access = getSystemSettingsAccess(user)
  if (access.systemSettings) return '/system-settings/site'
  if (access.payment) return '/system-settings/billing/payment'
  if (access.oauth) return '/system-settings/auth/oauth'
  return null
}

export function canAccessSystemSettingsPath(
  user: AuthUser | null | undefined,
  pathname: string
): boolean {
  const access = getSystemSettingsAccess(user)
  const normalizedPath = pathname.replace(/\/+$/, '') || '/'

  if (normalizedPath === '/system-settings') {
    return access.systemSettings || access.payment || access.oauth
  }
  if (normalizedPath === '/system-settings/billing') {
    return access.systemSettings || access.payment
  }
  if (normalizedPath === '/system-settings/auth') {
    return access.systemSettings || access.oauth
  }
  if (normalizedPath === '/system-settings/billing/payment') {
    return access.payment
  }
  if (
    normalizedPath === '/system-settings/auth/oauth' ||
    normalizedPath === '/system-settings/auth/custom-oauth'
  ) {
    return access.oauth
  }
  return normalizedPath.startsWith('/system-settings/') && access.systemSettings
}
