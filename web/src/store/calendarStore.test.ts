import { describe, it, expect, beforeEach } from 'vitest'
import { useCalendarStore } from './calendarStore'

beforeEach(() => {
  useCalendarStore.setState({
    visibleCalendars: {},
    currentView: 'month',
    currentDate: new Date().toISOString(),
    theme: 'system',
  })
})

describe('calendarStore', () => {
  it('defaults to month view', () => {
    expect(useCalendarStore.getState().currentView).toBe('month')
  })

  it('setVisible toggles calendar visibility', () => {
    useCalendarStore.getState().setVisible('cal1', true)
    expect(useCalendarStore.getState().visibleCalendars['cal1']).toBe(true)
    useCalendarStore.getState().setVisible('cal1', false)
    expect(useCalendarStore.getState().visibleCalendars['cal1']).toBe(false)
  })

  it('setView switches between month and week', () => {
    useCalendarStore.getState().setView('week')
    expect(useCalendarStore.getState().currentView).toBe('week')
  })

  it('setTheme updates theme', () => {
    useCalendarStore.getState().setTheme('dark')
    expect(useCalendarStore.getState().theme).toBe('dark')
  })
})
