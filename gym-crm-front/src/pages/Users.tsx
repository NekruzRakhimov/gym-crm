import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { usersApi, type CreateAdminInput } from '../api/users'
import { useAuthStore } from '../store/auth'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Label } from '../components/ui/label'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '../components/ui/dialog'
import { Spinner } from '../components/ui/spinner'
import { UserCog, Plus, Trash2 } from 'lucide-react'

const emptyForm: CreateAdminInput = { username: '', password: '', role: 'manager' }

export function Users() {
  const role = useAuthStore((s) => s.role)
  const navigate = useNavigate()

  if (role !== 'admin') {
    navigate('/', { replace: true })
    return null
  }

  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [form, setForm] = useState<CreateAdminInput>(emptyForm)

  const { data: users, isLoading } = useQuery({
    queryKey: ['admin-users'],
    queryFn: () => usersApi.list().then((r) => r.data),
  })

  const createMutation = useMutation({
    mutationFn: (data: CreateAdminInput) => usersApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-users'] })
      setDialogOpen(false)
      setForm(emptyForm)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => usersApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-users'] })
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    createMutation.mutate(form)
  }

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <UserCog className="w-6 h-6" />
          <h1 className="text-2xl font-bold">Пользователи</h1>
        </div>
        <Button onClick={() => setDialogOpen(true)}>
          <Plus className="w-4 h-4 mr-2" />
          Добавить пользователя
        </Button>
      </div>

      {isLoading ? (
        <div className="flex justify-center py-12">
          <Spinner />
        </div>
      ) : (
        <div className="rounded-md border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/50">
                <th className="px-4 py-3 text-left font-medium">ID</th>
                <th className="px-4 py-3 text-left font-medium">Имя пользователя</th>
                <th className="px-4 py-3 text-left font-medium">Роль</th>
                <th className="px-4 py-3 text-left font-medium">Создан</th>
                <th className="px-4 py-3 text-left font-medium">Действия</th>
              </tr>
            </thead>
            <tbody>
              {users?.map((user) => (
                <tr key={user.id} className="border-b last:border-0 hover:bg-muted/30">
                  <td className="px-4 py-3 text-muted-foreground">{user.id}</td>
                  <td className="px-4 py-3 font-medium">{user.username}</td>
                  <td className="px-4 py-3">
                    <span
                      className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                        user.role === 'admin'
                          ? 'bg-blue-100 text-blue-700'
                          : 'bg-gray-100 text-gray-600'
                      }`}
                    >
                      {user.role === 'admin' ? 'Администратор' : 'Менеджер'}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {new Date(user.created_at).toLocaleDateString('ru-RU')}
                  </td>
                  <td className="px-4 py-3">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => deleteMutation.mutate(user.id)}
                      disabled={deleteMutation.isPending}
                    >
                      <Trash2 className="w-4 h-4 text-destructive" />
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Добавить пользователя</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="username">Имя пользователя</Label>
              <Input
                id="username"
                value={form.username}
                onChange={(e) => setForm((f) => ({ ...f, username: e.target.value }))}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">Пароль</Label>
              <Input
                id="password"
                type="password"
                value={form.password}
                onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="role">Роль</Label>
              <select
                id="role"
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm transition-colors"
                value={form.role}
                onChange={(e) => setForm((f) => ({ ...f, role: e.target.value }))}
              >
                <option value="manager">Менеджер</option>
                <option value="admin">Администратор</option>
              </select>
            </div>
            {createMutation.isError && (
              <p className="text-sm text-destructive">Ошибка при создании пользователя</p>
            )}
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
                Отмена
              </Button>
              <Button type="submit" disabled={createMutation.isPending}>
                {createMutation.isPending ? <Spinner className="w-4 h-4 mr-2" /> : null}
                Создать
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
