import React from 'react'
import ReactDOM from 'react-dom/client'
import { MutationCache, QueryCache, QueryClient } from '@tanstack/react-query'
import { PersistQueryClientProvider } from '@tanstack/react-query-persist-client'
import App from './App'
import { useAuthStore } from './store/authStore'
import { ApiError } from './lib/api'
import { setupApiAuth } from './lib/setupApiAuth'
import {
  PERSIST_BUSTER,
  PERSIST_MAX_AGE,
  clearCacheOnSignOut,
  persister,
  shouldPersistQuery,
} from './lib/queryPersist'
import './index.css'

setupApiAuth()

function handleGlobalError(err: unknown) {
  if (err instanceof ApiError && (err.code === 'token_expired' || err.code === 'auth_failed')) {
    useAuthStore.getState().clearAuth()
  }
}

const queryClient = new QueryClient({
  queryCache: new QueryCache({ onError: handleGlobalError }),
  mutationCache: new MutationCache({ onError: handleGlobalError }),
  defaultOptions: {
    // gcTime must cover the persist maxAge or restored queries are dropped.
    queries: { retry: 1, staleTime: 30_000, gcTime: PERSIST_MAX_AGE },
  },
})

clearCacheOnSignOut(queryClient)

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <PersistQueryClientProvider
      client={queryClient}
      persistOptions={{
        persister,
        maxAge: PERSIST_MAX_AGE,
        buster: PERSIST_BUSTER,
        dehydrateOptions: { shouldDehydrateQuery: shouldPersistQuery },
      }}
    >
      <App />
    </PersistQueryClientProvider>
  </React.StrictMode>
)
