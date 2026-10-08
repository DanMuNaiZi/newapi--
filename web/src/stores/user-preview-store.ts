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
import { create } from 'zustand'

export interface UserPreviewTarget {
  id: number
  username: string
  display_name?: string
}

interface UserPreviewState {
  token: string | null
  expiresAt: number | null
  target: UserPreviewTarget | null
  start: (token: string, expiresAt: number, target: UserPreviewTarget) => void
  clear: () => void
}

export const useUserPreviewStore = create<UserPreviewState>()((set) => ({
  token: null,
  expiresAt: null,
  target: null,
  start: (token, expiresAt, target) => set({ token, expiresAt, target }),
  clear: () => set({ token: null, expiresAt: null, target: null }),
}))
