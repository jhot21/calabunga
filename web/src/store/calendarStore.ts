import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type View = 'month' | 'week'
type Theme = 'system' | 'light' | 'dark'

interface CalendarState {
  visibleCalendars: Record<string, boolean>
  currentView: View
  currentDate: string
  theme: Theme
  setVisible: (id: string, visible: boolean) => void
  setView: (view: View) => void
  setCurrentDate: (date: string) => void
  setTheme: (theme: Theme) => void
}

export const useCalendarStore = create<CalendarState>()(
  persist(
    (set) => ({
      visibleCalendars: {},
      currentView: 'month',
      currentDate: new Date().toISOString(),
      theme: 'system',
      setVisible: (id, visible) =>
        set((s) => ({ visibleCalendars: { ...s.visibleCalendars, [id]: visible } })),
      setView: (view) => set({ currentView: view }),
      setCurrentDate: (date) => set({ currentDate: date }),
      setTheme: (theme) => set({ theme }),
    }),
    {
      name: 'calabunga-calendar',
      partialize: (s) => ({
        visibleCalendars: s.visibleCalendars,
        currentView: s.currentView,
        theme: s.theme,
      }),
    }
  )
)
