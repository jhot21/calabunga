import { useEffect } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useAuthStore } from './store/authStore'
import { useCalendarStore } from './store/calendarStore'
import { configureApiAuth } from './lib/api'
import LoginPage from './pages/LoginPage'
import CalendarPage from './pages/CalendarPage'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const token = useAuthStore((s) => s.token)
  if (!token) return <Navigate to="/login" replace />
  return <>{children}</>
}

export default function App() {
  const theme = useCalendarStore((s) => s.theme)

  useEffect(() => {
    configureApiAuth(
      () => useAuthStore.getState().token,
      (newToken) => {
        const { expiresAt } = useAuthStore.getState()
        useAuthStore.getState().setAuth(newToken, expiresAt ?? '')
      }
    )
  }, [])

  useEffect(() => {
    const apply = () => {
      if (theme === 'system') {
        const dark = window.matchMedia('(prefers-color-scheme: dark)').matches
        document.documentElement.setAttribute('data-theme', dark ? 'dark' : 'light')
      } else {
        document.documentElement.setAttribute('data-theme', theme)
      }
    }
    apply()
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    mq.addEventListener('change', apply)
    return () => mq.removeEventListener('change', apply)
  }, [theme])

  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route
          path="/"
          element={
            <ProtectedRoute>
              <CalendarPage />
            </ProtectedRoute>
          }
        />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
