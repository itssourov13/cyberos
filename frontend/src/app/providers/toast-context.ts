import { createContext, useContext } from 'react'
import type { Tone } from '@/api/types'

export interface ToastCtx {
  push: (message: string, tone?: Tone) => void
}

export const ToastContext = createContext<ToastCtx>({ push: () => {} })

export const useToast = () => useContext(ToastContext)
