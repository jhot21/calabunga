import { describe, it, expect } from 'vitest'
import { applyEdit } from './eventCache'
import type { Event } from '../types/api'

const base: Event = {
  uid: 'u1', calendarId: '/c/', etag: '"e"', title: 'Standup', allDay: false,
  start: '2026-06-02T10:00:00Z', end: '2026-06-02T10:30:00Z',
}
const rec = { frequency: 'weekly' as const, interval: 1, byDay: ['TU'], end: { type: 'never' as const } }

describe('applyEdit', () => {
  it('replaces a single event outright', () => {
    const other: Event = { ...base, uid: 'u2' }
    const edited = { ...base, title: 'Renamed' }
    expect(applyEdit([base, other], edited)).toEqual([edited, other])
  })

  it('keeps every occurrence of a series on its own date', () => {
    const occ = (d: string): Event => ({ ...base, recurrence: rec, start: `2026-06-${d}T10:00:00Z`, end: `2026-06-${d}T10:30:00Z` })
    const old = [occ('02'), occ('09'), occ('16')]
    const edited = { ...occ('09'), title: 'Renamed' }

    const result = applyEdit(old, edited, '2026-06-09T10:00:00Z')

    expect(result.map((e) => e.title)).toEqual(['Renamed', 'Renamed', 'Renamed'])
    expect(result.map((e) => e.start)).toEqual([
      '2026-06-02T10:00:00Z', '2026-06-09T10:00:00Z', '2026-06-16T10:00:00Z',
    ])
  })

  it('applies a time change to the opened occurrence and keeps the others on their dates', () => {
    const occ = (d: string): Event => ({ ...base, recurrence: rec, start: `2026-06-${d}T10:00:00Z`, end: `2026-06-${d}T10:30:00Z` })
    const edited = { ...occ('09'), start: '2026-06-09T11:00:00Z', end: '2026-06-09T11:30:00Z' }

    const result = applyEdit([occ('02'), occ('09')], edited, '2026-06-09T10:00:00Z')

    expect(result[1].start).toBe('2026-06-09T11:00:00Z')
    expect(result[0].start).toBe('2026-06-02T10:00:00Z')
  })

  it('does not leak the request-only originalStart into cached events', () => {
    const edited = { ...base, recurrence: rec, originalStart: base.start }
    const result = applyEdit([{ ...base, recurrence: rec }], edited, base.start)
    expect(result[0]).not.toHaveProperty('originalStart')
  })
})
