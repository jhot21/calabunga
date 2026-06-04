import { Fragment, useRef, useEffect } from 'react'
import { isSameDay, isToday, format } from 'date-fns'
import { weekDays, eventTopPx, eventHeightPx, formatTime, HOUR_HEIGHT } from '../../lib/dateUtils'
import type { Event, Calendar } from '../../types/api'

const HOURS = Array.from({ length: 24 }, (_, i) => i)
const DOW_SHORT = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

interface WeekViewProps {
  currentDate: Date
  events: Event[]
  calendars: Calendar[]
  onSlotClick: (date: Date) => void
  onEventClick: (event: Event, anchor: HTMLElement) => void
}

export default function WeekView({
  currentDate, events, calendars, onSlotClick, onEventClick,
}: WeekViewProps) {
  const days = weekDays(currentDate)
  const calMap = new Map(calendars.map((c) => [c.id, c]))
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (scrollRef.current) scrollRef.current.scrollTop = 8 * HOUR_HEIGHT
  }, [])

  const timedForDay = (day: Date) => events.filter((e) => !e.allDay && isSameDay(new Date(e.start), day))
  const allDayForDay = (day: Date) => events.filter((e) => e.allDay && isSameDay(new Date(e.start), day))

  return (
    <div className="week-view">
      <div className="week-header">
        <div className="week-header-gutter" />
        {days.map((day, i) => {
          const dow = day.getDay()
          return (
            <div key={i} className="week-header-day">
              <div className={`week-header-day-name${(dow === 0 || dow === 6) ? ' weekend' : ''}`}>
                {DOW_SHORT[dow]}
              </div>
              <div className={`week-header-day-num${isToday(day) ? ' today' : ''}`}>
                {day.getDate()}
              </div>
            </div>
          )
        })}
      </div>

      <div className="week-all-day-row">
        <div className="week-all-day-gutter">all-day</div>
        {days.map((day, i) => (
          <div key={i} className="week-all-day-col">
            {allDayForDay(day).map((ev) => {
              const color = calMap.get(ev.calendarId)?.color ?? '#4285f4'
              return (
                <div
                  key={ev.uid}
                  className="event-chip all-day"
                  style={{ backgroundColor: color }}
                  onClick={(e) => { e.stopPropagation(); onEventClick(ev, e.currentTarget) }}
                >
                  <span className="event-chip-title">{ev.title}</span>
                </div>
              )
            })}
          </div>
        ))}
      </div>

      <div className="week-scroll" ref={scrollRef}>
        <div className="week-time-grid" style={{ height: HOUR_HEIGHT * 24 }}>
          {HOURS.map((hour) => (
            <Fragment key={hour}>
              <div className="week-hour-label" style={{ gridRow: hour + 1, gridColumn: 1 }}>
                {hour === 0 ? '' : format(new Date(2026, 0, 1, hour), 'h a')}
              </div>
              {days.map((day, di) => (
                <div
                  key={di}
                  className="week-hour-cell"
                  style={{ gridRow: hour + 1, gridColumn: di + 2 }}
                  onClick={() => {
                    const d = new Date(day)
                    d.setHours(hour, 0, 0, 0)
                    onSlotClick(d)
                  }}
                />
              ))}
            </Fragment>
          ))}

          {days.map((day, di) => {
            const dayEvents = timedForDay(day)
            if (dayEvents.length === 0) return null
            return (
              <div
                key={`col-${di}`}
                className="week-events-col"
                style={{ gridColumn: di + 2 }}
              >
                {dayEvents.map((ev) => {
                  const color = calMap.get(ev.calendarId)?.color ?? '#4285f4'
                  return (
                    <div
                      key={ev.uid}
                      className="week-event"
                      style={{
                        top: eventTopPx(ev.start),
                        height: eventHeightPx(ev.start, ev.end),
                        backgroundColor: color,
                        pointerEvents: 'auto',
                      }}
                      onClick={(e) => { e.stopPropagation(); onEventClick(ev, e.currentTarget) }}
                    >
                      <div className="week-event-title">{ev.title}</div>
                      <div className="week-event-time">
                        {formatTime(ev.start)} – {formatTime(ev.end)}
                      </div>
                    </div>
                  )
                })}
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
