import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { clientsApi, type AssignTariffInput, type Transaction } from '../api/clients'
import { tariffsApi } from '../api/tariffs'
import { type AccessEvent } from '../api/events'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Label } from '../components/ui/label'
import { Badge } from '../components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card'
import { Dialog, DialogHeader, DialogTitle, DialogContent, DialogFooter } from '../components/ui/dialog'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/table'
import { StatusBadge } from '../components/StatusBadge'
import { DirectionBadge } from '../components/DirectionBadge'
import { PhotoUpload } from '../components/PhotoUpload'
import { Spinner } from '../components/ui/spinner'
import { ArrowLeft, Trash2 } from 'lucide-react'
import { format } from 'date-fns'

export function ClientDetail() {
  const { id } = useParams<{ id: string }>()
  const clientId = Number(id)
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [activeTab, setActiveTab] = useState<'events' | 'payments' | 'transactions'>('events')
  const [eventsPage, setEventsPage] = useState(1)
  const [showAssign, setShowAssign] = useState(false)
  const [showDeposit, setShowDeposit] = useState(false)
  const [depositAmount, setDepositAmount] = useState('')
  const [depositDesc, setDepositDesc] = useState('')
  const [assignForm, setAssignForm] = useState<AssignTariffInput>({
    tariff_id: 0,
    start_date: new Date().toISOString().split('T')[0],
  })

  const { data: client, isLoading } = useQuery({
    queryKey: ['client', clientId],
    queryFn: () => clientsApi.getById(clientId).then((r) => r.data),
  })

  const { data: activeTariff } = useQuery({
    queryKey: ['client-active-tariff', clientId],
    queryFn: () => clientsApi.getActiveTariff(clientId).then((r) => r.data),
  })

  const { data: eventsData } = useQuery({
    queryKey: ['client-events', clientId, eventsPage],
    queryFn: (): Promise<{ items: AccessEvent[]; total: number }> =>
      clientsApi.getEvents(clientId, { page: eventsPage, limit: 20 }).then((r) => r.data as { items: AccessEvent[]; total: number }),
    enabled: activeTab === 'events',
  })

  const { data: payments } = useQuery({
    queryKey: ['client-payments', clientId],
    queryFn: () => clientsApi.getPayments(clientId).then((r) => r.data),
    enabled: activeTab === 'payments',
  })

  const { data: transactions } = useQuery({
    queryKey: ['client-transactions', clientId],
    queryFn: () => clientsApi.getTransactions(clientId).then(r => r.data),
    enabled: activeTab === 'transactions',
  })

  const { data: tariffs } = useQuery({
    queryKey: ['tariffs'],
    queryFn: () => tariffsApi.list().then((r) => r.data),
    enabled: showAssign,
  })

  const photoMutation = useMutation({
    mutationFn: (file: File) => clientsApi.uploadPhoto(clientId, file),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['client', clientId] }),
  })

  const blockMutation = useMutation({
    mutationFn: () => client?.is_active ? clientsApi.block(clientId) : clientsApi.unblock(clientId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['client', clientId] }),
  })

  const assignMutation = useMutation({
    mutationFn: (data: AssignTariffInput) => clientsApi.assignTariff(clientId, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['client', clientId] })
      qc.invalidateQueries({ queryKey: ['client-active-tariff', clientId] })
      qc.invalidateQueries({ queryKey: ['client-payments', clientId] })
      setShowAssign(false)
    },
  })

  const assignError = assignMutation.error
    ? (assignMutation.error as { response?: { data?: { error?: string } } }).response?.data?.error ?? 'Не удалось назначить тариф'
    : null

  const depositMutation = useMutation({
    mutationFn: () => clientsApi.deposit(clientId, { amount: Number(depositAmount), description: depositDesc || undefined }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['client', clientId] })
      qc.invalidateQueries({ queryKey: ['client-transactions', clientId] })
      setShowDeposit(false)
      setDepositAmount('')
      setDepositDesc('')
    },
  })

  const deleteMutation = useMutation({
    mutationFn: () => clientsApi.delete(clientId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['clients'] })
      navigate('/clients')
    },
  })

  const revokeMutation = useMutation({
    mutationFn: (tariffRecordId: number) => clientsApi.revokeTariff(clientId, tariffRecordId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['client-active-tariff', clientId] })
      qc.invalidateQueries({ queryKey: ['client-payments', clientId] })
    },
  })

  if (isLoading) return <div className="flex justify-center p-12"><Spinner /></div>
  if (!client) return <div className="p-6">Клиент не найден</div>

  return (
    <div className="p-6 space-y-6">
      <button onClick={() => navigate('/clients')} className="flex items-center gap-2 text-muted-foreground hover:text-foreground text-sm">
        <ArrowLeft className="w-4 h-4" />
        Назад к клиентам
      </button>

      {/* Header card */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex items-start gap-6">
            <PhotoUpload
              photoPath={client.photo_path}
              onUpload={(file) => photoMutation.mutate(file)}
              size={96}
            />
            <div className="flex-1">
              <div className="flex items-start justify-between">
                <div>
                  <h1 className="text-2xl font-bold">{client.full_name}</h1>
                  <p className="text-muted-foreground">{client.phone ?? 'Нет телефона'}</p>
                  {client.card_number && (
                    <p className="text-sm text-muted-foreground">Карта: {client.card_number}</p>
                  )}
                </div>
                <div className="flex items-center gap-3">
                  <StatusBadge active={client.is_active} />
                  <Button variant="outline" size="sm" onClick={() => setShowDeposit(true)}>
                    Пополнить
                  </Button>
                  <Button
                    variant={client.is_active ? 'destructive' : 'default'}
                    size="sm"
                    onClick={() => blockMutation.mutate()}
                    disabled={blockMutation.isPending}
                  >
                    {client.is_active ? 'Заблокировать' : 'Разблокировать'}
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      if (window.confirm(`Удалить клиента "${client.full_name}"? Это действие необратимо.`)) {
                        deleteMutation.mutate()
                      }
                    }}
                    disabled={deleteMutation.isPending}
                  >
                    <Trash2 className="w-4 h-4" />
                  </Button>
                </div>
              </div>
              <p className="text-xs text-muted-foreground mt-1">
                Зарегистрирован: {format(new Date(client.created_at), 'dd.MM.yyyy')}
              </p>
              <p className="text-sm font-medium mt-1">
                Баланс: <span className="text-green-600">{client.balance.toFixed(2)} сомони</span>
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Active tariff */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle>Активный тариф</CardTitle>
          <Button size="sm" onClick={() => setShowAssign(true)}>Назначить тариф</Button>
        </CardHeader>
        <CardContent>
          {activeTariff ? (
            <div className="flex items-start justify-between">
              <div className="space-y-1">
                <div className="font-semibold text-lg">{activeTariff.tariff_name}</div>
                <div className="text-sm text-muted-foreground">
                  {format(new Date(activeTariff.start_date), 'dd MMM yyyy')} —{' '}
                  {format(new Date(activeTariff.end_date), 'dd MMM yyyy')}
                </div>
                <div className="text-sm">
                  Визитов в день: {activeTariff.max_visits_per_day ?? 'Без ограничений'}
                </div>
              </div>
              <Button
                variant="destructive"
                size="sm"
                disabled={revokeMutation.isPending}
                onClick={() => {
                  if (window.confirm('Открепить тариф? Доступ будет закрыт на терминалах.')) {
                    revokeMutation.mutate(activeTariff.id)
                  }
                }}
              >
                Открепить
              </Button>
            </div>
          ) : (
            <p className="text-muted-foreground">Нет активного тарифа</p>
          )}
        </CardContent>
      </Card>

      {/* Tabs */}
      <div>
        <div className="flex border-b mb-4">
          {(['events', 'payments', 'transactions'] as const).map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-4 py-2 text-sm font-medium capitalize border-b-2 transition-colors ${
                activeTab === tab
                  ? 'border-primary text-primary'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
              }`}
            >
              {tab === 'events' ? 'История доступа' : tab === 'payments' ? 'Платежи' : 'Транзакции'}
            </button>
          ))}
        </div>

        {activeTab === 'events' && (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Время</TableHead>
                <TableHead>Терминал</TableHead>
                <TableHead>Направление</TableHead>
                <TableHead>Метод</TableHead>
                <TableHead>Результат</TableHead>
                <TableHead>Причина</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(eventsData?.items ?? []).map((e) => (
                <TableRow key={e.id}>
                  <TableCell className="text-sm">{format(new Date(e.event_time), 'dd.MM HH:mm:ss')}</TableCell>
                  <TableCell>{e.terminal_name ?? '—'}</TableCell>
                  <TableCell><DirectionBadge direction={e.direction} /></TableCell>
                  <TableCell>{e.auth_method ?? '—'}</TableCell>
                  <TableCell>
                    <Badge variant={e.access_granted ? 'success' : 'destructive'}>
                      {e.access_granted ? 'Разрешён' : 'Отказан'}
                    </Badge>
                  </TableCell>
                  <TableCell>{e.deny_reason ?? '—'}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}

        {activeTab === 'payments' && (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Дата</TableHead>
                <TableHead>Тариф</TableHead>
                <TableHead>Период</TableHead>
                <TableHead>Сумма</TableHead>
                <TableHead>Примечание</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(payments ?? []).map((p) => (
                <TableRow key={p.id}>
                  <TableCell>{format(new Date(p.created_at), 'dd.MM.yyyy')}</TableCell>
                  <TableCell className="font-medium">{p.tariff_name}</TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {format(new Date(p.start_date), 'dd.MM')} — {format(new Date(p.end_date), 'dd.MM.yyyy')}
                  </TableCell>
                  <TableCell>{p.paid_amount != null ? `${p.paid_amount}` : '—'}</TableCell>
                  <TableCell>{p.payment_note ?? '—'}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}

        {activeTab === 'transactions' && (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Дата</TableHead>
                <TableHead>Тип</TableHead>
                <TableHead>Сумма</TableHead>
                <TableHead>Описание</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(transactions ?? []).map((t) => (
                <TableRow key={t.id}>
                  <TableCell className="text-sm">{format(new Date(t.created_at), 'dd.MM.yyyy HH:mm')}</TableCell>
                  <TableCell>
                    <Badge variant={t.type === 'deposit' ? 'success' : 'destructive'}>
                      {t.type === 'deposit' ? '+ Пополнение' : '- Оплата'}
                    </Badge>
                  </TableCell>
                  <TableCell className={`font-medium ${t.type === 'deposit' ? 'text-green-600' : 'text-red-500'}`}>
                    {t.type === 'deposit' ? '+' : '-'}{t.amount.toFixed(2)}
                  </TableCell>
                  <TableCell>{t.description ?? '—'}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </div>

      {/* Deposit Dialog */}
      <Dialog open={showDeposit} onClose={() => setShowDeposit(false)}>
        <div className="w-[380px]">
          <DialogHeader>
            <DialogTitle>Пополнить баланс</DialogTitle>
          </DialogHeader>
          <DialogContent>
            <form id="deposit-form" className="space-y-4" onSubmit={(e) => { e.preventDefault(); depositMutation.mutate() }}>
              <div className="space-y-2">
                <Label>Сумма *</Label>
                <Input type="number" step="0.01" min="0.01" value={depositAmount} onChange={(e) => setDepositAmount(e.target.value)} onFocus={(e) => e.target.select()} placeholder="0.00" required />
              </div>
              <div className="space-y-2">
                <Label>Описание</Label>
                <Input value={depositDesc} onChange={(e) => setDepositDesc(e.target.value)} placeholder="Наличные, перевод..." />
              </div>
            </form>
          </DialogContent>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDeposit(false)}>Отмена</Button>
            <Button type="submit" form="deposit-form" disabled={depositMutation.isPending}>
              {depositMutation.isPending ? 'Пополнение...' : 'Пополнить'}
            </Button>
          </DialogFooter>
        </div>
      </Dialog>

      {/* Assign Tariff Dialog */}
      <Dialog open={showAssign} onClose={() => { setShowAssign(false); assignMutation.reset() }}>
        <div className="w-[420px]">
          <DialogHeader>
            <DialogTitle>Назначить тариф</DialogTitle>
          </DialogHeader>
          <DialogContent>
            <form
              id="assign-tariff-form"
              className="space-y-4"
              onSubmit={(e) => {
                e.preventDefault()
                assignMutation.mutate(assignForm)
              }}
            >
              <div className="space-y-2">
                <Label>Тариф *</Label>
                <select
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm"
                  value={assignForm.tariff_id || ''}
                  onChange={(e) => setAssignForm(f => ({ ...f, tariff_id: Number(e.target.value) }))}
                  required
                >
                  <option value="">Выберите тариф...</option>
                  {(tariffs ?? []).filter(t => t.active).map((t) => (
                    <option key={t.id} value={t.id}>
                      {t.name} ({t.duration_days}d — {t.price})
                    </option>
                  ))}
                </select>
              </div>
              <div className="space-y-2">
                <Label>Дата начала *</Label>
                <Input
                  type="date"
                  value={assignForm.start_date}
                  onChange={(e) => setAssignForm(f => ({ ...f, start_date: e.target.value }))}
                  required
                />
              </div>
              {assignForm.tariff_id > 0 && tariffs && (
                <p className="text-sm text-muted-foreground">
                  Будет списано: <span className="font-medium text-foreground">
                    {(tariffs.find(t => t.id === assignForm.tariff_id)?.price ?? 0).toFixed(2)} сомони
                  </span> с баланса клиента
                </p>
              )}
              {assignError && (
                <p className="text-sm text-destructive">{assignError}</p>
              )}
            </form>
          </DialogContent>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setShowAssign(false); assignMutation.reset() }}>Отмена</Button>
            <Button type="submit" form="assign-tariff-form" disabled={assignMutation.isPending}>
              {assignMutation.isPending ? 'Назначение...' : 'Назначить'}
            </Button>
          </DialogFooter>
        </div>
      </Dialog>
    </div>
  )
}
