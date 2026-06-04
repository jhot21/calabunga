import { describe, it, expect, beforeEach } from 'vitest'
import { useAuthStore } from './authStore'

beforeEach(() => {
  useAuthStore.setState({ token: null, expiresAt: null })
})

describe('authStore', () => {
  it('starts with no token', () => {
    expect(useAuthStore.getState().token).toBeNull()
  })

  it('setAuth stores token and expiresAt', () => {
    useAuthStore.getState().setAuth('tok123', '2026-06-04T10:00:00Z')
    expect(useAuthStore.getState().token).toBe('tok123')
    expect(useAuthStore.getState().expiresAt).toBe('2026-06-04T10:00:00Z')
  })

  it('clearAuth removes token', () => {
    useAuthStore.getState().setAuth('tok', 'exp')
    useAuthStore.getState().clearAuth()
    expect(useAuthStore.getState().token).toBeNull()
  })
})
