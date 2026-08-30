import { BrowserRouter } from 'react-router-dom'
import { AppRoutes } from '@/app/router/routes'

export default function App() {
  return (
    <BrowserRouter>
      <AppRoutes />
    </BrowserRouter>
  )
}
