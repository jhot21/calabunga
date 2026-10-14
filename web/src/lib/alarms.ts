// Notifications are stored as "minutes before the event starts".

export const MAX_ALARMS = 10 // keep in sync with maxAlarms in internal/caldav/alarms.go

export const ALARM_PRESETS = [
  { minutes: 10, label: '10 minutes before' },
  { minutes: 0, label: 'At time of event' },
  { minutes: 5, label: '5 minutes before' },
  { minutes: 15, label: '15 minutes before' },
  { minutes: 30, label: '30 minutes before' },
  { minutes: 60, label: '1 hour before' },
  { minutes: 120, label: '2 hours before' },
  { minutes: 1440, label: '1 day before' },
  { minutes: 2880, label: '2 days before' },
  { minutes: 10080, label: '1 week before' },
] as const

const plural = (n: number, unit: string) => `${n} ${unit}${n === 1 ? '' : 's'} before`

export function alarmLabel(minutes: number): string {
  const preset = ALARM_PRESETS.find((p) => p.minutes === minutes)
  if (preset) return preset.label
  if (minutes % 10080 === 0) return plural(minutes / 10080, 'week')
  if (minutes % 1440 === 0) return plural(minutes / 1440, 'day')
  if (minutes % 60 === 0) return plural(minutes / 60, 'hour')
  return plural(minutes, 'minute')
}

// The value a new notification row starts on: 10 minutes first, then the next
// preset not already in use, then the smallest free number of minutes.
export function nextAlarm(existing: number[]): number {
  const preset = ALARM_PRESETS.find((p) => !existing.includes(p.minutes))
  if (preset) return preset.minutes
  let m = 1
  while (existing.includes(m)) m++
  return m
}
