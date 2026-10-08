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
import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { EyeOff } from 'lucide-react'
import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { toIntlLocale } from '@/i18n/languages'
import { useAuthStore } from '@/stores/auth-store'
import { useUserPreviewStore } from '@/stores/user-preview-store'

export function UserPreviewBanner() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const token = useUserPreviewStore((state) => state.token)
  const expiresAt = useUserPreviewStore((state) => state.expiresAt)
  const target = useUserPreviewStore((state) => state.target)
  const clearPreview = useUserPreviewStore((state) => state.clear)
  const previousToken = useRef(token)

  useEffect(() => {
    const ended = previousToken.current && !token
    previousToken.current = token
    if (!ended) return
    queryClient.clear()
    if (useAuthStore.getState().auth.accessToken) {
      void navigate({ to: '/dashboard', replace: true })
    }
  }, [token, queryClient, navigate])

  useEffect(() => {
    if (!expiresAt) return
    const remaining = Math.max(0, expiresAt * 1000 - Date.now())
    const timeout = window.setTimeout(clearPreview, remaining)
    return () => window.clearTimeout(timeout)
  }, [clearPreview, expiresAt])

  if (!token || !target || !expiresAt) return null

  return (
    <div className='bg-amber-500 px-4 py-2 text-center text-sm text-black'>
      <span>
        {t('Viewing as {{username}} until {{time}}', {
          username: target.display_name || target.username,
          time: new Date(expiresAt * 1000).toLocaleTimeString(
            toIntlLocale(i18n.resolvedLanguage || i18n.language)
          ),
        })}
      </span>{' '}
      <span>{t('User preview is read-only')}</span>
      <Button
        type='button'
        variant='outline'
        size='sm'
        className='ml-3 border-black/30 bg-transparent text-black hover:bg-black/10'
        onClick={clearPreview}
      >
        <EyeOff />
        {t('Exit preview')}
      </Button>
    </div>
  )
}
