import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface AuthState {
  token: string | null
  expiresAt: string | null
  setAuth: (token: string, expiresAt: string) => void
  clearAuth: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      expiresAt: null,
      setAuth: (token, expiresAt) => set({ token, expiresAt }),
      clearAuth: () => set({ token: null, expiresAt: null }),
    }),
    { name: 'calabunga-auth' }
  )
)
