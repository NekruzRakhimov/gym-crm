import { useEffect, useRef, useCallback, useState } from 'react'
import { useAuthStore } from '../store/auth'

export type WSStatus = 'connecting' | 'connected' | 'disconnected'

const MIN_RECONNECT_MS = 3_000
const MAX_RECONNECT_MS = 30_000

export function useWebSocket(onMessage: (data: unknown) => void): WSStatus {
  const accessToken = useAuthStore((s) => s.accessToken)
  const ws = useRef<WebSocket | null>(null)
  const reconnectTimeout = useRef<number>()
  const reconnectDelay = useRef(MIN_RECONNECT_MS)
  const onMessageRef = useRef(onMessage)
  const [status, setStatus] = useState<WSStatus>('disconnected')
  onMessageRef.current = onMessage

  const connect = useCallback(() => {
    if (!accessToken) return
    // Close any existing socket before opening a new one
    if (ws.current) {
      ws.current.onclose = null
      ws.current.close()
      ws.current = null
    }
    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
    const url = `${protocol}://${window.location.host}/ws?token=${accessToken}`
    setStatus('connecting')
    const socket = new WebSocket(url)
    ws.current = socket

    socket.onopen = () => {
      setStatus('connected')
      reconnectDelay.current = MIN_RECONNECT_MS  // reset backoff on success
    }
    socket.onmessage = (e) => {
      try {
        onMessageRef.current(JSON.parse(e.data))
      } catch {
        // ignore
      }
    }
    socket.onclose = () => {
      setStatus('disconnected')
      // Exponential backoff: 3s → 4.5s → 6.75s → … → 30s max.
      // Prevents hammering a recovering backend with constant reconnects.
      reconnectTimeout.current = window.setTimeout(connect, reconnectDelay.current)
      reconnectDelay.current = Math.min(reconnectDelay.current * 1.5, MAX_RECONNECT_MS)
    }
    socket.onerror = () => {
      socket.close()
    }
  }, [accessToken])

  useEffect(() => {
    connect()
    return () => {
      clearTimeout(reconnectTimeout.current)
      if (ws.current) {
        ws.current.onclose = null // prevent cleanup-close from scheduling a reconnect
        ws.current.close()
        ws.current = null
      }
    }
  }, [connect])

  return status
}
