import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from 'react-router-dom'
import '@/design/tokens.css'
import '@/design/base.css'
import '@/design/ui.css'
import '@/design/layout.css'
import { Providers } from '@/app/providers/Providers'
import { router } from '@/app/router'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <Providers>
      <RouterProvider router={router} />
    </Providers>
  </StrictMode>,
)
