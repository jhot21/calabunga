import { useState } from 'react'
import { MdClose } from 'react-icons/md'
import { toDatetimeLocal, fromDatetimeLocal, toDateLocal } from '../../lib/dateUtils'
import { defaultRecurrence, WEEK_DAY_LABELS, FREQUENCIES } from '../../lib/rrule'
import { ALARM_PRESETS, MAX_ALARMS, alarmLabel, nextAlarm } from '../../lib/alarms'
import type { Event, Calendar, EditScope, Recurrence } from '../../types/api'

interface EventModalProps {
  event?: Event
  initialDate?: Date
  calendars: Calendar[]
  onClose: () => void
  onSave: (event: Event, scope: EditScope) => void
  onDelete?: (event: Event, scope: EditScope) => void
}

function blankEvent(calendars: Calendar[], initialDate?: Date): Event {
  const base = initialDate ?? new Date()
  const start = new Date(base)
  start.setMinutes(0, 0, 0)
  const end = new Date(start.getTime() + 3_600_000)
  return {
    uid: '', calendarId: calendars[0]?.id ?? '', etag: '',
    title: '', start: start.toISOString(), end: end.toISOString(), allDay: false,
    alarms: [], // no default notifications
  }
}

export default function EventModal({
  event, initialDate, calendars, onClose, onSave, onDelete,
}: EventModalProps) {
  const isEdit = !!event
  const [form, setForm] = useState<Event>(event ?? blankEvent(calendars, initialDate))
  const [scope, setScope] = useState<EditScope>('all')
  const [recurring, setRecurring] = useState(!!event?.recurrence)

  const set = <K extends keyof Event>(key: K, val: Event[K]) =>
    setForm((f) => ({ ...f, [key]: val }))

  const setRec = <K extends keyof Recurrence>(key: K, val: Recurrence[K]) =>
    setForm((f) => ({ ...f, recurrence: { ...(f.recurrence ?? defaultRecurrence()), [key]: val } }))

  const toggleByDay = (day: string) => {
    const cur = form.recurrence?.byDay ?? []
    const next = cur.includes(day) ? cur.filter((d) => d !== day) : [...cur, day]
    setRec('byDay', next)
  }

  const alarms = form.alarms ?? []
  const setAlarm = (i: number, minutes: number) =>
    set('alarms', alarms.map((m, j) => (j === i ? minutes : m)))
  const removeAlarm = (i: number) => set('alarms', alarms.filter((_, j) => j !== i))
  const addAlarm = () => set('alarms', [...alarms, nextAlarm(alarms)])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSave({ ...form, recurrence: recurring ? (form.recurrence ?? defaultRecurrence()) : undefined }, scope)
    onClose()
  }

  const rec = form.recurrence ?? defaultRecurrence()

  return (
    <div className="modal-backdrop" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal" role="dialog" aria-modal="true">
        <div className="modal-header">
          <span className="modal-title">{isEdit ? 'Edit event' : 'New event'}</span>
          <button className="modal-close" onClick={onClose}><MdClose size={18} /></button>
        </div>
        <form onSubmit={handleSubmit}>
          <div className="modal-body">
            <div className="modal-field">
              <label className="modal-field-label">Title</label>
              <input className="modal-input" type="text" value={form.title}
                onChange={(e) => set('title', e.target.value)} placeholder="Add title" autoFocus required />
            </div>

            <div className="modal-checkbox-row">
              <input id="allday" type="checkbox" checked={form.allDay}
                onChange={(e) => set('allDay', e.target.checked)} />
              <label htmlFor="allday">All day</label>
            </div>

            {form.allDay ? (
              <div className="modal-row">
                <div className="modal-field">
                  <label className="modal-field-label">Start date</label>
                  <input className="modal-input" type="date" value={toDateLocal(form.start)}
                    onChange={(e) => {
                      const newStart = new Date(e.target.value + 'T00:00:00')
                      setForm((f) => {
                        const duration = new Date(f.end).getTime() - new Date(f.start).getTime()
                        return { ...f, start: newStart.toISOString(), end: new Date(newStart.getTime() + duration).toISOString() }
                      })
                    }} required />
                </div>
                <div className="modal-field">
                  <label className="modal-field-label">End date</label>
                  <input className="modal-input" type="date" value={toDateLocal(form.end)}
                    onChange={(e) => set('end', new Date(e.target.value + 'T00:00:00').toISOString())} required />
                </div>
              </div>
            ) : (
              <div className="modal-row">
                <div className="modal-field">
                  <label className="modal-field-label">Start</label>
                  <input className="modal-input" type="datetime-local" value={toDatetimeLocal(form.start)}
                    onChange={(e) => {
                      const newStart = fromDatetimeLocal(e.target.value)
                      setForm((f) => {
                        const duration = new Date(f.end).getTime() - new Date(f.start).getTime()
                        return { ...f, start: newStart, end: new Date(new Date(newStart).getTime() + duration).toISOString() }
                      })
                    }} required />
                </div>
                <div className="modal-field">
                  <label className="modal-field-label">End</label>
                  <input className="modal-input" type="datetime-local" value={toDatetimeLocal(form.end)}
                    onChange={(e) => set('end', fromDatetimeLocal(e.target.value))} required />
                </div>
              </div>
            )}

            <div className="modal-field">
              <label className="modal-field-label">Calendar</label>
              <select className="modal-select" value={form.calendarId}
                onChange={(e) => set('calendarId', e.target.value)}>
                {calendars.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
              </select>
            </div>

            <div className="modal-checkbox-row">
              <input id="recurring" type="checkbox" checked={recurring}
                onChange={(e) => {
                  setRecurring(e.target.checked)
                  if (e.target.checked && !form.recurrence)
                    setForm((f) => ({ ...f, recurrence: defaultRecurrence() }))
                }} />
              <label htmlFor="recurring">Repeat</label>
            </div>

            {recurring && (
              <div className="modal-recurrence-section">
                <div className="modal-recurrence-grid">
                  <div className="modal-field" style={{ marginBottom: 0 }}>
                    <label className="modal-field-label">Frequency</label>
                    <select className="modal-select" value={rec.frequency}
                      onChange={(e) => setRec('frequency', e.target.value as Recurrence['frequency'])}>
                      {FREQUENCIES.map((f) => (
                        <option key={f} value={f}>{f.charAt(0).toUpperCase() + f.slice(1)}</option>
                      ))}
                    </select>
                  </div>
                  <div className="modal-field" style={{ marginBottom: 0 }}>
                    <label className="modal-field-label">Every</label>
                    <input className="modal-input" type="number" min={1} max={99} value={rec.interval}
                      onChange={(e) => setRec('interval', parseInt(e.target.value) || 1)} />
                  </div>
                </div>

                {rec.frequency === 'weekly' && (
                  <div>
                    <div className="modal-field-label" style={{ marginTop: 12 }}>On days</div>
                    <div className="modal-byday-row">
                      {WEEK_DAY_LABELS.map(({ key, label }) => (
                        <button key={key} type="button"
                          className={`modal-day-btn${(rec.byDay ?? []).includes(key) ? ' active' : ''}`}
                          onClick={() => toggleByDay(key)}>
                          {label.slice(0, 2)}
                        </button>
                      ))}
                    </div>
                  </div>
                )}

                <div style={{ marginTop: 12 }}>
                  <div className="modal-field-label">Ends</div>
                  <div className="modal-recurrence-grid" style={{ alignItems: 'end' }}>
                    <select className="modal-select" value={rec.end.type}
                      onChange={(e) => setRec('end', { type: e.target.value as 'never' | 'date' | 'count' })}>
                      <option value="never">Never</option>
                      <option value="date">On date</option>
                      <option value="count">After</option>
                    </select>
                    {rec.end.type === 'date' && (
                      <input className="modal-input" type="date" value={rec.end.value ?? ''}
                        onChange={(e) => setRec('end', { type: 'date', value: e.target.value })} required />
                    )}
                    {rec.end.type === 'count' && (
                      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                        <input className="modal-input" type="number" min={1} style={{ width: 80 }}
                          value={rec.end.value ?? '1'}
                          onChange={(e) => setRec('end', { type: 'count', value: e.target.value })} />
                        <span style={{ fontSize: 13 }}>occurrences</span>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            )}

            {isEdit && event?.recurrence && (
              <div className="modal-scope-section">
                <div className="modal-scope-title">Edit recurring event</div>
                <div className="modal-scope-options">
                  {(['this', 'thisAndFollowing', 'all'] as EditScope[]).map((s) => (
                    <label key={s} className="modal-scope-option">
                      <input type="radio" name="scope" value={s} checked={scope === s} onChange={() => setScope(s)} />
                      {s === 'this' && 'This event'}
                      {s === 'thisAndFollowing' && 'This and following events'}
                      {s === 'all' && 'All events'}
                    </label>
                  ))}
                </div>
              </div>
            )}

            <div className="modal-field">
              <label className="modal-field-label">Notifications</label>
              {alarms.map((minutes, i) => (
                <div key={i} className="modal-alarm-row">
                  <select className="modal-select" aria-label={`Notification ${i + 1}`} value={minutes}
                    onChange={(e) => setAlarm(i, parseInt(e.target.value))}>
                    {!ALARM_PRESETS.some((p) => p.minutes === minutes) && (
                      <option value={minutes}>{alarmLabel(minutes)}</option>
                    )}
                    {ALARM_PRESETS.map((p) => (
                      <option key={p.minutes} value={p.minutes} disabled={alarms.includes(p.minutes) && p.minutes !== minutes}>
                        {p.label}
                      </option>
                    ))}
                  </select>
                  <button type="button" className="modal-close" aria-label={`Remove notification ${i + 1}`}
                    onClick={() => removeAlarm(i)}><MdClose size={18} /></button>
                </div>
              ))}
              <button type="button" className="modal-btn modal-btn-secondary modal-alarm-add"
                disabled={alarms.length >= MAX_ALARMS} onClick={addAlarm}>
                Add notification
              </button>
            </div>

            <div className="modal-field">
              <label className="modal-field-label">Description</label>
              <textarea className="modal-input" rows={3} value={form.description ?? ''}
                onChange={(e) => set('description', e.target.value)} placeholder="Add description" />
            </div>
            <div className="modal-field">
              <label className="modal-field-label">Location</label>
              <input className="modal-input" type="text" value={form.location ?? ''}
                onChange={(e) => set('location', e.target.value)} placeholder="Add location" />
            </div>
          </div>

          <div className="modal-footer">
            {isEdit && onDelete && (
              <button type="button" className="modal-btn modal-btn-danger"
                onClick={() => { onDelete!(event, scope); onClose() }}>
                Delete
              </button>
            )}
            <button type="button" className="modal-btn modal-btn-secondary" onClick={onClose}>Cancel</button>
            <button type="submit" className="modal-btn modal-btn-primary">{isEdit ? 'Save' : 'Create'}</button>
          </div>
        </form>
      </div>
    </div>
  )
}
