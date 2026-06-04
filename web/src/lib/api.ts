// src/lib/api.ts
import type { Calendar, ConnectRequest, ConnectResponse, Event, EditScope } from '../types/api'

let getToken: () => string | null = () => null
let setToken: (t: string) => void = () => {}

export function configureApiAuth(
  getter: () => string | null,
  setter: (t: string) => void
): void {
  getToken = getter
  setToken = setter
}

export class ApiError extends Error {
  constructor(
    public readonly code: string,
    message: string,
    public readonly status: number
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {}
  const token = getToken()
  if (token) headers['Authorization'] = `Bearer ${token}`
  if (body !== undefined) headers['Content-Type'] = 'application/json'

  const res = await fetch(`/api${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  const refresh = res.headers.get('X-Refresh-Token')
  if (refresh) setToken(refresh)

  if (!res.ok) {
    let err = { code: 'unknown', message: res.statusText }
    try { err = await res.json() } catch { /* ignore */ }
    throw new ApiError(err.code, err.message, res.status)
  }

  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

export const api = {
  connect: (body: ConnectRequest) =>
    request<ConnectResponse>('POST', '/connect', body),

  calendars: () =>
    request<Calendar[]>('GET', '/calendars'),

  events: (params: { calendarId: string; start: string; end: string }) =>
    request<Event[]>('GET',
      `/events?calendarId=${encodeURIComponent(params.calendarId)}&start=${encodeURIComponent(params.start)}&end=${encodeURIComponent(params.end)}`
    ),

  createEvent: (event: Event) =>
    request<Event>('POST', '/events', event),

  updateEvent: (uid: string, event: Event, scope: EditScope, calendarId: string) =>
    request<Event>(
      'PUT',
      `/events/${encodeURIComponent(uid)}?editScope=${scope}&calendarId=${encodeURIComponent(calendarId)}`,
      event
    ),

  deleteEvent: (uid: string, scope: EditScope, calendarId: string, etag: string) =>
    request<void>(
      'DELETE',
      `/events/${encodeURIComponent(uid)}?editScope=${scope}&calendarId=${encodeURIComponent(calendarId)}&etag=${encodeURIComponent(etag)}`
    ),
}
