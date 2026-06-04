import { useEffect, useRef } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../lib/api'
import { useCalendarStore } from '../store/calendarStore'
import type { Calendar } from '../types/api'

export function useCalendars() {
  const { visibleCalendars, setVisible } = useCalendarStore()
  const queryClient = useQueryClient()
  const prevCtags = useRef<Record<string, string>>({})

  const query = useQuery({
    queryKey: ['calendars'],
    queryFn: () => api.calendars(),
    refetchInterval: 60_000,
    staleTime: 30_000,
  })

  useEffect(() => {
    if (!query.data) return

    query.data.forEach((cal) => {
      if (!(cal.id in visibleCalendars)) setVisible(cal.id, true)
    })

    query.data.forEach((cal) => {
      const prev = prevCtags.current[cal.id]
      if (prev !== undefined && prev !== cal.ctag) {
        queryClient.invalidateQueries({ queryKey: ['events', cal.id] })
      }
      prevCtags.current[cal.id] = cal.ctag
    })
  }, [query.data])

  const calendars: Calendar[] = query.data ?? []
  const visibleIds = calendars
    .filter((c) => visibleCalendars[c.id] !== false)
    .map((c) => c.id)

  return {
    calendars,
    visibleIds,
    visibleCalendars,
    setVisible,
    isLoading: query.isLoading,
  }
}
