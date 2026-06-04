import { formatTime } from '../../lib/dateUtils'
import type { Event, Calendar } from '../../types/api'

interface EventChipProps {
  event: Event
  calendar?: Calendar
  onClick: (event: Event, anchor: HTMLElement) => void
}

export default function EventChip({ event, calendar, onClick }: EventChipProps) {
  const color = calendar?.color ?? '#4285f4'
  return (
    <div
      className="event-chip"
      style={{ backgroundColor: color }}
      onClick={(e) => { e.stopPropagation(); onClick(event, e.currentTarget) }}
      title={event.title}
    >
      {!event.allDay && <span className="event-chip-time">{formatTime(event.start)}</span>}
      <span className="event-chip-title">{event.title}</span>
    </div>
  )
}
