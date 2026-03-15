import { useRef } from 'react'
import { Camera } from 'lucide-react'

interface Props {
  photoPath: string | null
  onUpload: (file: File) => void
  size?: number
}

export function PhotoUpload({ photoPath, onUpload, size = 80 }: Props) {
  const inputRef = useRef<HTMLInputElement>(null)
  const src = photoPath ? `/uploads/${photoPath.split('/').pop()}` : null

  return (
    <div
      onClick={() => inputRef.current?.click()}
      className="cursor-pointer rounded-full bg-muted flex items-center justify-center overflow-hidden border-2 border-dashed border-border hover:border-primary transition-colors relative group"
      style={{ width: size, height: size }}
    >
      {src ? (
        <>
          <img src={src} alt="Photo" className="w-full h-full object-cover" />
          <div className="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center">
            <Camera className="w-5 h-5 text-white" />
          </div>
        </>
      ) : (
        <Camera className="w-6 h-6 text-muted-foreground" />
      )}
      <input
        ref={inputRef}
        type="file"
        accept="image/jpeg,image/jpg"
        className="hidden"
        onChange={(e) => {
          if (e.target.files?.[0]) onUpload(e.target.files[0])
        }}
      />
    </div>
  )
}
