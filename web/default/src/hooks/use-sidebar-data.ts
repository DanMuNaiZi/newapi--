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
  Activity,
  Box,
  CreditCard,
  FileText,
  FlaskConical,
  HandHeart,
  Key,
  LayoutDashboard,
  ListTodo,
  Medal,
  Megaphone,
  MessageSquare,
  Radio,
  ServerCog,
  Settings,
  Ticket,
  Trophy,
  User,
  Users,
  Wallet,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import type { NavItem, SidebarData } from '@/components/layout/types'
import {
  ADMIN_PERMISSION_ACTIONS,
  ADMIN_PERMISSION_RESOURCES,
  hasPermission,
} from '@/lib/admin-permissions'
import { ROLE } from '@/lib/roles'
import { isUserPreviewActive } from '@/lib/user-preview'
import { useAuthStore } from '@/stores/auth-store'

/**
 * Root navigation groups for the application sidebar.
 *
 * These are shown when the URL does not match any nested sidebar view
 * registered in `layout/lib/sidebar-view-registry.ts`.
 */
export function useSidebarData(): SidebarData {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const readOnlyPreview = isUserPreviewActive()
  const isRoot = user?.role === ROLE.SUPER_ADMIN
  const can = (resource: string, action = ADMIN_PERMISSION_ACTIONS.READ) =>
    isRoot || hasPermission(user, resource, action)

  const adminItems: NavItem[] = []
  if (can(ADMIN_PERMISSION_RESOURCES.CHANNEL)) {
    adminItems.push({
      title: t('Channels'),
      url: '/channels',
      icon: Radio,
    })
  }
  if (
    can(ADMIN_PERMISSION_RESOURCES.MODEL) ||
    can(ADMIN_PERMISSION_RESOURCES.DEPLOYMENT)
  ) {
    adminItems.push({
      title: t('Models'),
      url: '/models/metadata',
      icon: Box,
    })
  }
  if (can(ADMIN_PERMISSION_RESOURCES.USER)) {
    adminItems.push({
      title: t('Users'),
      url: '/users',
      icon: Users,
    })
  }
  if (can(ADMIN_PERMISSION_RESOURCES.REDEMPTION)) {
    adminItems.push({
      title: t('Redemption Codes'),
      url: '/redemption-codes',
      icon: Ticket,
    })
  }
  if (can(ADMIN_PERMISSION_RESOURCES.LOTTERY)) {
    adminItems.push({
      title: t('Lottery Management'),
      url: '/lotteries/admin',
      icon: Trophy,
    })
  }
  if (can(ADMIN_PERMISSION_RESOURCES.PUBLIC_POOL)) {
    adminItems.push({
      title: t('Public pool management'),
      url: '/public-pool/admin',
      icon: HandHeart,
    })
  }
  if (can(ADMIN_PERMISSION_RESOURCES.REFERRAL_CAMPAIGN)) {
    adminItems.push({
      title: t('Referral campaigns'),
      url: '/referral-campaigns/admin',
      icon: Megaphone,
    })
  }
  if (can(ADMIN_PERMISSION_RESOURCES.SUBSCRIPTION)) {
    adminItems.push({
      title: t('Subscriptions'),
      url: '/subscriptions',
      icon: CreditCard,
    })
  }
  if (can(ADMIN_PERMISSION_RESOURCES.SYSTEM_INFO)) {
    adminItems.push({
      title: t('System Info'),
      url: '/system-info',
      icon: ServerCog,
    })
  }
  if (
    can(ADMIN_PERMISSION_RESOURCES.SYSTEM_SETTINGS) ||
    can(ADMIN_PERMISSION_RESOURCES.PAYMENT) ||
    can(ADMIN_PERMISSION_RESOURCES.OAUTH)
  ) {
    adminItems.push({
      title: t('System Settings'),
      url: '/system-settings/site',
      activeUrls: ['/system-settings'],
      icon: Settings,
    })
  }

  const generalItems: NavItem[] = [
    {
      title: t('Overview'),
      url: '/dashboard/overview',
      icon: Activity,
    },
    {
      title: t('Dashboard'),
      url: '/dashboard/models',
      icon: LayoutDashboard,
    },
  ]
  if (!readOnlyPreview) {
    generalItems.push({
      title: t('API Keys'),
      url: '/keys',
      icon: Key,
    })
  }
  generalItems.push(
    {
      title: t('Usage Logs'),
      url: '/usage-logs/common',
      icon: FileText,
    },
    {
      title: t('Usage Rankings'),
      url: '/usage-rankings',
      icon: Medal,
    },
    {
      title: t('Public Pool'),
      url: '/public-pool',
      icon: HandHeart,
    }
  )
  if (
    isRoot ||
    user?.role === ROLE.ADMIN ||
    can(ADMIN_PERMISSION_RESOURCES.TASK_LOG)
  ) {
    generalItems.push({
      title: t('Task Logs'),
      url: '/usage-logs/task',
      activeUrls: ['/usage-logs/drawing'],
      configUrls: ['/usage-logs/drawing', '/usage-logs/task'],
      icon: ListTodo,
    })
  }

  const navGroups: SidebarData['navGroups'] = []
  if (!readOnlyPreview) {
    navGroups.push({
      id: 'chat',
      title: t('Chat'),
      items: [
        {
          title: t('Playground'),
          url: '/playground',
          icon: FlaskConical,
        },
        {
          title: t('Chat'),
          icon: MessageSquare,
          type: 'chat-presets',
        },
      ],
    })
  }
  navGroups.push(
    {
      id: 'general',
      title: t('General'),
      items: generalItems,
    },
    {
      id: 'personal',
      title: t('Personal'),
      items: [
        {
          title: t('Wallet'),
          url: '/wallet',
          icon: Wallet,
        },
        {
          title: t('Profile'),
          url: '/profile',
          icon: User,
        },
      ],
    },
    {
      id: 'admin',
      title: t('Admin'),
      items: adminItems,
    }
  )

  return {
    navGroups,
  }
}
