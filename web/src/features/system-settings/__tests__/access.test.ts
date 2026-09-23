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
import type { TFunction } from 'i18next'
import { describe, expect, test } from 'vitest'

import { getSystemSettingsNavGroups } from '@/components/layout/config/system-settings.config'
import type { AuthUser } from '@/stores/auth-store'

import {
  canAccessSystemSettingsPath,
  getSystemSettingsAccess,
  getSystemSettingsLandingPath,
} from '../utils/access'

function userWithPermissions(
  adminPermissions: NonNullable<
    NonNullable<AuthUser['permissions']>['admin_permissions']
  >
): AuthUser {
  return {
    id: 1,
    username: 'settings-admin',
    role: 10,
    permissions: { admin_permissions: adminPermissions },
  }
}

describe('system settings access', () => {
  test('payment readers land on and can only open the payment section', () => {
    const user = userWithPermissions({ payment: { read: true } })

    expect(getSystemSettingsLandingPath(user)).toBe(
      '/system-settings/billing/payment'
    )
    expect(
      canAccessSystemSettingsPath(user, '/system-settings/billing/payment')
    ).toBe(true)
    expect(
      canAccessSystemSettingsPath(user, '/system-settings/billing/quota')
    ).toBe(false)
    expect(canAccessSystemSettingsPath(user, '/system-settings/site')).toBe(
      false
    )

    const t = ((key: string) => key) as TFunction
    expect(
      getSystemSettingsNavGroups(t, getSystemSettingsAccess(user))
    ).toEqual([
      {
        id: 'system-administration',
        title: 'System Administration',
        items: [
          {
            title: 'Billing & Payment',
            icon: expect.anything(),
            items: [
              {
                title: 'Payment Gateway',
                url: '/system-settings/billing/payment',
              },
            ],
          },
        ],
      },
    ])
  })

  test('oauth readers can open only oauth sections', () => {
    const user = userWithPermissions({ oauth: { read: true } })

    expect(getSystemSettingsLandingPath(user)).toBe(
      '/system-settings/auth/oauth'
    )
    expect(
      canAccessSystemSettingsPath(user, '/system-settings/auth/oauth')
    ).toBe(true)
    expect(
      canAccessSystemSettingsPath(user, '/system-settings/auth/custom-oauth')
    ).toBe(true)
    expect(
      canAccessSystemSettingsPath(user, '/system-settings/auth/basic-auth')
    ).toBe(false)
  })

  test('system settings readers cannot enter payment or oauth sections without their resources', () => {
    const user = userWithPermissions({ system_settings: { read: true } })

    expect(getSystemSettingsLandingPath(user)).toBe('/system-settings/site')
    expect(canAccessSystemSettingsPath(user, '/system-settings/site')).toBe(
      true
    )
    expect(
      canAccessSystemSettingsPath(user, '/system-settings/billing/quota')
    ).toBe(true)
    expect(
      canAccessSystemSettingsPath(user, '/system-settings/billing/payment')
    ).toBe(false)
    expect(
      canAccessSystemSettingsPath(user, '/system-settings/auth/oauth')
    ).toBe(false)
  })

  test('super administrators retain access to every settings section', () => {
    const user: AuthUser = {
      id: 1,
      username: 'root',
      role: 100,
    }

    expect(getSystemSettingsAccess(user)).toEqual({
      systemSettings: true,
      payment: true,
      oauth: true,
    })
    expect(
      canAccessSystemSettingsPath(user, '/system-settings/billing/payment')
    ).toBe(true)
    expect(
      canAccessSystemSettingsPath(user, '/system-settings/auth/custom-oauth')
    ).toBe(true)
  })
})
