import api from '../lib/axios'

export interface DashboardStats {
  inside_now: number
  today_entries: number
  today_exits: number
  today_denied: number
}

export const dashboardApi = {
  getStats: () => api.get<DashboardStats>('/api/dashboard/stats'),
}
