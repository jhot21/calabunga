import { useQueries, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, ApiError } from '../lib/api'
import { useUIStore } from '../store/uiStore'
import { applyEdit } from '../lib/eventCache'
import type { Event, EditScope } from '../types/api'

export function useAllEvents(calendarIds: string[], start: Date, end: Date) {
  const startISO = start.toISOString()
  const endISO = end.toISOString()

  const results = useQueries({
    queries: calendarIds.map((calId) => ({
      queryKey: ['events', calId, startISO, endISO],
      queryFn: () => api.events({ calendarId: calId, start: startISO, end: endISO }),
      enabled: calendarIds.length > 0,
      staleTime: 30_000,
    })),
  })

  const allEvents = results.flatMap((r) => r.data ?? [])
  const isLoading = results.some((r) => r.isLoading)
  return { events: allEvents, isLoading }
}

export function useCreateEvent() {
  const queryClient = useQueryClient()
  const addToast = useUIStore((s) => s.addToast)

  return useMutation({
    mutationFn: (event: Event) => api.createEvent(event),
    onMutate: async (newEvent) => {
      await queryClient.cancelQueries({ queryKey: ['events', newEvent.calendarId] })
      const snapshots = queryClient.getQueriesData<Event[]>({ queryKey: ['events', newEvent.calendarId] })
      queryClient.setQueriesData<Event[]>(
        { queryKey: ['events', newEvent.calendarId] },
        (old) => [...(old ?? []), { ...newEvent, uid: `optimistic-${Date.now()}` }]
      )
      return { snapshots }
    },
    onError: (err, _vars, ctx) => {
      ctx?.snapshots.forEach(([key, data]) => queryClient.setQueryData(key, data))
      const msg = err instanceof ApiError ? err.message : 'Failed to create event'
      addToast(msg)
    },
    onSettled: (_data, _err, vars) => {
      queryClient.invalidateQueries({ queryKey: ['events', vars.calendarId] })
    },
  })
}

export function useUpdateEvent() {
  const queryClient = useQueryClient()
  const addToast = useUIStore((s) => s.addToast)

  return useMutation({
    mutationFn: ({ event, scope, originalStart }: { event: Event; scope: EditScope; originalStart?: string }) =>
      api.updateEvent(event.uid, { ...event, originalStart }, scope, event.calendarId),
    onMutate: async ({ event, originalStart }) => {
      await queryClient.cancelQueries({ queryKey: ['events', event.calendarId] })
      const snapshots = queryClient.getQueriesData<Event[]>({ queryKey: ['events', event.calendarId] })
      queryClient.setQueriesData<Event[]>(
        { queryKey: ['events', event.calendarId] },
        (old) => applyEdit(old, event, originalStart)
      )
      return { snapshots }
    },
    onError: (err, vars, ctx) => {
      ctx?.snapshots.forEach(([key, data]) => queryClient.setQueryData(key, data))
      if (err instanceof ApiError && err.code === 'conflict') {
        addToast('Event was modified externally', {
          label: 'Refresh',
          onClick: () =>
            queryClient.invalidateQueries({ queryKey: ['events', vars.event.calendarId] }),
        })
      } else {
        const msg = err instanceof ApiError ? err.message : 'Failed to update event'
        addToast(msg)
      }
    },
    onSettled: (_data, _err, vars) => {
      queryClient.invalidateQueries({ queryKey: ['events', vars.event.calendarId] })
    },
  })
}

export function useDeleteEvent() {
  const queryClient = useQueryClient()
  const addToast = useUIStore((s) => s.addToast)

  return useMutation({
    mutationFn: ({ event, scope }: { event: Event; scope: EditScope }) =>
      api.deleteEvent(event.uid, scope, event.calendarId, event.etag),
    onMutate: async ({ event }) => {
      await queryClient.cancelQueries({ queryKey: ['events', event.calendarId] })
      const snapshots = queryClient.getQueriesData<Event[]>({ queryKey: ['events', event.calendarId] })
      queryClient.setQueriesData<Event[]>(
        { queryKey: ['events', event.calendarId] },
        (old) => (old ?? []).filter((e) => e.uid !== event.uid)
      )
      return { snapshots }
    },
    onError: (err, vars, ctx) => {
      ctx?.snapshots.forEach(([key, data]) => queryClient.setQueryData(key, data))
      if (err instanceof ApiError && err.code === 'conflict') {
        addToast('Event was modified externally', {
          label: 'Refresh',
          onClick: () =>
            queryClient.invalidateQueries({ queryKey: ['events', vars.event.calendarId] }),
        })
      } else {
        const msg = err instanceof ApiError ? err.message : 'Failed to delete event'
        addToast(msg)
      }
    },
    onSettled: (_data, _err, vars) => {
      queryClient.invalidateQueries({ queryKey: ['events', vars.event.calendarId] })
    },
  })
}
