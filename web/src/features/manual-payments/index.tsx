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
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  ADMIN_PERMISSION_ACTIONS,
  ADMIN_PERMISSION_RESOURCES,
  hasPermission,
} from '@/lib/admin-permissions'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { ManualPaymentOrdersTable } from './components/manual-payment-orders-table'

export function ManualPayments() {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const isRoot = user?.role === ROLE.SUPER_ADMIN
  const canReadTopUps =
    isRoot ||
    hasPermission(
      user,
      ADMIN_PERMISSION_RESOURCES.USER,
      ADMIN_PERMISSION_ACTIONS.READ
    )
  const canReadSubscriptions =
    isRoot ||
    hasPermission(
      user,
      ADMIN_PERMISSION_RESOURCES.SUBSCRIPTION,
      ADMIN_PERMISSION_ACTIONS.READ
    )
  const canOperateTopUps =
    isRoot ||
    hasPermission(
      user,
      ADMIN_PERMISSION_RESOURCES.USER,
      ADMIN_PERMISSION_ACTIONS.QUOTA
    )
  const canOperateSubscriptions =
    isRoot ||
    hasPermission(
      user,
      ADMIN_PERMISSION_RESOURCES.SUBSCRIPTION,
      ADMIN_PERMISSION_ACTIONS.OPERATE
    )
  const defaultTab = canReadTopUps ? 'topup' : 'subscription'

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>
        {t('admin.manualPayments.title')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <Tabs defaultValue={defaultTab} className='flex h-full flex-col gap-4'>
          <TabsList className='w-fit'>
            {canReadTopUps ? (
              <TabsTrigger value='topup'>
                {t('admin.manualPayments.topupTab')}
              </TabsTrigger>
            ) : null}
            {canReadSubscriptions ? (
              <TabsTrigger value='subscription'>
                {t('admin.manualPayments.subscriptionTab')}
              </TabsTrigger>
            ) : null}
          </TabsList>
          {canReadTopUps ? (
            <TabsContent value='topup' className='min-h-0 flex-1'>
              <ManualPaymentOrdersTable
                kind='topup'
                canOperate={canOperateTopUps}
              />
            </TabsContent>
          ) : null}
          {canReadSubscriptions ? (
            <TabsContent value='subscription' className='min-h-0 flex-1'>
              <ManualPaymentOrdersTable
                kind='subscription'
                canOperate={canOperateSubscriptions}
              />
            </TabsContent>
          ) : null}
        </Tabs>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
