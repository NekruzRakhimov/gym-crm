import api from '../lib/axios'

export interface Client {
  id: number
  full_name: string
  phone: string | null
  photo_path: string | null
  card_number: string | null
  is_active: boolean
  created_at: string
  active_tariff_name?: string | null
  active_tariff_end?: string | null
  balance: number
}

export interface Transaction {
  id: number
  client_id: number
  type: 'deposit' | 'payment'
  amount: number
  description: string | null
  client_tariff_id: number | null
  created_at: string
}

export interface ClientTariffDetail {
  id: number
  client_id: number
  tariff_id: number
  start_date: string
  end_date: string
  paid_amount: number | null
  payment_note: string | null
  created_at: string
  tariff_name: string
  duration_days: number
  max_visits_per_day: number | null
}

export interface CreateClientInput {
  full_name: string
  phone?: string | null
  card_number?: string | null
}

export interface AssignTariffInput {
  tariff_id: number
  start_date: string
}

export const clientsApi = {
  list: (params: { search?: string; page?: number; limit?: number }) =>
    api.get<{ items: Client[]; total: number }>('/api/clients', { params }),
  getById: (id: number) => api.get<Client>(`/api/clients/${id}`),
  create: (data: CreateClientInput) => api.post<Client>('/api/clients', data),
  update: (id: number, data: CreateClientInput) => api.put<Client>(`/api/clients/${id}`, data),
  uploadPhoto: (id: number, file: File) => {
    const form = new FormData()
    form.append('photo', file)
    return api.post(`/api/clients/${id}/photo`, form)
  },
  block: (id: number) => api.post(`/api/clients/${id}/block`),
  unblock: (id: number) => api.post(`/api/clients/${id}/unblock`),
  getEvents: (id: number, params: { page?: number; limit?: number }) =>
    api.get('/api/clients/' + id + '/events', { params }),
  getPayments: (id: number) =>
    api.get<ClientTariffDetail[]>('/api/clients/' + id + '/payments'),
  assignTariff: (id: number, data: AssignTariffInput) =>
    api.post('/api/clients/' + id + '/assign-tariff', data),
  revokeTariff: (id: number, tariffRecordId: number) =>
    api.delete(`/api/clients/${id}/tariffs/${tariffRecordId}`),
  getActiveTariff: (id: number) =>
    api.get<ClientTariffDetail | null>('/api/clients/' + id + '/active-tariff'),
  deposit: (id: number, data: { amount: number; description?: string }) =>
    api.post('/api/clients/' + id + '/deposit', data),
  getTransactions: (id: number) =>
    api.get<Transaction[]>('/api/clients/' + id + '/transactions'),
}
