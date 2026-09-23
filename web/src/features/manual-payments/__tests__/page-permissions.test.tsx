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
import { render, screen } from '@testing-library/react'
import { afterEach, describe, expect, test, vi } from 'vitest'

import { useAuthStore } from '@/stores/auth-store'

import { ManualPayments } from '../index'

vi.mock('../components/manual-payment-orders-table', () => ({
  ManualPaymentOrdersTable: (props: { kind: string; canOperate: boolean }) => (
    <div data-testid={`${props.kind}-orders`}>{String(props.canOperate)}</div>
  ),
}))

afterEach(() => {
  useAuthStore.getState().auth.reset()
})

describe('ManualPayments permissions', () => {
  test('shows only top-up orders and permits actions with UserQuota', () => {
    useAuthStore.getState().auth.setUser({
      id: 1,
      username: 'topup-admin',
      role: 10,
      permissions: {
        admin_permissions: {
          user: { read: true, quota: true },
        },
      },
    })

    render(<ManualPayments />)

    expect(
      screen.getByRole('tab', { name: 'admin.manualPayments.topupTab' })
    ).toBeInTheDocument()
    expect(
      screen.queryByRole('tab', {
        name: 'admin.manualPayments.subscriptionTab',
      })
    ).not.toBeInTheDocument()
    expect(screen.getByTestId('topup-orders')).toHaveTextContent('true')
  })

  test('keeps subscription actions read-only without SubscriptionOperate', () => {
    useAuthStore.getState().auth.setUser({
      id: 2,
      username: 'subscription-viewer',
      role: 10,
      permissions: {
        admin_permissions: {
          subscription: { read: true, operate: false },
        },
      },
    })

    render(<ManualPayments />)

    expect(
      screen.getByRole('tab', {
        name: 'admin.manualPayments.subscriptionTab',
      })
    ).toBeInTheDocument()
    expect(screen.queryByTestId('topup-orders')).not.toBeInTheDocument()
    expect(screen.getByTestId('subscription-orders')).toHaveTextContent('false')
  })
})
