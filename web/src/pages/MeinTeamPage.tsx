import { useState, useEffect, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { api } from '../lib/api'
import PersonChip from '../components/PersonChip'
import { useLiveUpdates } from '../hooks/useLiveUpdates'

interface TrainerEntry { userId: number; name: string }
interface PlayerEntry { userId: number; name: string; jerseyNumber: number | null }
interface ParentEntry { userId: number; name: string; children: string[] }

interface TeamRoster {
  team: { id: number; name: string; display_short?: string; display_long?: string }
  trainers: TrainerEntry[]
  players: PlayerEntry[]
  parents: ParentEntry[]
  extended_players: PlayerEntry[]
  extended_parents: ParentEntry[]
}

interface MyTeam { id: number; name: string }


type RosterTab = 'team' | 'trainer' | 'eltern'

const TABS: { id: RosterTab; label: string }[] = [
  { id: 'team', label: 'Team' },
  { id: 'trainer', label: 'Trainer' },
  { id: 'eltern', label: 'Eltern' },
]

function RosterSection({ roster }: { roster: TeamRoster }) {
  const [activeTab, setActiveTab] = useState<RosterTab>('team')

  return (
    <>
      <div className="flex gap-1 mb-3">
        {TABS.map(tab => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`px-3 py-1 rounded-md text-sm font-medium transition-colors ${
              activeTab === tab.id
                ? 'bg-brand-yellow text-brand-black'
                : 'text-brand-text-muted hover:text-brand-text'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {activeTab === 'team' && (
        <>
          {roster.players.length === 0 ? (
            <p className="text-sm text-brand-text-muted">— keine Einträge —</p>
          ) : (
            <div className="overflow-x-auto -mx-5 px-5">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left">
                    <th className="pb-2 pr-4 text-xs text-brand-text-muted font-medium">#</th>
                    <th className="pb-2 text-xs text-brand-text-muted font-medium">Name</th>
                  </tr>
                </thead>
                <tbody>
                  {roster.players.map((p, i) => (
                    <tr key={i} className="border-t border-brand-border-subtle">
                      <td className="py-2 pr-4 text-brand-text-muted w-8">
                        {p.jerseyNumber != null ? p.jerseyNumber : '–'}
                      </td>
                      <td className="py-2">
                        <PersonChip userId={p.userId || undefined} name={p.name} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          {roster.extended_players?.length > 0 && (
            <div className="mt-5">
              <p className="text-xs font-semibold text-brand-text-muted uppercase tracking-wide mb-2">
                Erweiterter Kader
              </p>
              <div className="overflow-x-auto -mx-5 px-5">
                <table className="w-full text-sm">
                  <tbody>
                    {roster.extended_players.map((p, i) => (
                      <tr key={i} className="border-t border-brand-border-subtle">
                        <td className="py-2 pr-4 text-brand-text-muted w-8">
                          {p.jerseyNumber != null ? p.jerseyNumber : '–'}
                        </td>
                        <td className="py-2">
                          <PersonChip userId={p.userId || undefined} name={p.name} />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </>
      )}

      {activeTab === 'trainer' && (
        roster.trainers.length === 0 ? (
          <p className="text-sm text-brand-text-muted">— keine Einträge —</p>
        ) : (
          <table className="w-full text-sm">
            <tbody>
              {roster.trainers.map((t, i) => (
                <tr key={i} className="border-b border-brand-border-subtle last:border-0">
                  <td className="py-2">
                    <PersonChip userId={t.userId || undefined} name={t.name} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )
      )}

      {activeTab === 'eltern' && (
        roster.parents.length === 0 && (roster.extended_parents?.length ?? 0) === 0 ? (
          <p className="text-sm text-brand-text-muted">— keine Einträge —</p>
        ) : (
          <>
            {roster.parents.length > 0 && (
              <table className="w-full text-sm">
                <tbody>
                  {roster.parents.map((p, i) => (
                    <tr key={i} className="border-b border-brand-border-subtle last:border-0">
                      <td className="py-2">
                        <PersonChip userId={p.userId || undefined} name={p.name} />
                        {p.children.length > 0 && (
                          <p className="text-xs text-brand-text-muted mt-0.5">{p.children.join(', ')}</p>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
            {roster.extended_parents?.length > 0 && (
              <div className="mt-5">
                <p className="text-xs font-semibold text-brand-text-muted uppercase tracking-wide mb-2">
                  Erweiterter Kader
                </p>
                <table className="w-full text-sm">
                  <tbody>
                    {roster.extended_parents.map((p, i) => (
                      <tr key={i} className="border-b border-brand-border-subtle last:border-0">
                        <td className="py-2">
                          <PersonChip userId={p.userId || undefined} name={p.name} />
                          {p.children.length > 0 && (
                            <p className="text-xs text-brand-text-muted mt-0.5">{p.children.join(', ')}</p>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </>
        )
      )}
    </>
  )
}

export default function MeinTeamPage() {
  const [searchParams] = useSearchParams()
  const focusTeamId = searchParams.get('team') ? Number(searchParams.get('team')) : null

  const [myTeams, setMyTeams] = useState<MyTeam[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // On-Demand-Rosters: erst beim Aufklappen/Fokus geladen, dann in der Session
  // behalten (kein Re-Fetch beim erneuten Aufklappen).
  const [rosters, setRosters] = useState<Record<number, TeamRoster>>({})
  const [expanded, setExpanded] = useState<Set<number>>(new Set())
  const [rosterErrors, setRosterErrors] = useState<Record<number, string>>({})

  const loadRoster = useCallback(async (teamId: number) => {
    try {
      const r = await api.get(`/teams/${teamId}/roster`)
      setRosters(prev => ({ ...prev, [teamId]: r.data as TeamRoster }))
    } catch (err) {
      setRosterErrors(prev => ({ ...prev, [teamId]: err instanceof Error ? err.message : 'Fehler beim Laden' }))
    }
  }, [])

  const toggleTeam = useCallback((teamId: number) => {
    setExpanded(prev => {
      const next = new Set(prev)
      if (next.has(teamId)) next.delete(teamId)
      else next.add(teamId)
      return next
    })
  }, [])

  // On-Demand-Laden: sobald ein Team aufgeklappt ist und sein Roster weder im
  // Session-Cache noch als Fehler vorliegt, wird es geladen. Bereits geladene
  // Rosters bleiben erhalten (kein Re-Fetch beim erneuten Aufklappen).
  useEffect(() => {
    for (const teamId of expanded) {
      if (!rosters[teamId] && !rosterErrors[teamId]) loadRoster(teamId)
    }
  }, [expanded, rosters, rosterErrors, loadRoster])

  const loadTeams = useCallback(() => {
    api.get('/teams/my')
      .then(res => {
        const teams: MyTeam[] = res.data ?? []
        setMyTeams(teams)
        setLoading(false)
        // Fokus-/Einzelteam wird automatisch aufgeklappt (→ Roster lädt on-demand
        // via Effekt); alle anderen bleiben eingeklappt und laden erst bei Fokus.
        const autoOpen = focusTeamId != null
          ? teams.filter(t => t.id === focusTeamId)
          : teams.length === 1 ? teams : []
        if (autoOpen.length > 0) setExpanded(new Set(autoOpen.map(t => t.id)))
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [focusTeamId])

  // Nur die Team-Liste eager laden; Rosters folgen on-demand.
  // focusTeamId/loadTeams stabil; bewusst nur bei Fokuswechsel neu laufen.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { loadTeams() }, [focusTeamId])

  // Bei Mitglieds-/Kader-Änderungen: bereits geladene Rosters aktualisieren.
  useLiveUpdates(event => {
    if (event === 'members' || event === 'kader') {
      for (const idStr of Object.keys(rosters)) loadRoster(Number(idStr))
    }
  })

  if (loading) {
    return (
      <div className="max-w-3xl mx-auto space-y-3">
        {[1, 2].map(i => <div key={i} className="h-32 bg-brand-border-subtle rounded-xl animate-pulse" />)}
      </div>
    )
  }

  if (error) {
    return (
      <div className="max-w-3xl mx-auto py-8 text-center">
        <p className="text-sm text-brand-text-muted">{error}</p>
      </div>
    )
  }

  return (
    <div className="max-w-3xl mx-auto">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-brand-text">Mein Team</h1>
        {myTeams.length > 1 && (
          <p className="text-sm text-brand-text-muted mt-0.5">{myTeams.length} Teams</p>
        )}
      </div>

      {myTeams.length === 0 ? (
        <p className="text-sm text-brand-text-muted">Kein Team zugeordnet.</p>
      ) : (
        <div className="space-y-4">
          {myTeams.map(team => {
            const isOpen = expanded.has(team.id)
            const roster = rosters[team.id]
            const rosterError = rosterErrors[team.id]
            return (
              <div key={team.id} className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
                <button
                  onClick={() => toggleTeam(team.id)}
                  aria-expanded={isOpen}
                  className="w-full flex items-center justify-between px-5 py-4 hover:bg-brand-border-subtle transition-colors min-h-[44px]"
                >
                  <h2 className="text-lg font-bold text-brand-text text-left">{roster?.team.display_long || team.name}</h2>
                  {isOpen
                    ? <ChevronDown className="w-5 h-5 text-brand-text-muted shrink-0" />
                    : <ChevronRight className="w-5 h-5 text-brand-text-muted shrink-0" />
                  }
                </button>
                {isOpen && (
                  <div className="px-5 py-4 border-t border-brand-border-subtle">
                    {rosterError ? (
                      <p className="text-sm text-brand-danger">{rosterError}</p>
                    ) : roster ? (
                      <RosterSection roster={roster} />
                    ) : (
                      <div className="h-20 bg-brand-border-subtle rounded-lg animate-pulse" />
                    )}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
