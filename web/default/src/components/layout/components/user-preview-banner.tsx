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

import { Eye, LogOut } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  clearUserPreviewSession,
  getUserPreviewSession,
} from '@/lib/user-preview'

export function UserPreviewBanner() {
  const { t } = useTranslation()
  const [session] = useState(() => getUserPreviewSession())

  if (!session) return null

  const exitPreview = () => {
    clearUserPreviewSession()
    window.location.replace('/dashboard/overview')
  }

  return (
    <div className='flex h-10 shrink-0 items-center justify-center gap-3 border-b border-amber-500/40 bg-amber-500/10 px-4 text-sm text-amber-950 dark:text-amber-100'>
      <Eye className='size-4' aria-hidden='true' />
      <span className='truncate font-medium'>
        {t('Read-only preview of user {{username}}', {
          username: session.display_name || session.username,
        })}
      </span>
      <Button
        type='button'
        variant='outline'
        size='sm'
        className='h-7'
        onClick={exitPreview}
      >
        <LogOut className='size-3.5' />
        {t('Exit preview')}
      </Button>
    </div>
  )
}
