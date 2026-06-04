import type { Recurrence, RecurrenceEnd } from '../types/api'

const DAY_LABELS: Record<string, string> = {
  SU: 'Sun', MO: 'Mon', TU: 'Tue', WE: 'Wed', TH: 'Thu', FR: 'Fri', SA: 'Sat',
}

export function recurrenceLabel(r: Recurrence): string {
  switch (r.frequency) {
    case 'daily':
      return r.interval > 1 ? `Every ${r.interval} days` : 'Daily'
    case 'weekly': {
      const days = (r.byDay ?? []).map((d) => DAY_LABELS[d] ?? d).join(', ')
      const freq = r.interval > 1 ? `Every ${r.interval} weeks` : 'Weekly'
      return days ? `${freq} on ${days}` : freq
    }
    case 'monthly':
      return r.interval > 1 ? `Every ${r.interval} months` : 'Monthly'
    case 'yearly':
      return r.interval > 1 ? `Every ${r.interval} years` : 'Annually'
    default:
      return 'Recurring'
  }
}

export function recurrenceEndLabel(end: RecurrenceEnd): string {
  switch (end.type) {
    case 'never': return ''
    case 'date':  return `until ${end.value}`
    case 'count': return `${end.value} time${end.value === '1' ? '' : 's'}`
    default:      return ''
  }
}

export function defaultRecurrence(): Recurrence {
  return { frequency: 'weekly', interval: 1, byDay: ['MO'], end: { type: 'never' } }
}

export const WEEK_DAYS = ['SU', 'MO', 'TU', 'WE', 'TH', 'FR', 'SA'] as const
export const WEEK_DAY_LABELS = WEEK_DAYS.map((d) => ({ key: d, label: DAY_LABELS[d] }))
export const FREQUENCIES = ['daily', 'weekly', 'monthly', 'yearly'] as const
