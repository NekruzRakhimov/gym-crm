import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { terminalsApi, type Terminal, type CreateTerminalInput } from '../api/terminals'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Label } from '../components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card'
import { Dialog, DialogHeader, DialogTitle, DialogContent, DialogFooter } from '../components/ui/dialog'
import { DirectionBadge } from '../components/DirectionBadge'
import { Spinner } from '../components/ui/spinner'
import { Plus, RefreshCw, DoorOpen, Wifi, ShieldCheck, Pencil } from 'lucide-react'

const emptyForm: CreateTerminalInput = { name: '', ip: '', port: 80, username: 'admin', password: '', direction: 'entry' }

function TerminalStatus({ terminalId }: { terminalId: number }) {
  const { data, isLoading } = useQuery({
    queryKey: ['terminal-status', terminalId],
    queryFn: () => terminalsApi.getStatus(terminalId).then((r) => r.data),
    refetchInterval: 30_000,
  })

  if (isLoading) return <Spinner className="w-4 h-4" />

  return (
    <div className={`flex items-center gap-1.5 text-sm ${data?.online ? 'text-green-600' : 'text-red-500'}`}>
      <div className={`w-2 h-2 rounded-full ${data?.online ? 'bg-green-500' : 'bg-red-500'}`} />
      {data?.online ? 'Онлайн' : 'Офлайн'}
    </div>
  )
}

export function Terminals() {
  const qc = useQueryClient()
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState<CreateTerminalInput>(emptyForm)
  const [editingTerminal, setEditingTerminal] = useState<Terminal | null>(null)
  const [editForm, setEditForm] = useState<CreateTerminalInput>(emptyForm)

  const { data: terminals, isLoading } = useQuery({
    queryKey: ['terminals'],
    queryFn: () => terminalsApi.list().then((r) => r.data),
  })

  const createMutation = useMutation({
    mutationFn: terminalsApi.create,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['terminals'] }); setShowForm(false); setForm(emptyForm) },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: CreateTerminalInput }) => terminalsApi.update(id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['terminals'] }); setEditingTerminal(null) },
  })

  const openDoorMutation = useMutation({
    mutationFn: terminalsApi.openDoor,
  })

  const webhookMutation = useMutation({
    mutationFn: terminalsApi.setupWebhook,
  })

  const syncMutation = useMutation({
    mutationFn: terminalsApi.sync,
  })

  const [remoteVerifyInfo, setRemoteVerifyInfo] = useState<{ steps: string[]; webhook_url: string } | null>(null)

  const remoteVerifyMutation = useMutation({
    mutationFn: terminalsApi.enableRemoteVerify,
    onSuccess: (res) => setRemoteVerifyInfo(res.data),
  })

  const handleAction = (fn: (id: number) => unknown, id: number, label: string) => {
    if (window.confirm(`${label} — терминал #${id}?`)) {
      fn(id)
    }
  }

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Терминалы</h1>
        <Button onClick={() => setShowForm(true)}>
          <Plus className="w-4 h-4 mr-2" />
          Добавить терминал
        </Button>
      </div>

      {isLoading ? (
        <div className="flex justify-center py-8"><Spinner /></div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {(terminals ?? []).map((t: Terminal) => (
            <Card key={t.id}>
              <CardHeader className="pb-2">
                <div className="flex items-start justify-between">
                  <CardTitle className="text-base">{t.name}</CardTitle>
                  <div className="flex items-center gap-2">
                    <TerminalStatus terminalId={t.id} />
                    <button
                      onClick={() => { setEditingTerminal(t); setEditForm({ name: t.name, ip: t.ip, port: t.port, username: t.username, password: '', direction: t.direction }) }}
                      className="text-muted-foreground hover:text-foreground"
                    >
                      <Pencil className="w-3.5 h-3.5" />
                    </button>
                  </div>
                </div>
                <div className="flex items-center gap-2 mt-1">
                  <DirectionBadge direction={t.direction} />
                  <span className="text-xs text-muted-foreground font-mono">{t.ip}:{t.port}</span>
                </div>
              </CardHeader>
              <CardContent>
                <div className="flex flex-wrap gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleAction(openDoorMutation.mutate, t.id, 'Открыть дверь')}
                    disabled={openDoorMutation.isPending}
                  >
                    <DoorOpen className="w-3.5 h-3.5 mr-1.5" />
                    Открыть дверь
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleAction(webhookMutation.mutate, t.id, 'Настроить вебхук')}
                    disabled={webhookMutation.isPending}
                  >
                    <Wifi className="w-3.5 h-3.5 mr-1.5" />
                    Вебхук
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleAction(syncMutation.mutate, t.id, 'Синхронизировать всех клиентов')}
                    disabled={syncMutation.isPending}
                  >
                    <RefreshCw className="w-3.5 h-3.5 mr-1.5" />
                    Синхронизировать
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleAction(remoteVerifyMutation.mutate, t.id, 'Включить Remote Verification')}
                    disabled={remoteVerifyMutation.isPending}
                  >
                    <ShieldCheck className="w-3.5 h-3.5 mr-1.5" />
                    Remote Verify
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Dialog open={!!remoteVerifyInfo} onClose={() => setRemoteVerifyInfo(null)}>
        <div className="w-[480px]">
          <DialogHeader>
            <DialogTitle>Настройка Remote Verification</DialogTitle>
          </DialogHeader>
          <DialogContent>
            <p className="text-sm text-muted-foreground mb-3">
              Эту настройку нужно включить вручную в веб-интерфейсе терминала:
            </p>
            <ol className="space-y-2">
              {remoteVerifyInfo?.steps.map((step, i) => (
                <li key={i} className="flex gap-2 text-sm">
                  <span className="font-mono text-xs bg-muted rounded px-1.5 py-0.5 h-fit">{i + 1}</span>
                  <span>{step}</span>
                </li>
              ))}
            </ol>
            {remoteVerifyInfo?.webhook_url && (
              <div className="mt-4 p-3 bg-muted rounded text-xs font-mono break-all">
                {remoteVerifyInfo.webhook_url}
              </div>
            )}
            <p className="text-xs text-green-600 mt-3">Сервер уже готов принимать запросы верификации.</p>
          </DialogContent>
          <DialogFooter>
            <Button onClick={() => setRemoteVerifyInfo(null)}>Понятно</Button>
          </DialogFooter>
        </div>
      </Dialog>

      <Dialog open={!!editingTerminal} onClose={() => setEditingTerminal(null)}>
        <div className="w-[420px]">
          <DialogHeader>
            <DialogTitle>Редактировать терминал</DialogTitle>
          </DialogHeader>
          <DialogContent>
            <form
              id="edit-terminal-form"
              className="space-y-4"
              onSubmit={(e) => { e.preventDefault(); updateMutation.mutate({ id: editingTerminal!.id, data: editForm }) }}
            >
              <div className="space-y-2">
                <Label>Название *</Label>
                <Input value={editForm.name} onChange={(e) => setEditForm(f => ({ ...f, name: e.target.value }))} required />
              </div>
              <div className="grid grid-cols-3 gap-2">
                <div className="col-span-2 space-y-2">
                  <Label>IP-адрес *</Label>
                  <Input value={editForm.ip} onChange={(e) => setEditForm(f => ({ ...f, ip: e.target.value }))} placeholder="192.168.1.100" required />
                </div>
                <div className="space-y-2">
                  <Label>Порт</Label>
                  <Input type="number" value={editForm.port} onChange={(e) => setEditForm(f => ({ ...f, port: Number(e.target.value) }))} />
                </div>
              </div>
              <div className="space-y-2">
                <Label>Логин *</Label>
                <Input value={editForm.username} onChange={(e) => setEditForm(f => ({ ...f, username: e.target.value }))} required />
              </div>
              <div className="space-y-2">
                <Label>Пароль</Label>
                <Input type="password" value={editForm.password} onChange={(e) => setEditForm(f => ({ ...f, password: e.target.value }))} placeholder="Оставьте пустым, чтобы не менять" />
              </div>
              <div className="space-y-2">
                <Label>Направление *</Label>
                <select
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm"
                  value={editForm.direction}
                  onChange={(e) => setEditForm(f => ({ ...f, direction: e.target.value as 'entry' | 'exit' }))}
                >
                  <option value="entry">Вход</option>
                  <option value="exit">Выход</option>
                </select>
              </div>
            </form>
          </DialogContent>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditingTerminal(null)}>Отмена</Button>
            <Button type="submit" form="edit-terminal-form" disabled={updateMutation.isPending}>
              {updateMutation.isPending ? 'Сохранение...' : 'Сохранить'}
            </Button>
          </DialogFooter>
        </div>
      </Dialog>

      <Dialog open={showForm} onClose={() => { setShowForm(false); setForm(emptyForm) }}>
        <div className="w-[420px]">
          <DialogHeader>
            <DialogTitle>Добавить терминал</DialogTitle>
          </DialogHeader>
          <DialogContent>
            <form
              id="terminal-form"
              className="space-y-4"
              onSubmit={(e) => { e.preventDefault(); createMutation.mutate(form) }}
            >
              <div className="space-y-2">
                <Label>Название *</Label>
                <Input value={form.name} onChange={(e) => setForm(f => ({ ...f, name: e.target.value }))} required />
              </div>
              <div className="grid grid-cols-3 gap-2">
                <div className="col-span-2 space-y-2">
                  <Label>IP-адрес *</Label>
                  <Input value={form.ip} onChange={(e) => setForm(f => ({ ...f, ip: e.target.value }))} placeholder="192.168.1.100" required />
                </div>
                <div className="space-y-2">
                  <Label>Порт</Label>
                  <Input type="number" value={form.port} onChange={(e) => setForm(f => ({ ...f, port: Number(e.target.value) }))} />
                </div>
              </div>
              <div className="space-y-2">
                <Label>Логин *</Label>
                <Input value={form.username} onChange={(e) => setForm(f => ({ ...f, username: e.target.value }))} required />
              </div>
              <div className="space-y-2">
                <Label>Пароль *</Label>
                <Input type="password" value={form.password} onChange={(e) => setForm(f => ({ ...f, password: e.target.value }))} required />
              </div>
              <div className="space-y-2">
                <Label>Направление *</Label>
                <select
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm"
                  value={form.direction}
                  onChange={(e) => setForm(f => ({ ...f, direction: e.target.value as 'entry' | 'exit' }))}
                >
                  <option value="entry">Вход</option>
                  <option value="exit">Выход</option>
                </select>
              </div>
            </form>
          </DialogContent>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setShowForm(false); setForm(emptyForm) }}>Отмена</Button>
            <Button type="submit" form="terminal-form" disabled={createMutation.isPending}>
              {createMutation.isPending ? 'Добавление...' : 'Добавить'}
            </Button>
          </DialogFooter>
        </div>
      </Dialog>
    </div>
  )
}
