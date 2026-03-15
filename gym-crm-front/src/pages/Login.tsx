import { useState } from 'react'
import { useLogin } from '../hooks/useAuth'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Label } from '../components/ui/label'
import { Card, CardHeader, CardTitle, CardContent } from '../components/ui/card'
import { Dumbbell } from 'lucide-react'

export function Login() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const login = useLogin()

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    login.mutate({ username, password })
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-muted/30">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <div className="flex justify-center mb-2">
            <Dumbbell className="w-10 h-10 text-primary" />
          </div>
          <CardTitle className="text-2xl">Gym CRM</CardTitle>
          <p className="text-muted-foreground text-sm">Войдите в систему</p>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="username">Логин</Label>
              <Input
                id="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="admin"
                required
                autoFocus
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">Пароль</Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                required
              />
            </div>
            {login.isError && (
              <p className="text-sm text-destructive">Неверный логин или пароль</p>
            )}
            <Button type="submit" className="w-full" disabled={login.isPending}>
              {login.isPending ? 'Вход...' : 'Войти'}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
