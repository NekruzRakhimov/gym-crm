import { useState, useCallback } from 'react'
import { useWebSocket } from '../hooks/useWebSocket'
import { type AccessEvent } from '../api/events'
import { DirectionBadge } from './DirectionBadge'
import { Badge } from './ui/badge'
import { format } from 'date-fns'

const denyReasonRu = (reason: string | null | undefined): string => {
  const map: Record<string, string> = {
    no_tariff: 'Нет тарифа',
    expired: 'Тариф истёк',
    limit_reached: 'Лимит визитов',
    blocked: 'Заблокирован',
    unknown: 'Неизвестен',
  }
  return reason ? (map[reason] ?? reason) : 'Отказан'
}

const statusLabel: Record<string, string> = {
  connected: 'Подключено',
  connecting: 'Подключение...',
  disconnected: 'Нет соединения',
}

const statusColor: Record<string, string> = {
  connected: 'bg-green-500',
  connecting: 'bg-yellow-400',
  disconnected: 'bg-red-500',
}

export function LiveFeed() {
  const [events, setEvents] = useState<AccessEvent[]>([])

  const onMessage = useCallback((data: unknown) => {
    const msg = data as { type: string; data: AccessEvent }
    if (msg.type === 'access_event') {
      setEvents((prev) => [msg.data, ...prev].slice(0, 50))
    }
  }, [])

  const wsStatus = useWebSocket(onMessage)

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2 pb-1">
        <span className={`w-2 h-2 rounded-full ${statusColor[wsStatus]}`} />
        <span className="text-xs text-muted-foreground">{statusLabel[wsStatus]}</span>
      </div>
      {events.length === 0 && (
        <p className="text-muted-foreground text-sm py-4 text-center">Ожидание событий...</p>
      )}
      {events.map((e) => (
        <div key={e.id} className="flex items-center gap-3 p-3 rounded-lg border bg-card text-sm">
          <div className="w-9 h-9 rounded-full bg-muted flex items-center justify-center overflow-hidden flex-shrink-0">
            {e.client_photo ? (
              <img
                src={`/uploads/${e.client_photo.split('/').pop()}`}
                className="w-full h-full object-cover"
                alt=""
              />
            ) : (
              <span className="text-xs text-muted-foreground font-bold">?</span>
            )}
          </div>
          <div className="flex-1 min-w-0">
            <div className="font-medium truncate">{e.client_name ?? 'Неизвестен'}</div>
            <div className="text-muted-foreground text-xs">{e.terminal_name ?? 'Неизвестный терминал'}</div>
          </div>
          <DirectionBadge direction={e.direction} />
          {e.auth_method && (
            <span className="text-xs text-muted-foreground hidden sm:inline">{e.auth_method}</span>
          )}
          <Badge variant={e.access_granted ? 'success' : 'destructive'}>
            {e.access_granted ? 'Разрешён' : (denyReasonRu(e.deny_reason) ?? 'Отказан')}
          </Badge>
          <span className="text-xs text-muted-foreground flex-shrink-0">
            {format(new Date(e.event_time), 'HH:mm:ss')}
          </span>
        </div>
      ))}
    </div>
  )
}
