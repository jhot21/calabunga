import { useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api, ApiError } from '../lib/api'
import { useAuthStore } from '../store/authStore'
import { useUIStore } from '../store/uiStore'
import type { ConnectRequest } from '../types/api'

export function useAuth() {
  const { token, expiresAt, setAuth, clearAuth } = useAuthStore()
  const addToast = useUIStore((s) => s.addToast)
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const connectMutation = useMutation({
    mutationFn: (req: ConnectRequest) => api.connect(req),
    onSuccess: (data, req) => {
      // A new login may be a different account or server; never show the old one's events.
      queryClient.clear()
      localStorage.setItem('caldav_url', req.url)
      localStorage.setItem('caldav_username', req.username)
      setAuth(data.token, data.expiresAt)
      navigate('/')
    },
    onError: (err) => {
      const msg = err instanceof ApiError ? err.message : 'Connection failed'
      addToast(msg)
    },
  })

  const logout = useCallback(() => {
    clearAuth()
    queryClient.clear()
    navigate('/login')
  }, [clearAuth, navigate, queryClient])

  return {
    token,
    expiresAt,
    isAuthenticated: !!token,
    connect: connectMutation,
    logout,
  }
}
