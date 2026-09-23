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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, test, vi } from 'vitest'

import { SubscriptionPurchaseDialog } from '@/features/subscriptions/components/dialogs/subscription-purchase-dialog'
import type { PlanRecord } from '@/features/subscriptions/types'

import { createManualWechatSubscription } from '../api'
import type { ManualWechatOrder } from '../types'

vi.mock('../api', () => ({
  createManualWechatSubscription: vi.fn(),
}))

vi.mock('@/hooks/use-system-config', () => ({
  useSystemConfig: () => ({
    currency: { quotaPerUnit: 500_000 },
  }),
}))

const plan: PlanRecord = {
  plan: {
    id: 10,
    title: 'Pro',
    price_amount: 36.5,
    currency: 'CNY',
    duration_unit: 'month',
    duration_value: 1,
    quota_reset_period: 'monthly',
    enabled: true,
    sort_order: 1,
    allow_balance_pay: true,
    allow_wallet_overflow: true,
    max_purchase_per_user: 0,
    total_amount: 1_000,
  },
}

const order: ManualWechatOrder = {
  kind: 'subscription',
  trade_no: 'MWX-S-ABC123456789',
  user_id: 123,
  amount_cny: 36.5,
  currency: 'CNY',
  status: 'pending',
  created_at: 1_780_000_000,
  reused: false,
  purpose: { plan_id: 10, plan_title: 'Pro' },
  qr_code_image: 'data:image/png;base64,AA==',
  qr_code_expires_at: 1_900_000_000,
  instructions: 'Include the order number in the payment remark.',
}

describe('manual WeChat subscription payment', () => {
  test('creates one pending order on a double click and opens the QR dialog', async () => {
    const user = userEvent.setup()
    let resolveRequest:
      | ((value: { success: true; data: ManualWechatOrder }) => void)
      | undefined
    vi.mocked(createManualWechatSubscription).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveRequest = resolve
        })
    )
    const onOpenChange = vi.fn()
    const queryClient = new QueryClient({
      defaultOptions: { mutations: { retry: false } },
    })
    render(
      <QueryClientProvider client={queryClient}>
        <SubscriptionPurchaseDialog
          open
          onOpenChange={onOpenChange}
          plan={plan}
          enableManualWechat
          userQuota={0}
        />
      </QueryClientProvider>
    )

    await user.dblClick(
      screen.getByRole('button', {
        name: 'payment.manualWechat.displayName',
      })
    )

    expect(createManualWechatSubscription).toHaveBeenCalledTimes(1)
    expect(vi.mocked(createManualWechatSubscription).mock.calls[0]?.[0]).toBe(
      10
    )

    await act(async () => {
      resolveRequest?.({ success: true, data: order })
    })

    expect(await screen.findAllByText(order.trade_no)).toHaveLength(2)
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })
})
