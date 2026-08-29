import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../lib/api'
import TrainingDiaryStatsView from '../components/TrainingDiaryStatsView'
import { buildTeamShortNames } from '../lib/teamName'
import { HEADER_FIELD } from '../lib/buttonStyles'

interface TeamRef {
  id: number
  name: string
  age_class: string
  gender: string
  team_number: number
  group_count: number
  is_active: boolean
}

export default function TeamTrainingstagebuchPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const teamId = id ? Number(id) : null

  const [teams, setTeams] = useState<TeamRef[]>([])

  useEffect(() => {
    api
      .get<TeamRef[]>('/teams')
      .then(r => {
        const list = r.data ?? []
        setTeams(list)
        if (teamId == null && list.length > 0) {
          navigate(`/team/${list[0].id}/trainingstagebuch`, { replace: true })
        }
      })
      .catch(() => {})
    // Teams nur beim ersten Mount laden — Muster TeamAnwesenheitPage.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const shortNames = useMemo(() => buildTeamShortNames(teams), [teams])

  return (
    <div className="max-w-3xl space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-semibold text-brand-text">Trainingstagebuch</h1>
        <select
          value={teamId ?? ''}
          onChange={e => navigate(`/team/${e.target.value}/trainingstagebuch`)}
          aria-label="Mannschaft wählen"
          className={`${HEADER_FIELD} w-24 shrink-0`}
        >
          <option value="" disabled>
            Teams
          </option>
          {teams
            .filter(t => t.is_active)
            .map(t => (
              <option key={t.id} value={t.id}>
                {shortNames.get(t.id) ?? t.name}
              </option>
            ))}
        </select>
      </div>

      {teamId == null ? (
        <p className="text-sm text-brand-text-muted">Keine Mannschaft ausgewählt.</p>
      ) : (
        <TrainingDiaryStatsView teamId={teamId} />
      )}
    </div>
  )
}
