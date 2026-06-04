import { useEffect } from 'react'
import { useUIStore } from '../store/uiStore'

export default function ToastList() {
  const { toasts, dismissToast } = useUIStore()

  useEffect(() => {
    if (toasts.length === 0) return
    const timer = setTimeout(() => dismissToast(toasts[0].id), 5_000)
    return () => clearTimeout(timer)
  }, [toasts])

  if (toasts.length === 0) return null

  return (
    <div className="toast-list" aria-live="polite">
      {toasts.map((t) => (
        <div key={t.id} className="toast">
          <span style={{ flex: 1 }}>{t.message}</span>
          {t.action && (
            <button className="toast-action" onClick={t.action.onClick}>{t.action.label}</button>
          )}
          <button className="toast-action toast-dismiss" onClick={() => dismissToast(t.id)} aria-label="Dismiss">✕</button>
        </div>
      ))}
    </div>
  )
}
