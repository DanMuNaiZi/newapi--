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
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import {
  getManualWechatPaymentSettings,
  updateManualWechatPaymentSettings,
} from '../api'
import { ManualWechatSettingsCard } from '../components/manual-wechat-settings-card'

vi.mock('../api', () => ({
  getManualWechatPaymentSettings: vi.fn(),
  updateManualWechatPaymentSettings: vi.fn(),
}))

const emptySettings = {
  enabled: false,
  qr_code_image: '',
  expires_at: 0,
  instructions: 'Include the order number in the payment remark.',
}

function renderSettingsCard() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <ManualWechatSettingsCard />
    </QueryClientProvider>
  )
}

describe('ManualWechatSettingsCard', () => {
  beforeEach(() => {
    useAuthStore.getState().auth.setUser({
      id: 1,
      username: 'root',
      role: ROLE.SUPER_ADMIN,
    })
    vi.mocked(getManualWechatPaymentSettings).mockResolvedValue({
      success: true,
      data: emptySettings,
    })
    vi.mocked(updateManualWechatPaymentSettings).mockResolvedValue({
      success: true,
      data: emptySettings,
    })
  })

  afterEach(() => {
    useAuthStore.getState().auth.reset()
  })

  test('previews an uploaded QR code and clearing it also disables payment', async () => {
    const user = userEvent.setup()
    renderSettingsCard()

    const fileInput = await screen.findByLabelText(
      'payment.manualWechat.settings.qrCodeFile'
    )
    await user.upload(
      fileInput,
      new File(['png'], 'wechat.png', { type: 'image/png' })
    )

    expect(
      await screen.findByRole('img', {
        name: 'payment.manualWechat.settings.qrCodePreview',
      })
    ).toHaveAttribute('src', 'data:image/png;base64,cG5n')

    const enabled = screen.getByRole('switch', {
      name: 'payment.manualWechat.settings.enabled',
    })
    await user.click(enabled)
    expect(enabled).toHaveAttribute('aria-checked', 'true')

    await user.click(
      screen.getByRole('button', {
        name: 'payment.manualWechat.settings.clearQrCode',
      })
    )

    expect(
      screen.queryByRole('img', {
        name: 'payment.manualWechat.settings.qrCodePreview',
      })
    ).not.toBeInTheDocument()
    expect(enabled).toHaveAttribute('aria-checked', 'false')
  })

  test('does not submit an enabled configuration without an image', async () => {
    const user = userEvent.setup()
    renderSettingsCard()

    const enabled = await screen.findByRole('switch', {
      name: 'payment.manualWechat.settings.enabled',
    })
    await user.click(enabled)
    await user.click(
      screen.getByRole('button', {
        name: 'payment.manualWechat.settings.save',
      })
    )

    expect(
      await screen.findByText('payment.manualWechat.settings.imageRequired')
    ).toBeInTheDocument()
    await waitFor(() =>
      expect(updateManualWechatPaymentSettings).not.toHaveBeenCalled()
    )
  })

  test('keeps payment settings read-only without PaymentWrite', async () => {
    useAuthStore.getState().auth.setUser({
      id: 2,
      username: 'payment-reader',
      role: 10,
      permissions: {
        admin_permissions: {
          payment: { read: true, write: false },
        },
      },
    })

    renderSettingsCard()

    expect(
      await screen.findByRole('switch', {
        name: 'payment.manualWechat.settings.enabled',
      })
    ).toHaveAttribute('aria-disabled', 'true')
    expect(
      screen.getByRole('button', {
        name: 'payment.manualWechat.settings.uploadQrCode',
      })
    ).toBeDisabled()
    expect(
      screen.getByLabelText('payment.manualWechat.settings.expiresAt')
    ).toBeDisabled()
    expect(
      screen.getByLabelText('payment.manualWechat.settings.instructions')
    ).toBeDisabled()
    expect(
      screen.getByRole('button', {
        name: 'payment.manualWechat.settings.save',
      })
    ).toBeDisabled()
  })
})
