import { useEffect, useState } from 'react'
import { Trash2, Car, Users, X } from 'lucide-react'
import { api } from '../lib/api'

interface CarpoolEntry {
  id: number
  userName: string
  plaetze?: number
  treffpunkt?: string
  notiz?: string
  isOwn: boolean
}

interface GameCarpoolData {
  game: { id: number; date: string; opponent: string; team: string; eventType: string }
  biete: CarpoolEntry[]
  suche: CarpoolEntry[]
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

function EntryCard({ entry, onDelete }: { entry: CarpoolEntry; onDelete: (id: number) => void }) {
  return (
    <div className="flex items-start justify-between gap-2 py-2 border-b border-brand-border-subtle last:border-0">
      <div className="min-w-0">
        <span className="text-sm font-medium text-brand-text">{entry.userName}</span>
        {entry.plaetze != null && (
          <span className="ml-2 text-xs text-brand-text-muted">{entry.plaetze} Platz/Plätze</span>
        )}
        {entry.treffpunkt && (
          <p className="text-xs text-brand-text-muted mt-0.5">Treffpunkt: {entry.treffpunkt}</p>
        )}
        {entry.notiz && (
          <p className="text-xs text-brand-text-muted mt-0.5">{entry.notiz}</p>
        )}
      </div>
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
    setSaving(true)
    setError(null)
    try {
      await api.post('/mitfahrgelegenheiten', {
        gameId,
        typ,
        plaetze: typ === 'biete' && plaetze ? parseInt(plaetze) : null,
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

          {typ === 'biete' && (
            <div>
              <label className="block text-sm font-medium text-brand-text mb-1">Freie Plätze</label>
              <input
                type="number"
                min="1"
                value={plaetze}
                onChange={e => setPlaetze(e.target.value)}
                placeholder="z. B. 3"
                className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
              />
              {vehicleSeats && !plaetze && (
                <p className="text-xs text-brand-text-muted mt-1">
                  Laut Profil: {vehicleSeats} Plätze
                </p>
              )}
            </div>
          )}

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
}

function GameCard({ data, onDelete, onOpenForm }: GameCardProps) {
  const [activeTab, setActiveTab] = useState<'biete' | 'suche'>('biete')
  const hasOwn = [...data.biete, ...data.suche].some(e => e.isOwn)

  return (
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
      <div className="px-4 py-3 border-b border-brand-border-subtle">
        <div className="flex items-start justify-between gap-2">
          <div>
            <p className="text-xs text-brand-text-muted">{formatDate(data.game.date)}</p>
            <h2 className="text-sm font-semibold text-brand-text">{data.game.team} vs. {data.game.opponent}</h2>
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
              : data.biete.map(e => <EntryCard key={e.id} entry={e} onDelete={onDelete} />)
            : data.suche.length === 0
              ? <p className="text-sm text-brand-text-muted py-2">Noch keine Mitfahrgesuche.</p>
              : data.suche.map(e => <EntryCard key={e.id} entry={e} onDelete={onDelete} />)
          }
        </div>
      </div>

      {/* Desktop: two columns */}
      <div className="hidden sm:grid grid-cols-2 divide-x divide-brand-border-subtle">
        <div className="px-4 py-3">
          <p className="text-xs font-semibold uppercase tracking-wider text-brand-text-muted mb-2">Fahrangebote ({data.biete.length})</p>
          {data.biete.length === 0
            ? <p className="text-sm text-brand-text-muted">Noch keine Angebote.</p>
            : data.biete.map(e => <EntryCard key={e.id} entry={e} onDelete={onDelete} />)
          }
        </div>
        <div className="px-4 py-3">
          <p className="text-xs font-semibold uppercase tracking-wider text-brand-text-muted mb-2">Mitfahrgesuche ({data.suche.length})</p>
          {data.suche.length === 0
            ? <p className="text-sm text-brand-text-muted">Noch keine Gesuche.</p>
            : data.suche.map(e => <EntryCard key={e.id} entry={e} onDelete={onDelete} />)
          }
        </div>
      </div>

      {hasOwn && (
        <div className="px-4 py-2 border-t border-brand-border-subtle bg-brand-surface-card">
          <span className="text-xs text-brand-text-muted">Du bist eingetragen.{' '}</span>
          <button
            onClick={() => onOpenForm(data.game.id, 'biete')}
            className="text-xs text-brand-text underline hover:no-underline"
          >
            Ändern
          </button>
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
  const [response, setResponse] = useState<ListResponse>({ games: [] })
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<EventTab>('auswärts')
  const [modal, setModal] = useState<{ gameId: number; typ: 'biete' | 'suche' } | null>(null)

  const load = () => {
    setLoading(true)
    api.get('/mitfahrgelegenheiten')
      .then(res => {
        setResponse(res.data ?? { games: [] })
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

  const tabGames = response.games.filter(d => d.game.eventType === activeTab)

  const countForTab = (tab: EventTab) => response.games.filter(d => d.game.eventType === tab).length

  const tabs: EventTab[] = ['auswärts', 'heim', 'generisch']

  return (
    <div className="max-w-3xl mx-auto">
      <h1 className="text-2xl font-bold text-brand-text mb-6">Mitfahrgelegenheiten</h1>

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
          {/* Tab navigation */}
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
              {activeTab === 'auswärts' && 'Keine Auswärtsspiele geplant.'}
              {activeTab === 'heim' && 'Keine Heimspiele geplant.'}
              {activeTab === 'generisch' && 'Keine Events geplant.'}
            </p>
          ) : (
            <div className="space-y-4">
              {tabGames.map(d => (
                <GameCard
                  key={d.game.id}
                  data={d}
                  onDelete={handleDelete}
                  onOpenForm={(gameId, typ) => setModal({ gameId, typ })}
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
