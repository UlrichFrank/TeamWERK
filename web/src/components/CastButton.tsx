import { useState } from 'react'
import { Cast } from 'lucide-react'
import { isCastAvailable, loadCastSDK, startCastSession } from '../lib/cast'
import { BTN_SMALL } from '../lib/buttonStyles'

// CastButton bietet dem User eine Wurftaste zum Chromecast/Google-TV. Das
// Cast-SDK wird erst **nach dem ersten Klick** geladen — bis dahin ist der
// Button lediglich ein „Ich könnte casten"-Hinweis. Grund: DSGVO-neutral
// (keine passive Google-Verbindung beim Seitenaufruf).
//
// Vor dem Klick zeigen wir den Button optimistisch. Scheitert der Klick, bleibt
// **immer** ein sichtbarer Hinweis stehen — früher verschwand der Button
// kommentarlos, und ein CSP-Block des SDK sah genauso aus wie ein Browser ohne
// Cast-Unterstützung.

export const MSG_UNAVAILABLE =
  'Chromecast ist in diesem Browser nicht verfügbar. Casten geht nur aus Chrome auf dem Computer oder auf Android.'
export const MSG_LOAD_FAILED =
  'Die Chromecast-Funktion konnte nicht geladen werden (Werbeblocker oder Netzwerk?). Bitte erneut versuchen.'
export const MSG_START_FAILED = 'Wiedergabe auf dem Chromecast konnte nicht gestartet werden.'

// Die Cast-API lehnt mit einem chrome.cast.ErrorCode-String ab ("cancel",
// "timeout", "receiver_unavailable", …), nicht mit einem Error-Objekt.
function castErrorCode(e: unknown): string {
  if (typeof e === 'string') return e
  if (e instanceof Error) return e.message
  if (e && typeof e === 'object' && 'code' in e) return String((e as { code: unknown }).code)
  return String(e)
}

export function CastButton({ masterURL }: { masterURL: string }) {
  // unavailable wird erst nach einem Ladeversuch gesetzt; ist die API schon vor
  // dem Mount injiziert, ist der Button ohnehin sichtbar.
  const [unavailable, setUnavailable] = useState(false)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function handleClick() {
    setError('')
    setBusy(true)
    try {
      if (!isCastAvailable()) {
        const result = await loadCastSDK()
        if (result === 'unavailable') {
          setUnavailable(true)
          return
        }
        if (result === 'load_failed') {
          setError(MSG_LOAD_FAILED)
          return
        }
      }
      await startCastSession(masterURL)
    } catch (e) {
      const code = castErrorCode(e)
      // User hat den Geräte-Dialog geschlossen — kein Fehler, still bleiben.
      if (/cancel/i.test(code)) return
      console.warn('[cast] Session fehlgeschlagen:', code)
      setError(MSG_START_FAILED)
    } finally {
      setBusy(false)
    }
  }

  if (unavailable) {
    return (
      <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
        {MSG_UNAVAILABLE}
      </div>
    )
  }

  return (
    <div>
      <button
        type="button"
        onClick={handleClick}
        disabled={busy}
        aria-label="Auf Chromecast wiedergeben"
        className={`${BTN_SMALL} inline-flex items-center gap-1.5`}
      >
        <Cast className="w-4 h-4" />
        Cast
      </button>
      {error && (
        <div className="mt-2 p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
          {error}
        </div>
      )}
    </div>
  )
}

export default CastButton
