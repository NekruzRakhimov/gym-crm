import { useQuery } from '@tanstack/react-query'
import { financeApi } from '../api/finance'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/table'
import { Spinner } from '../components/ui/spinner'

export function Finance() {
  const { data: stats, isLoading } = useQuery({
    queryKey: ['finance-stats'],
    queryFn: () => financeApi.getStats().then(r => r.data),
    refetchInterval: 60_000,
  })

  if (isLoading) return <div className="flex justify-center p-12"><Spinner /></div>
  if (!stats) return null

  return (
    <div className="p-6 space-y-6">
      <h1 className="text-2xl font-bold">Финансы</h1>

      {/* Summary cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="pb-2"><CardTitle className="text-sm text-muted-foreground">Общая выручка</CardTitle></CardHeader>
          <CardContent><p className="text-2xl font-bold text-green-600">{stats.total_revenue.toLocaleString()} сомони</p></CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2"><CardTitle className="text-sm text-muted-foreground">Всего клиентов</CardTitle></CardHeader>
          <CardContent><p className="text-2xl font-bold">{stats.total_clients}</p></CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2"><CardTitle className="text-sm text-muted-foreground">Активных клиентов</CardTitle></CardHeader>
          <CardContent><p className="text-2xl font-bold text-blue-600">{stats.active_clients}</p></CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2"><CardTitle className="text-sm text-muted-foreground">Среднее на клиента</CardTitle></CardHeader>
          <CardContent><p className="text-2xl font-bold">{stats.total_clients > 0 ? (stats.total_revenue / stats.total_clients).toLocaleString(undefined, { maximumFractionDigits: 0 }) : 0} сомони</p></CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Monthly revenue */}
        <Card>
          <CardHeader><CardTitle>Выручка по месяцам</CardTitle></CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Месяц</TableHead>
                  <TableHead className="text-right">Сумма</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {stats.monthly_revenue.map((m) => (
                  <TableRow key={m.month}>
                    <TableCell>{m.month}</TableCell>
                    <TableCell className="text-right font-medium text-green-600">{m.revenue.toLocaleString()} сомони</TableCell>
                  </TableRow>
                ))}
                {stats.monthly_revenue.length === 0 && (
                  <TableRow><TableCell colSpan={2} className="text-center text-muted-foreground">Нет данных</TableCell></TableRow>
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        {/* Top tariffs */}
        <Card>
          <CardHeader><CardTitle>Топ тарифы</CardTitle></CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Тариф</TableHead>
                  <TableHead className="text-right">Продаж</TableHead>
                  <TableHead className="text-right">Выручка</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {stats.top_tariffs.map((t) => (
                  <TableRow key={t.tariff_name}>
                    <TableCell className="font-medium">{t.tariff_name}</TableCell>
                    <TableCell className="text-right">{t.count}</TableCell>
                    <TableCell className="text-right text-green-600">{t.revenue.toLocaleString()} сомони</TableCell>
                  </TableRow>
                ))}
                {stats.top_tariffs.length === 0 && (
                  <TableRow><TableCell colSpan={3} className="text-center text-muted-foreground">Нет данных</TableCell></TableRow>
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
