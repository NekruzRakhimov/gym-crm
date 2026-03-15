import { NavLink, Outlet } from 'react-router-dom'
import { useLogout } from '../hooks/useAuth'
import { useAuthStore } from '../store/auth'
import {
  LayoutDashboard,
  Users,
  CreditCard,
  Activity,
  Monitor,
  LogOut,
  Dumbbell,
  TrendingUp,
  UserCog,
} from 'lucide-react'
import { cn } from '../lib/utils'

const navItems = [
  { to: '/', label: 'Главная', icon: LayoutDashboard, end: true, adminOnly: false },
  { to: '/clients', label: 'Клиенты', icon: Users, adminOnly: false },
  { to: '/tariffs', label: 'Тарифы', icon: CreditCard, adminOnly: false },
  { to: '/events', label: 'События', icon: Activity, adminOnly: false },
  { to: '/terminals', label: 'Терминалы', icon: Monitor, adminOnly: false },
  { to: '/finance', label: 'Финансы', icon: TrendingUp, adminOnly: true },
  { to: '/users', label: 'Пользователи', icon: UserCog, adminOnly: true },
]

export function Layout() {
  const logout = useLogout()
  const role = useAuthStore((s) => s.role)

  const visibleItems = navItems.filter((item) => !item.adminOnly || role === 'admin')

  return (
    <div className="flex h-screen bg-background">
      {/* Sidebar */}
      <aside className="w-60 border-r bg-card flex flex-col">
        <div className="h-16 flex items-center px-6 border-b">
          <Dumbbell className="w-6 h-6 text-primary mr-2" />
          <span className="font-bold text-lg">Gym CRM</span>
        </div>
        <nav className="flex-1 py-4 space-y-1 px-3">
          {visibleItems.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                cn(
                  'flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium transition-colors',
                  isActive
                    ? 'bg-primary text-primary-foreground'
                    : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
                )
              }
            >
              <Icon className="w-4 h-4" />
              {label}
            </NavLink>
          ))}
        </nav>
        <div className="p-3 border-t">
          <button
            onClick={logout}
            className="flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium text-muted-foreground hover:bg-accent hover:text-accent-foreground w-full transition-colors"
          >
            <LogOut className="w-4 h-4" />
            Выйти
          </button>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
    </div>
  )
}
