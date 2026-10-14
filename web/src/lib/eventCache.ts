import type { Event } from '../types/api'

// Optimistic update for a saved edit. For a recurring event the cache holds one
// expanded entry per occurrence, all sharing a uid; replacing them all with the
// edited occurrence would stack the whole series onto one day until the refetch.
export function applyEdit(old: Event[] | undefined, edited: Event, originalStart?: string): Event[] {
  const { originalStart: _drop, ...clean } = edited
  return (old ?? []).map((e) => {
    if (e.uid !== edited.uid) return e
    if (!edited.recurrence) return clean
    if (originalStart !== undefined && e.start === originalStart) return clean
    return { ...clean, start: e.start, end: e.end }
  })
}
