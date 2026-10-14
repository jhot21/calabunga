import { describe, it, expect, beforeEach } from 'vitest'
import { QueryClient } from '@tanstack/react-query'
import { useAuthStore } from '../store/authStore'
import { PERSIST_KEY, clearCacheOnSignOut, shouldPersistQuery } from './queryPersist'

beforeEach(() => {
  localStorage.clear()
  useAuthStore.setState({ token: null, expiresAt: null })
})

describe('shouldPersistQuery', () => {
  it('persists only successful calendars and events queries', () => {
    const qc = new QueryClient()
    qc.setQueryData(['calendars'], [])
    qc.setQueryData(['events', 'cal', 's', 'e'], [])
    qc.setQueryData(['something-else'], 1)
    const find = (key: unknown[]) => qc.getQueryCache().find({ queryKey: key })!

    expect(shouldPersistQuery(find(['calendars']))).toBe(true)
    expect(shouldPersistQuery(find(['events', 'cal', 's', 'e']))).toBe(true)
    expect(shouldPersistQuery(find(['something-else']))).toBe(false)
  })
})

describe('clearCacheOnSignOut', () => {
  it('wipes the cache and stored copy when the token is cleared', () => {
    const qc = new QueryClient()
    qc.setQueryData(['events', 'cal', 's', 'e'], [{ uid: 'a' }])
    localStorage.setItem(PERSIST_KEY, '{"stale":true}')
    useAuthStore.setState({ token: 'tok', expiresAt: 'exp' })

    clearCacheOnSignOut(qc)
    expect(localStorage.getItem(PERSIST_KEY)).not.toBeNull()

    useAuthStore.getState().clearAuth()

    expect(localStorage.getItem(PERSIST_KEY)).toBeNull()
    expect(qc.getQueryCache().getAll()).toHaveLength(0)
  })

  it('keeps the cache across a token refresh', () => {
    const qc = new QueryClient()
    localStorage.setItem(PERSIST_KEY, '{"keep":true}')
    useAuthStore.setState({ token: 'tok', expiresAt: 'exp' })

    clearCacheOnSignOut(qc)
    useAuthStore.getState().setAuth('refreshed', 'exp')

    expect(localStorage.getItem(PERSIST_KEY)).not.toBeNull()
  })

  it('discards a stored copy left behind when there is no session', () => {
    localStorage.setItem(PERSIST_KEY, '{"orphan":true}')

    clearCacheOnSignOut(new QueryClient())

    expect(localStorage.getItem(PERSIST_KEY)).toBeNull()
  })
})
