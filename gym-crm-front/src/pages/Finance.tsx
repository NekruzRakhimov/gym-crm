import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { format, startOfMonth, endOfMonth, subMonths } from 'date-fns'
import { financeApi } from '../api/finance'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/table'
import { Spinner } from '../components/ui/spinner'
import { Input } from '../components/ui/input'
import { Button } from '../components/ui/button'

type QuickRange = 'this_month' | 'last_month' | 'last_3_months' | 'all'

function getQuickRange(range: QuickRange): { from: string; to: string } | null {
  const today = new Date()
  switch (range) {
    case 'this_month':
      return {
        from: format(startOfMonth(today), 'yyyy-MM-dd'),
        to: format(endOfMonth(today), 'yyyy-MM-dd'),
      }
    case 'last_month': {
      const prev = subMonths(today, 1)
      return {
        from: format(startOfMonth(prev), 'yyyy-MM-dd'),
        to: format(endOfMonth(prev), 'yyyy-MM-dd'),
      }
    }
    case 'last_3_months':
      return {
        from: format(startOfMonth(subMonths(today, 2)), 'yyyy-MM-dd'),
        to: format(endOfMonth(today), 'yyyy-MM-dd'),
      }
    case 'all':
      return null
  }
}

const quickButtons: { label: string; value: QuickRange }[] = [
  { label: 'Этот месяц', value: 'this_month' },
  { label: 'Прошлый месяц', value: 'last_month' },
  { label: '3 месяца', value: 'last_3_months' },
  { label: 'За всё время', value: 'all' },
]

export function Finance() {
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [activeQuick, setActiveQuick] = useState<QuickRange | null>(null)
  const [exporting, setExporting] = useState(false)

  async function handleExport() {
    setExporting(true)
    try {
      const params = from || to ? { from: from || undefined, to: to || undefined } : undefined
      const res = await financeApi.exportExcel(params)
      const url = URL.createObjectURL(new Blob([res.data]))
      const a = document.createElement('a')
      a.href = url
      a.download = `finance_${format(new Date(), 'yyyy-MM-dd')}.xlsx`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      setTimeout(() => URL.revokeObjectURL(url), 100)
    } catch {
      alert('Ошибка при экспорте. Попробуйте ещё раз.')
    } finally {
      setExporting(false)
    }
  }

  const params = from || to ? { from: from || undefined, to: to || undefined } : undefined

  const { data: stats, isLoading } = useQuery({
    queryKey: ['finance-stats', from, to],
    queryFn: () => financeApi.getStats(params).then(r => r.data),
    refetchInterval: 60_000,
  })

  function applyQuick(range: QuickRange) {
    setActiveQuick(range)
    const r = getQuickRange(range)
    if (r) {
      setFrom(r.from)
      setTo(r.to)
    } else {
      setFrom('')
      setTo('')
    }
  }

  function handleFromChange(val: string) {
    setFrom(val)
    setActiveQuick(null)
  }

  function handleToChange(val: string) {
    setTo(val)
    setActiveQuick(null)
  }

  const periodLabel = from || to
    ? `${from || '...'} — ${to || '...'}`
    : 'За всё время'

  const breakdownLabel = (from || to) ? 'Выручка по дням' : 'Выручка по месяцам'

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Финансы</h1>
        <Button onClick={handleExport} disabled={exporting} variant="outline" size="sm">
          {exporting ? 'Экспорт...' : 'Экспорт Excel'}
        </Button>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="pt-4">
          <div className="flex flex-wrap items-end gap-3">
            <div className="flex gap-2">
              {quickButtons.map(({ label, value }) => (
                <Button
                  key={value}
                  size="sm"
                  variant={activeQuick === value ? 'default' : 'outline'}
                  onClick={() => applyQuick(value)}
                >
                  {label}
                </Button>
              ))}
            </div>
            <div className="flex items-center gap-2 ml-auto">
              <span className="text-sm text-muted-foreground">С</span>
              <Input
                type="date"
                className="w-38"
                value={from}
                onChange={e => handleFromChange(e.target.value)}
              />
              <span className="text-sm text-muted-foreground">По</span>
              <Input
                type="date"
                className="w-38"
                value={to}
                onChange={e => handleToChange(e.target.value)}
              />
              {(from || to) && (
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => { setFrom(''); setTo(''); setActiveQuick(null) }}
                >
                  Сбросить
                </Button>
              )}
            </div>
          </div>
          <p className="text-xs text-muted-foreground mt-2">Период: {periodLabel}</p>
        </CardContent>
      </Card>

      {isLoading ? (
        <div className="flex justify-center p-12"><Spinner /></div>
      ) : !stats ? null : (
        <>
          {/* Summary cards */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <Card>
              <CardHeader className="pb-2"><CardTitle className="text-sm text-muted-foreground">Выручка</CardTitle></CardHeader>
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
              <CardContent>
                <p className="text-2xl font-bold">
                  {stats.total_clients > 0
                    ? (stats.total_revenue / stats.total_clients).toLocaleString(undefined, { maximumFractionDigits: 0 })
                    : 0} сомони
                </p>
              </CardContent>
            </Card>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* Revenue breakdown */}
            <Card>
              <CardHeader><CardTitle>{breakdownLabel}</CardTitle></CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{(from || to) ? 'День' : 'Месяц'}</TableHead>
                      <TableHead className="text-right">Сумма</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {(stats.monthly_revenue ?? []).map((m) => (
                      <TableRow key={m.month}>
                        <TableCell>{m.month}</TableCell>
                        <TableCell className="text-right font-medium text-green-600">
                          {m.revenue.toLocaleString()} сомони
                        </TableCell>
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
                    {(stats.top_tariffs ?? []).map((t) => (
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
        </>
      )}
    </div>
  )
}
