import { useEffect, useRef, useCallback } from 'react'
import { useAuthStore } from '../store/auth'

export function useWebSocket(onMessage: (data: unknown) => void) {
  const accessToken = useAuthStore((s) => s.accessToken)
  const ws = useRef<WebSocket | null>(null)
  const reconnectTimeout = useRef<number>()
  const onMessageRef = useRef(onMessage)
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
    const socket = new WebSocket(url)
    ws.current = socket

    socket.onmessage = (e) => {
      try {
        onMessageRef.current(JSON.parse(e.data))
      } catch {
        // ignore
      }
    }
    socket.onclose = () => {
      reconnectTimeout.current = window.setTimeout(connect, 3000)
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
}
