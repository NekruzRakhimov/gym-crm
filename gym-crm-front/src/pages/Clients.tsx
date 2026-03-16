import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { clientsApi, type CreateClientInput } from '../api/clients'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Label } from '../components/ui/label'
import { Badge } from '../components/ui/badge'
import { Dialog, DialogHeader, DialogTitle, DialogContent, DialogFooter } from '../components/ui/dialog'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/table'
import { StatusBadge } from '../components/StatusBadge'
import { PhotoUpload } from '../components/PhotoUpload'
import { Spinner } from '../components/ui/spinner'
import { Plus, Search } from 'lucide-react'
import { format } from 'date-fns'

export function Clients() {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [page, setPage] = useState(1)
  const [showAdd, setShowAdd] = useState(false)
  const [form, setForm] = useState<CreateClientInput>({ full_name: '', phone: null, card_number: null })
  const [pendingPhoto, setPendingPhoto] = useState<File | null>(null)
  const [pendingPhotoUrl, setPendingPhotoUrl] = useState<string | null>(null)

  useEffect(() => {
    const t = setTimeout(() => { setDebouncedSearch(search); setPage(1) }, 300)
    return () => clearTimeout(t)
  }, [search])

  const { data, isLoading } = useQuery({
    queryKey: ['clients', debouncedSearch, page],
    queryFn: () => clientsApi.list({ search: debouncedSearch, page, limit: 20 }).then((r) => r.data),
  })

  const uploadPhotoMutation = useMutation({
    mutationFn: ({ id, file }: { id: number; file: File }) => clientsApi.uploadPhoto(id, file),
    onSettled: () => qc.invalidateQueries({ queryKey: ['clients'] }),
  })

  const createMutation = useMutation({
    mutationFn: clientsApi.create,
    onSuccess: (res) => {
      if (pendingPhoto) {
        uploadPhotoMutation.mutate({ id: res.data.id, file: pendingPhoto })
      } else {
        qc.invalidateQueries({ queryKey: ['clients'] })
      }
      setShowAdd(false)
      setForm({ full_name: '', phone: null, card_number: null })
      setPendingPhoto(null)
      if (pendingPhotoUrl) { URL.revokeObjectURL(pendingPhotoUrl); setPendingPhotoUrl(null) }
    },
  })

  const total = data?.total ?? 0
  const totalPages = Math.ceil(total / 20)

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Клиенты</h1>
        <Button onClick={() => setShowAdd(true)}>
          <Plus className="w-4 h-4 mr-2" />
          Добавить клиента
        </Button>
      </div>

      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
        <Input
          placeholder="Поиск по имени или телефону..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="pl-9"
        />
      </div>

      {isLoading ? (
        <div className="flex justify-center py-8"><Spinner /></div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Фото</TableHead>
              <TableHead>Имя</TableHead>
              <TableHead>Телефон</TableHead>
              <TableHead>Активный тариф</TableHead>
              <TableHead>Статус</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(data?.items ?? []).map((client) => (
              <TableRow
                key={client.id}
                className="cursor-pointer"
                onClick={() => navigate(`/clients/${client.id}`)}
              >
                <TableCell>
                  <div className="w-9 h-9 rounded-full bg-muted overflow-hidden flex items-center justify-center">
                    {client.photo_path ? (
                      <img
                        src={`/uploads/${client.photo_path.split('/').pop()}`}
                        alt=""
                        className="w-full h-full object-cover"
                      />
                    ) : (
                      <span className="text-xs font-bold text-muted-foreground">
                        {client.full_name.charAt(0).toUpperCase()}
                      </span>
                    )}
                  </div>
                </TableCell>
                <TableCell className="font-medium">{client.full_name}</TableCell>
                <TableCell>{client.phone ?? '—'}</TableCell>
                <TableCell>
                  {client.active_tariff_name ? (
                    <div>
                      <div className="font-medium text-sm">{client.active_tariff_name}</div>
                      <div className="text-xs text-muted-foreground">
                        до {client.active_tariff_end ? format(new Date(client.active_tariff_end), 'dd.MM.yyyy') : ''}
                      </div>
                    </div>
                  ) : (
                    <Badge variant="outline">Нет тарифа</Badge>
                  )}
                </TableCell>
                <TableCell>
                  <StatusBadge active={client.is_active} />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2">
          <Button variant="outline" size="sm" disabled={page === 1} onClick={() => setPage(p => p - 1)}>
            Назад
          </Button>
          <span className="text-sm text-muted-foreground">Страница {page} из {totalPages}</span>
          <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage(p => p + 1)}>
            Вперёд
          </Button>
        </div>
      )}

      <Dialog open={showAdd} onClose={() => { setShowAdd(false); setPendingPhoto(null); if (pendingPhotoUrl) { URL.revokeObjectURL(pendingPhotoUrl); setPendingPhotoUrl(null) } }}>
        <div className="w-[420px]">
          <DialogHeader>
            <DialogTitle>Новый клиент</DialogTitle>
          </DialogHeader>
          <DialogContent>
            <form
              id="add-client-form"
              className="space-y-4"
              onSubmit={(e) => {
                e.preventDefault()
                createMutation.mutate(form)
              }}
            >
              <div className="flex justify-center">
                <PhotoUpload
                  photoPath={null}
                  previewSrc={pendingPhotoUrl}
                  onUpload={(file) => {
                    if (pendingPhotoUrl) URL.revokeObjectURL(pendingPhotoUrl)
                    setPendingPhoto(file)
                    setPendingPhotoUrl(URL.createObjectURL(file))
                  }}
                  size={96}
                />
              </div>
              <div className="space-y-2">
                <Label>ФИО *</Label>
                <Input
                  value={form.full_name}
                  onChange={(e) => setForm(f => ({ ...f, full_name: e.target.value }))}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label>Телефон</Label>
                <Input
                  value={form.phone ?? ''}
                  onChange={(e) => setForm(f => ({ ...f, phone: e.target.value || null }))}
                />
              </div>
              <div className="space-y-2">
                <Label>Номер карты</Label>
                <Input
                  value={form.card_number ?? ''}
                  onChange={(e) => setForm(f => ({ ...f, card_number: e.target.value || null }))}
                />
              </div>
              {createMutation.isError && (
                <p className="text-sm text-destructive">Не удалось создать клиента</p>
              )}
            </form>
          </DialogContent>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setShowAdd(false); setPendingPhoto(null); if (pendingPhotoUrl) { URL.revokeObjectURL(pendingPhotoUrl); setPendingPhotoUrl(null) } }}>Отмена</Button>
            <Button type="submit" form="add-client-form" disabled={createMutation.isPending}>
              {createMutation.isPending ? 'Создание...' : 'Создать'}
            </Button>
          </DialogFooter>
        </div>
      </Dialog>
    </div>
  )
}
