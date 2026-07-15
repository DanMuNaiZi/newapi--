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
import {
  isUserDeleted,
  USER_ROLE,
  USER_STATUS,
} from '@/features/users/constants'
import type { User } from '@/features/users/types'

export function isEligibleLotteryUser(user: User): boolean {
  return (
    user.role === USER_ROLE.USER &&
    user.status === USER_STATUS.ENABLED &&
    !isUserDeleted(user)
  )
}

export function mergeLotteryUsers(
  selectedUsers: User[],
  searchResults: User[]
): User[] {
  const usersById = new Map<number, User>()
  for (const item of selectedUsers) {
    usersById.set(item.id, item)
  }
  for (const item of searchResults) {
    if (!usersById.has(item.id)) {
      usersById.set(item.id, item)
    }
  }
  return [...usersById.values()]
}

export function formatLotteryUserLabel(user: User): string {
  const name = user.display_name.trim() || user.username
  return `${name} (${user.username}) | #${user.id} | ${user.group}`
}
