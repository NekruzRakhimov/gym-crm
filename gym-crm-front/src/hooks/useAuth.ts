import { useMutation } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { authApi } from '../api/auth'
import { useAuthStore } from '../store/auth'

export function useLogin() {
  const setAccessToken = useAuthStore((s) => s.setAccessToken)
  const navigate = useNavigate()
  return useMutation({
    mutationFn: authApi.login,
    onSuccess: (res) => {
      sessionStorage.setItem('session_active', '1')
      setAccessToken(res.data.access_token)
      navigate('/')
    },
  })
}

export function useLogout() {
  const logout = useAuthStore((s) => s.logout)
  const navigate = useNavigate()
  return async () => {
    try {
      await authApi.logout()
    } catch {
      // ignore
    }
    sessionStorage.removeItem('session_active')
    logout()
    navigate('/login')
  }
}
