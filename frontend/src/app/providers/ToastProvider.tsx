import { useCallback, useMemo, useRef, useState, type ReactNode } from 'react'
import { ToastContext } from './toast-context'
import type { Tone } from '@/api/types'

interface ToastItem {
  id: number
  message: string
  tone: Tone
}

export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<ToastItem[]>([])
  const seq = useRef(0)

  const push = useCallback((message: string, tone: Tone = 'neutral') => {
    const id = ++seq.current
    setItems((list) => [...list, { id, message, tone }])
    window.setTimeout(() => setItems((list) => list.filter((t) => t.id !== id)), 4200)
  }, [])

  const value = useMemo(() => ({ push }), [push])

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="toasts" role="status" aria-live="polite">
        {items.map((t) => (
          <div key={t.id} className="toast" data-tone={t.tone}>
            {t.message}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  )
}
