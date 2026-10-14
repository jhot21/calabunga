import { describe, it, expect } from 'vitest'
import { alarmLabel, nextAlarm, ALARM_PRESETS, MAX_ALARMS } from './alarms'

describe('alarmLabel', () => {
  it('labels the presets', () => {
    expect(alarmLabel(0)).toBe('At time of event')
    expect(alarmLabel(10)).toBe('10 minutes before')
    expect(alarmLabel(60)).toBe('1 hour before')
    expect(alarmLabel(1440)).toBe('1 day before')
    expect(alarmLabel(10080)).toBe('1 week before')
  })

  it('labels values set by other clients in the largest whole unit', () => {
    expect(alarmLabel(1)).toBe('1 minute before')
    expect(alarmLabel(45)).toBe('45 minutes before')
    expect(alarmLabel(90)).toBe('90 minutes before')
    expect(alarmLabel(180)).toBe('3 hours before')
    expect(alarmLabel(4320)).toBe('3 days before')
    expect(alarmLabel(20160)).toBe('2 weeks before')
  })
})

describe('nextAlarm', () => {
  it('starts with 10 minutes', () => {
    expect(nextAlarm([])).toBe(10)
  })

  it('picks the next unused preset', () => {
    expect(nextAlarm([10])).not.toBe(10)
    expect(ALARM_PRESETS.map((p) => p.minutes)).toContain(nextAlarm([10]))
  })

  it('never duplicates an existing alarm', () => {
    const all = ALARM_PRESETS.map((p) => p.minutes)
    expect(all).not.toContain(nextAlarm(all))
  })

  it('exposes the same limit the server enforces', () => {
    expect(MAX_ALARMS).toBe(10)
  })
})
