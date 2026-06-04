import {
  startOfMonth, startOfWeek, endOfWeek, addDays,
  eachDayOfInterval, getHours, getMinutes, format, parseISO,
} from 'date-fns'

export const HOUR_HEIGHT = 64
export const MIN_EVENT_HEIGHT = 20

export function calendarGrid(year: number, month: number): Date[][] {
  const first = startOfMonth(new Date(year, month))
  const start = startOfWeek(first, { weekStartsOn: 0 })
  const days = eachDayOfInterval({ start, end: addDays(start, 41) })
  const weeks: Date[][] = []
  for (let i = 0; i < 42; i += 7) weeks.push(days.slice(i, i + 7))
  return weeks
}

export function weekDays(date: Date): Date[] {
  const start = startOfWeek(date, { weekStartsOn: 0 })
  return eachDayOfInterval({ start, end: addDays(start, 6) })
}

export function monthRange(date: Date): { start: Date; end: Date } {
  const first = startOfMonth(date)
  const start = startOfWeek(first, { weekStartsOn: 0 })
  return { start, end: addDays(start, 41) }
}

export function weekRange(date: Date): { start: Date; end: Date } {
  return {
    start: startOfWeek(date, { weekStartsOn: 0 }),
    end: endOfWeek(date, { weekStartsOn: 0 }),
  }
}

export function eventTopPx(startISO: string): number {
  const d = parseISO(startISO)
  return (getHours(d) * 60 + getMinutes(d)) / 60 * HOUR_HEIGHT
}

export function eventHeightPx(startISO: string, endISO: string): number {
  const ms = parseISO(endISO).getTime() - parseISO(startISO).getTime()
  return Math.max(ms / 3_600_000 * HOUR_HEIGHT, MIN_EVENT_HEIGHT)
}

export function formatTime(iso: string): string {
  return format(parseISO(iso), 'h:mm a')
}

export function formatDate(date: Date): string {
  return format(date, 'MMM d')
}

export function formatMonthYear(date: Date): string {
  return format(date, 'MMMM yyyy')
}

export function formatWeekTitle(date: Date): string {
  const start = startOfWeek(date, { weekStartsOn: 0 })
  const end = endOfWeek(date, { weekStartsOn: 0 })
  if (start.getMonth() === end.getMonth()) return format(start, 'MMMM yyyy')
  return `${format(start, 'MMM')} – ${format(end, 'MMM yyyy')}`
}

export function toDatetimeLocal(utcIso: string): string {
  const d = new Date(utcIso)
  return new Date(d.getTime() - d.getTimezoneOffset() * 60_000)
    .toISOString().slice(0, 16)
}

export function fromDatetimeLocal(localStr: string): string {
  return new Date(localStr).toISOString()
}

export function toDateLocal(utcIso: string): string {
  const d = new Date(utcIso)
  return new Date(d.getTime() - d.getTimezoneOffset() * 60_000)
    .toISOString().slice(0, 10)
}
