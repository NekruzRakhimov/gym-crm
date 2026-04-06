import { useQuery } from '@tanstack/react-query'
import { dashboardApi } from '../api/dashboard'
import { LiveFeed } from '../components/LiveFeed'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card'
import { Users, LogIn, LogOut, ShieldX } from 'lucide-react'

export function Dashboard() {
  const { data: stats } = useQuery({
    queryKey: ['dashboard-stats'],
    queryFn: () => dashboardApi.getStats().then((r) => r.data),
    refetchInterval: 30_000,
  })

  const statCards = [
    {
      label: 'Сейчас внутри',
      value: stats?.inside_now ?? '-',
      icon: Users,
      color: 'text-blue-600',
    },
    {
      label: 'Входов сегодня',
      value: stats?.today_entries ?? '-',
      icon: LogIn,
      color: 'text-green-600',
    },
    {
      label: 'Выходов сегодня',
      value: stats?.today_exits ?? '-',
      icon: LogOut,
      color: 'text-orange-600',
    },
    {
      label: 'Отказов сегодня',
      value: stats?.today_denied ?? '-',
      icon: ShieldX,
      color: 'text-red-600',
    },
  ]

  return (
    <div className="p-6 space-y-6">
      <h1 className="text-2xl font-bold">Главная</h1>

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        {statCards.map(({ label, value, icon: Icon, color }) => (
          <Card key={label}>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">{label}</CardTitle>
              <Icon className={`w-5 h-5 ${color}`} />
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold">{value}</div>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Живая лента событий</CardTitle>
        </CardHeader>
        <CardContent>
          <LiveFeed />
        </CardContent>
      </Card>
    </div>
  )
}
