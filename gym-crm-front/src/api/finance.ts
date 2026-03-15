import api from '../lib/axios'

export interface FinanceStats {
  total_revenue: number
  monthly_revenue: { month: string; revenue: number }[]
  top_tariffs: { tariff_name: string; count: number; revenue: number }[]
  total_clients: number
  active_clients: number
}

export const financeApi = {
  getStats: () => api.get<FinanceStats>('/api/finance/stats'),
}
