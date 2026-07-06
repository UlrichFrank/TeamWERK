import { useEffect, useState } from 'react'
import { AlertTriangle, CheckCircle2 } from 'lucide-react'
import { api } from '../../lib/api'

const CARD = 'bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6'
const BTN_DANGER =
  'bg-brand-danger text-white rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed'
const BTN_PRIMARY =
  'bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed'
const ALERT_INFO = 'p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text'
const ALERT_ERR = 'p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger'

interface AdminStatus {
  enabled: boolean
  updated_at?: string
  updated_by_name?: string
}

export default function WartungsmodusPage() {
  const [status, setStatus] = useState<AdminStatus | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError(null)
    api
      .get<AdminStatus>('/admin/maintenance-mode')
      .then((r) => setStatus(r.data))
      .catch(() => setError('Zustand konnte nicht geladen werden.'))
  }
  useEffect(load, [])

  const toggle = async () => {
    if (!status) return
    setBusy(true)
    setError(null)
    try {
      await api.post('/admin/maintenance-mode', { enabled: !status.enabled })
      load()
    } catch {
      setError('Umschalten fehlgeschlagen.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2">
        <h1 className="text-2xl font-semibold text-brand-text">Wartungsmodus</h1>
      </div>

      <div className={CARD}>
        <p className="text-sm text-brand-text-muted mb-4">
          Bei aktivem Wartungsmodus lehnt der Server alle Schreibzugriffe außer Anmelde-Routen mit HTTP 503 ab.
          Admins können weiterhin schreiben. Alle Sessions sehen einen persistenten Banner.
        </p>

        {status === null && !error && (
          <p className="text-sm text-brand-text-muted">Lade Zustand …</p>
        )}

        {status && (
          <div className="flex items-start gap-3 mb-4">
            {status.enabled ? (
              <AlertTriangle className="w-6 h-6 shrink-0 text-brand-danger" />
            ) : (
              <CheckCircle2 className="w-6 h-6 shrink-0 text-brand-green" />
            )}
            <div>
              <p className="text-base font-medium text-brand-text">
                Zustand: {status.enabled ? 'Ein' : 'Aus'}
              </p>
              {status.updated_at && (
                <p className="text-xs text-brand-text-muted">
                  Zuletzt geändert: {status.updated_at}
                  {status.updated_by_name ? ` (${status.updated_by_name})` : ''}
                </p>
              )}
            </div>
          </div>
        )}

        {error && <div className={`${ALERT_ERR} mb-4`}>{error}</div>}

        {status && (
          <button
            onClick={toggle}
            disabled={busy}
            className={status.enabled ? BTN_PRIMARY : BTN_DANGER}
          >
            {status.enabled ? 'Wartungsmodus ausschalten' : 'Wartungsmodus einschalten'}
          </button>
        )}
      </div>

      <div className={ALERT_INFO}>
        <p>
          <strong>Notfall-Fallback per CLI:</strong>{' '}
          <code>teamwerk maintenance on|off --db /var/lib/teamwerk/teamwerk.db</code>{' '}
          direkt auf dem Server. Der laufende Prozess übernimmt den Zustand binnen 10 s.
        </p>
      </div>
    </div>
  )
}
