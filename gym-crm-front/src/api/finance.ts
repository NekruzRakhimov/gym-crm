import api from '../lib/axios'

export interface FinanceStats {
  total_revenue: number
  monthly_revenue: { month: string; revenue: number }[]
  top_tariffs: { tariff_name: string; count: number; revenue: number }[]
  total_clients: number
  active_clients: number
}

export const financeApi = {
  getStats: (params?: { from?: string; to?: string }) =>
    api.get<FinanceStats>('/api/finance/stats', { params }),
}
