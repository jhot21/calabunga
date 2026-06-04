import { isSameDay, isToday } from 'date-fns'
import { calendarGrid } from '../../lib/dateUtils'
import EventChip from './EventChip'
import type { Event, Calendar } from '../../types/api'

const DOW_LABELS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']
const MAX_CHIPS = 3

interface MonthViewProps {
  currentDate: Date
  events: Event[]
  calendars: Calendar[]
  onDayClick: (date: Date) => void
  onEventClick: (event: Event, anchor: HTMLElement) => void
}

export default function MonthView({
  currentDate, events, calendars, onDayClick, onEventClick,
}: MonthViewProps) {
  const grid = calendarGrid(currentDate.getFullYear(), currentDate.getMonth())
  const calMap = new Map(calendars.map((c) => [c.id, c]))

  const eventsForDay = (day: Date) =>
    events
      .filter((e) => isSameDay(new Date(e.start), day))
      .sort((a, b) => a.start.localeCompare(b.start))

  return (
    <div className="month-view">
      <div className="month-dow-header">
        {DOW_LABELS.map((d, i) => (
          <div key={d} className={`month-dow-label${i === 0 || i === 6 ? ' weekend' : ''}`}>
            {d}
          </div>
        ))}
      </div>
      <div className="month-grid">
        {grid.flat().map((day, i) => {
          const isOther = day.getMonth() !== currentDate.getMonth()
          const dayEvents = eventsForDay(day)
          const overflow = dayEvents.length - MAX_CHIPS
          const dow = day.getDay()
          return (
            <div
              key={i}
              className={`month-cell${isOther ? ' other-month' : ''}`}
              onClick={() => onDayClick(day)}
            >
              <div className={`month-day-num${isToday(day) ? ' today' : ''}${(dow === 0 || dow === 6) ? ' weekend' : ''}`}>
                {day.getDate()}
              </div>
              {dayEvents.slice(0, MAX_CHIPS).map((ev) => (
                <EventChip key={ev.uid} event={ev} calendar={calMap.get(ev.calendarId)} onClick={onEventClick} />
              ))}
              {overflow > 0 && <div className="month-more">{overflow} more</div>}
            </div>
          )
        })}
      </div>
    </div>
  )
}
