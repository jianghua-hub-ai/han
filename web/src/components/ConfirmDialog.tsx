import { useEffect } from 'react'

export interface ConfirmDialogProps {
  open: boolean
  title: string
  message: string
  confirmLabel: string
  pending?: boolean
  onConfirm(): void
  onCancel(): void
}

/** Minimal accessible confirmation layer; Escape and the cancel action close it. */
export function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel,
  pending = false,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  useEffect(() => {
    if (!open) {
      return
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onCancel()
      }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [open, onCancel])

  if (!open) {
    return null
  }
  return (
    <div role="dialog" aria-modal="true" aria-label={title}>
      <h2>{title}</h2>
      <p>{message}</p>
      <button type="button" onClick={onCancel} disabled={pending}>
        取消
      </button>
      <button type="button" onClick={onConfirm} disabled={pending}>
        {pending ? '处理中…' : confirmLabel}
      </button>
    </div>
  )
}
