import { createAsyncStoragePersister } from '@tanstack/query-async-storage-persister'
import type { Query, QueryClient } from '@tanstack/react-query'
import { useAuthStore } from '../store/authStore'

export const PERSIST_KEY = 'calabunga-query-cache'
export const PERSIST_MAX_AGE = 24 * 60 * 60 * 1000
// Bump to discard caches written by an incompatible shape of the API types.
export const PERSIST_BUSTER = 'v4'

const PERSISTED_ROOTS = new Set(['calendars', 'events'])

export const persister = createAsyncStoragePersister({
  storage: window.localStorage,
  key: PERSIST_KEY,
})

export function shouldPersistQuery(query: Query): boolean {
  return query.state.status === 'success' && PERSISTED_ROOTS.has(String(query.queryKey[0]))
}

// Cached events are private calendar data, so they must not outlive the session
// (logout, expiry, or a rejected token all end in clearAuth).
export function clearCacheOnSignOut(queryClient: QueryClient): () => void {
  const wipe = () => {
    queryClient.clear()
    try {
      window.localStorage.removeItem(PERSIST_KEY)
    } catch {
      /* storage unavailable */
    }
  }

  if (!useAuthStore.getState().token) wipe()

  return useAuthStore.subscribe((state, prev) => {
    if (prev.token && !state.token) wipe()
  })
}
