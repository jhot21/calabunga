import { describe, it, expect } from 'vitest'
import { recurrenceLabel, recurrenceEndLabel, defaultRecurrence } from './rrule'
import type { Recurrence } from '../types/api'

describe('recurrenceLabel', () => {
  it('daily', () => {
    const r: Recurrence = { frequency: 'daily', interval: 1, end: { type: 'never' } }
    expect(recurrenceLabel(r)).toBe('Daily')
  })

  it('every 2 days', () => {
    const r: Recurrence = { frequency: 'daily', interval: 2, end: { type: 'never' } }
    expect(recurrenceLabel(r)).toBe('Every 2 days')
  })

  it('weekly on MO, WE, FR', () => {
    const r: Recurrence = { frequency: 'weekly', interval: 1, byDay: ['MO', 'WE', 'FR'], end: { type: 'never' } }
    expect(recurrenceLabel(r)).toBe('Weekly on Mon, Wed, Fri')
  })

  it('monthly', () => {
    const r: Recurrence = { frequency: 'monthly', interval: 1, end: { type: 'never' } }
    expect(recurrenceLabel(r)).toBe('Monthly')
  })
})

describe('recurrenceEndLabel', () => {
  it('never returns empty string', () => {
    expect(recurrenceEndLabel({ type: 'never' })).toBe('')
  })

  it('count shows count', () => {
    expect(recurrenceEndLabel({ type: 'count', value: '5' })).toBe('5 times')
  })

  it('date shows date', () => {
    expect(recurrenceEndLabel({ type: 'date', value: '2026-12-31' })).toBe('until 2026-12-31')
  })
})
