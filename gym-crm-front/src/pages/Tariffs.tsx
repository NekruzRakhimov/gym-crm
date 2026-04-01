import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { tariffsApi, type Tariff, type CreateTariffInput } from '../api/tariffs'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Label } from '../components/ui/label'
import { Badge } from '../components/ui/badge'
import { Dialog, DialogHeader, DialogTitle, DialogContent, DialogFooter } from '../components/ui/dialog'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/table'
import { Spinner } from '../components/ui/spinner'
import { Plus, Pencil, Trash2 } from 'lucide-react'

const PRESET_SCHEDULE_LABELS: Record<string, string> = {
  all: 'Все дни',
  weekdays: 'Будни (Пн–Пт)',
  weekends: 'Выходные (Сб–Вс)',
  even: 'Чётные дни',
  odd: 'Нечётные дни',
}

const DAY_OPTIONS = [
  { value: 'mon', label: 'Пн' },
  { value: 'tue', label: 'Вт' },
  { value: 'wed', label: 'Ср' },
  { value: 'thu', label: 'Чт' },
  { value: 'fri', label: 'Пт' },
  { value: 'sat', label: 'Сб' },
  { value: 'sun', label: 'Вс' },
]

const DAY_LABEL: Record<string, string> = Object.fromEntries(DAY_OPTIONS.map(d => [d.value, d.label]))

const isPreset = (s: string) => s in PRESET_SCHEDULE_LABELS

function scheduleLabel(s: string): string {
  if (PRESET_SCHEDULE_LABELS[s]) return PRESET_SCHEDULE_LABELS[s]
  // custom comma-separated days
  return s.split(',').map(d => DAY_LABEL[d] || d).join(', ')
}

const emptyForm: CreateTariffInput = {
  name: '', duration_days: 30, max_visits_per_day: null, price: 0,
  schedule_days: 'all', time_from: null, time_to: null,
}

export function Tariffs() {
  const qc = useQueryClient()
  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<Tariff | null>(null)
  const [form, setForm] = useState<CreateTariffInput>(emptyForm)
  const [deleteId, setDeleteId] = useState<number | null>(null)
  const [timeError, setTimeError] = useState<string | null>(null)

  // Derived UI state from form values
  const customDays = !isPreset(form.schedule_days ?? 'all')
  const selectedDays = customDays ? (form.schedule_days ?? '').split(',').filter(Boolean) : []
  const allDay = !form.time_from && !form.time_to

  const toggleDay = (day: string) => {
    const set = new Set(selectedDays)
    if (set.has(day)) { set.delete(day) } else { set.add(day) }
    // preserve canonical order
    const ordered = DAY_OPTIONS.map(d => d.value).filter(d => set.has(d))
    setForm(f => ({ ...f, schedule_days: ordered.length > 0 ? ordered.join(',') : 'all' }))
  }

  const { data: tariffs, isLoading } = useQuery({
    queryKey: ['tariffs'],
    queryFn: () => tariffsApi.list().then((r) => r.data),
  })

  const createMutation = useMutation({
    mutationFn: tariffsApi.create,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['tariffs'] }); closeForm() },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: CreateTariffInput }) => tariffsApi.update(id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['tariffs'] }); closeForm() },
  })

  const deleteMutation = useMutation({
    mutationFn: tariffsApi.delete,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['tariffs'] }); setDeleteId(null) },
  })

  const toggleMutation = useMutation({
    mutationFn: tariffsApi.toggle,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tariffs'] }),
  })

  const openEdit = (t: Tariff) => {
    setEditing(t)
    setForm({
      name: t.name, duration_days: t.duration_days, max_visits_per_day: t.max_visits_per_day,
      price: t.price, schedule_days: t.schedule_days || 'all',
      time_from: t.time_from, time_to: t.time_to,
    })
    setShowForm(true)
  }

  const closeForm = () => {
    setShowForm(false)
    setEditing(null)
    setForm(emptyForm)
    setTimeError(null)
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (form.time_from && form.time_to && form.time_from >= form.time_to) {
      setTimeError('Время начала должно быть раньше времени окончания')
      return
    }
    setTimeError(null)
    if (editing) updateMutation.mutate({ id: editing.id, data: form })
    else createMutation.mutate(form)
  }

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Тарифы</h1>
        <Button onClick={() => setShowForm(true)}>
          <Plus className="w-4 h-4 mr-2" />
          Добавить тариф
        </Button>
      </div>

      {isLoading ? (
        <div className="flex justify-center py-8"><Spinner /></div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Название</TableHead>
              <TableHead>Длительность</TableHead>
              <TableHead>Визитов в день</TableHead>
              <TableHead>Расписание</TableHead>
              <TableHead>Цена</TableHead>
              <TableHead>Статус</TableHead>
              <TableHead>Действия</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(tariffs ?? []).map((t) => (
              <TableRow key={t.id}>
                <TableCell className="font-medium">{t.name}</TableCell>
                <TableCell>{t.duration_days} дн.</TableCell>
                <TableCell>{t.max_visits_per_day ?? 'Без ограничений'}</TableCell>
                <TableCell className="text-sm">
                  <div>{scheduleLabel(t.schedule_days ?? 'all')}</div>
                  {t.time_from && t.time_to && (
                    <div className="text-muted-foreground">{t.time_from}–{t.time_to}</div>
                  )}
                </TableCell>
                <TableCell>{t.price.toLocaleString()} сомони</TableCell>
                <TableCell>
                  <Badge variant={t.active ? 'success' : 'secondary'}>{t.active ? 'Активен' : 'Неактивен'}</Badge>
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    <Button variant="outline" size="sm" onClick={() => toggleMutation.mutate(t.id)}>
                      {t.active ? 'Отключить' : 'Включить'}
                    </Button>
                    <Button variant="ghost" size="icon" onClick={() => openEdit(t)}>
                      <Pencil className="w-4 h-4" />
                    </Button>
                    <Button variant="ghost" size="icon" onClick={() => setDeleteId(t.id)}>
                      <Trash2 className="w-4 h-4 text-destructive" />
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {/* Add/Edit Dialog */}
      <Dialog open={showForm} onClose={closeForm}>
        <div className="w-[460px]">
          <DialogHeader>
            <DialogTitle>{editing ? 'Редактировать тариф' : 'Добавить тариф'}</DialogTitle>
          </DialogHeader>
          <DialogContent>
            <form id="tariff-form" onSubmit={handleSubmit} className="space-y-4">
              {/* Name */}
              <div className="space-y-2">
                <Label>Название *</Label>
                <Input value={form.name} onChange={(e) => setForm(f => ({ ...f, name: e.target.value }))} required />
              </div>

              {/* Duration */}
              <div className="space-y-2">
                <Label>Длительность (дней) *</Label>
                <Input
                  type="number" min={1} value={form.duration_days}
                  onChange={(e) => setForm(f => ({ ...f, duration_days: Number(e.target.value) }))}
                  required
                />
              </div>

              {/* Max visits */}
              <div className="space-y-2">
                <Label>Макс. визитов в день (пусто = без ограничений)</Label>
                <Input
                  type="number" min={1} value={form.max_visits_per_day ?? ''}
                  onChange={(e) => setForm(f => ({ ...f, max_visits_per_day: e.target.value ? Number(e.target.value) : null }))}
                />
              </div>

              {/* Price */}
              <div className="space-y-2">
                <Label>Цена *</Label>
                <Input
                  type="number" step="0.01" min={0} value={form.price}
                  onChange={(e) => setForm(f => ({ ...f, price: Number(e.target.value) }))}
                  required
                />
              </div>

              {/* Days of access */}
              <div className="space-y-2">
                <Label>Дни доступа</Label>
                <div className="flex gap-2 mb-2">
                  <Button
                    type="button" size="sm"
                    variant={!customDays ? 'default' : 'outline'}
                    onClick={() => setForm(f => ({ ...f, schedule_days: 'all' }))}
                  >
                    Пресет
                  </Button>
                  <Button
                    type="button" size="sm"
                    variant={customDays ? 'default' : 'outline'}
                    onClick={() => setForm(f => ({ ...f, schedule_days: 'mon,tue,wed,thu,fri' }))}
                  >
                    Выбрать дни
                  </Button>
                </div>

                {!customDays ? (
                  <select
                    className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm"
                    value={form.schedule_days ?? 'all'}
                    onChange={(e) => setForm(f => ({ ...f, schedule_days: e.target.value }))}
                  >
                    {Object.entries(PRESET_SCHEDULE_LABELS).map(([val, label]) => (
                      <option key={val} value={val}>{label}</option>
                    ))}
                  </select>
                ) : (
                  <div className="flex gap-1.5">
                    {DAY_OPTIONS.map(({ value, label }) => {
                      const active = selectedDays.includes(value)
                      return (
                        <button
                          key={value}
                          type="button"
                          onClick={() => toggleDay(value)}
                          className={`w-9 h-9 rounded-md text-sm font-medium border transition-colors ${
                            active
                              ? 'bg-primary text-primary-foreground border-primary'
                              : 'border-input text-muted-foreground hover:text-foreground'
                          }`}
                        >
                          {label}
                        </button>
                      )
                    })}
                  </div>
                )}
              </div>

              {/* Time of access */}
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <Label>Время доступа</Label>
                  <label className="flex items-center gap-2 text-sm cursor-pointer">
                    <input
                      type="checkbox"
                      checked={allDay}
                      onChange={(e) => {
                        if (e.target.checked) {
                          setForm(f => ({ ...f, time_from: null, time_to: null }))
                        } else {
                          setForm(f => ({ ...f, time_from: '06:00', time_to: '22:00' }))
                        }
                      }}
                    />
                    Весь день
                  </label>
                </div>
                {!allDay && (
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <Input
                        type="time" value={form.time_from ?? ''}
                        onChange={(e) => { setTimeError(null); setForm(f => ({ ...f, time_from: e.target.value || null })) }}
                        className="flex-1"
                      />
                      <span className="text-muted-foreground">—</span>
                      <Input
                        type="time" value={form.time_to ?? ''}
                        onChange={(e) => { setTimeError(null); setForm(f => ({ ...f, time_to: e.target.value || null })) }}
                        className="flex-1"
                      />
                    </div>
                    {timeError && <p className="text-sm text-destructive">{timeError}</p>}
                  </div>
                )}
              </div>
            </form>
          </DialogContent>
          <DialogFooter>
            <Button variant="outline" onClick={closeForm}>Отмена</Button>
            <Button type="submit" form="tariff-form" disabled={createMutation.isPending || updateMutation.isPending}>
              {editing ? 'Сохранить' : 'Создать'}
            </Button>
          </DialogFooter>
        </div>
      </Dialog>

      {/* Delete Confirmation */}
      <Dialog open={deleteId !== null} onClose={() => setDeleteId(null)}>
        <div className="w-[360px]">
          <DialogHeader>
            <DialogTitle>Удалить тариф</DialogTitle>
          </DialogHeader>
          <DialogContent>
            <p className="text-muted-foreground">Вы уверены, что хотите удалить этот тариф? Это действие нельзя отменить.</p>
          </DialogContent>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteId(null)}>Отмена</Button>
            <Button
              variant="destructive"
              onClick={() => deleteId !== null && deleteMutation.mutate(deleteId)}
              disabled={deleteMutation.isPending}
            >
              Удалить
            </Button>
          </DialogFooter>
        </div>
      </Dialog>
    </div>
  )
}
