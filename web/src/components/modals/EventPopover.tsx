import { format } from 'date-fns'
import { MdEdit, MdDelete, MdClose, MdSchedule, MdRepeat, MdNotes, MdLocationOn } from 'react-icons/md'
import { recurrenceLabel } from '../../lib/rrule'
import type { Event, Calendar, EditScope } from '../../types/api'

interface EventPopoverProps {
  event: Event
  calendar?: Calendar
  anchorRect: DOMRect
  onClose: () => void
  onEdit: (event: Event) => void
  onDelete: (event: Event, scope: EditScope) => void
}

export default function EventPopover({
  event, calendar, anchorRect, onClose, onEdit, onDelete,
}: EventPopoverProps) {
  const POPOVER_W = 320
  const POPOVER_H = 260
  let left = anchorRect.right + 8
  let top = anchorRect.top
  if (left + POPOVER_W > window.innerWidth - 8) left = anchorRect.left - POPOVER_W - 8
  if (top + POPOVER_H > window.innerHeight - 8) top = window.innerHeight - POPOVER_H - 8

  const startDate = new Date(event.start)
  const endDate = new Date(event.end)
  const color = calendar?.color ?? '#4285f4'
  const timeStr = event.allDay
    ? format(startDate, 'EEEE, MMMM d')
    : `${format(startDate, 'EEE, MMM d')} · ${format(startDate, 'h:mm a')} – ${format(endDate, 'h:mm a')}`

  return (
    <>
      <div className="popover-backdrop" onClick={onClose} />
      <div className="popover" style={{ left: Math.max(8, left), top: Math.max(8, top) }}>
        <div className="popover-header">
          <button
            className="popover-action-btn"
            onClick={() => { onEdit(event); onClose() }}
            title="Edit"
          ><MdEdit size={18} /></button>
          <button
            className="popover-action-btn"
            onClick={() => { onDelete(event, 'all'); onClose() }}
            title="Delete"
          ><MdDelete size={18} /></button>
          <button className="popover-action-btn" onClick={onClose} title="Close"><MdClose size={18} /></button>
        </div>
        <div style={{ height: 4, backgroundColor: color }} />
        <div className="popover-body">
          <div className="popover-title">{event.title}</div>
          <div className="popover-meta"><MdSchedule size={16} className="popover-icon" /><span>{timeStr}</span></div>
          {event.recurrence && (
            <div className="popover-meta">
              <MdRepeat size={16} className="popover-icon" />
              <span>{recurrenceLabel(event.recurrence)}</span>
            </div>
          )}
          {calendar && (
            <div className="popover-meta">
              <span style={{ width: 12, height: 12, borderRadius: 3, backgroundColor: color, display: 'inline-block', flexShrink: 0 }} />
              <span>{calendar.name}</span>
            </div>
          )}
          {event.description && (
            <div className="popover-meta"><MdNotes size={16} className="popover-icon" /><span style={{ whiteSpace: 'pre-wrap' }}>{event.description}</span></div>
          )}
          {event.location && (
            <div className="popover-meta"><MdLocationOn size={16} className="popover-icon" /><span>{event.location}</span></div>
          )}
        </div>
      </div>
    </>
  )
}
