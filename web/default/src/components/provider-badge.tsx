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
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'

import { RemoteIcon } from './remote-icon'
import { StatusBadge, type StatusBadgeProps } from './status-badge'

type ProviderBadgeProps = Omit<StatusBadgeProps, 'children' | 'label'> & {
  iconKey?: string | null
  iconSize?: number
  iconUrl?: string | null
  label: string
  /** Color the label text by provider name. Set false for a neutral label. */
  colorText?: boolean
}

export function ProviderBadge({
  className,
  iconKey,
  iconSize = 14,
  iconUrl,
  label,
  colorText = true,
  ...badgeProps
}: ProviderBadgeProps) {
  const providerIcon = iconKey ? getLobeIcon(iconKey, iconSize) : null
  const icon = iconUrl ? (
    <RemoteIcon
      src={iconUrl}
      className='rounded-sm'
      style={{ width: iconSize, height: iconSize }}
      fallback={providerIcon}
    />
  ) : (
    providerIcon
  )

  return (
    <div
      data-slot='provider-badge'
      className={cn('flex max-w-full min-w-0 items-center gap-1.5', className)}
    >
      {icon && <span className='flex shrink-0 items-center'>{icon}</span>}
      <StatusBadge
        label={label}
        autoColor={colorText ? label : undefined}
        variant={colorText ? undefined : 'neutral'}
        size='sm'
        className={cn('min-w-0 shrink overflow-hidden', !icon && 'pl-0')}
        {...badgeProps}
      />
    </div>
  )
}
