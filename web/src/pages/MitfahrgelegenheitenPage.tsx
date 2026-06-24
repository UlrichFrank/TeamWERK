import { useEffect, useMemo, useState } from 'react'
import { useLocation, useSearchParams } from 'react-router-dom'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import { Trash2, Car, Users, X, Check, UserPlus, Home, Plane, Calendar, UserCheck } from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'
import { api } from '../lib/api'
import NumberSpinner from '../components/NumberSpinner'
import PersonChip from '../components/PersonChip'
import { getEventColors } from '../lib/eventColors'
import EventTypeFilter, { type EventTypeFilterEntry } from '../components/EventTypeFilter'
import { buildTeamShortNames, type TeamForName } from '../lib/teamName'
import { useCompactHeader } from '../hooks/useCompactHeader'

interface CarpoolEntry {
  id: number
  userId: number
  userName: string
  photoUrl?: string
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
  bietePhotoUrl?: string
  suchePhotoUrl?: string
  bieteUserId: number
  sucheUserId: number
  anzahl: number
  status: 'pending' | 'confirmed'
  initiertVon: 'biete' | 'suche'
  bieteIsOwn: boolean
  sucheIsOwn: boolean
}

interface GameCarpoolData {
  game: {
    id: number
    date: string
    time: string
    opponent: string
    team: string
    teamIds: number[]
    eventType: string
  }
  biete: CarpoolEntry[]
  suche: CarpoolEntry[]
  paarungen: PaarungEntry[]
}

interface ChildUser {
  userId: number
  name: string
}

interface ListResponse {
  games: GameCarpoolData[]
  vehicleSeats?: number | null
  children: ChildUser[]
}

function formatDate(iso: string) {
  if (iso.length >= 10) {
    const d = new Date(iso.slice(0, 10) + 'T12:00:00')
    return d.toLocaleDateString('de-DE', { weekday: 'short', day: '2-digit', month: '2-digit', year: 'numeric' })
  }
  return iso
}

function gameTitle(game: GameCarpoolData['game'], teamShort?: string): string {
  const team = teamShort ?? game.team
  if (game.eventType === 'generisch') return `${team} - ${game.opponent}`
  if (game.eventType === 'heim') return `${team} - Team Stuttgart vs ${game.opponent}`
  return `${team} - ${game.opponent} vs Team Stuttgart`
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
  onOpenQuickPair: (side: 'biete' | 'suche', counterpartId: number) => void
  onConfirm: (paarungId: number) => void
  onReject: (paarungId: number) => void
}

function EntryCard({ entry, typ, paarungen, myBieteIds, mySucheIds, onDelete, onRequest, onOpenQuickPair, onConfirm, onReject }: EntryCardProps) {
  const free = typ === 'biete' ? freePlaetze(entry, paarungen) : null

  // Paarungen that involve this entry
  const entryPaarungen = paarungen.filter(p =>
    typ === 'biete' ? p.bieteId === entry.id : p.sucheId === entry.id
  )

  // Can a sucher request this biete entry? Kein eigener Suche-Eintrag mehr nötig —
  // fehlt er, legt der einseitige Request ihn beim Bestätigen des Mini-Dialogs an.
  const canRequestAsBiete = typ === 'biete' && !entry.isOwn && free !== null && free > 0 &&
    !paarungen.some(p => p.bieteId === entry.id && (p.bieteIsOwn || p.sucheIsOwn) &&
      (p.status === 'pending' || p.status === 'confirmed'))

  // Can a bieter invite this suche entry? Kein eigener Biete-Eintrag mehr nötig.
  // Ein Gesuch darf nur eine aktive Paarung haben — deshalb reicht es, wenn
  // irgendeine aktive Paarung für dieses Gesuch existiert (egal von wem).
  const canInviteAsSuche = typ === 'suche' && !entry.isOwn &&
    !paarungen.some(p => p.sucheId === entry.id &&
      (p.status === 'pending' || p.status === 'confirmed'))

  return (
    <div id={`${typ}-${entry.id}`} className="py-2 border-b border-brand-border-subtle last:border-0 scroll-mt-24">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2 flex-wrap">
            <PersonChip userId={entry.userId} name={entry.userName} photoUrl={entry.photoUrl} />
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
                // Eigener Gesuch-Eintrag vorhanden → direkt paaren; sonst Mini-Dialog
                // (legt den Suche-Spiegel beim Bestätigen einseitig an).
                if (mySucheIds.length > 0) onRequest(entry.id, mySucheIds[0])
                else onOpenQuickPair('suche', entry.id)
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
              onClick={() => {
                // Eigener Biete-Eintrag vorhanden → direkt; sonst Mini-Dialog.
                if (myBieteIds.length > 0) onRequest(myBieteIds[0], entry.id)
                else onOpenQuickPair('biete', entry.id)
              }}
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
  initialBiete?: CarpoolEntry | null
  initialSuche?: CarpoolEntry | null
  vehicleSeats?: number | null
  children?: ChildUser[]
  onClose: () => void
  onSaved: () => void
}

function fieldsFromEntry(entry: CarpoolEntry | null | undefined, fallbackPlaetze: string) {
  return {
    plaetze: entry?.plaetze != null ? String(entry.plaetze) : fallbackPlaetze,
    treffpunkt: entry?.treffpunkt ?? '',
    notiz: entry?.notiz ?? '',
  }
}

function FormModal({ gameId, initialTyp, initialBiete, initialSuche, vehicleSeats, children, onClose, onSaved }: FormModalProps) {
  const startTyp = initialTyp ?? 'biete'
  const startEntry = startTyp === 'biete' ? initialBiete : initialSuche
  const startFields = fieldsFromEntry(startEntry, startTyp === 'biete' ? String(vehicleSeats ?? 1) : '1')

  const [typ, setTyp] = useState<'biete' | 'suche'>(startTyp)
  const [forUserId, setForUserId] = useState<number | null>(null)
  const [plaetze, setPlaetze] = useState(startFields.plaetze)
  const [treffpunkt, setTreffpunkt] = useState(startFields.treffpunkt)
  const [notiz, setNotiz] = useState(startFields.notiz)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const switchTyp = (next: 'biete' | 'suche') => {
    const entry = next === 'biete' ? initialBiete : initialSuche
    const fallback = next === 'biete' ? String(vehicleSeats ?? 1) : '1'
    const fields = fieldsFromEntry(entry, fallback)
    setTyp(next)
    setPlaetze(fields.plaetze)
    setTreffpunkt(fields.treffpunkt)
    setNotiz(fields.notiz)
  }

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
        ...(forUserId !== null ? { forUserId } : {}),
        plaetze: typ === 'suche' ? (parseInt(plaetze) || 1) : (plaetze ? parseInt(plaetze) : null),
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
              onClick={() => switchTyp('biete')}
              className={`flex-1 py-2.5 sm:py-2 text-sm font-medium rounded-md border transition-colors ${typ === 'biete' ? 'bg-brand-yellow text-brand-black border-brand-yellow' : 'border-brand-border text-brand-text-muted hover:border-brand-text'}`}
            >
              Ich biete Mitfahrt
            </button>
            <button
              type="button"
              onClick={() => switchTyp('suche')}
              className={`flex-1 py-2.5 sm:py-2 text-sm font-medium rounded-md border transition-colors ${typ === 'suche' ? 'bg-brand-yellow text-brand-black border-brand-yellow' : 'border-brand-border text-brand-text-muted hover:border-brand-text'}`}
            >
              Ich suche Mitfahrt
            </button>
          </div>

          {children && children.length > 0 && (
            <div>
              <label className="block text-sm font-medium text-brand-text mb-1">Für wen?</label>
              <select
                value={forUserId ?? ''}
                onChange={e => setForUserId(e.target.value === '' ? null : Number(e.target.value))}
                className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
              >
                <option value="">Ich selbst</option>
                {children.map(c => (
                  <option key={c.userId} value={c.userId}>{c.name}</option>
                ))}
              </select>
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-brand-text mb-1">
              {typ === 'biete' ? 'Freie Plätze' : 'Anzahl Personen'}
              {typ === 'suche' && <span className="text-brand-danger ml-1">*</span>}
            </label>
            <NumberSpinner
              value={parseInt(plaetze) || 1}
              min={1}
              max={8}
              onChange={v => setPlaetze(String(v))}
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

interface QuickPairModalProps {
  side: 'biete' | 'suche'
  counterpartId: number
  children?: ChildUser[]
  vehicleSeats?: number | null
  onClose: () => void
  onSaved: () => void
}

// QuickPairModal: schlanker Dialog für die One-Click-Paarung ohne vorhandenen
// eigenen Eintrag. Fragt nur Plätze ab (auf der Mitfahren-Seite zusätzlich für
// wen) und postet den einseitigen Paarungs-Request; der Spiegel-Eintrag entsteht
// atomar im Backend.
function QuickPairModal({ side, counterpartId, children, vehicleSeats, onClose, onSaved }: QuickPairModalProps) {
  const isRide = side === 'suche'
  const [plaetze, setPlaetze] = useState<number>(isRide ? 1 : (vehicleSeats ?? 1))
  const [forUserId, setForUserId] = useState<number | null>(null)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setError(null)
    try {
      const body = isRide
        ? { bieteId: counterpartId, plaetze, ...(forUserId != null ? { forUserId } : {}) }
        : { sucheId: counterpartId, plaetze }
      await api.post('/mitfahrt-paarungen', body)
      onSaved()
      onClose()
    } catch (err: unknown) {
      const status = (err as { response?: { status: number } })?.response?.status
      if (status === 409) setError('Keine freien Plätze mehr oder bereits eine Anfrage vorhanden.')
      else if (status === 403) setError('Dazu fehlt dir die Berechtigung.')
      else setError('Fehler. Bitte erneut versuchen.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-4 bg-black/40">
      <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-md">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-base font-semibold text-brand-text">{isRide ? 'Mitfahren' : 'Platz anbieten'}</h2>
          <button onClick={onClose} aria-label="Schließen" className="text-brand-text-muted hover:text-brand-text transition-colors">
            <X className="w-5 h-5" />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="space-y-4">
          {isRide && children && children.length > 0 && (
            <div>
              <label className="block text-sm font-medium text-brand-text mb-1">Für wen?</label>
              <select
                value={forUserId ?? ''}
                onChange={e => setForUserId(e.target.value === '' ? null : Number(e.target.value))}
                className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
              >
                <option value="">Ich selbst</option>
                {children.map(c => (
                  <option key={c.userId} value={c.userId}>{c.name}</option>
                ))}
              </select>
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-brand-text mb-1">
              {isRide ? 'Anzahl Personen' : 'Freie Plätze'}
            </label>
            <NumberSpinner value={plaetze} min={1} max={8} onChange={setPlaetze} />
          </div>

          {error && (
            <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{error}</p>
          )}

          <button
            type="submit"
            disabled={saving}
            className="w-full bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {saving ? 'Senden…' : (isRide ? 'Mitfahrt anfragen' : 'Platz anbieten')}
          </button>
        </form>
      </div>
    </div>
  )
}

interface GameCardProps {
  data: GameCarpoolData
  teamShortNames: Map<number, string>
  focusTab?: 'biete' | 'suche'
  onDelete: (id: number) => void
  onOpenForm: (gameId: number, typ: 'biete' | 'suche') => void
  onRequest: (bieteId: number, sucheId: number) => void
  onOpenQuickPair: (side: 'biete' | 'suche', counterpartId: number) => void
  onConfirm: (paarungId: number) => void
  onReject: (paarungId: number) => void
}

function GameCard({ data, teamShortNames, focusTab, onDelete, onOpenForm, onRequest, onOpenQuickPair, onConfirm, onReject }: GameCardProps) {
  const [activeTab, setActiveTab] = useState<'biete' | 'suche'>(focusTab ?? 'biete')
  useEffect(() => { if (focusTab) setActiveTab(focusTab) }, [focusTab])
  const teamIds = data.game.teamIds ?? []
  const shorts = teamIds.map(id => teamShortNames.get(id)).filter((s): s is string => !!s).sort()
  const teamShort = shorts.length > 0 ? shorts.join(' / ') : undefined
  const hasOwnBiete = (data.biete ?? []).some(e => e.isOwn)
  const hasOwn = hasOwnBiete || (data.suche ?? []).some(e => e.isOwn)

  const myBieteIds = (data.biete ?? []).filter(e => e.isOwn).map(e => e.id)
  const mySucheIds = (data.suche ?? []).filter(e => e.isOwn).map(e => e.id)

  const confirmedPaarungen = (data.paarungen ?? []).filter(p => p.status === 'confirmed')

  const entryCardProps = { paarungen: data.paarungen, myBieteIds, mySucheIds, onDelete, onRequest, onOpenQuickPair, onConfirm, onReject }
  const colors = getEventColors(data.game.eventType)
  const Icon = data.game.eventType === 'heim' ? Home : data.game.eventType === 'auswärts' ? Plane : Calendar

  return (
    <div id={`game-${data.game.id}`} className={`rounded-xl shadow border-t-4 overflow-hidden scroll-mt-24 ${colors.card.bg} ${colors.card.border}`}>
      <div className="px-4 py-3 border-b border-brand-border-subtle">
        <div className="flex items-start justify-between gap-2">
          <div className="flex items-start gap-3 min-w-0">
            <Icon className={`w-5 h-5 mt-0.5 shrink-0 ${colors.card.icon}`} />
            <div>
              <p className="text-xs text-brand-text-muted">{formatDate(data.game.date)}</p>
              <h2 className="text-sm font-semibold text-brand-text">{gameTitle(data.game, teamShort)}</h2>
            </div>
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
            Fahrangebote ({(data.biete ?? []).length})
          </button>
          <button
            onClick={() => setActiveTab('suche')}
            className={`flex-1 py-2 text-sm font-medium transition-colors ${activeTab === 'suche' ? 'text-brand-text border-b-2 border-brand-yellow' : 'text-brand-text-muted'}`}
          >
            Mitfahrgesuche ({(data.suche ?? []).length})
          </button>
        </div>
        <div className="px-4 py-2">
          {activeTab === 'biete'
            ? (data.biete ?? []).length === 0
              ? <p className="text-sm text-brand-text-muted py-2">Noch keine Fahrangebote.</p>
              : (data.biete ?? []).map(e => <EntryCard key={e.id} entry={e} typ="biete" {...entryCardProps} />)
            : (data.suche ?? []).length === 0
              ? <p className="text-sm text-brand-text-muted py-2">Noch keine Mitfahrgesuche.</p>
              : (data.suche ?? []).map(e => <EntryCard key={e.id} entry={e} typ="suche" {...entryCardProps} />)
          }
        </div>
      </div>

      {/* Desktop: two columns */}
      <div className="hidden sm:grid grid-cols-2 divide-x divide-brand-border-subtle">
        <div className="px-4 py-3">
          <p className="text-xs font-semibold uppercase tracking-wider text-brand-text-muted mb-2">Fahrangebote ({(data.biete ?? []).length})</p>
          {(data.biete ?? []).length === 0
            ? <p className="text-sm text-brand-text-muted">Noch keine Angebote.</p>
            : (data.biete ?? []).map(e => <EntryCard key={e.id} entry={e} typ="biete" {...entryCardProps} />)
          }
        </div>
        <div className="px-4 py-3">
          <p className="text-xs font-semibold uppercase tracking-wider text-brand-text-muted mb-2">Mitfahrgesuche ({(data.suche ?? []).length})</p>
          {(data.suche ?? []).length === 0
            ? <p className="text-sm text-brand-text-muted">Noch keine Gesuche.</p>
            : (data.suche ?? []).map(e => <EntryCard key={e.id} entry={e} typ="suche" {...entryCardProps} />)
          }
        </div>
      </div>

      {/* Confirmed pairings — visible to all */}
      {confirmedPaarungen.length > 0 && (
        <div className="px-4 py-3 border-t border-brand-border-subtle">
          <p className="text-xs font-semibold uppercase tracking-wider text-brand-text-muted mb-2">Fahrgemeinschaften</p>
          <div className="space-y-1">
            {confirmedPaarungen.map(p => (
              <div key={p.id} id={`paarung-${p.id}`} className="flex items-center gap-2 text-xs text-brand-text scroll-mt-24">
                <Check className="w-3 h-3 text-green-600 flex-shrink-0" />
                <span className="flex items-center gap-1 flex-wrap">
                  <PersonChip userId={p.sucheUserId} name={p.sucheName} photoUrl={p.suchePhotoUrl} />
                  {p.anzahl > 1 && ` (${p.anzahl} Personen)`}
                  {' '}fährt mit <PersonChip userId={p.bieteUserId} name={p.bieteName} photoUrl={p.bietePhotoUrl} />
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

const EVENT_TYPES = ['heim', 'auswärts', 'generisch'] as const
type EventTypeFilter = typeof EVENT_TYPES[number]
const ALL_TYPES = new Set<string>(EVENT_TYPES)

function parseFilters(sp: URLSearchParams) {
  const team = parseInt(sp.get('team') ?? '') || null
  const typesRaw = sp.get('types')
  const types = typesRaw
    ? (() => {
        const parsed = new Set(typesRaw.split(',').filter(t => ALL_TYPES.has(t)))
        return parsed.size > 0 ? parsed : new Set(ALL_TYPES)
      })()
    : new Set<string>(ALL_TYPES)
  const mine = sp.get('mine') === '1'
  return { team, types, mine }
}

export default function MitfahrgelegenheitenPage() {
  const { user } = useAuth()
  const [response, setResponse] = useState<ListResponse>({ games: [], children: [] })
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [modal, setModal] = useState<{ gameId: number; typ: 'biete' | 'suche' } | null>(null)
  const [quickPair, setQuickPair] = useState<{ side: 'biete' | 'suche'; counterpartId: number } | null>(null)
  const [allTeams, setAllTeams] = useState<TeamForName[]>([])
  const [searchParams, setSearchParams] = useSearchParams()
  const { team: filterTeamId, types: filterTypes, mine: viewMine } = parseFilters(searchParams)
  const location = useLocation()
  const compact = useCompactHeader(950)

  const focus = useMemo(() => {
    const m = /^#(paarung|biete|suche)-(\d+)$/.exec(location.hash)
    if (!m) return null
    return { kind: m[1] as 'paarung' | 'biete' | 'suche', id: Number(m[2]) }
  }, [location.hash])
  const teamShortNames = useMemo(() => buildTeamShortNames(allTeams), [allTeams])

  void user // used to re-render when auth changes

  const updateFilter = (patch: { team?: number | null; types?: Set<string>; mine?: boolean }) => {
    const next = new URLSearchParams(searchParams)
    if ('team' in patch) {
      if (patch.team === null) next.delete('team')
      else next.set('team', String(patch.team))
    }
    if ('types' in patch && patch.types) {
      const isDefault = patch.types.size === ALL_TYPES.size && [...ALL_TYPES].every(t => patch.types!.has(t))
      if (isDefault) next.delete('types')
      else next.set('types', [...patch.types].join(','))
    }
    if ('mine' in patch) {
      if (patch.mine) next.set('mine', '1')
      else next.delete('mine')
    }
    setSearchParams(next, { replace: true })
  }

  const toggleType = (type: EventTypeFilter) => {
    const next = new Set(filterTypes)
    if (next.has(type)) next.delete(type); else next.add(type)
    updateFilter({ types: next })
  }

  const load = (silent = false, teamId?: number | null) => {
    if (!silent) setLoading(true)
    const tid = teamId !== undefined ? teamId : filterTeamId
    const url = tid != null ? `/mitfahrgelegenheiten?team_id=${tid}` : '/mitfahrgelegenheiten'
    api.get(url)
      .then(res => {
        setResponse({ games: res.data?.games ?? [], vehicleSeats: res.data?.vehicleSeats, children: res.data?.children ?? [] })
        setLoading(false)
      })
      .catch(() => { setError('Fehler beim Laden.'); setLoading(false) })
  }

  useEffect(() => {
    api.get('/teams').then(res => {
      const list = Array.isArray(res.data) ? res.data : (res.data?.teams ?? [])
      setAllTeams(list)
    }).catch(() => {})
  }, [])
  useEffect(() => {
    load()
  }, [filterTeamId]) // eslint-disable-line react-hooks/exhaustive-deps
  useLiveUpdates((event) => { if (event === 'mitfahrgelegenheiten') load(true) })

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

  const teamKey = (d: GameCarpoolData): string => {
    const ids = d.game.teamIds ?? []
    if (ids.length === 0) return d.game.team ?? ''
    const shorts = ids
      .map(id => teamShortNames.get(id))
      .filter((s): s is string => !!s)
      .sort()
    if (shorts.length === 0) return d.game.team ?? ''
    return shorts.join(',')
  }

  const sortKey = (d: GameCarpoolData): string => {
    const date = (d.game.date ?? '').slice(0, 10)
    const time = d.game.time ?? '00:00'
    return `${date}T${time}|${teamKey(d)}`
  }

  const childIdSet = useMemo(() => new Set((response.children ?? []).map(c => c.userId)), [response.children])

  const mineMatches = (d: GameCarpoolData): boolean =>
    [...(d.biete ?? []), ...(d.suche ?? [])].some(e => e.isOwn || childIdSet.has(e.userId)) ||
    (d.paarungen ?? []).some(p => p.bieteIsOwn || p.sucheIsOwn || childIdSet.has(p.bieteUserId) || childIdSet.has(p.sucheUserId))

  const visibleGames = response.games
    .filter(d => filterTypes.has(d.game.eventType))
    .filter(d => !viewMine || mineMatches(d))
    .sort((a, b) => sortKey(a).localeCompare(sortKey(b)))

  const focusGameId = useMemo(() => {
    if (!focus) return null
    const game = response.games.find(g =>
      focus.kind === 'paarung' ? (g.paarungen ?? []).some(p => p.id === focus.id)
      : focus.kind === 'biete' ? (g.biete ?? []).some(e => e.id === focus.id)
      : (g.suche ?? []).some(e => e.id === focus.id)
    )
    return game?.game.id ?? null
  }, [focus, response.games])

  useEffect(() => {
    if (!focus || loading) return
    const target = document.getElementById(`${focus.kind}-${focus.id}`)
    if (!target) return
    target.scrollIntoView({ behavior: 'smooth', block: 'center' })
    target.classList.add('ring-2', 'ring-brand-yellow', 'rounded-md')
    const timer = setTimeout(() => {
      target.classList.remove('ring-2', 'ring-brand-yellow', 'rounded-md')
    }, 2200)
    return () => clearTimeout(timer)
  }, [focus, loading, focusGameId])

  const TYPE_PILLS: EventTypeFilterEntry[] = [
    ['heim',      'Heim',      <Home className="w-3.5 h-3.5" />],
    ['auswärts',  'Auswärts',  <Plane className="w-3.5 h-3.5" />],
    ['generisch', 'Sonstiges', <Calendar className="w-3.5 h-3.5" />],
  ]

  return (
    <div>
      <div className="flex items-center gap-2 mb-6 flex-wrap">
        <h1 className="text-2xl font-bold text-brand-text shrink-0">Mitfahrten</h1>
        <div className="flex items-center gap-1.5 flex-1 flex-nowrap min-w-0">
          {allTeams.length > 1 && (
            <select
              value={filterTeamId ?? ''}
              onChange={e => updateFilter({ team: e.target.value === '' ? null : Number(e.target.value) })}
              className="border border-brand-border rounded-md px-2 py-1.5 text-xs text-brand-text bg-white focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow w-24 shrink-0"
            >
              <option value="">Teams</option>
              {allTeams.map(t => (
                <option key={t.id} value={t.id}>{teamShortNames.get(t.id) ?? `Team ${t.id}`}</option>
              ))}
            </select>
          )}
          <EventTypeFilter
            types={TYPE_PILLS}
            active={filterTypes}
            onToggle={t => toggleType(t as EventTypeFilter)}
            compact={compact}
            ariaLabel="Mitfahrten-Typ-Filter"
          />

          <button
            onClick={() => updateFilter({ mine: !viewMine })}
            aria-label="Meine"
            className={`flex items-center gap-1 rounded-md py-1.5 text-xs font-medium border transition-colors shrink-0 ${compact ? 'px-2' : 'px-3'} ${
              viewMine
                ? 'bg-brand-yellow text-brand-black border-brand-yellow'
                : 'bg-white text-brand-text-muted border-brand-border hover:border-brand-text hover:text-brand-text'
            }`}
          >
            <UserCheck className="w-3.5 h-3.5" />
            {!compact && <span>Meine</span>}
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
        visibleGames.length === 0 ? (
          <p className="text-sm text-brand-text-muted">
            {viewMine
              ? 'Du bist bei keinem Spiel eingetragen.'
              : filterTypes.size === 0
              ? 'Keine Event-Typen ausgewählt.'
              : 'Keine Spiele deines Teams geplant.'}
          </p>
        ) : (
          <div className="space-y-4">
            {visibleGames.map(d => (
              <GameCard
                key={d.game.id}
                data={d}
                teamShortNames={teamShortNames}
                focusTab={focus && d.game.id === focusGameId && (focus.kind === 'biete' || focus.kind === 'suche') ? focus.kind : undefined}
                onDelete={handleDelete}
                onOpenForm={(gameId, typ) => setModal({ gameId, typ })}
                onRequest={handleRequest}
                onOpenQuickPair={(side, counterpartId) => setQuickPair({ side, counterpartId })}
                onConfirm={handleConfirm}
                onReject={handleReject}
              />
            ))}
          </div>
        )
      )}

      {modal && (() => {
        const gameData = response.games.find(g => g.game.id === modal.gameId)
        const ownBiete = gameData?.biete.find(e => e.isOwn) ?? null
        const ownSuche = gameData?.suche.find(e => e.isOwn) ?? null
        return (
          <FormModal
            gameId={modal.gameId}
            initialTyp={modal.typ}
            initialBiete={ownBiete}
            initialSuche={ownSuche}
            vehicleSeats={response.vehicleSeats}
            children={response.children}
            onClose={() => setModal(null)}
            onSaved={load}
          />
        )
      })()}

      {quickPair && (
        <QuickPairModal
          side={quickPair.side}
          counterpartId={quickPair.counterpartId}
          children={response.children}
          vehicleSeats={response.vehicleSeats}
          onClose={() => setQuickPair(null)}
          onSaved={load}
        />
      )}
    </div>
  )
}
