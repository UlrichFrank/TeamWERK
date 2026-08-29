import { useEffect, useState } from 'react'
import { Download, X } from 'lucide-react'
import { api } from '../lib/api'
import { errorStatus } from '../lib/errors'
import { useEscapeKey } from '../lib/useEscapeKey'
import { BTN_PRIMARY } from '../lib/buttonStyles'

// Dienst-CSV über einen wählbaren Zeitraum (GET /api/duty-slots/export).
//
// Bewusst ein eigener kleiner Dialog statt eines Direkt-Downloads des
// angezeigten Monats: der Zeitraum, für den man Dienste plant, deckt sich fast
// nie mit einem Kalendermonat (Vorrunde, Restsaison, ein Spieltags-Wochenende).
// Vorbelegt wird trotzdem der Monat, aus dem der Dialog aufgerufen wurde.
//
// Der Server liefert die Datei als Blob; es gibt keine Vorschau — die Datei ist
// die Vorschau. Inhaltlich trägt sie KEINE Belegung und keine Namen, sie darf
// deshalb auch an Ausrichter ohne TeamWERK-Zugang weitergegeben werden.

const INPUT = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'
const BTN_SECONDARY = 'px-4 py-2.5 sm:py-2 border border-brand-border rounded-md text-sm text-brand-text hover:bg-brand-surface-card disabled:opacity-40 disabled:cursor-not-allowed transition-colors'

interface Props {
  isOpen: boolean
  /** Erster Tag des im Kalender angezeigten Monats (YYYY-MM-DD) — Vorbelegung „Von". */
  monthStart: string
  /** Letzter Tag des im Kalender angezeigten Monats (YYYY-MM-DD) — Vorbelegung „Bis". */
  monthEnd: string
  onClose: () => void
}

export default function DutyExportModal({ isOpen, monthStart, monthEnd, onClose }: Props) {
  const [from, setFrom] = useState(monthStart)
  const [to, setTo] = useState(monthEnd)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEscapeKey(isOpen && !busy ? onClose : null)

  // Beim Öffnen auf den gerade angezeigten Monat zurücksetzen: der Dialog wird
  // aus einer Monatsansicht heraus aufgerufen, ein Zeitraum aus dem letzten
  // Aufruf wäre dort die falsche Vorbelegung.
  useEffect(() => {
    if (!isOpen) return
    // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync beim Öffnen (Prop-getrieben), kein Ableitungs-Bug
    setFrom(monthStart)
    setTo(monthEnd)
    setError(null)
  }, [isOpen, monthStart, monthEnd])

  const rangeInvalid = !from || !to || from > to

  const download = async () => {
    setBusy(true)
    setError(null)
    try {
      const r = await api.get(`/duty-slots/export?from=${from}&to=${to}`, { responseType: 'blob' })
      const url = URL.createObjectURL(r.data)
      const a = document.createElement('a')
      a.href = url
      a.download = `dienste_${from}_${to}.csv`
      a.click()
      URL.revokeObjectURL(url)
      onClose()
    } catch (e) {
      setError(errorStatus(e) === 403
        ? 'Keine Berechtigung für den Dienst-Export.'
        : 'Der Export ist fehlgeschlagen.')
    } finally {
      setBusy(false)
    }
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/40" onClick={busy ? undefined : onClose} />
      <div className="relative bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow max-w-lg mx-4 w-full">
        <div className="flex items-center justify-between px-6 py-4 border-b border-brand-border-subtle">
          <h2 className="text-lg font-bold text-brand-text">Dienste als CSV</h2>
          <button onClick={onClose} aria-label="Schließen" className="text-brand-text-muted hover:text-brand-text transition-colors">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-6 space-y-4">
          {error && (
            <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
              {error}
            </div>
          )}

          <div className="flex flex-wrap items-end gap-3">
            <div className="flex-1 min-w-[8rem]">
              <label htmlFor="duty-export-from" className="block text-xs text-brand-text-muted mb-1">Von</label>
              <input id="duty-export-from" type="date" className={INPUT} value={from} onChange={e => setFrom(e.target.value)} />
            </div>
            <div className="flex-1 min-w-[8rem]">
              <label htmlFor="duty-export-to" className="block text-xs text-brand-text-muted mb-1">Bis</label>
              <input id="duty-export-to" type="date" className={INPUT} value={to} onChange={e => setTo(e.target.value)} />
            </div>
          </div>

          <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
            Die Datei enthält je Dienst Datum, Beginn, Ende und Dauer sowie den Termin dazu:
            Ausrichter des Spieltags, Anwurfzeiten des Tages, Heimspiel am Vor-/Folgetag und die
            am Diensttyp eingestellte Regel für beide Fälle. Ohne Belegung und ohne Namen.
          </div>

          {rangeInvalid && from && to && (
            <p className="text-sm text-brand-danger">„Von" muss vor „Bis" liegen.</p>
          )}
        </div>

        <div className="flex items-center justify-end gap-2 px-6 py-4 border-t border-brand-border-subtle">
          <button onClick={onClose} disabled={busy} className={BTN_SECONDARY}>Abbrechen</button>
          <button onClick={download} disabled={busy || rangeInvalid} className={`${BTN_PRIMARY} flex items-center gap-1.5`}>
            <Download className="w-4 h-4" />
            {busy ? 'Wird erzeugt…' : 'Herunterladen'}
          </button>
        </div>
      </div>
    </div>
  )
}
