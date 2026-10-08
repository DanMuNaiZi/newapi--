/* Copyright (C) 2023-2026 QuantumNous */
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'
import { useUserPreviewStore } from '@/stores/user-preview-store'

import { Profile } from '..'
import type { UserProfile } from '../types'

const target: UserProfile = {
  id: 2,
  username: 'target',
  display_name: 'Target User',
  role: 1,
  group: 'default',
  quota: 1_000_000,
  used_quota: 500_000,
  request_count: 3,
  status: 1,
  aff_count: 0,
  aff_quota: 0,
  aff_history_quota: 0,
  created_time: 0,
}

beforeEach(() => {
  localStorage.clear()
  useAuthStore.getState().auth.setUser({ id: 1, username: 'root', role: 100 })
  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/status') {
      return { data: { success: true, data: { checkin_enabled: true } } }
    }
    if (url === '/api/user/checkin') {
      return { data: { success: true, data: { records: [] } } }
    }
    return { data: { success: true, data: target } }
  })
})

afterEach(() => {
  vi.restoreAllMocks()
  useUserPreviewStore.getState().clear()
  useAuthStore.getState().auth.reset()
})

test('preview displays the target balance without settings or check-in controls', async () => {
  useUserPreviewStore.getState().start('preview-token', 2_000_000_000, target)
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <Profile />
    </QueryClientProvider>
  )

  expect(
    await screen.findByRole('heading', { name: 'Target User' })
  ).toBeVisible()
  expect(screen.getByText('Current Balance')).toBeVisible()
  expect(screen.getByText('API Requests')).toBeVisible()
  expect(screen.queryByText('Settings & Preferences')).not.toBeInTheDocument()
  expect(screen.queryByText('Language Preferences')).not.toBeInTheDocument()
  expect(screen.queryByText('Daily Check-in')).not.toBeInTheDocument()
  expect(
    screen.queryByText('Sidebar Personal Settings')
  ).not.toBeInTheDocument()
  expect(useAuthStore.getState().auth.user?.id).toBe(1)
  await waitFor(() => expect(api.get).toHaveBeenCalledWith('/api/user/self'))
  expect(
    vi
      .mocked(api.get)
      .mock.calls.some(([url]) => url.startsWith('/api/user/checkin'))
  ).toBe(false)
  client.clear()
})

test('the ordinary profile keeps editable settings and language preferences', async () => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <Profile />
    </QueryClientProvider>
  )

  expect(
    await screen.findByRole('heading', { name: 'Target User' })
  ).toBeVisible()
  expect(screen.getByText('Settings & Preferences')).toBeVisible()
  expect(screen.getByText('Language Preferences')).toBeVisible()
  expect(screen.getByText('Sidebar Personal Settings')).toBeVisible()
  client.clear()
})
