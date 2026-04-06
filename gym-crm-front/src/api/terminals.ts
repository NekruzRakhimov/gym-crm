import api from '../lib/axios'

export interface Terminal {
  id: number
  name: string
  ip: string
  port: number
  username: string
  direction: 'entry' | 'exit'
  active: boolean
  created_at: string
}

export interface CreateTerminalInput {
  name: string
  ip: string
  port?: number
  username: string
  password: string
  direction: 'entry' | 'exit'
}

export const terminalsApi = {
  list: () => api.get<Terminal[]>('/api/terminals'),
  create: (data: CreateTerminalInput) => api.post<Terminal>('/api/terminals', data),
  update: (id: number, data: CreateTerminalInput) => api.put<Terminal>(`/api/terminals/${id}`, data),
  delete: (id: number) => api.delete(`/api/terminals/${id}`),
  getStatus: (id: number) => api.get<{ online: boolean }>(`/api/terminals/${id}/status`),
  openDoor: (id: number) => api.post(`/api/terminals/${id}/open-door`),
  setupWebhook: (id: number) => api.post(`/api/terminals/${id}/setup-webhook`),
  sync: (id: number) => api.post(`/api/terminals/${id}/sync`),
  enableRemoteVerify: (id: number) => api.post(`/api/terminals/${id}/enable-remote-verify`),
}
