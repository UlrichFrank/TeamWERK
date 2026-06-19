import { useState, useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
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
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden mb-4">
      <div className="px-5 py-4 border-b border-brand-border-subtle">
        <h2 className="text-lg font-bold text-brand-text mb-3">{roster.team.display_long || roster.team.name}</h2>
        <div className="flex gap-1">
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
      </div>

      <div className="px-5 py-4">
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
          roster.parents.length === 0 ? (
            <p className="text-sm text-brand-text-muted">— keine Einträge —</p>
          ) : (
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
          )
        )}
      </div>
    </div>
  )
}

export default function MeinTeamPage() {
  const [searchParams] = useSearchParams()
  const focusTeamId = searchParams.get('team') ? Number(searchParams.get('team')) : null

  const [myTeams, setMyTeams] = useState<MyTeam[]>([])
  const [rosters, setRosters] = useState<TeamRoster[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const loadRosters = () => {
    api.get('/teams/my')
      .then(res => {
        const teams: MyTeam[] = res.data ?? []
        setMyTeams(teams)
        const toLoad = focusTeamId != null ? teams.filter(t => t.id === focusTeamId) : teams
        return Promise.all(toLoad.map(t => api.get(`/teams/${t.id}/roster`).then(r => r.data as TeamRoster)))
      })
      .then(rosterData => {
        setRosters(rosterData)
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }

  // loadRosters kapselt focusTeamId, soll nur bei dessen Änderung neu laufen
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { loadRosters() }, [focusTeamId])

  useLiveUpdates(event => { if (event === 'members' || event === 'kader') loadRosters() })

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

      {rosters.length === 0 ? (
        <p className="text-sm text-brand-text-muted">Kein Team zugeordnet.</p>
      ) : (
        rosters.map(r => <RosterSection key={r.team.id} roster={r} />)
      )}
    </div>
  )
}
