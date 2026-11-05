import type { ReactNode } from 'react'
import { ApiProvider } from '@/api'
import { ThemeProvider } from './ThemeProvider'
import { ToastProvider } from './ToastProvider'

export function Providers({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider>
      <ToastProvider>
        <ApiProvider>{children}</ApiProvider>
      </ToastProvider>
    </ThemeProvider>
  )
}
