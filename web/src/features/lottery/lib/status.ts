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
import type { TFunction } from 'i18next'

import type { LotteryPlanStatus } from '../types'

export function getLotteryPlanStatusLabel(
  t: TFunction,
  status: LotteryPlanStatus
): string {
  switch (status) {
    case 'draft':
      return t('Lottery status draft')
    case 'scheduled':
      return t('Lottery status scheduled')
    case 'open':
      return t('Lottery status open')
    case 'drawing':
      return t('Lottery status drawing')
    case 'finished':
      return t('Lottery status finished')
    case 'cancelled':
      return t('Lottery status cancelled')
  }
}

export function getLotteryRewardStatusLabel(
  t: TFunction,
  status: string
): string {
  switch (status) {
    case 'pending':
      return t('Lottery reward pending claim')
    case 'pending_auto':
      return t('Lottery reward delivery pending')
    case 'issued':
      return t('Lottery redemption code issued')
    case 'fulfilled':
      return t('Lottery reward fulfilled')
    default:
      return t('Unknown lottery reward status')
  }
}
