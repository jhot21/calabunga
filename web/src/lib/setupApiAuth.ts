import { useAuthStore } from '../store/authStore'
import { configureApiAuth } from './api'

// Must run before the first render: child effects fire before App's, so wiring
// this in an effect sends the first request unauthenticated and costs a 401 plus
// React Query's retry delay on every reload.
export function setupApiAuth(): void {
  configureApiAuth(
    () => useAuthStore.getState().token,
    (newToken) => {
      const { expiresAt } = useAuthStore.getState()
      useAuthStore.getState().setAuth(newToken, expiresAt ?? '')
    }
  )
}
