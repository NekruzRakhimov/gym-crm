import api from '../lib/axios'

export interface LoginInput {
  username: string
  password: string
}

export interface LoginResponse {
  access_token: string
}

export const authApi = {
  login: (data: LoginInput) => api.post<LoginResponse>('/api/auth/login', data),
  refresh: () => api.post<LoginResponse>('/api/auth/refresh'),
  logout: () => api.post('/api/auth/logout'),
}
