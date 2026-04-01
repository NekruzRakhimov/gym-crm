import { useEffect, useState } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import axios from 'axios'
import { useAuthStore } from './store/auth'
import { Layout } from './components/Layout'
import { Login } from './pages/Login'
import { Dashboard } from './pages/Dashboard'
import { Clients } from './pages/Clients'
import { ClientDetail } from './pages/ClientDetail'
import { Tariffs } from './pages/Tariffs'
import { Events } from './pages/Events'
import { Terminals } from './pages/Terminals'
import { Finance } from './pages/Finance'
import { Users as UsersPage } from './pages/Users'
import { Spinner } from './components/ui/spinner'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  if (!isAuthenticated) return <Navigate to="/login" replace />
  return <>{children}</>
}

function PublicRoute({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  if (isAuthenticated) return <Navigate to="/" replace />
  return <>{children}</>
}

function AdminRoute({ children }: { children: React.ReactNode }) {
  const role = useAuthStore((s) => s.role)
  if (role !== 'admin') return <Navigate to="/clients" replace />
  return <>{children}</>
}

export default function App() {
  const setAccessToken = useAuthStore((s) => s.setAccessToken)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    axios
      .post<{ access_token: string }>('/api/auth/refresh', {}, { withCredentials: true })
      .then((res) => setAccessToken(res.data.access_token))
      .catch(() => setAccessToken(null))
      .finally(() => setLoading(false))
  }, [setAccessToken])

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Spinner />
      </div>
    )
  }

  return (
    <BrowserRouter>
      <Routes>
        <Route
          path="/login"
          element={
            <PublicRoute>
              <Login />
            </PublicRoute>
          }
        />
        <Route
          element={
            <ProtectedRoute>
              <Layout />
            </ProtectedRoute>
          }
        >
          <Route index element={<AdminRoute><Dashboard /></AdminRoute>} />
          <Route path="clients" element={<Clients />} />
          <Route path="clients/:id" element={<ClientDetail />} />
          <Route path="tariffs" element={<AdminRoute><Tariffs /></AdminRoute>} />
          <Route path="events" element={<AdminRoute><Events /></AdminRoute>} />
          <Route path="terminals" element={<AdminRoute><Terminals /></AdminRoute>} />
          <Route path="finance" element={<AdminRoute><Finance /></AdminRoute>} />
          <Route path="users" element={<AdminRoute><UsersPage /></AdminRoute>} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
