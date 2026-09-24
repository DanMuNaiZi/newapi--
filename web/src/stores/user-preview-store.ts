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
