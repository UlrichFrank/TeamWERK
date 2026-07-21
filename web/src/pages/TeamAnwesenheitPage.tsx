import { useEffect, useMemo, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { AlertTriangle, Check, MinusCircle, X, ChevronRight, EyeOff, Eye } from 'lucide-react'
import { api } from '../lib/api'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import { buildTeamShortNames } from '../lib/teamName'

interface MemberCounts {
  member_id: number
  member_name: string
  training_present: number
  training_missed: number
  training_excused: number
  game_present: number
  game_missed: number
  game_excused: number
}

interface Averages {
  training_present: number
  training_missed: number
  training_excused: number
  game_present: number
  game_missed: number
  game_excused: number
}

interface TeamStats {
  team_id: number
  team_name: string
  season_id: number
  start_date: string
  end_date: string
  regular_members: MemberCounts[]
  extended_members: MemberCounts[]
  regular_averages: Averages
  extended_averages: Averages
}

interface OpenItem {
  event_type: 'training' | 'game'
  event_id: number
  date: string
  title: string
}

interface OpenResponse {
  open: OpenItem[]
  excluded: OpenItem[]
}

interface TeamRef {
  id: number
  name: string
  age_class: string
  gender: string
  team_number: number
  group_count: number
  is_active: boolean
}

function quote(present: number, missed: number): string {
  const denom = present + missed
  if (denom === 0) return '–'
  return `${Math.round((present / denom) * 100)}%`
}

function fmtDate(iso: string) {
  const d = iso.slice(0, 10).split('-')
  return d.length === 3 ? `${d[2]}.${d[1]}.${d[0]}` : iso
}

// Drei Säulen-Zähler mit Quote, kompakt nebeneinander.
function PillarCell({ present, excused, missed }: { present: number; excused: number; missed: number }) {
  return (
    <div className="flex items-center gap-2 text-sm text-brand-text">
      <span className="inline-flex items-center gap-0.5"><Check className="w-4 h-4 text-brand-green" />{present}</span>
      <span className="inline-flex items-center gap-0.5"><MinusCircle className="w-4 h-4 text-brand-yellow" />{excused}</span>
      <span className="inline-flex items-center gap-0.5"><X className="w-4 h-4 text-brand-danger" />{missed}</span>
      <span className="text-brand-text-muted">{quote(present, missed)}</span>
    </div>
  )
}

function StatTable({ title, members, averages, onMember }: {
  title: string
  members: MemberCounts[]
  averages: Averages
  onMember: (id: number) => void
}) {
  if (members.length === 0) return null
  return (
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
      <div className="px-6 py-4 border-b border-brand-border-subtle">
        <h2 className="font-semibold text-brand-text">{title}</h2>
      </div>

      {/* Desktop-Tabelle */}
      <table className="hidden sm:table w-full">
        <thead>
          <tr className="border-b border-brand-border-subtle">
            <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Spieler</th>
            <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Trainings</th>
            <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Spiele</th>
          </tr>
        </thead>
        <tbody>
          {members.map(m => (
            <tr
              key={m.member_id}
              onClick={() => onMember(m.member_id)}
              className="border-b border-brand-border-subtle last:border-0 hover:bg-brand-table-select transition-colors cursor-pointer"
            >
              <td className="px-4 py-3 text-sm font-medium text-brand-text">{m.member_name}</td>
              <td className="px-4 py-3"><PillarCell present={m.training_present} excused={m.training_excused} missed={m.training_missed} /></td>
              <td className="px-4 py-3"><PillarCell present={m.game_present} excused={m.game_excused} missed={m.game_missed} /></td>
            </tr>
          ))}
          <tr className="border-t-2 border-brand-border bg-brand-surface-card">
            <td className="px-4 py-3 text-sm font-semibold text-brand-text-muted uppercase tracking-wide">Ø Team</td>
            <td className="px-4 py-3"><PillarCell present={Math.round(averages.training_present * 10) / 10} excused={Math.round(averages.training_excused * 10) / 10} missed={Math.round(averages.training_missed * 10) / 10} /></td>
            <td className="px-4 py-3"><PillarCell present={Math.round(averages.game_present * 10) / 10} excused={Math.round(averages.game_excused * 10) / 10} missed={Math.round(averages.game_missed * 10) / 10} /></td>
          </tr>
        </tbody>
      </table>

      {/* Mobile-Cards */}
      <div className="sm:hidden divide-y divide-brand-border-subtle">
        {members.map(m => (
          <button
            key={m.member_id}
            onClick={() => onMember(m.member_id)}
            className="w-full text-left px-4 py-3 flex items-start justify-between gap-2 hover:bg-brand-table-select transition-colors"
          >
            <div className="min-w-0">
              <div className="font-medium text-brand-text mb-1">{m.member_name}</div>
              <div className="text-xs text-brand-text-muted mb-0.5">Trainings</div>
              <PillarCell present={m.training_present} excused={m.training_excused} missed={m.training_missed} />
              <div className="text-xs text-brand-text-muted mt-1 mb-0.5">Spiele</div>
              <PillarCell present={m.game_present} excused={m.game_excused} missed={m.game_missed} />
            </div>
            <ChevronRight className="w-5 h-5 text-brand-text-subtle shrink-0 mt-1" />
          </button>
        ))}
        <div className="px-4 py-3 bg-brand-surface-card">
          <div className="font-semibold text-brand-text-muted uppercase text-xs tracking-wide mb-1">Ø Team — Trainings</div>
          <PillarCell present={Math.round(averages.training_present * 10) / 10} excused={Math.round(averages.training_excused * 10) / 10} missed={Math.round(averages.training_missed * 10) / 10} />
          <div className="font-semibold text-brand-text-muted uppercase text-xs tracking-wide mt-1 mb-1">Ø Team — Spiele</div>
          <PillarCell present={Math.round(averages.game_present * 10) / 10} excused={Math.round(averages.game_excused * 10) / 10} missed={Math.round(averages.game_missed * 10) / 10} />
        </div>
      </div>
    </div>
  )
}

export default function TeamAnwesenheitPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const teamId = id ? Number(id) : null

  const [teams, setTeams] = useState<TeamRef[]>([])
  const [stats, setStats] = useState<TeamStats | null>(null)
  const [open, setOpen] = useState<OpenItem[]>([])
  const [excluded, setExcluded] = useState<OpenItem[]>([])
  const [showOpen, setShowOpen] = useState(false)
  const [showExcluded, setShowExcluded] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api.get('/teams').then(r => {
      const list: TeamRef[] = r.data ?? []
      setTeams(list)
      // Ohne explizit gewähltes Team direkt auf das erste verfügbare springen.
      if (teamId == null && list.length > 0) navigate(`/team/${list[0].id}/anwesenheit`, { replace: true })
    }).catch(() => {})
    // Nur beim ersten Mount Teams laden.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const load = (silent = false) => {
    if (teamId == null) return
    if (!silent) setLoading(true)
    Promise.all([
      api.get(`/teams/${teamId}/attendance-stats`),
      api.get(`/teams/${teamId}/attendance-open`),
    ])
      .then(([statsRes, openRes]) => {
        setStats(statsRes.data)
        const openData: OpenResponse = openRes.data ?? { open: [], excluded: [] }
        setOpen(openData.open ?? [])
        setExcluded(openData.excluded ?? [])
        setError(null)
      })
      .catch(() => setError('Statistik konnte nicht geladen werden.'))
      .finally(() => { if (!silent) setLoading(false) })
  }

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-/Abhängigkeits-getrieben), kein Ableitungs-Bug
    load()
    // load kapselt teamId, soll nur bei dessen Änderung neu laufen
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [teamId])

  useLiveUpdates((event) => { if (event === 'attendance-changed') load(true) })

  const openMember = (memberId: number) => navigate(`/profil/anwesenheit?member=${memberId}`)

  // Stats-Response liefert team_name autoritativ — fängt Teams ohne aktiven Kader ab,
  // die nicht in der /teams-Dropdown-Liste auftauchen.
  const teamName = stats?.team_name ?? teams.find(t => t.id === teamId)?.name
  const shortNames = useMemo(() => buildTeamShortNames(teams), [teams])

  return (
    <div className="max-w-3xl space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-bold text-brand-text">Anwesenheit{teamName ? ` — ${teamName}` : ''}</h1>
        {teams.filter(t => t.is_active).length > 1 && (
          <select
            value={teamId ?? ''}
            onChange={e => navigate(`/team/${e.target.value}/anwesenheit`)}
            className="border border-brand-border rounded-md px-2 py-1.5 text-xs text-brand-text bg-white focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow w-24 shrink-0"
          >
            <option value="" disabled>Teams</option>
            {teams.filter(t => t.is_active).map(t => (
              <option key={t.id} value={t.id}>{shortNames.get(t.id) ?? t.name}</option>
            ))}
          </select>
        )}
      </div>

      {teamId == null && (
        <p className="text-sm text-brand-text-muted">Kein Team ausgewählt.</p>
      )}

      {error && (
        <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{error}</div>
      )}

      {loading && teamId != null && <p className="text-brand-text-muted text-sm p-4">Laden…</p>}

      {!loading && stats && (
        <>
          {open.length > 0 && (
            <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
              <button
                onClick={() => setShowOpen(s => !s)}
                className="w-full flex items-center gap-2 px-6 py-4 text-left hover:bg-brand-table-select transition-colors"
              >
                <AlertTriangle className="w-5 h-5 text-brand-danger shrink-0" />
                <span className="flex-1 text-sm font-medium text-brand-text">
                  {open.length} offene {open.length === 1 ? 'Erfassung' : 'Erfassungen'}
                </span>
                <ChevronRight className={`w-5 h-5 text-brand-text-subtle transition-transform ${showOpen ? 'rotate-90' : ''}`} />
              </button>
              {showOpen && (
                <ul className="border-t border-brand-border-subtle divide-y divide-brand-border-subtle">
                  {open.map(it => (
                    <li key={`${it.event_type}-${it.event_id}`} className="flex items-center gap-2 px-4 py-2.5 hover:bg-brand-table-select transition-colors">
                      <button
                        onClick={() => navigate(`/termine/${it.event_type === 'training' ? 'training' : 'spiel'}/${it.event_id}`)}
                        className="flex-1 text-left flex items-center justify-between gap-2"
                      >
                        <span className="text-sm text-brand-text">
                          {fmtDate(it.date)} · {it.title}
                          <span className="text-brand-text-muted"> ({it.event_type === 'training' ? 'Training' : 'Spiel'})</span>
                        </span>
                        <ChevronRight className="w-4 h-4 text-brand-text-subtle shrink-0" />
                      </button>
                      <button
                        onClick={() => {
                          const url = it.event_type === 'training'
                            ? `/training-sessions/${it.event_id}/attendance-excluded`
                            : `/games/${it.event_id}/attendance-excluded`
                          api.post(url).catch(() => {})
                        }}
                        title="Aus Statistik ausschließen"
                        className="text-brand-text-muted hover:text-brand-danger transition-colors shrink-0"
                        aria-label="Aus Statistik ausschließen"
                      >
                        <EyeOff className="w-4 h-4" />
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )}

          {excluded.length > 0 && (
            <div className="bg-brand-surface-card rounded-xl shadow overflow-hidden border border-brand-border-subtle">
              <button
                onClick={() => setShowExcluded(s => !s)}
                className="w-full flex items-center gap-2 px-6 py-4 text-left hover:bg-brand-table-select transition-colors"
              >
                <EyeOff className="w-5 h-5 text-brand-text-muted shrink-0" />
                <span className="flex-1 text-sm text-brand-text-muted">
                  {excluded.length} ausgeschlossene {excluded.length === 1 ? 'Erfassung' : 'Erfassungen'}
                </span>
                <ChevronRight className={`w-5 h-5 text-brand-text-subtle transition-transform ${showExcluded ? 'rotate-90' : ''}`} />
              </button>
              {showExcluded && (
                <ul className="border-t border-brand-border-subtle divide-y divide-brand-border-subtle">
                  {excluded.map(it => (
                    <li key={`${it.event_type}-${it.event_id}`} className="flex items-center gap-2 px-4 py-2.5 hover:bg-brand-table-select transition-colors">
                      <span className="flex-1 text-sm text-brand-text-muted">
                        {fmtDate(it.date)} · {it.title}
                        <span className="text-brand-text-subtle"> ({it.event_type === 'training' ? 'Training' : 'Spiel'})</span>
                      </span>
                      <button
                        onClick={() => {
                          const url = it.event_type === 'training'
                            ? `/training-sessions/${it.event_id}/attendance-excluded`
                            : `/games/${it.event_id}/attendance-excluded`
                          api.delete(url).catch(() => {})
                        }}
                        title="Wieder einschließen"
                        className="text-brand-text-muted hover:text-brand-text transition-colors shrink-0"
                        aria-label="Wieder einschließen"
                      >
                        <Eye className="w-4 h-4" />
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )}

          <StatTable title="Stammkader" members={stats.regular_members} averages={stats.regular_averages} onMember={openMember} />
          {stats.extended_members.length > 0 && (
            <StatTable title={`Erweiterter Kader (${stats.extended_members.length})`} members={stats.extended_members} averages={stats.extended_averages} onMember={openMember} />
          )}
        </>
      )}
    </div>
  )
}
