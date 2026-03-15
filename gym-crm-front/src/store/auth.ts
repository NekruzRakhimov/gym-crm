import { create } from 'zustand'

interface AuthState {
  accessToken: string | null
  role: string | null
  isAuthenticated: boolean
  setAccessToken: (token: string | null) => void
  logout: () => void
}

function parseRole(token: string | null): string | null {
  if (!token) return null
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    return payload.role ?? null
  } catch {
    return null
  }
}

export const useAuthStore = create<AuthState>((set) => ({
  accessToken: null,
  role: null,
  isAuthenticated: false,
  setAccessToken: (token) =>
    set({ accessToken: token, isAuthenticated: !!token, role: parseRole(token) }),
  logout: () => set({ accessToken: null, isAuthenticated: false, role: null }),
}))
