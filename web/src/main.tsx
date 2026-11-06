import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { AppRouter } from './app/router'
import { SessionProvider } from './app/session'

const container = document.getElementById('root')
if (!container) throw new Error('root element missing')

createRoot(container).render(
  <StrictMode>
    <SessionProvider>
      <AppRouter />
    </SessionProvider>
  </StrictMode>,
)
