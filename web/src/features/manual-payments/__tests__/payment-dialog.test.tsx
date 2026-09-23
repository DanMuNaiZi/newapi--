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
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, test, vi } from 'vitest'

import { ManualWechatPaymentDialog } from '../components/manual-wechat-payment-dialog'
import { isManualWechatQRCodeExpired } from '../lib/manual-wechat-order'
import type { ManualWechatOrder } from '../types'

const order: ManualWechatOrder = {
  kind: 'topup',
  trade_no: 'MWX-T-ABC123456789',
  user_id: 123,
  amount_cny: 36.5,
  currency: 'CNY',
  status: 'pending',
  created_at: 1_780_000_000,
  reused: false,
  purpose: { topup_amount: 5 },
  qr_code_image: 'data:image/png;base64,AA==',
  qr_code_expires_at: 1_800_000_000,
  instructions: 'Include the order number in the payment remark.',
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('ManualWechatPaymentDialog', () => {
  test('shows the exact order details needed for manual reconciliation', () => {
    render(
      <ManualWechatPaymentDialog open onOpenChange={vi.fn()} order={order} />
    )

    expect(screen.getByText('¥36.50 CNY')).toBeInTheDocument()
    expect(screen.getAllByText(order.trade_no)).toHaveLength(2)
    expect(screen.getByText('123')).toBeInTheDocument()
    expect(screen.getByText('5')).toBeInTheDocument()
    expect(
      screen.getByRole('img', { name: 'payment.manualWechat.qrCodeAlt' })
    ).toHaveAttribute('src', order.qr_code_image)
  })

  test('closes without marking the order successful when the user says payment is complete', async () => {
    const user = userEvent.setup()
    const onOpenChange = vi.fn()
    render(
      <ManualWechatPaymentDialog
        open
        onOpenChange={onOpenChange}
        order={order}
      />
    )

    await user.click(
      screen.getByRole('button', {
        name: 'payment.manualWechat.completedPayment',
      })
    )

    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  test('downloads the validated QR image with the order number as its filename', async () => {
    const user = userEvent.setup()
    const click = vi
      .spyOn(HTMLAnchorElement.prototype, 'click')
      .mockImplementation(() => undefined)
    render(
      <ManualWechatPaymentDialog open onOpenChange={vi.fn()} order={order} />
    )

    await user.click(
      screen.getByRole('button', {
        name: 'payment.manualWechat.saveQrCode',
      })
    )

    expect(click).toHaveBeenCalledTimes(1)
  })

  test('stops payment guidance when the configured QR code has expired', () => {
    render(
      <ManualWechatPaymentDialog
        open
        onOpenChange={vi.fn()}
        order={{ ...order, qr_code_expires_at: 1 }}
      />
    )

    expect(
      screen.getByText('payment.manualWechat.qrExpired')
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', {
        name: 'payment.manualWechat.completedPayment',
      })
    ).toBeDisabled()
    expect(
      screen.getByRole('button', {
        name: 'payment.manualWechat.saveQrCode',
      })
    ).toBeDisabled()
  })
})

describe('isManualWechatQRCodeExpired', () => {
  test('treats zero as no expiry and expires at the configured second', () => {
    expect(isManualWechatQRCodeExpired(0, 100)).toBe(false)
    expect(isManualWechatQRCodeExpired(101, 100)).toBe(false)
    expect(isManualWechatQRCodeExpired(100, 100)).toBe(true)
    expect(isManualWechatQRCodeExpired(99, 100)).toBe(true)
  })
})
