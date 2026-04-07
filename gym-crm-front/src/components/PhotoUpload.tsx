import { useRef, useState, useCallback, useEffect } from 'react'
import { Camera, Upload, X, RefreshCw } from 'lucide-react'
import { Dialog, DialogHeader, DialogTitle, DialogContent, DialogFooter } from './ui/dialog'
import { Button } from './ui/button'

interface Props {
  photoPath: string | null
  onUpload: (file: File) => void
  size?: number
  previewSrc?: string | null  // override src directly (e.g. blob URL)
}

type Mode = 'choose' | 'camera' | 'preview'

export function PhotoUpload({ photoPath, onUpload, size = 80, previewSrc }: Props) {
  const fileInputRef = useRef<HTMLInputElement>(null)
  const videoRef = useRef<HTMLVideoElement>(null)
  const canvasRef = useRef<HTMLCanvasElement>(null)

  const [open, setOpen] = useState(false)
  const [mode, setMode] = useState<Mode>('choose')
  const [previewUrl, setPreviewUrl] = useState<string | null>(null)
  const [capturedFile, setCapturedFile] = useState<File | null>(null)
  const [cameraError, setCameraError] = useState<string | null>(null)
  const [stream, setStream] = useState<MediaStream | null>(null)

  const src = previewSrc ?? (photoPath ? `/uploads/${photoPath.split('/').pop()}` : null)

  const stopStream = useCallback(() => {
    stream?.getTracks().forEach((t) => t.stop())
    setStream(null)
  }, [stream])

  const close = useCallback(() => {
    stopStream()
    setOpen(false)
    setMode('choose')
    setPreviewUrl(null)
    setCapturedFile(null)
    setCameraError(null)
  }, [stopStream])

  const startCamera = useCallback(async () => {
    setCameraError(null)
    setMode('camera')
    if (!navigator.mediaDevices?.getUserMedia) {
      setCameraError(
        'Камера недоступна. Сайт должен быть открыт по HTTPS или на localhost.'
      )
      return
    }
    try {
      const s = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: 'user', width: { ideal: 1280 }, height: { ideal: 720 } },
      })
      setStream(s)
    } catch {
      setCameraError('Нет доступа к камере. Проверьте разрешения браузера.')
    }
  }, [])

  // Attach stream to video element when both are ready
  useEffect(() => {
    if (mode === 'camera' && stream && videoRef.current) {
      videoRef.current.srcObject = stream
    }
  }, [mode, stream])

  const capture = useCallback(() => {
    const video = videoRef.current
    const canvas = canvasRef.current
    if (!video || !canvas) return

    canvas.width = video.videoWidth
    canvas.height = video.videoHeight
    canvas.getContext('2d')!.drawImage(video, 0, 0)

    canvas.toBlob((blob) => {
      if (!blob) return
      const file = new File([blob], 'capture.jpg', { type: 'image/jpeg' })
      const url = URL.createObjectURL(blob)
      stopStream()
      setCapturedFile(file)
      setPreviewUrl(url)
      setMode('preview')
    }, 'image/jpeg', 0.92)
  }, [stopStream])

  const retake = useCallback(() => {
    if (previewUrl) URL.revokeObjectURL(previewUrl)
    setPreviewUrl(null)
    setCapturedFile(null)
    startCamera()
  }, [previewUrl, startCamera])

  const confirm = useCallback(() => {
    if (capturedFile) {
      onUpload(capturedFile)
      close()
    }
  }, [capturedFile, onUpload, close])

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    e.target.value = ''

    // Convert any image (PNG, WebP, etc.) to JPEG via canvas, then show preview
    const img = new Image()
    const objectUrl = URL.createObjectURL(file)
    img.onload = () => {
      const canvas = canvasRef.current
      if (!canvas) return
      canvas.width = img.naturalWidth
      canvas.height = img.naturalHeight
      canvas.getContext('2d')!.drawImage(img, 0, 0)
      URL.revokeObjectURL(objectUrl)
      canvas.toBlob((blob) => {
        if (!blob) return
        const jpegFile = new File([blob], 'photo.jpg', { type: 'image/jpeg' })
        const previewUrl = URL.createObjectURL(blob)
        setCapturedFile(jpegFile)
        setPreviewUrl(previewUrl)
        setMode('preview')
      }, 'image/jpeg', 0.92)
    }
    img.src = objectUrl
  }

  return (
    <>
      <div
        onClick={() => setOpen(true)}
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
      </div>

      <Dialog open={open} onClose={close}>
        <div className="w-[480px]">
          <DialogHeader>
            <div className="flex items-center justify-between pr-2">
              <DialogTitle>
                {mode === 'choose' && 'Добавить фото'}
                {mode === 'camera' && 'Камера'}
                {mode === 'preview' && 'Подтвердить фото'}
              </DialogTitle>
              <button onClick={close} className="text-muted-foreground hover:text-foreground">
                <X className="w-4 h-4" />
              </button>
            </div>
          </DialogHeader>

          <DialogContent>
            {/* Choose mode */}
            {mode === 'choose' && (
              <div className="flex gap-4 py-2">
                <button
                  onClick={startCamera}
                  className="flex-1 flex flex-col items-center gap-3 p-6 rounded-lg border-2 border-dashed border-border hover:border-primary hover:bg-muted/50 transition-all"
                >
                  <Camera className="w-8 h-8 text-muted-foreground" />
                  <span className="text-sm font-medium">Сфотографировать</span>
                  <span className="text-xs text-muted-foreground text-center">Включить камеру и сделать снимок</span>
                </button>
                <button
                  onClick={() => fileInputRef.current?.click()}
                  className="flex-1 flex flex-col items-center gap-3 p-6 rounded-lg border-2 border-dashed border-border hover:border-primary hover:bg-muted/50 transition-all"
                >
                  <Upload className="w-8 h-8 text-muted-foreground" />
                  <span className="text-sm font-medium">Загрузить файл</span>
                  <span className="text-xs text-muted-foreground text-center">Выбрать фото с устройства</span>
                </button>
              </div>
            )}

            {/* Camera mode */}
            {mode === 'camera' && (
              <div className="space-y-3">
                {cameraError ? (
                  <div className="text-sm text-destructive text-center py-8">{cameraError}</div>
                ) : (
                  <div className="relative rounded-lg overflow-hidden bg-black aspect-video">
                    <video
                      ref={videoRef}
                      autoPlay
                      playsInline
                      muted
                      className="w-full h-full object-cover"
                    />
                  </div>
                )}
              </div>
            )}

            {/* Preview mode */}
            {mode === 'preview' && previewUrl && (
              <div className="flex justify-center">
                <img
                  src={previewUrl}
                  alt="Preview"
                  className="rounded-lg max-h-64 object-contain"
                />
              </div>
            )}
          </DialogContent>

          <DialogFooter>
            {mode === 'choose' && (
              <Button variant="outline" onClick={close}>Отмена</Button>
            )}

            {mode === 'camera' && (
              <>
                <Button variant="outline" onClick={() => { stopStream(); setMode('choose') }}>Назад</Button>
                <Button onClick={capture} disabled={!!cameraError}>
                  <Camera className="w-4 h-4 mr-2" />
                  Сделать снимок
                </Button>
              </>
            )}

            {mode === 'preview' && (
              <>
                <Button variant="outline" onClick={retake}>
                  <RefreshCw className="w-4 h-4 mr-2" />
                  Переснять
                </Button>
                <Button onClick={confirm}>Использовать</Button>
              </>
            )}
          </DialogFooter>
        </div>
      </Dialog>

      {/* Hidden file input */}
      <input
        ref={fileInputRef}
        type="file"
        accept="image/jpeg,image/jpg,image/png,image/webp"
        className="hidden"
        onChange={handleFileChange}
      />

      {/* Off-screen canvas for capture */}
      <canvas ref={canvasRef} className="hidden" />
    </>
  )
}
