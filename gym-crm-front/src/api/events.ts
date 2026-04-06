import api from '../lib/axios'

export interface AccessEvent {
  id: number
  client_id: number | null
  terminal_id: number | null
  direction: 'entry' | 'exit'
  auth_method: string | null
  access_granted: boolean
  deny_reason: string | null
  event_time: string
  client_name?: string | null
  client_photo?: string | null
  terminal_name?: string | null
}

export interface EventsFilter {
  from?: string
  to?: string
  client_id?: number
  terminal_id?: number
  direction?: string
  granted?: boolean
  page?: number
  limit?: number
}

export const eventsApi = {
  list: (params: EventsFilter) =>
    api.get<{ items: AccessEvent[]; total: number }>('/api/events', { params }),
}
