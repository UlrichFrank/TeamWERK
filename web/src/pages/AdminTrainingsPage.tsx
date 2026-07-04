import { useEffect, useMemo, useState } from 'react'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import { useNavigate } from 'react-router-dom'
import { Plus, Trash2, Edit2, ChevronDown, ChevronRight, Dumbbell, AlertTriangle, X } from 'lucide-react'
import { api } from '../lib/api'
import { buildTeamShortNames } from '../lib/teamName'
import VenuePicker from '../components/VenuePicker'
import MapsLink from '../components/MapsLink'
import RsvpDefaultsEditor, { type RsvpDefault } from '../components/RsvpDefaultsEditor'
import { errorMessage } from '../lib/errors'

const WEEKDAY_LABELS = ['Montag', 'Dienstag', 'Mittwoch', 'Donnerstag', 'Freitag', 'Samstag', 'Sonntag']
const WEEKDAY_SHORT = ['Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa', 'So']

const INPUT = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'

interface VenueRef {
  id: number
  name: string
  street: string
  city: string
  postal_code: string
  note: string
}

interface Series {
  id: number
  team_id: number
  season_id: number
  name: string
  venue?: VenueRef | null
  day_of_week: number
  start_time: string
  end_time: string
  valid_from: string
  valid_until: string
  note: string
  team_name: string
  session_count: number
  rsvp_default_players: RsvpDefault
  rsvp_default_extended: RsvpDefault
  rsvp_require_reason: number
}

interface Team { id: number; name: string; age_class: string; gender: string; team_number: number; group_count: number }
interface Season { id: number; name: string; is_active: boolean }
interface StandaloneSession {
  id: number
  team_id: number
  date: string
  start_time: string
  end_time: string
  venue?: VenueRef | null
  note: string
  status: string
  cancel_reason: string
  rsvp_default_players?: RsvpDefault
  rsvp_default_extended?: RsvpDefault
  rsvp_require_reason?: number
}

type SeriesModal = {
  id?: number
  team_id: number
  season_id: number
  name: string
  venue_id: number | null
  day_of_week: number
  start_time: string
  end_time: string
  valid_from: string
  valid_until: string
  note: string
  rsvp_default_players: RsvpDefault
  rsvp_default_extended: RsvpDefault
  rsvp_require_reason: number
}

type SessionModal = {
  id?: number
  team_id: number
  season_id: number
  date: string
  start_time: string
  end_time: string
  venue_id: number | null
  note: string
  status: string
  cancel_reason: string
  rsvp_default_players: RsvpDefault
  rsvp_default_extended: RsvpDefault
  rsvp_require_reason: number
}

function fmtDate(iso: string) {
  const d = new Date(iso.slice(0, 10) + 'T12:00:00')
  return `${WEEKDAY_SHORT[d.getDay() === 0 ? 6 : d.getDay() - 1]} ${String(d.getDate()).padStart(2,'0')}.${String(d.getMonth()+1).padStart(2,'0')}.${d.getFullYear()}`
}

export default function AdminTrainingsPage() {
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState<'serien' | 'einzeltermine'>('serien')
  const [series, setSeries] = useState<Series[]>([])
  const [standalone, setStandalone] = useState<StandaloneSession[]>([])
  const [teams, setTeams] = useState<Team[]>([])
  const [seasons, setSeasons] = useState<Season[]>([])
  const teamShortNames = useMemo(() => buildTeamShortNames(teams), [teams])
  const [expandedSeries, setExpandedSeries] = useState<Set<number>>(new Set())

  const [seriesModal, setSeriesModal] = useState<SeriesModal | null>(null)
  const [editScope, setEditScope] = useState<'all' | 'this_and_following'>('this_and_following')
  const [editFromDate, setEditFromDate] = useState('')

  const [sessionModal, setSessionModal] = useState<SessionModal | null>(null)

  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const [deleteConfirm, setDeleteConfirm] = useState<{ type: 'series' | 'session'; id: number } | null>(null)
  const [deleteScope, setDeleteScope] = useState<'future' | 'all'>('future')

  const activeSeasonId = seasons.find(s => s.is_active)?.id ?? 0
  const isNewSeries = seriesModal !== null && seriesModal.id === undefined
  const isNewSession = sessionModal !== null && sessionModal.id === undefined

  const loadSeries = () => api.get('/training-series').then(r => setSeries(r.data ?? []))
  const loadStandalone = () => {
    const from = new Date(Date.now() - 90 * 24 * 60 * 60 * 1000).toISOString().slice(0, 10)
    const to = new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toISOString().slice(0, 10)
    // Serverseitiger Filter (exclude_series=1) statt Client-filter(series_id===null);
    // Antwort ist {items,total}. Limit großzügig, da hier alle Einzeltermine gebraucht werden.
    api.get(`/training-sessions?from=${from}&to=${to}&exclude_series=1&limit=500`).then(r => {
      const items = Array.isArray(r.data?.items) ? r.data.items : (Array.isArray(r.data) ? r.data : [])
      setStandalone(items)
    })
  }

  useEffect(() => {
    Promise.all([api.get('/teams'), api.get('/seasons')]).then(([t, s]) => {
      setTeams(t.data ?? [])
      setSeasons(s.data ?? [])
    })
    loadSeries()
    loadStandalone()
  }, [])

  useLiveUpdates(event => { if (event === 'trainings') { loadSeries(); loadStandalone() } })

  const openNewSeries = () => {
    setError('')
    setEditScope('this_and_following')
    setEditFromDate('')
    setSeriesModal({ team_id: 0, season_id: activeSeasonId, name: '', venue_id: null, day_of_week: 0, start_time: '18:00', end_time: '19:30', valid_from: '', valid_until: '', note: '', rsvp_default_players: 'none', rsvp_default_extended: 'none', rsvp_require_reason: 1 })
  }

  const openEditSeries = (s: Series) => {
    setError('')
    setEditScope('this_and_following')
    setEditFromDate('')
    setSeriesModal({ id: s.id, team_id: s.team_id, season_id: s.season_id, name: s.name, venue_id: s.venue?.id ?? null, day_of_week: s.day_of_week, start_time: s.start_time, end_time: s.end_time, valid_from: s.valid_from.slice(0, 10), valid_until: s.valid_until.slice(0, 10), note: s.note, rsvp_default_players: s.rsvp_default_players ?? 'none', rsvp_default_extended: s.rsvp_default_extended ?? 'none', rsvp_require_reason: s.rsvp_require_reason ?? 1 })
  }

  const openNewSession = () => {
    setError('')
    setSessionModal({ team_id: 0, season_id: activeSeasonId, date: '', start_time: '18:00', end_time: '19:30', venue_id: null, note: '', status: 'active', cancel_reason: '', rsvp_default_players: 'none', rsvp_default_extended: 'none', rsvp_require_reason: 1 })
  }

  const openEditSession = (s: StandaloneSession) => {
    setError('')
    setSessionModal({ id: s.id, team_id: s.team_id, season_id: 0, date: s.date.slice(0, 10), start_time: s.start_time, end_time: s.end_time, venue_id: s.venue?.id ?? null, note: s.note, status: s.status, cancel_reason: s.cancel_reason, rsvp_default_players: s.rsvp_default_players ?? 'none', rsvp_default_extended: s.rsvp_default_extended ?? 'none', rsvp_require_reason: s.rsvp_require_reason ?? 1 })
  }

  const handleSubmitSeries = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!seriesModal) return
    setError('')
    if (isNewSeries && !seriesModal.team_id) { setError('Bitte Team wählen.'); return }
    const seasonId = seriesModal.season_id || activeSeasonId
    if (isNewSeries && !seasonId) { setError('Keine aktive Saison vorhanden.'); return }
    setSaving(true)
    try {
      if (isNewSeries) {
        await api.post('/training-series', { ...seriesModal, season_id: seasonId })
      } else {
        await api.put(`/training-series/${seriesModal.id}`, {
          name: seriesModal.name,
          venue_id: seriesModal.venue_id,
          day_of_week: seriesModal.day_of_week,
          start_time: seriesModal.start_time,
          end_time: seriesModal.end_time,
          valid_from: seriesModal.valid_from,
          valid_until: seriesModal.valid_until,
          note: seriesModal.note,
          rsvp_default_players: seriesModal.rsvp_default_players,
          rsvp_default_extended: seriesModal.rsvp_default_extended,
          rsvp_require_reason: seriesModal.rsvp_require_reason,
          scope: editScope,
          from_date: editScope === 'this_and_following' ? editFromDate : undefined,
        })
      }
      setSeriesModal(null)
      loadSeries()
    } catch (e) {
      setError(errorMessage(e, 'Fehler beim Speichern.'))
    } finally {
      setSaving(false)
    }
  }

  const handleSubmitSession = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!sessionModal) return
    setError('')
    if (isNewSession && !sessionModal.team_id) { setError('Bitte Team wählen.'); return }
    const seasonId = sessionModal.season_id || activeSeasonId
    if (isNewSession && !seasonId) { setError('Keine aktive Saison vorhanden.'); return }
    setSaving(true)
    try {
      if (isNewSession) {
        await api.post('/training-sessions', { ...sessionModal, season_id: seasonId })
      } else {
        await api.put(`/training-sessions/${sessionModal.id}`, {
          date: sessionModal.date,
          start_time: sessionModal.start_time,
          end_time: sessionModal.end_time,
          venue_id: sessionModal.venue_id,
          note: sessionModal.note,
          status: sessionModal.status,
          cancel_reason: sessionModal.cancel_reason,
          rsvp_default_players: sessionModal.rsvp_default_players,
          rsvp_default_extended: sessionModal.rsvp_default_extended,
          rsvp_require_reason: sessionModal.rsvp_require_reason,
        })
      }
      setSessionModal(null)
      loadStandalone()
    } catch (e) {
      setError(errorMessage(e, 'Fehler beim Speichern.'))
    } finally {
      setSaving(false)
    }
  }

  const handleDeleteSeries = async (id: number) => {
    await api.delete(`/training-series/${id}?scope=${deleteScope === 'all' ? 'all' : 'future'}`)
    setDeleteConfirm(null)
    setDeleteScope('future')
    loadSeries()
  }

  const handleDeleteSession = async (id: number) => {
    await api.delete(`/training-sessions/${id}`)
    setDeleteConfirm(null)
    loadStandalone()
  }

  const toggleExpand = (id: number) => {
    setExpandedSeries(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id); else next.add(id)
      return next
    })
  }

  return (
    <div className="max-w-3xl">
      <h1 className="text-2xl font-bold text-brand-text mb-6">Trainings verwalten</h1>

      {/* Tabs */}
      <div className="flex gap-1 mb-6 bg-brand-surface-card rounded-lg p-1 border border-brand-border-subtle w-fit">
        {(['serien', 'einzeltermine'] as const).map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-sm font-medium rounded-md transition-colors ${
              activeTab === tab ? 'bg-brand-yellow text-brand-black' : 'text-brand-text-muted hover:text-brand-text'
            }`}
          >
            {tab === 'serien' ? 'Trainingsserien' : 'Einzeltermine'}
          </button>
        ))}
      </div>

      {error && !seriesModal && !sessionModal && (
        <div className="mb-4 p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger flex items-start gap-2">
          <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" />
          <span>{error}</span>
          <button onClick={() => setError('')} className="ml-auto"><X className="w-4 h-4" /></button>
        </div>
      )}

      {/* Delete confirmation modal */}
      {deleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50" onClick={() => { setDeleteConfirm(null); setDeleteScope('future') }}>
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-sm" onClick={e => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-4">
              <h2 className="font-semibold text-brand-text text-lg">
                {deleteConfirm.type === 'series' ? 'Serie löschen' : 'Einzeltermin löschen'}
              </h2>
              <button onClick={() => { setDeleteConfirm(null); setDeleteScope('future') }} className="p-1 text-brand-text-muted hover:text-brand-text rounded transition-colors">
                <X className="w-5 h-5" />
              </button>
            </div>

            {deleteConfirm.type === 'series' ? (
              <div className="space-y-2 mb-6">
                <p className="text-sm text-brand-text-muted mb-3">Welche Sessions sollen gelöscht werden?</p>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input type="radio" name="deleteScope" checked={deleteScope === 'future'} onChange={() => setDeleteScope('future')} />
                  <span className="text-sm text-brand-text">Nur zukünftige Sessions</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input type="radio" name="deleteScope" checked={deleteScope === 'all'} onChange={() => setDeleteScope('all')} />
                  <span className="text-sm text-brand-text">Alle Sessions (inkl. vergangene)</span>
                </label>
              </div>
            ) : (
              <p className="text-sm text-brand-text-muted mb-6">Dieser Einzeltermin wird unwiderruflich gelöscht.</p>
            )}

            <div className="flex gap-2 justify-end">
              <button
                onClick={() => { setDeleteConfirm(null); setDeleteScope('future') }}
                className="bg-white border border-brand-border text-brand-text rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-surface-card transition-colors"
              >
                Abbrechen
              </button>
              <button
                onClick={() => deleteConfirm.type === 'series' ? handleDeleteSeries(deleteConfirm.id) : handleDeleteSession(deleteConfirm.id)}
                className="bg-brand-danger text-white rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors"
              >
                Löschen
              </button>
            </div>
          </div>
        </div>
      )}

      {/* === SERIEN TAB === */}
      {activeTab === 'serien' && (
        <div className="space-y-4">
          <div className="flex justify-end">
            <button
              onClick={openNewSeries}
              className="flex items-center gap-2 bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
            >
              <Plus className="w-4 h-4" /> Neue Serie
            </button>
          </div>

          {series.length === 0 ? (
            <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-8 text-center">
              <Dumbbell className="w-10 h-10 mx-auto mb-3 text-brand-text-subtle" />
              <p className="text-brand-text-muted">Noch keine Trainingsserien angelegt.</p>
            </div>
          ) : (
            <div className="space-y-3">
              {series.map(s => (
                <div key={s.id} className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
                  <div className="p-4 cursor-pointer hover:bg-white/40 transition-colors" onClick={() => toggleExpand(s.id)}>
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex items-start gap-3 min-w-0">
                        {expandedSeries.has(s.id)
                          ? <ChevronDown className="w-4 h-4 mt-1 text-brand-text-muted shrink-0" />
                          : <ChevronRight className="w-4 h-4 mt-1 text-brand-text-muted shrink-0" />}
                        <div className="min-w-0">
                          <p className="font-semibold text-brand-text">{s.name}</p>
                          <p className="text-sm text-brand-text-muted">
                            {WEEKDAY_LABELS[s.day_of_week]} · {s.start_time}–{s.end_time}
                            {s.venue ? ` · ${s.venue.name}` : ''}
                          </p>
                          <p className="text-xs text-brand-text-subtle mt-0.5">
                            {s.team_name} · {s.session_count} Termine · bis {s.valid_until.slice(0, 10)}
                          </p>
                        </div>
                      </div>
                      <div className="flex gap-1 shrink-0" onClick={e => e.stopPropagation()}>
                        <button onClick={() => openEditSeries(s)} aria-label="Bearbeiten"
                          className="p-2 text-brand-text-muted hover:text-brand-text hover:bg-white rounded-lg transition-colors">
                          <Edit2 className="w-4 h-4" />
                        </button>
                        <button onClick={() => setDeleteConfirm({ type: 'series', id: s.id })} aria-label="Löschen"
                          className="p-2 text-brand-text-muted hover:text-brand-danger hover:bg-brand-danger-light rounded-lg transition-colors">
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* === EINZELTERMINE TAB === */}
      {activeTab === 'einzeltermine' && (
        <div className="space-y-4">
          <div className="flex justify-end">
            <button
              onClick={openNewSession}
              className="flex items-center gap-2 bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
            >
              <Plus className="w-4 h-4" /> Neuer Termin
            </button>
          </div>

          {standalone.length === 0 ? (
            <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-8 text-center">
              <Dumbbell className="w-10 h-10 mx-auto mb-3 text-brand-text-subtle" />
              <p className="text-brand-text-muted">Keine Einzeltermine vorhanden.</p>
            </div>
          ) : (
            <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
              <ul className="divide-y divide-brand-border-subtle">
                {standalone.map(s => (
                  <li key={s.id}
                    className="px-4 py-3 flex items-center justify-between gap-4 hover:bg-white/40 transition-colors cursor-pointer"
                    onClick={() => navigate(`/trainings/${s.id}`)}
                  >
                    <div className="min-w-0">
                      <p className="text-sm font-medium text-brand-text">{fmtDate(s.date)}</p>
                      <div className="flex items-center gap-2 text-xs text-brand-text-muted">
                        <span>{s.start_time}–{s.end_time}</span>
                        {s.venue && <MapsLink venue={s.venue} />}
                      </div>
                      {s.status === 'cancelled' && <span className="text-xs text-brand-danger">Abgesagt</span>}
                    </div>
                    <div className="flex gap-1 shrink-0" onClick={e => e.stopPropagation()}>
                      <button onClick={() => openEditSession(s)} aria-label="Bearbeiten"
                        className="p-2 text-brand-text-muted hover:text-brand-text hover:bg-white rounded-lg transition-colors">
                        <Edit2 className="w-4 h-4" />
                      </button>
                      <button onClick={() => setDeleteConfirm({ type: 'session', id: s.id })} aria-label="Löschen"
                        className="p-2 text-brand-text-muted hover:text-brand-danger hover:bg-brand-danger-light rounded-lg transition-colors">
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}

      {/* === SERIES MODAL === */}
      {seriesModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50" onClick={() => setSeriesModal(null)}>
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto" onClick={e => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-4">
              <h2 className="font-semibold text-brand-text text-lg">
                {isNewSeries ? 'Neue Trainingsserie' : 'Serie bearbeiten'}
              </h2>
              <button onClick={() => setSeriesModal(null)} className="p-1 text-brand-text-muted hover:text-brand-text rounded transition-colors">
                <X className="w-5 h-5" />
              </button>
            </div>

            {error && (
              <div className="mb-4 p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger flex items-center gap-2">
                <AlertTriangle className="w-4 h-4 shrink-0" />
                <span>{error}</span>
              </div>
            )}

            <form onSubmit={handleSubmitSeries} className="space-y-4">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-medium text-brand-text-muted mb-1">Bezeichnung</label>
                  <input required value={seriesModal.name}
                    onChange={e => setSeriesModal(f => f ? { ...f, name: e.target.value } : f)}
                    className={INPUT} placeholder="z.B. Dienstag-Training" />
                </div>
                <div>
                  <label className="block text-xs font-medium text-brand-text-muted mb-1">Team</label>
                  {isNewSeries ? (
                    <select required value={seriesModal.team_id}
                      onChange={e => setSeriesModal(f => f ? { ...f, team_id: Number(e.target.value) } : f)}
                      className={INPUT}>
                      <option value={0}>– Team wählen –</option>
                      {teams.map(t => <option key={t.id} value={t.id}>{teamShortNames.get(t.id) ?? t.name}</option>)}
                    </select>
                  ) : (
                    <p className="text-sm text-brand-text py-2">{teamShortNames.get(seriesModal.team_id) ?? teams.find(t => t.id === seriesModal.team_id)?.name ?? '–'}</p>
                  )}
                </div>
                <div>
                  <label className="block text-xs font-medium text-brand-text-muted mb-1">Wochentag</label>
                  <select value={seriesModal.day_of_week}
                    onChange={e => setSeriesModal(f => f ? { ...f, day_of_week: Number(e.target.value) } : f)}
                    className={INPUT}>
                    {WEEKDAY_LABELS.map((l, i) => <option key={i} value={i}>{l}</option>)}
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-brand-text-muted mb-1">Ort</label>
                  <VenuePicker value={seriesModal.venue_id} onChange={v => setSeriesModal(f => f ? { ...f, venue_id: v } : f)} />
                </div>
                <div>
                  <label className="block text-xs font-medium text-brand-text-muted mb-1">Beginn</label>
                  <input type="time" required value={seriesModal.start_time}
                    onChange={e => setSeriesModal(f => f ? { ...f, start_time: e.target.value } : f)}
                    className={INPUT} />
                </div>
                <div>
                  <label className="block text-xs font-medium text-brand-text-muted mb-1">Ende</label>
                  <input type="time" required value={seriesModal.end_time}
                    onChange={e => setSeriesModal(f => f ? { ...f, end_time: e.target.value } : f)}
                    className={INPUT} />
                </div>
                <div>
                  <label className="block text-xs font-medium text-brand-text-muted mb-1">Gültig ab</label>
                  <input type="date" required value={seriesModal.valid_from}
                    onChange={e => setSeriesModal(f => f ? { ...f, valid_from: e.target.value } : f)}
                    className={INPUT} />
                </div>
                <div>
                  <label className="block text-xs font-medium text-brand-text-muted mb-1">Gültig bis</label>
                  <input type="date" required value={seriesModal.valid_until}
                    onChange={e => setSeriesModal(f => f ? { ...f, valid_until: e.target.value } : f)}
                    className={INPUT} />
                </div>
                <div className="sm:col-span-2">
                  <label className="block text-xs font-medium text-brand-text-muted mb-1">Hinweis (optional)</label>
                  <input value={seriesModal.note}
                    onChange={e => setSeriesModal(f => f ? { ...f, note: e.target.value } : f)}
                    className={INPUT} placeholder="Hinweis für alle Termine dieser Serie" />
                </div>
                <div className="sm:col-span-2">
                  <RsvpDefaultsEditor
                    idPrefix="series"
                    defaultPlayers={seriesModal.rsvp_default_players}
                    defaultExtended={seriesModal.rsvp_default_extended}
                    requireReason={seriesModal.rsvp_require_reason === 1}
                    onChangePlayers={v => setSeriesModal(f => f ? { ...f, rsvp_default_players: v } : f)}
                    onChangeExtended={v => setSeriesModal(f => f ? { ...f, rsvp_default_extended: v } : f)}
                    onChangeRequireReason={v => setSeriesModal(f => f ? { ...f, rsvp_require_reason: v ? 1 : 0 } : f)}
                  />
                </div>
              </div>

              {!isNewSeries && (
                <div className="pt-3 border-t border-brand-border-subtle space-y-2">
                  <label className="block text-xs font-medium text-brand-text-muted">Welche Termine ändern?</label>
                  <div className="flex gap-4 text-sm">
                    <label className="flex items-center gap-2 cursor-pointer">
                      <input type="radio" checked={editScope === 'this_and_following'} onChange={() => setEditScope('this_and_following')} />
                      Ab Datum
                    </label>
                    <label className="flex items-center gap-2 cursor-pointer">
                      <input type="radio" checked={editScope === 'all'} onChange={() => setEditScope('all')} />
                      Alle
                    </label>
                  </div>
                  {editScope === 'this_and_following' && (
                    <input type="date" required value={editFromDate}
                      onChange={e => setEditFromDate(e.target.value)}
                      className={INPUT} />
                  )}
                </div>
              )}

              <div className="flex gap-2 pt-2">
                <button type="submit" disabled={saving}
                  className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40">
                  {saving ? 'Speichern…' : isNewSeries ? 'Serie anlegen' : 'Speichern'}
                </button>
                <button type="button" onClick={() => setSeriesModal(null)}
                  className="bg-white border border-brand-border text-brand-text rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-surface-card transition-colors">
                  Abbrechen
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* === SESSION MODAL === */}
      {sessionModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50" onClick={() => setSessionModal(null)}>
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto" onClick={e => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-4">
              <h2 className="font-semibold text-brand-text text-lg">
                {isNewSession ? 'Neuer Einzeltermin' : 'Termin bearbeiten'}
              </h2>
              <button onClick={() => setSessionModal(null)} className="p-1 text-brand-text-muted hover:text-brand-text rounded transition-colors">
                <X className="w-5 h-5" />
              </button>
            </div>

            {error && (
              <div className="mb-4 p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger flex items-center gap-2">
                <AlertTriangle className="w-4 h-4 shrink-0" />
                <span>{error}</span>
              </div>
            )}

            <form onSubmit={handleSubmitSession} className="space-y-4">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-medium text-brand-text-muted mb-1">Team</label>
                  {isNewSession ? (
                    <select required value={sessionModal.team_id}
                      onChange={e => setSessionModal(f => f ? { ...f, team_id: Number(e.target.value) } : f)}
                      className={INPUT}>
                      <option value={0}>– Team wählen –</option>
                      {teams.map(t => <option key={t.id} value={t.id}>{teamShortNames.get(t.id) ?? t.name}</option>)}
                    </select>
                  ) : (
                    <p className="text-sm text-brand-text py-2">{teamShortNames.get(sessionModal.team_id) ?? teams.find(t => t.id === sessionModal.team_id)?.name ?? '–'}</p>
                  )}
                </div>
                <div>
                  <label className="block text-xs font-medium text-brand-text-muted mb-1">Datum</label>
                  <input type="date" required value={sessionModal.date}
                    onChange={e => setSessionModal(f => f ? { ...f, date: e.target.value } : f)}
                    className={INPUT} />
                </div>
                <div>
                  <label className="block text-xs font-medium text-brand-text-muted mb-1">Beginn</label>
                  <input type="time" required value={sessionModal.start_time}
                    onChange={e => setSessionModal(f => f ? { ...f, start_time: e.target.value } : f)}
                    className={INPUT} />
                </div>
                <div>
                  <label className="block text-xs font-medium text-brand-text-muted mb-1">Ende</label>
                  <input type="time" required value={sessionModal.end_time}
                    onChange={e => setSessionModal(f => f ? { ...f, end_time: e.target.value } : f)}
                    className={INPUT} />
                </div>
                <div className="sm:col-span-2">
                  <label className="block text-xs font-medium text-brand-text-muted mb-1">Ort</label>
                  <VenuePicker value={sessionModal.venue_id} onChange={v => setSessionModal(f => f ? { ...f, venue_id: v } : f)} />
                </div>
                <div className="sm:col-span-2">
                  <label className="block text-xs font-medium text-brand-text-muted mb-1">Hinweis</label>
                  <input value={sessionModal.note}
                    onChange={e => setSessionModal(f => f ? { ...f, note: e.target.value } : f)}
                    className={INPUT} placeholder="Optionaler Hinweis" />
                </div>
                <div className="sm:col-span-2">
                  <RsvpDefaultsEditor
                    idPrefix="session"
                    defaultPlayers={sessionModal.rsvp_default_players}
                    defaultExtended={sessionModal.rsvp_default_extended}
                    requireReason={sessionModal.rsvp_require_reason === 1}
                    onChangePlayers={v => setSessionModal(f => f ? { ...f, rsvp_default_players: v } : f)}
                    onChangeExtended={v => setSessionModal(f => f ? { ...f, rsvp_default_extended: v } : f)}
                    onChangeRequireReason={v => setSessionModal(f => f ? { ...f, rsvp_require_reason: v ? 1 : 0 } : f)}
                  />
                </div>
                {!isNewSession && (
                  <>
                    <div className="sm:col-span-2">
                      <label className="block text-xs font-medium text-brand-text-muted mb-1">Status</label>
                      <select value={sessionModal.status}
                        onChange={e => setSessionModal(f => f ? { ...f, status: e.target.value } : f)}
                        className={INPUT}>
                        <option value="active">Aktiv</option>
                        <option value="cancelled">Abgesagt</option>
                      </select>
                    </div>
                    {sessionModal.status === 'cancelled' && (
                      <div className="sm:col-span-2">
                        <label className="block text-xs font-medium text-brand-text-muted mb-1">Absagegrund (optional)</label>
                        <input value={sessionModal.cancel_reason}
                          onChange={e => setSessionModal(f => f ? { ...f, cancel_reason: e.target.value } : f)}
                          className={INPUT} placeholder="Grund für die Absage" />
                      </div>
                    )}
                  </>
                )}
              </div>
              <div className="flex gap-2 pt-2">
                <button type="submit" disabled={saving}
                  className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40">
                  {saving ? 'Speichern…' : isNewSession ? 'Termin anlegen' : 'Speichern'}
                </button>
                <button type="button" onClick={() => setSessionModal(null)}
                  className="bg-white border border-brand-border text-brand-text rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-surface-card transition-colors">
                  Abbrechen
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
