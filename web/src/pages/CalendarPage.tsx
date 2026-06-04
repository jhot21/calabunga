import { useState, useCallback } from 'react'
import { useCalendarStore } from '../store/calendarStore'
import { useCalendars } from '../hooks/useCalendars'
import { useAllEvents, useCreateEvent, useUpdateEvent, useDeleteEvent } from '../hooks/useEvents'
import { monthRange, weekRange } from '../lib/dateUtils'
import TopBar from '../components/layout/TopBar'
import Sidebar from '../components/layout/Sidebar'
import MonthView from '../components/views/MonthView'
import WeekView from '../components/views/WeekView'
import EventModal from '../components/modals/EventModal'
import EventPopover from '../components/modals/EventPopover'
import ToastList from '../components/Toast'
import type { Event, EditScope } from '../types/api'

export default function CalendarPage() {
  const { currentView, currentDate } = useCalendarStore()
  const currentDateObj = new Date(currentDate)

  const { calendars, visibleIds, visibleCalendars, setVisible } = useCalendars()
  const { start, end } = currentView === 'month'
    ? monthRange(currentDateObj)
    : weekRange(currentDateObj)
  const { events } = useAllEvents(visibleIds, start, end)

  const createEvent = useCreateEvent()
  const updateEvent = useUpdateEvent()
  const deleteEvent = useDeleteEvent()

  const [modalEvent, setModalEvent] = useState<Event | undefined>()
  const [modalDate, setModalDate] = useState<Date | undefined>()
  const [modalOpen, setModalOpen] = useState(false)
  const [popoverEvent, setPopoverEvent] = useState<Event | null>(null)
  const [popoverRect, setPopoverRect] = useState<DOMRect | null>(null)

  const openCreateModal = useCallback((date?: Date) => {
    setModalEvent(undefined); setModalDate(date); setModalOpen(true)
  }, [])

  const openEditModal = useCallback((ev: Event) => {
    setModalEvent(ev); setModalDate(undefined); setModalOpen(true)
  }, [])

  const openPopover = useCallback((ev: Event, anchor: HTMLElement) => {
    setPopoverEvent(ev); setPopoverRect(anchor.getBoundingClientRect())
  }, [])

  const closePopover = useCallback(() => {
    setPopoverEvent(null); setPopoverRect(null)
  }, [])

  const handleSave = (ev: Event, scope: EditScope) => {
    if (ev.uid) updateEvent.mutate({ event: ev, scope })
    else createEvent.mutate(ev)
  }

  const handleDelete = (ev: Event, scope: EditScope) =>
    deleteEvent.mutate({ event: ev, scope })

  const calMap = new Map(calendars.map((c) => [c.id, c]))

  return (
    <div className="calendar-page">
      <TopBar onNewEvent={() => openCreateModal()} />
      <Sidebar
        calendars={calendars}
        visibleCalendars={visibleCalendars}
        onToggleCalendar={setVisible}
        onNewEvent={() => openCreateModal()}
      />
      <main className="view-container">
        {currentView === 'month' ? (
          <MonthView
            currentDate={currentDateObj}
            events={events}
            calendars={calendars}
            onDayClick={(date) => openCreateModal(date)}
            onEventClick={openPopover}
          />
        ) : (
          <WeekView
            currentDate={currentDateObj}
            events={events}
            calendars={calendars}
            onSlotClick={(date) => openCreateModal(date)}
            onEventClick={openPopover}
          />
        )}
      </main>

      {modalOpen && (
        <EventModal
          event={modalEvent}
          initialDate={modalDate}
          calendars={calendars}
          onClose={() => setModalOpen(false)}
          onSave={handleSave}
          onDelete={modalEvent ? handleDelete : undefined}
        />
      )}

      {popoverEvent && popoverRect && (
        <EventPopover
          event={popoverEvent}
          calendar={calMap.get(popoverEvent.calendarId)}
          anchorRect={popoverRect}
          onClose={closePopover}
          onEdit={(ev) => { closePopover(); openEditModal(ev) }}
          onDelete={(ev, scope) => { handleDelete(ev, scope); closePopover() }}
        />
      )}

      <ToastList />
    </div>
  )
}
