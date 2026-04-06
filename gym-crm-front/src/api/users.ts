import api from '../lib/axios'

export interface AdminUser {
  id: number
  username: string
  role: string
  created_at: string
}

export interface CreateAdminInput {
  username: string
  password: string
  role: string
}

export const usersApi = {
  list: () => api.get<AdminUser[]>('/api/users'),
  create: (data: CreateAdminInput) => api.post<AdminUser>('/api/users', data),
  delete: (id: number) => api.delete(`/api/users/${id}`),
}
