import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { eventsApi, type EventsFilter } from '../api/events'
import { Badge } from '../components/ui/badge'
import { Input } from '../components/ui/input'
import { Button } from '../components/ui/button'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/table'
import { DirectionBadge } from '../components/DirectionBadge'
import { Spinner } from '../components/ui/spinner'
import { format } from 'date-fns'

export function Events() {
  const [filter, setFilter] = useState<EventsFilter>({ page: 1, limit: 20 })

  const { data, isLoading } = useQuery({
    queryKey: ['events', filter],
    queryFn: () => eventsApi.list(filter).then((r) => r.data),
  })

  const total = data?.total ?? 0
  const totalPages = Math.ceil(total / 20)
  const page = filter.page ?? 1

  const updateFilter = (updates: Partial<EventsFilter>) => {
    setFilter(f => ({ ...f, ...updates, page: 1 }))
  }

  return (
    <div className="p-6 space-y-4">
      <h1 className="text-2xl font-bold">События доступа</h1>

      {/* Filter bar */}
      <div className="flex flex-wrap gap-3">
        <div className="flex items-center gap-2">
          <label className="text-sm text-muted-foreground">С:</label>
          <Input
            type="date"
            className="w-36"
            value={filter.from ?? ''}
            onChange={(e) => updateFilter({ from: e.target.value || undefined })}
          />
        </div>
        <div className="flex items-center gap-2">
          <label className="text-sm text-muted-foreground">По:</label>
          <Input
            type="date"
            className="w-36"
            value={filter.to ?? ''}
            onChange={(e) => updateFilter({ to: e.target.value || undefined })}
          />
        </div>
        <select
          className="h-9 rounded-md border border-input bg-transparent px-3 text-sm"
          value={filter.direction ?? ''}
          onChange={(e) => updateFilter({ direction: e.target.value || undefined })}
        >
          <option value="">Все направления</option>
          <option value="entry">Вход</option>
          <option value="exit">Выход</option>
        </select>
        <select
          className="h-9 rounded-md border border-input bg-transparent px-3 text-sm"
          value={filter.granted === undefined ? '' : filter.granted ? 'true' : 'false'}
          onChange={(e) => updateFilter({ granted: e.target.value === '' ? undefined : e.target.value === 'true' })}
        >
          <option value="">Все результаты</option>
          <option value="true">Разрешён</option>
          <option value="false">Отказан</option>
        </select>
        <Button
          variant="outline"
          size="sm"
          onClick={() => setFilter({ page: 1, limit: 20 })}
        >
          Сбросить
        </Button>
      </div>

      {isLoading ? (
        <div className="flex justify-center py-8"><Spinner /></div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Время</TableHead>
              <TableHead>Клиент</TableHead>
              <TableHead>Терминал</TableHead>
              <TableHead>Направление</TableHead>
              <TableHead>Метод</TableHead>
              <TableHead>Результат</TableHead>
              <TableHead>Причина</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(data?.items ?? []).map((e) => (
              <TableRow key={e.id}>
                <TableCell className="text-sm whitespace-nowrap">
                  {format(new Date(e.event_time), 'dd.MM.yyyy HH:mm:ss')}
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    <div className="w-7 h-7 rounded-full bg-muted overflow-hidden flex items-center justify-center flex-shrink-0">
                      {e.client_photo ? (
                        <img src={`/uploads/${e.client_photo.split('/').pop()}`} className="w-full h-full object-cover" alt="" />
                      ) : (
                        <span className="text-xs text-muted-foreground">?</span>
                      )}
                    </div>
                    <span>{e.client_name ?? 'Неизвестен'}</span>
                  </div>
                </TableCell>
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

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={page <= 1}
            onClick={() => setFilter(f => ({ ...f, page: (f.page ?? 1) - 1 }))}
          >
            Назад
          </Button>
          <span className="text-sm text-muted-foreground">Страница {page} из {totalPages} ({total} всего)</span>
          <Button
            variant="outline"
            size="sm"
            disabled={page >= totalPages}
            onClick={() => setFilter(f => ({ ...f, page: (f.page ?? 1) + 1 }))}
          >
            Вперёд
          </Button>
        </div>
      )}
    </div>
  )
}
