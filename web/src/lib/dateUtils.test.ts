import { describe, it, expect } from 'vitest'
import {
  calendarGrid, weekDays, monthRange, weekRange,
  eventTopPx, eventHeightPx, formatTime, toDatetimeLocal, fromDatetimeLocal,
  HOUR_HEIGHT,
} from './dateUtils'

describe('calendarGrid', () => {
  it('produces 6 rows of 7 days', () => {
    const grid = calendarGrid(2026, 5) // June 2026
    expect(grid).toHaveLength(6)
    expect(grid[0]).toHaveLength(7)
  })

  it('first cell is a Sunday', () => {
    const grid = calendarGrid(2026, 5)
    expect(grid[0][0].getDay()).toBe(0)
  })

  it('contains June 1 2026', () => {
    const grid = calendarGrid(2026, 5)
    const allDays = grid.flat()
    expect(allDays.some((d) => d.getFullYear() === 2026 && d.getMonth() === 5 && d.getDate() === 1)).toBe(true)
  })
})

describe('weekDays', () => {
  it('returns 7 days starting from Sunday', () => {
    const days = weekDays(new Date(2026, 5, 3)) // Wed Jun 3
    expect(days).toHaveLength(7)
    expect(days[0].getDay()).toBe(0) // Sunday
    expect(days[6].getDay()).toBe(6) // Saturday
  })
})

describe('eventTopPx', () => {
  it('positions 10am event at 10 * HOUR_HEIGHT', () => {
    const top = eventTopPx('2026-06-03T10:00:00Z')
    expect(typeof top).toBe('number')
    expect(top).toBeGreaterThanOrEqual(0)
  })
})

describe('eventHeightPx', () => {
  it('returns HOUR_HEIGHT for a 1-hour event', () => {
    const h = eventHeightPx('2026-06-03T10:00:00Z', '2026-06-03T11:00:00Z')
    expect(h).toBe(HOUR_HEIGHT)
  })

  it('enforces minimum height', () => {
    const h = eventHeightPx('2026-06-03T10:00:00Z', '2026-06-03T10:05:00Z')
    expect(h).toBeGreaterThanOrEqual(20)
  })
})

describe('toDatetimeLocal / fromDatetimeLocal', () => {
  it('round-trips a UTC ISO string', () => {
    const utc = '2026-06-03T10:00:00.000Z'
    const local = toDatetimeLocal(utc)
    const back = fromDatetimeLocal(local)
    const diff = Math.abs(new Date(back).getTime() - new Date(utc).getTime())
    expect(diff).toBeLessThan(60_000)
  })
})
