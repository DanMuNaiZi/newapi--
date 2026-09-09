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

export const USER_PREVIEW_TOKEN_KEY = 'new-api-user-preview-token'
export const USER_PREVIEW_SESSION_KEY = 'new-api-user-preview-session'

export interface UserPreviewSession {
  actor_user_id: number
  target_user_id: number
  username: string
  display_name?: string
  expires_at: number
}

export function getUserPreviewToken(): string | null {
  if (typeof window === 'undefined') return null
  const token = window.sessionStorage.getItem(USER_PREVIEW_TOKEN_KEY)
  const session = getUserPreviewSession()
  if (!token || !session) return null
  return token
}

export function getUserPreviewSession(): UserPreviewSession | null {
  if (typeof window === 'undefined') return null
  const raw = window.sessionStorage.getItem(USER_PREVIEW_SESSION_KEY)
  if (!raw) return null
  try {
    const session = JSON.parse(raw) as UserPreviewSession
    if (
      !session.target_user_id ||
      !session.actor_user_id ||
      !session.username ||
      !session.expires_at
    ) {
      clearUserPreviewSession()
      return null
    }
    return session
  } catch {
    clearUserPreviewSession()
    return null
  }
}

export function isUserPreviewActive(): boolean {
  return Boolean(getUserPreviewToken())
}

export function clearUserPreviewSession(): void {
  if (typeof window === 'undefined') return
  window.sessionStorage.removeItem(USER_PREVIEW_TOKEN_KEY)
  window.sessionStorage.removeItem(USER_PREVIEW_SESSION_KEY)
}
