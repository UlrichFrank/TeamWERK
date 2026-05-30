import { useEffect, useState } from 'react'
import { Trash2, Car, Users, X, Check, UserPlus } from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'
import { api } from '../lib/api'

interface CarpoolEntry {
  id: number
  userName: string
  plaetze?: number
  treffpunkt?: string
  notiz?: string
  isOwn: boolean
}

interface PaarungEntry {
  id: number
  bieteId: number
  sucheId: number
  bieteName: string
  sucheName: string
  anzahl: number
  status: 'pending' | 'confirmed'
  initiertVon: 'biete' | 'suche'
  bieteIsOwn: boolean
  sucheIsOwn: boolean
}

interface GameCarpoolData {
  game: { id: number; date: string; opponent: string; team: string; eventType: string }
  biete: CarpoolEntry[]
  suche: CarpoolEntry[]
  paarungen: PaarungEntry[]
}

interface ListResponse {
  games: GameCarpoolData[]
  vehicleSeats?: number | null
}

type EventTab = 'auswärts' | 'heim' | 'generisch'

function formatDate(iso: string) {
  if (iso.length >= 10) {
    const d = new Date(iso.slice(0, 10) + 'T12:00:00')
    return d.toLocaleDateString('de-DE', { weekday: 'short', day: '2-digit', month: '2-digit', year: 'numeric' })
  }
  return iso
}

function gameTitle(game: GameCarpoolData['game']): string {
  if (game.eventType === 'generisch') return `${game.team} - ${game.opponent}`
  if (game.eventType === 'heim') return `${game.team} - Team Stuttgart vs ${game.opponent}`
  return `${game.team} - ${game.opponent} vs Team Stuttgart`
}

// How many seats the bieter still has available (pending + confirmed count against capacity)
function freePlaetze(bieteEntry: CarpoolEntry, paarungen: PaarungEntry[]): number {
  if (bieteEntry.plaetze == null) return 0
  const used = paarungen
    .filter(p => p.bieteId === bieteEntry.id && (p.status === 'pending' || p.status === 'confirmed'))
    .reduce((sum, p) => sum + p.anzahl, 0)
  return bieteEntry.plaetze - used
}

interface EntryCardProps {
  entry: CarpoolEntry
  typ: 'biete' | 'suche'
  paarungen: PaarungEntry[]
  myBieteIds: number[]
  mySucheIds: number[]
  onDelete: (id: number) => void
  onRequest: (bieteId: number, sucheId: number) => void
  onConfirm: (paarungId: number) => void
  onReject: (paarungId: number) => void
}

function EntryCard({ entry, typ, paarungen, myBieteIds, mySucheIds, onDelete, onRequest, onConfirm, onReject }: EntryCardProps) {
  const free = typ === 'biete' ? freePlaetze(entry, paarungen) : null

  // Paarungen that involve this entry
  const entryPaarungen = paarungen.filter(p =>
    typ === 'biete' ? p.bieteId === entry.id : p.sucheId === entry.id
  )

  // Can a sucher request this biete entry?
  const canRequestAsBiete = typ === 'biete' && !entry.isOwn && free !== null && free > 0 &&
    mySucheIds.length > 0 &&
    !paarungen.some(p => p.bieteId === entry.id && (p.bieteIsOwn || p.sucheIsOwn) &&
      (p.status === 'pending' || p.status === 'confirmed'))

  // Can a bieter invite this suche entry? (from one of their own biete entries)
  const canInviteAsSuche = typ === 'suche' && !entry.isOwn && myBieteIds.length > 0 &&
    !paarungen.some(p => p.sucheId === entry.id && (p.bieteIsOwn || p.sucheIsOwn) &&
      (p.status === 'pending' || p.status === 'confirmed'))

  return (
    <div className="py-2 border-b border-brand-border-subtle last:border-0">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-sm font-medium text-brand-text">{entry.userName}</span>
            {entry.plaetze != null && (
              <span className="text-xs text-brand-text-muted">
                {typ === 'biete'
                  ? `${free}/${entry.plaetze} Plätze frei`
                  : `${entry.plaetze} Person${entry.plaetze > 1 ? 'en' : ''}`}
              </span>
            )}
          </div>
          {entry.treffpunkt && (
            <p className="text-xs text-brand-text-muted mt-0.5">Treffpunkt: {entry.treffpunkt}</p>
          )}
          {entry.notiz && (
            <p className="text-xs text-brand-text-muted mt-0.5">{entry.notiz}</p>
          )}
        </div>
        <div className="flex items-center gap-1 flex-shrink-0">
          {canRequestAsBiete && (
            <button
              onClick={() => {
                const mySuccheId = mySucheIds[0]
                onRequest(entry.id, mySuccheId)
              }}
              title="Mitfahrt anfragen"
              className="bg-brand-yellow text-brand-black rounded-md px-2 py-1 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors min-h-[44px] sm:min-h-0 flex items-center gap-1"
            >
              <UserPlus className="w-3 h-3" />
              <span className="hidden sm:inline">Anfragen</span>
            </button>
          )}
          {canInviteAsSuche && (
            <button
              onClick={() => onRequest(myBieteIds[0], entry.id)}
              title="Mitnahme anbieten"
              className="bg-brand-yellow text-brand-black rounded-md px-2 py-1 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors min-h-[44px] sm:min-h-0 flex items-center gap-1"
            >
              <Car className="w-3 h-3" />
              <span className="hidden sm:inline">Einladen</span>
            </button>
          )}
          {entry.isOwn && (
            <button
              onClick={() => onDelete(entry.id)}
              aria-label="Eintrag löschen"
              className="flex-shrink-0 p-1.5 text-brand-text-muted hover:text-brand-danger transition-colors min-h-[44px] min-w-[44px] flex items-center justify-center"
            >
              <Trash2 className="w-4 h-4" />
            </button>
          )}
        </div>
      </div>

      {/* Pending paarungen involving this entry where I can act */}
      {entryPaarungen.filter(p => p.status === 'pending').map(p => {
        const iCanConfirm = (p.initiertVon === 'suche' && p.bieteIsOwn) ||
                            (p.initiertVon === 'biete' && p.sucheIsOwn)
        const iInitiated = (p.initiertVon === 'biete' && p.bieteIsOwn) ||
                           (p.initiertVon === 'suche' && p.sucheIsOwn)
        if (!iCanConfirm && !iInitiated) return null
        return (
          <div key={p.id} className="mt-2 flex items-center gap-2 bg-brand-border-subtle/40 rounded-md px-2 py-1.5">
            <span className="text-xs text-brand-text-muted flex-1">
              {iCanConfirm
                ? `Anfrage: ${p.initiertVon === 'suche' ? p.sucheName : p.bieteName} (${p.anzahl} Platz${p.anzahl > 1 ? 'e' : ''})`
                : `Ausstehend: Anfrage an ${p.initiertVon === 'suche' ? p.bieteName : p.sucheName}`}
            </span>
            {iCanConfirm && (
              <>
                <button
                  onClick={() => onConfirm(p.id)}
                  aria-label="Bestätigen"
                  className="p-1 text-brand-text-muted hover:text-green-600 transition-colors"
                >
                  <Check className="w-4 h-4" />
                </button>
                <button
                  onClick={() => onReject(p.id)}
                  aria-label="Ablehnen"
                  className="p-1 text-brand-text-muted hover:text-brand-danger transition-colors"
                >
                  <X className="w-4 h-4" />
                </button>
              </>
            )}
            {iInitiated && (
              <button
                onClick={() => onReject(p.id)}
                aria-label="Anfrage zurückziehen"
                className="p-1 text-brand-text-muted hover:text-brand-danger transition-colors"
              >
                <X className="w-4 h-4" />
              </button>
            )}
          </div>
        )
      })}
    </div>
  )
}

interface FormModalProps {
  gameId: number
  initialTyp?: 'biete' | 'suche'
  vehicleSeats?: number | null
  onClose: () => void
  onSaved: () => void
}

function FormModal({ gameId, initialTyp, vehicleSeats, onClose, onSaved }: FormModalProps) {
  const [typ, setTyp] = useState<'biete' | 'suche'>(initialTyp ?? 'biete')
  const [plaetze, setPlaetze] = useState(() =>
    initialTyp === 'biete' && vehicleSeats ? String(vehicleSeats) : ''
  )
  const [treffpunkt, setTreffpunkt] = useState('')
  const [notiz, setNotiz] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (typ === 'suche' && (!plaetze || parseInt(plaetze) < 1)) {
      setError('Bitte Anzahl Personen angeben.')
      return
    }
    setSaving(true)
    setError(null)
    try {
      await api.post('/mitfahrgelegenheiten', {
        gameId,
        typ,
        plaetze: plaetze ? parseInt(plaetze) : null,
        treffpunkt,
        notiz,
      })
      onSaved()
      onClose()
    } catch {
      setError('Fehler beim Speichern. Bitte erneut versuchen.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-4 bg-black/40">
      <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-md">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-base font-semibold text-brand-text">Mitfahrgelegenheit eintragen</h2>
          <button onClick={onClose} aria-label="Schließen" className="text-brand-text-muted hover:text-brand-text transition-colors">
            <X className="w-5 h-5" />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => { setTyp('biete'); if (!plaetze && vehicleSeats) setPlaetze(String(vehicleSeats)) }}
              className={`flex-1 py-2.5 sm:py-2 text-sm font-medium rounded-md border transition-colors ${typ === 'biete' ? 'bg-brand-yellow text-brand-black border-brand-yellow' : 'border-brand-border text-brand-text-muted hover:border-brand-text'}`}
            >
              Ich biete Mitfahrt
            </button>
            <button
              type="button"
              onClick={() => setTyp('suche')}
              className={`flex-1 py-2.5 sm:py-2 text-sm font-medium rounded-md border transition-colors ${typ === 'suche' ? 'bg-brand-yellow text-brand-black border-brand-yellow' : 'border-brand-border text-brand-text-muted hover:border-brand-text'}`}
            >
              Ich suche Mitfahrt
            </button>
          </div>

          <div>
            <label className="block text-sm font-medium text-brand-text mb-1">
              {typ === 'biete' ? 'Freie Plätze' : 'Anzahl Personen'}
              {typ === 'suche' && <span className="text-brand-danger ml-1">*</span>}
            </label>
            <input
              type="number"
              min="1"
              value={plaetze}
              onChange={e => setPlaetze(e.target.value)}
              placeholder={typ === 'biete' ? 'z. B. 3' : 'z. B. 2'}
              required={typ === 'suche'}
              className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
            />
            {typ === 'biete' && vehicleSeats && !plaetze && (
              <p className="text-xs text-brand-text-muted mt-1">Laut Profil: {vehicleSeats} Plätze</p>
            )}
          </div>

          <div>
            <label className="block text-sm font-medium text-brand-text mb-1">Treffpunkt <span className="font-normal text-brand-text-muted">(optional)</span></label>
            <input
              type="text"
              value={treffpunkt}
              onChange={e => setTreffpunkt(e.target.value)}
              placeholder="z. B. Halle um 09:00 Uhr"
              className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-brand-text mb-1">Notiz <span className="font-normal text-brand-text-muted">(optional)</span></label>
            <input
              type="text"
              value={notiz}
              onChange={e => setNotiz(e.target.value)}
              placeholder="z. B. Parkplatz Ost"
              className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
            />
          </div>

          {error && (
            <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{error}</p>
          )}

          <button
            type="submit"
            disabled={saving}
            className="w-full bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {saving ? 'Speichern…' : 'Speichern'}
          </button>
        </form>
      </div>
    </div>
  )
}

interface GameCardProps {
  data: GameCarpoolData
  onDelete: (id: number) => void
  onOpenForm: (gameId: number, typ: 'biete' | 'suche') => void
  onRequest: (bieteId: number, sucheId: number) => void
  onConfirm: (paarungId: number) => void
  onReject: (paarungId: number) => void
}

function GameCard({ data, onDelete, onOpenForm, onRequest, onConfirm, onReject }: GameCardProps) {
  const [activeTab, setActiveTab] = useState<'biete' | 'suche'>('biete')
  const hasOwnBiete = data.biete.some(e => e.isOwn)
  const hasOwn = hasOwnBiete || data.suche.some(e => e.isOwn)

  const myBieteIds = data.biete.filter(e => e.isOwn).map(e => e.id)
  const mySucheIds = data.suche.filter(e => e.isOwn).map(e => e.id)

  const confirmedPaarungen = data.paarungen.filter(p => p.status === 'confirmed')

  const entryCardProps = { paarungen: data.paarungen, myBieteIds, mySucheIds, onDelete, onRequest, onConfirm, onReject }

  return (
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
      <div className="px-4 py-3 border-b border-brand-border-subtle">
        <div className="flex items-start justify-between gap-2">
          <div>
            <p className="text-xs text-brand-text-muted">{formatDate(data.game.date)}</p>
            <h2 className="text-sm font-semibold text-brand-text">{gameTitle(data.game)}</h2>
          </div>
          {!hasOwn && (
            <div className="flex gap-2 flex-shrink-0">
              <button
                onClick={() => onOpenForm(data.game.id, 'biete')}
                className="bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors min-h-[44px] sm:min-h-0"
              >
                <span className="hidden sm:inline">Ich biete Mitfahrt</span>
                <Car className="w-4 h-4 sm:hidden" />
              </button>
              <button
                onClick={() => onOpenForm(data.game.id, 'suche')}
                className="bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors min-h-[44px] sm:min-h-0"
              >
                <span className="hidden sm:inline">Ich suche Mitfahrt</span>
                <Users className="w-4 h-4 sm:hidden" />
              </button>
            </div>
          )}
          {hasOwn && (
            <button
              onClick={() => onOpenForm(data.game.id, hasOwnBiete ? 'suche' : 'biete')}
              className="bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors min-h-[44px] sm:min-h-0"
            >
              <span className="hidden sm:inline">Eintrag hinzufügen</span>
              <span className="sm:hidden">+</span>
            </button>
          )}
        </div>
      </div>

      {/* Mobile: tabs */}
      <div className="sm:hidden">
        <div className="flex border-b border-brand-border-subtle">
          <button
            onClick={() => setActiveTab('biete')}
            className={`flex-1 py-2 text-sm font-medium transition-colors ${activeTab === 'biete' ? 'text-brand-text border-b-2 border-brand-yellow' : 'text-brand-text-muted'}`}
          >
            Fahrangebote ({data.biete.length})
          </button>
          <button
            onClick={() => setActiveTab('suche')}
            className={`flex-1 py-2 text-sm font-medium transition-colors ${activeTab === 'suche' ? 'text-brand-text border-b-2 border-brand-yellow' : 'text-brand-text-muted'}`}
          >
            Mitfahrgesuche ({data.suche.length})
          </button>
        </div>
        <div className="px-4 py-2">
          {activeTab === 'biete'
            ? data.biete.length === 0
              ? <p className="text-sm text-brand-text-muted py-2">Noch keine Fahrangebote.</p>
              : data.biete.map(e => <EntryCard key={e.id} entry={e} typ="biete" {...entryCardProps} />)
            : data.suche.length === 0
              ? <p className="text-sm text-brand-text-muted py-2">Noch keine Mitfahrgesuche.</p>
              : data.suche.map(e => <EntryCard key={e.id} entry={e} typ="suche" {...entryCardProps} />)
          }
        </div>
      </div>

      {/* Desktop: two columns */}
      <div className="hidden sm:grid grid-cols-2 divide-x divide-brand-border-subtle">
        <div className="px-4 py-3">
          <p className="text-xs font-semibold uppercase tracking-wider text-brand-text-muted mb-2">Fahrangebote ({data.biete.length})</p>
          {data.biete.length === 0
            ? <p className="text-sm text-brand-text-muted">Noch keine Angebote.</p>
            : data.biete.map(e => <EntryCard key={e.id} entry={e} typ="biete" {...entryCardProps} />)
          }
        </div>
        <div className="px-4 py-3">
          <p className="text-xs font-semibold uppercase tracking-wider text-brand-text-muted mb-2">Mitfahrgesuche ({data.suche.length})</p>
          {data.suche.length === 0
            ? <p className="text-sm text-brand-text-muted">Noch keine Gesuche.</p>
            : data.suche.map(e => <EntryCard key={e.id} entry={e} typ="suche" {...entryCardProps} />)
          }
        </div>
      </div>

      {/* Confirmed pairings — visible to all */}
      {confirmedPaarungen.length > 0 && (
        <div className="px-4 py-3 border-t border-brand-border-subtle">
          <p className="text-xs font-semibold uppercase tracking-wider text-brand-text-muted mb-2">Fahrgemeinschaften</p>
          <div className="space-y-1">
            {confirmedPaarungen.map(p => (
              <div key={p.id} className="flex items-center gap-2 text-xs text-brand-text">
                <Check className="w-3 h-3 text-green-600 flex-shrink-0" />
                <span>
                  <span className="font-medium">{p.sucheName}</span>
                  {p.anzahl > 1 && ` (${p.anzahl} Personen)`}
                  {' '}fährt mit <span className="font-medium">{p.bieteName}</span>
                </span>
                {(p.bieteIsOwn || p.sucheIsOwn) && (
                  <button
                    onClick={() => onReject(p.id)}
                    aria-label="Paarung stornieren"
                    className="ml-auto p-1 text-brand-text-muted hover:text-brand-danger transition-colors"
                  >
                    <X className="w-3 h-3" />
                  </button>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

const TAB_LABELS: Record<EventTab, string> = {
  'auswärts': 'Auswärtsspiele',
  'heim': 'Heimspiele',
  'generisch': 'Events',
}

export default function MitfahrgelegenheitenPage() {
  const { user } = useAuth()
  const [response, setResponse] = useState<ListResponse>({ games: [] })
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<EventTab>('auswärts')
  const [modal, setModal] = useState<{ gameId: number; typ: 'biete' | 'suche' } | null>(null)
  const [viewMine, setViewMine] = useState(false)

  void user // used to re-render when auth changes

  const load = () => {
    setLoading(true)
    api.get('/mitfahrgelegenheiten')
      .then(res => {
        setResponse({ games: res.data?.games ?? [], vehicleSeats: res.data?.vehicleSeats })
        setLoading(false)
      })
      .catch(() => { setError('Fehler beim Laden.'); setLoading(false) })
  }

  useEffect(() => { load() }, [])

  const handleDelete = async (id: number) => {
    try {
      await api.delete(`/mitfahrgelegenheiten/${id}`)
      load()
    } catch {
      alert('Fehler beim Löschen.')
    }
  }

  const handleRequest = async (bieteId: number, sucheId: number) => {
    try {
      await api.post('/mitfahrt-paarungen', { bieteId, sucheId })
      load()
    } catch (err: unknown) {
      const status = (err as { response?: { status: number } })?.response?.status
      if (status === 409) {
        alert('Keine freien Plätze mehr oder bereits eine Anfrage vorhanden.')
      } else {
        alert('Fehler beim Anfragen.')
      }
    }
  }

  const handleConfirm = async (paarungId: number) => {
    try {
      await api.post(`/mitfahrt-paarungen/${paarungId}/confirm`)
      load()
    } catch {
      alert('Fehler beim Bestätigen.')
    }
  }

  const handleReject = async (paarungId: number) => {
    try {
      await api.post(`/mitfahrt-paarungen/${paarungId}/reject`)
      load()
    } catch {
      alert('Fehler beim Ablehnen.')
    }
  }

  const filteredGames = viewMine
    ? response.games.filter(d =>
        [...d.biete, ...d.suche].some(e => e.isOwn) ||
        d.paarungen.some(p => p.bieteIsOwn || p.sucheIsOwn)
      )
    : response.games
  const tabGames = filteredGames.filter(d => d.game.eventType === activeTab)
  const countForTab = (tab: EventTab) => filteredGames.filter(d => d.game.eventType === tab).length

  const tabs: EventTab[] = ['auswärts', 'heim', 'generisch']

  return (
    <div className="max-w-3xl mx-auto">
      <div className="flex items-center justify-between mb-6 flex-wrap gap-2">
        <h1 className="text-2xl font-bold text-brand-text">Mitfahrgelegenheiten</h1>
        <div className="flex rounded-lg border border-brand-border-subtle overflow-hidden text-sm">
          <button
            onClick={() => setViewMine(false)}
            className={`px-3 py-1.5 ${!viewMine ? 'bg-brand-yellow text-brand-black font-medium' : 'text-brand-text-muted hover:bg-brand-border-subtle'}`}
          >
            Alle
          </button>
          <button
            onClick={() => setViewMine(true)}
            className={`px-3 py-1.5 border-l border-brand-border-subtle ${viewMine ? 'bg-brand-yellow text-brand-black font-medium' : 'text-brand-text-muted hover:bg-brand-border-subtle'}`}
          >
            Meine
          </button>
        </div>
      </div>

      {loading && (
        <div className="space-y-3">
          {[1, 2].map(i => <div key={i} className="h-40 bg-brand-border-subtle rounded-xl animate-pulse" />)}
        </div>
      )}

      {!loading && error && (
        <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{error}</p>
      )}

      {!loading && !error && (
        <>
          <div className="flex gap-1 mb-6 border-b border-brand-border-subtle">
            {tabs.map(tab => (
              <button
                key={tab}
                onClick={() => setActiveTab(tab)}
                className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
                  activeTab === tab
                    ? 'border-brand-yellow text-brand-text'
                    : 'border-transparent text-brand-text-muted hover:text-brand-text'
                }`}
              >
                {TAB_LABELS[tab]}
                {countForTab(tab) > 0 && (
                  <span className="ml-1.5 text-xs text-brand-text-subtle">({countForTab(tab)})</span>
                )}
              </button>
            ))}
          </div>

          {tabGames.length === 0 ? (
            <p className="text-sm text-brand-text-muted">
              {viewMine
                ? 'Du bist bei keinem Spiel eingetragen.'
                : activeTab === 'auswärts' ? 'Keine Auswärtsspiele geplant.'
                : activeTab === 'heim' ? 'Keine Heimspiele geplant.'
                : 'Keine Events geplant.'}
            </p>
          ) : (
            <div className="space-y-4">
              {tabGames.map(d => (
                <GameCard
                  key={d.game.id}
                  data={d}
                  onDelete={handleDelete}
                  onOpenForm={(gameId, typ) => setModal({ gameId, typ })}
                  onRequest={handleRequest}
                  onConfirm={handleConfirm}
                  onReject={handleReject}
                />
              ))}
            </div>
          )}
        </>
      )}

      {modal && (
        <FormModal
          gameId={modal.gameId}
          initialTyp={modal.typ}
          vehicleSeats={response.vehicleSeats}
          onClose={() => setModal(null)}
          onSaved={load}
        />
      )}
    </div>
  )
}
