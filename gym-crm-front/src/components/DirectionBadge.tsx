import { Badge } from './ui/badge'

interface Props {
  direction: 'entry' | 'exit'
}

export function DirectionBadge({ direction }: Props) {
  return (
    <Badge variant={direction === 'entry' ? 'default' : 'secondary'}>
      {direction === 'entry' ? '↑ Вход' : '↓ Выход'}
    </Badge>
  )
}
