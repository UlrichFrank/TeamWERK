import { AlertTriangle } from 'lucide-react'
import { useMaintenanceStatus } from '../hooks/useMaintenanceStatus'

/**
 * Persistenter, nicht schließbarer Wartungsmodus-Hinweis. Sichtbar, sobald der
 * Server im Wartungsmodus ist (`GET /api/maintenance-status` liefert
 * `{enabled: true}` oder SSE-Event `settings-changed` triggert einen Reload).
 * Der Banner wird oberhalb des `TransitionalHostnameBanner` gemountet, damit
 * er auf jedem Host (Primär- wie Alias-URL) und auf der Login-Seite sichtbar
 * ist.
 */
export default function MaintenanceBanner() {
  const { enabled } = useMaintenanceStatus()
  if (!enabled) return null

  return (
    <div
      role="status"
      aria-live="polite"
      className="flex items-start sm:items-center gap-2 sm:gap-4 bg-brand-danger-light border-b border-brand-danger/30 text-brand-text text-sm px-4 py-3"
    >
      <AlertTriangle className="w-5 h-5 shrink-0 text-brand-danger" />
      <p>
        <strong>Wartungsmodus aktiv.</strong>{' '}
        Änderungen sind gerade nicht möglich. Lesen bleibt möglich; bitte gleich
        noch einmal versuchen.
      </p>
    </div>
  )
}
