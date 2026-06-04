import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'
import type { ConnectRequest } from '../types/api'

const TTL_OPTIONS = [
  { label: '8 hours',  value: '8h' },
  { label: '24 hours', value: '24h' },
  { label: '7 days',   value: '168h' },
  { label: '30 days',  value: '720h' },
]

export default function LoginPage() {
  const { isAuthenticated, connect } = useAuth()
  const navigate = useNavigate()
  const [form, setForm] = useState<ConnectRequest>({
    url: '', username: '', password: '', ttl: '24h',
  })

  useEffect(() => {
    if (isAuthenticated) navigate('/')
  }, [isAuthenticated, navigate])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    connect.mutate(form)
  }

  return (
    <div className="login-page">
      <div className="login-card">
        <div className="login-logo">cala<span>bunga</span></div>
        <div className="login-tagline">Connect to your CalDAV server</div>

        {connect.error && (
          <div className="login-error">
            {connect.error instanceof Error ? connect.error.message : 'Connection failed'}
          </div>
        )}

        <form onSubmit={handleSubmit}>
          <label className="login-label">Server URL</label>
          <input
            className="login-input"
            type="url"
            placeholder="https://dav.example.com"
            value={form.url}
            onChange={(e) => setForm((f) => ({ ...f, url: e.target.value }))}
            required
          />

          <label className="login-label">Username</label>
          <input
            className="login-input"
            type="text"
            placeholder="you@example.com"
            value={form.username}
            onChange={(e) => setForm((f) => ({ ...f, username: e.target.value }))}
            required
          />

          <label className="login-label">Password</label>
          <input
            className="login-input"
            type="password"
            placeholder="••••••••••••"
            value={form.password}
            onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))}
            required
          />

          <div className="login-ttl-row">
            <span className="login-ttl-label">Keep me signed in for</span>
            <select
              className="login-select"
              value={form.ttl}
              onChange={(e) => setForm((f) => ({ ...f, ttl: e.target.value }))}
            >
              {TTL_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>{o.label}</option>
              ))}
            </select>
          </div>

          <button className="login-btn" type="submit" disabled={connect.isPending}>
            {connect.isPending ? 'Connecting…' : 'Connect'}
          </button>
        </form>

        <div className="login-hint">
          Credentials are encrypted into a signed token and never stored on the server.
        </div>
      </div>
    </div>
  )
}
