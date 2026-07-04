import { useCallback, useEffect, useState } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'
import { ChevronLeft } from 'lucide-react'
import { api } from '../lib/api'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import MarkdownRenderer from '../components/MarkdownRenderer'

interface DutyTypeItem {
  id: number
  name: string
  instruction_md?: string
  instruction_updated_at?: string
}

// Der globale AppShell-Zurück-Button erscheint nur, wenn window.history.state.idx > 0.
// Bei Deep-Link/Cold-Start (App aus dem App-Switcher wiederhergestellt, Push-Klick,
// direkter URL-Aufruf) fehlt er — auf iOS-PWA ohne Browser-Chrome eine Sackgasse.
// Selbes Muster wie <FileViewer>: page-lokaler Fallback nur genau dann rendern.
function useIsColdStart(): boolean {
  const location = useLocation()
  const [coldStart, setColdStart] = useState(() =>
    typeof window === 'undefined' ? false : (window.history.state?.idx ?? 0) === 0,
  )
  useEffect(() => {
    setColdStart((window.history.state?.idx ?? 0) === 0)
  }, [location])
  return coldStart
}

function FallbackBackButton() {
  return (
    <Link
      to="/dienste"
      className="inline-flex items-center gap-1 text-sm text-brand-text-muted hover:text-brand-text mb-4"
    >
      <ChevronLeft className="w-4 h-4" /> Zur Dienstbörse
    </Link>
  )
}

export default function DutyInstructionPage() {
  const { typeId } = useParams<{ typeId: string }>()
  const [item, setItem] = useState<DutyTypeItem | null>(null)
  const [notFound, setNotFound] = useState(false)
  const coldStart = useIsColdStart()

  const load = useCallback(async () => {
    if (!typeId) return
    // Volltext aus dem Detail-Pfad (die Typen-Liste liefert ihn nicht mehr).
    try {
      const { data } = await api.get<DutyTypeItem>(`/duty-types/${typeId}/instruction`)
      setItem(data ?? null)
      setNotFound(!data)
    } catch {
      setItem(null)
      setNotFound(true)
    }
  }, [typeId])

  // eslint-disable-next-line react-hooks/set-state-in-effect -- Laden-beim-Mount; setState liegt hinter await in load(), kein synchroner Ableitungs-Bug
  useEffect(() => { load() }, [load])
  useLiveUpdates(event => { if (event === 'duties') load() })

  if (notFound) {
    return (
      <div className="max-w-3xl">
        {coldStart && <FallbackBackButton />}
        <p className="text-sm text-brand-text-muted">Diensttyp nicht gefunden.</p>
      </div>
    )
  }

  if (!item) {
    return <p className="text-sm text-brand-text-muted">Lade Anleitung…</p>
  }

  const updated = item.instruction_updated_at ? new Date(item.instruction_updated_at) : null
  const updatedLabel = updated
    ? updated.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric' })
    : ''

  return (
    <div className="max-w-3xl">
      {coldStart && <FallbackBackButton />}
      <h1 className="text-2xl font-bold mb-1">Anleitung: {item.name}</h1>
      {updatedLabel && (
        <p className="text-xs text-brand-text-muted mb-6">Zuletzt geändert am {updatedLabel}</p>
      )}

      {item.instruction_md ? (
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
          <MarkdownRenderer markdown={item.instruction_md} />
        </div>
      ) : (
        <p className="text-sm text-brand-text-muted italic">Für diesen Dienst gibt es noch keine Anleitung.</p>
      )}
    </div>
  )
}
