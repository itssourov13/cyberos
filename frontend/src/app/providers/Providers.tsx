import type { ReactNode } from 'react'
import { ApiProvider } from '@/api'
import { ThemeProvider } from './ThemeProvider'
import { SessionProvider } from './SessionProvider'
import { ToastProvider } from './ToastProvider'

export function Providers({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider>
      <ToastProvider>
        <ApiProvider>
          <SessionProvider>{children}</SessionProvider>
        </ApiProvider>
      </ToastProvider>
    </ThemeProvider>
  )
}
