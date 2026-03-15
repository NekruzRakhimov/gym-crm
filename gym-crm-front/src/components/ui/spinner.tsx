import { cn } from '../../lib/utils'

export function Spinner({ className }: { className?: string }) {
  return (
    <div
      className={cn(
        'inline-block h-6 w-6 animate-spin rounded-full border-2 border-solid border-current border-r-transparent',
        className
      )}
    />
  )
}
