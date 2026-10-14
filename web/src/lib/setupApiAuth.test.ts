import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useAuthStore } from '../store/authStore'
import { api } from './api'
import { setupApiAuth } from './setupApiAuth'

beforeEach(() => {
  useAuthStore.setState({ token: null, expiresAt: null })
  vi.restoreAllMocks()
})

describe('setupApiAuth', () => {
  it('sends the persisted token on the first request, before any component mounts', async () => {
    useAuthStore.setState({ token: 'persisted-tok', expiresAt: 'exp' })
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response('[]', { status: 200 })
    )

    setupApiAuth()
    await api.calendars()

    const headers = fetchMock.mock.calls[0][1]!.headers as Record<string, string>
    expect(headers['Authorization']).toBe('Bearer persisted-tok')
  })

  it('stores a refreshed token while keeping expiresAt', async () => {
    useAuthStore.setState({ token: 'old', expiresAt: 'exp' })
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response('[]', { status: 200, headers: { 'X-Refresh-Token': 'new' } })
    )

    setupApiAuth()
    await api.calendars()

    expect(useAuthStore.getState().token).toBe('new')
    expect(useAuthStore.getState().expiresAt).toBe('exp')
  })
})
