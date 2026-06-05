import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { api } from '../lib/api'

interface TrainerEntry { name: string; email: string }
interface PlayerEntry { name: string; jerseyNumber: number | null; status: string; email: string }
interface ParentEntry { name: string; email: string; children: string[] }

interface TeamRoster {
  team: { id: number; name: string }
  trainers: TrainerEntry[]
  players: PlayerEntry[]
  parents: ParentEntry[]
}

interface MyTeam { id: number; name: string }

const STATUS_LABELS: Record<string, string> = {
  aktiv: 'Aktiv',
  verletzt: 'Verletzt',
  pausiert: 'Pausiert',
  ausgetreten: 'Ausgetreten',
  passiv: 'Passiv',
}

function StatusBadge({ status }: { status: string }) {
  const cls = status === 'aktiv'
    ? 'bg-brand-success-light text-brand-success'
    : status === 'verletzt'
    ? 'bg-brand-danger-light text-brand-danger'
    : 'bg-brand-border-subtle text-brand-text-muted'
  return (
    <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${cls}`}>
      {STATUS_LABELS[status] ?? status}
    </span>
  )
}

function RosterSection({ roster }: { roster: TeamRoster }) {
  return (
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden mb-4">
      <div className="px-5 py-4 border-b border-brand-border-subtle">
        <h2 className="text-lg font-bold text-brand-text">{roster.team.name}</h2>
      </div>

      {/* Trainers */}
      {roster.trainers.length > 0 && (
        <div className="px-5 py-4 border-b border-brand-border-subtle">
          <h3 className="text-xs font-semibold uppercase tracking-wider text-brand-text-muted mb-3">Trainer</h3>
          <table className="w-full text-sm">
            <tbody>
              {roster.trainers.map((t, i) => (
                <tr key={i} className="border-b border-brand-border-subtle last:border-0">
                  <td className="py-2 pr-4 text-brand-text font-medium">{t.name}</td>
                  <td className="py-2 text-brand-text-muted">
                    {t.email && <a href={`mailto:${t.email}`} className="hover:text-brand-text transition-colors">{t.email}</a>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Players */}
      {roster.players.length > 0 && (
        <div className="px-5 py-4 border-b border-brand-border-subtle">
          <h3 className="text-xs font-semibold uppercase tracking-wider text-brand-text-muted mb-3">Spieler</h3>
          <div className="overflow-x-auto -mx-5 px-5">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left">
                  <th className="pb-2 pr-4 text-xs text-brand-text-muted font-medium">#</th>
                  <th className="pb-2 pr-4 text-xs text-brand-text-muted font-medium">Name</th>
                  <th className="pb-2 pr-4 text-xs text-brand-text-muted font-medium hidden sm:table-cell">Status</th>
                  <th className="pb-2 text-xs text-brand-text-muted font-medium hidden sm:table-cell">E-Mail</th>
                </tr>
              </thead>
              <tbody>
                {roster.players.map((p, i) => (
                  <tr key={i} className="border-t border-brand-border-subtle">
                    <td className="py-2 pr-4 text-brand-text-muted w-8">
                      {p.jerseyNumber != null ? p.jerseyNumber : '–'}
                    </td>
                    <td className="py-2 pr-4 text-brand-text font-medium">
                      {p.name}
                      <span className="sm:hidden ml-2">
                        <StatusBadge status={p.status} />
                      </span>
                    </td>
                    <td className="py-2 pr-4 hidden sm:table-cell">
                      <StatusBadge status={p.status} />
                    </td>
                    <td className="py-2 text-brand-text-muted hidden sm:table-cell">
                      {p.email && <a href={`mailto:${p.email}`} className="hover:text-brand-text transition-colors">{p.email}</a>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Parents */}
      {roster.parents.length > 0 && (
        <div className="px-5 py-4">
          <h3 className="text-xs font-semibold uppercase tracking-wider text-brand-text-muted mb-3">Eltern</h3>
          <table className="w-full text-sm">
            <tbody>
              {roster.parents.map((p, i) => (
                <tr key={i} className="border-b border-brand-border-subtle last:border-0">
                  <td className="py-2 pr-4">
                    <p className="text-brand-text font-medium">{p.name}</p>
                    {p.children.length > 0 && (
                      <p className="text-xs text-brand-text-muted">{p.children.join(', ')}</p>
                    )}
                  </td>
                  <td className="py-2 text-brand-text-muted">
                    {p.email && <a href={`mailto:${p.email}`} className="hover:text-brand-text transition-colors">{p.email}</a>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
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

  useEffect(() => {
    api.get('/teams/my')
      .then(res => {
        const teams: MyTeam[] = res.data ?? []
        setMyTeams(teams)

        // If a specific team is requested via ?team=id, only load that one
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
  }, [focusTeamId])

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
