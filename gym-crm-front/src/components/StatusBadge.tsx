import { Badge } from './ui/badge'

interface Props {
  active: boolean
  label?: [string, string]
}

export function StatusBadge({ active, label = ['Активен', 'Заблокирован'] }: Props) {
  return (
    <Badge variant={active ? 'success' : 'destructive'}>
      {active ? label[0] : label[1]}
    </Badge>
  )
}
