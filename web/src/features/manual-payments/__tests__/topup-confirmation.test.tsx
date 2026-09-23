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
import { describe, expect, test, vi } from 'vitest'

import { PaymentConfirmDialog } from '@/features/wallet/components/dialogs/payment-confirm-dialog'
import { PAYMENT_TYPES } from '@/features/wallet/constants'

describe('manual WeChat top-up confirmation', () => {
  test('labels the amount as CNY before creating the pending order', () => {
    render(
      <PaymentConfirmDialog
        open
        onOpenChange={vi.fn()}
        onConfirm={vi.fn()}
        topupAmount={5}
        paymentAmount={36.5}
        paymentMethod={{
          name: 'Manual WeChat',
          type: PAYMENT_TYPES.MANUAL_WECHAT,
        }}
        calculating={false}
        processing={false}
      />
    )

    expect(screen.getByText('¥36.50 CNY')).toBeInTheDocument()
  })
})
