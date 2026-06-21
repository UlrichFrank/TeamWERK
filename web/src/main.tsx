import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App'
import { initTelemetry } from './lib/telemetry'

initTelemetry(
  import.meta.env.VITE_MATOMO_URL,
  Number(import.meta.env.VITE_MATOMO_SITE_ID),
)

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
