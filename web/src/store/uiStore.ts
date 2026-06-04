import { create } from 'zustand'

export interface Toast {
  id: string
  message: string
  action?: { label: string; onClick: () => void }
}

interface UIState {
  toasts: Toast[]
  addToast: (message: string, action?: Toast['action']) => void
  dismissToast: (id: string) => void
}

export const useUIStore = create<UIState>((set) => ({
  toasts: [],
  addToast: (message, action) =>
    set((s) => ({
      toasts: [...s.toasts, { id: `${Date.now()}-${Math.random()}`, message, action }],
    })),
  dismissToast: (id) =>
    set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })),
}))
