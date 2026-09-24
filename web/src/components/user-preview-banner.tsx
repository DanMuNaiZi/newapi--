import { EyeOff } from 'lucide-react'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'

import { Button } from '@/components/ui/button'
import { useUserPreviewStore } from '@/stores/user-preview-store'

export function UserPreviewBanner() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const token = useUserPreviewStore((state) => state.token)
  const expiresAt = useUserPreviewStore((state) => state.expiresAt)
  const target = useUserPreviewStore((state) => state.target)
  const clearPreview = useUserPreviewStore((state) => state.clear)

  useEffect(() => {
    if (!expiresAt) return
    const remaining = Math.max(0, expiresAt * 1000 - Date.now())
    const timeout = window.setTimeout(clearPreview, remaining)
    return () => window.clearTimeout(timeout)
  }, [clearPreview, expiresAt])

  if (!token || !target || !expiresAt) return null

  const exitPreview = () => {
    clearPreview()
    queryClient.clear()
    void navigate({ to: '/dashboard', replace: true })
  }

  return (
    <div className='bg-amber-500 px-4 py-2 text-center text-sm text-black'>
      <span>
        {t('Viewing as {{username}} until {{time}}', {
          username: target.display_name || target.username,
          time: new Date(expiresAt * 1000).toLocaleTimeString(),
        })}
      </span>{' '}
      <Button
        type='button'
        variant='outline'
        size='sm'
        className='ml-3 border-black/30 bg-transparent text-black hover:bg-black/10'
        onClick={exitPreview}
      >
        <EyeOff />
        {t('Exit preview')}
      </Button>
    </div>
  )
}
