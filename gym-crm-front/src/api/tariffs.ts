import api from '../lib/axios'

export interface Tariff {
  id: number
  name: string
  duration_days: number
  max_visit_days: number | null
  price: number
  active: boolean
  schedule_days: string        // all | weekdays | weekends | even | odd
  time_from: string | null     // "HH:MM" or null
  time_to: string | null       // "HH:MM" or null
  created_at: string
}

export interface CreateTariffInput {
  name: string
  duration_days: number
  max_visit_days?: number | null
  price: number
  schedule_days?: string
  time_from?: string | null
  time_to?: string | null
}

export const tariffsApi = {
  list: () => api.get<Tariff[]>('/api/tariffs'),
  create: (data: CreateTariffInput) => api.post<Tariff>('/api/tariffs', data),
  update: (id: number, data: CreateTariffInput) => api.put<Tariff>(`/api/tariffs/${id}`, data),
  delete: (id: number) => api.delete(`/api/tariffs/${id}`),
  toggle: (id: number) => api.patch<Tariff>(`/api/tariffs/${id}/toggle`),
}
