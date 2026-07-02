import { useRef, useState } from 'react'
import { Cast } from 'lucide-react'
import { isCastAvailable, loadCastSDK, startCastSession } from '../lib/cast'

// CastButton bietet dem User eine Wurftaste zum Chromecast/Google-TV. Das
// Cast-SDK wird erst **nach dem ersten Klick** geladen — bis dahin ist der
// Button lediglich ein „Ich könnte casten"-Hinweis. Grund: DSGVO-neutral
// (keine passive Google-Verbindung beim Seitenaufruf).
//
// Der Button erscheint nur, wenn die Cast-API tatsächlich verfügbar ist — auf
// Safari/Firefox bleibt er unsichtbar, dort greift AirPlay bzw. Cast ist nicht
// unterstützt.
export function CastButton({ masterURL }: { masterURL: string }) {
  // available=null → noch nicht geprüft. Nach `loadCastSDK` steht available=true/false.
  // Initial-Check als useState-Initializer, damit kein Effect nötig ist
  // (react-hooks/set-state-in-effect-Regel).
  const [available, setAvailable] = useState<boolean | null>(() =>
    isCastAvailable() ? true : null,
  )
  const [error, setError] = useState('')
  const attemptedLoadRef = useRef(false)

  async function handleClick() {
    setError('')
    if (!attemptedLoadRef.current) {
      attemptedLoadRef.current = true
      const ok = await loadCastSDK()
      setAvailable(ok && isCastAvailable())
      if (!ok) {
        setError('Cast-SDK konnte nicht geladen werden.')
        return
      }
    }
    try {
      await startCastSession(masterURL)
    } catch (e) {
      // User hat Session-Dialog abgebrochen — kein Fehler, still bleiben.
      if (e instanceof Error && /cancel/i.test(e.message)) return
      setError('Cast konnte nicht gestartet werden.')
    }
  }

  // Verstecken, wenn ein früherer Check ergeben hat, dass die API nicht da ist
  // (nur nach Klick möglich). Vor Klick zeigen wir den Button optimistisch —
  // Chrome hat sehr wahrscheinlich Cast, Safari/Firefox verstecken sich erst
  // nach dem ersten (dann fehlschlagenden) Klick.
  if (available === false) return null

  return (
    <div>
      <button
        type="button"
        onClick={handleClick}
        aria-label="Auf Chromecast wiedergeben"
        className="bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed inline-flex items-center gap-1.5"
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
