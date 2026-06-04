import { useState } from 'react'
import { addMonths, subMonths, isSameDay, isToday, getMonth, getYear, format } from 'date-fns'
import { useCalendarStore } from '../../store/calendarStore'
import { calendarGrid } from '../../lib/dateUtils'
import type { Calendar } from '../../types/api'

const DOW = ['S', 'M', 'T', 'W', 'T', 'F', 'S']

interface SidebarProps {
  calendars: Calendar[]
  visibleCalendars: Record<string, boolean>
  onToggleCalendar: (id: string, visible: boolean) => void
  onNewEvent: () => void
}

export default function Sidebar({
  calendars, visibleCalendars, onToggleCalendar, onNewEvent,
}: SidebarProps) {
  const { currentDate, setCurrentDate } = useCalendarStore()
  const current = new Date(currentDate)
  const [miniDate, setMiniDate] = useState(
    () => new Date(current.getFullYear(), current.getMonth(), 1)
  )

  const grid = calendarGrid(getYear(miniDate), getMonth(miniDate))

  const handleDayClick = (day: Date) => {
    const next = new Date(current)
    next.setFullYear(day.getFullYear(), day.getMonth(), day.getDate())
    setCurrentDate(next.toISOString())
  }

  return (
    <aside className="sidebar">
      <button className="sidebar-add-btn" onClick={onNewEvent}>
        <span style={{ fontSize: 24, lineHeight: 1 }}>+</span>
        <span>New event</span>
      </button>

      <div className="mini-month">
        <div className="mini-month-header">
          <span className="mini-month-title">{format(miniDate, 'MMMM yyyy')}</span>
          <div style={{ display: 'flex', gap: 4 }}>
            <button className="mini-month-nav" onClick={() => setMiniDate((d) => subMonths(d, 1))}>‹</button>
            <button className="mini-month-nav" onClick={() => setMiniDate((d) => addMonths(d, 1))}>›</button>
          </div>
        </div>
        <div className="mini-month-grid">
          {DOW.map((d, i) => <div key={i} className="mini-month-dow">{d}</div>)}
          {grid.flat().map((day, i) => {
            const isOther = day.getMonth() !== miniDate.getMonth()
            const isSel = isSameDay(day, current)
            const isTod = isToday(day)
            let cls = 'mini-month-day'
            if (isOther) cls += ' other-month'
            if (isTod) cls += ' today'
            if (isSel) cls += ' selected'
            return (
              <button key={i} className={cls} onClick={() => handleDayClick(day)}>
                {day.getDate()}
              </button>
            )
          })}
        </div>
      </div>

      <div className="calendar-list-title">My calendars</div>
      {calendars.map((cal) => {
        const visible = visibleCalendars[cal.id] !== false
        return (
          <div
            key={cal.id}
            className="calendar-item"
            onClick={() => onToggleCalendar(cal.id, !visible)}
          >
            <div
              className={`calendar-swatch${visible ? '' : ' unchecked'}`}
              style={{ backgroundColor: cal.color }}
            />
            <span className="calendar-name">{cal.name}</span>
          </div>
        )
      })}
    </aside>
  )
}
