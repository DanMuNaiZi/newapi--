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
import { useEffect, useState, type CSSProperties, type ReactNode } from 'react'

import { cn } from '@/lib/utils'

type RemoteIconProps = {
  alt?: string
  className?: string
  fallback: ReactNode
  imageClassName?: string
  src?: null | string
  style?: CSSProperties
}

export function RemoteIcon(props: RemoteIconProps) {
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    setFailed(false)
  }, [props.src])

  return (
    <span
      className={cn(
        'inline-flex shrink-0 items-center justify-center overflow-hidden',
        props.className
      )}
      style={props.style}
    >
      {props.src && !failed ? (
        <img
          src={props.src}
          alt={props.alt ?? ''}
          className={cn('h-full w-full object-cover', props.imageClassName)}
          loading='lazy'
          referrerPolicy='no-referrer'
          onError={() => setFailed(true)}
        />
      ) : (
        props.fallback
      )}
    </span>
  )
}
