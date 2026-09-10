import { useCallback, useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { AlertTriangle } from 'lucide-react'
import { api } from '../lib/api'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import { useCompactHeader } from '../hooks/useCompactHeader'
import TeamFilter from '../components/TeamFilter'
import {
  effectiveTeamIds,
  parseTeamIds,
  serializeTeamIds,
  toggleTeamId,
  type TeamFilterOption,
} from '../lib/teamFilter'

// Siehe openspec/changes/dienste-familien-rangliste. GET /api/duty-fairness/rangliste
// liefert serverseitig bereits sortiert (rank 1..n, keine geteilten Plätze) und
// maskiert Namen für Standard-Nutzer — die Seite rendert nur, was ankommt.

interface RanglisteRow {
  rank: number
  memberId: number | null
  name: string | null
  isOwn: boolean
  geleistet: number
  vorhersage: number
}

interface RanglisteBlock {
  teamId: number
  teamLabel: string
  soll: number
  rows: RanglisteRow[]
}

interface RanglisteResponse {
  teams: TeamFilterOption[]
  blocks: RanglisteBlock[]
}

// de-DE, höchstens eine Nachkommastelle (Geschwister-/Kader-Aufteilung kann
// Bruchzahlen ergeben) — kein unnötiges ",0" bei ganzen Zahlen.
function formatDiensteZahl(n: number): string {
  return n.toLocaleString('de-DE', { maximumFractionDigits: 1 })
}

function RanglisteRowView({ row, scale, soll }: { row: RanglisteRow; scale: number; soll: number }) {
  const total = row.geleistet + row.vorhersage
  const geleistetPct = scale > 0 ? (row.geleistet / scale) * 100 : 0
  const vorhersagePct = scale > 0 ? (row.vorhersage / scale) * 100 : 0
  const sollPct = scale > 0 ? Math.min(100, (soll / scale) * 100) : 0
  // Anonymisierte Zeilen: nur ein Strich — die Platzierung steht schon in der
  // Rang-Spalte links, ein zweites „Platz N" wäre doppelt.
  const displayName = row.name ?? '-'

  return (
    <div className={`flex items-center gap-2 sm:gap-3 py-2 px-2 rounded-md ${row.isOwn ? 'bg-brand-yellow/20' : ''}`}>
      <span className="w-6 shrink-0 text-xs text-brand-text-muted text-right">{row.rank}.</span>
      <span
        className={`w-20 sm:w-36 shrink-0 truncate text-sm ${row.isOwn ? 'font-semibold text-brand-text' : 'text-brand-text'}`}
        title={displayName}
      >
        {displayName}
      </span>
      <div
        className="relative flex-1 h-2.5 bg-brand-border-subtle rounded-full overflow-hidden min-w-0"
        title={`Geleistet: ${formatDiensteZahl(row.geleistet)} · Eingetragen: ${formatDiensteZahl(row.vorhersage)} · Fair-Anteil: ${formatDiensteZahl(soll)}`}
      >
        <div className="h-full bg-brand-green" style={{ width: `${geleistetPct}%` }} aria-hidden="true" />
        <div className="absolute inset-y-0 h-full bg-brand-info" style={{ left: `${geleistetPct}%`, width: `${vorhersagePct}%` }} aria-hidden="true" />
        {soll > 0 && (
          <div
            className="absolute inset-y-0 w-px bg-brand-text"
            style={{ left: `${sollPct}%` }}
            aria-hidden="true"
          />
        )}
      </div>
      <span className="w-12 sm:w-14 shrink-0 text-right text-xs text-brand-text-muted whitespace-nowrap">
        {formatDiensteZahl(total)}
      </span>
    </div>
  )
}

function RanglisteBlockCard({ block }: { block: RanglisteBlock }) {
  const maxRowTotal = block.rows.reduce((m, r) => Math.max(m, r.geleistet + r.vorhersage), 0)
  const scale = Math.max(block.soll, maxRowTotal)

  return (
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
      <div className="flex items-baseline justify-between mb-1 gap-2 flex-wrap">
        <h2 className="font-semibold text-brand-text">{block.teamLabel}</h2>
        <span className="text-xs text-brand-text-muted">
          Fair-Anteil: {formatDiensteZahl(block.soll)} {block.soll === 1 ? 'Dienst' : 'Dienste'} pro Kind
        </span>
      </div>

      {block.rows.length === 0 ? (
        <p className="text-sm text-brand-text-muted py-2">Keine Kinder in diesem Kader.</p>
      ) : (
        <div className="mt-3 space-y-1">
          {block.rows.map(row => (
            <RanglisteRowView key={row.memberId ?? `rank-${row.rank}`} row={row} scale={scale} soll={block.soll} />
          ))}
        </div>
      )}
    </div>
  )
}

function Legend() {
  return (
    <p className="text-xs text-brand-text-muted mb-4 flex items-center gap-4 flex-wrap">
      <span className="inline-flex items-center gap-1.5">
        <span className="inline-block w-2.5 h-2.5 rounded-full bg-brand-green" aria-hidden="true" /> Geleistet
      </span>
      <span className="inline-flex items-center gap-1.5">
        <span className="inline-block w-2.5 h-2.5 rounded-full bg-brand-info" aria-hidden="true" /> Eingetragen
      </span>
      <span className="inline-flex items-center gap-1.5">
        <span className="inline-block w-0.5 h-3 bg-brand-text" aria-hidden="true" /> Fair-Anteil
      </span>
    </p>
  )
}

export default function DienstRanglistePage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const teamParam = searchParams.get('team')
  const filterTeamIds = useMemo(() => parseTeamIds(teamParam), [teamParam])

  const [teamOptions, setTeamOptions] = useState<TeamFilterOption[]>([])
  const [blocks, setBlocks] = useState<RanglisteBlock[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const compact = useCompactHeader(950)

  const load = useCallback(() => {
    setLoading(true)
    setError(null)
    const params = new URLSearchParams()
    if (filterTeamIds.size > 0) params.set('team', [...filterTeamIds].join(','))
    const qs = params.toString()
    const url = qs ? `/duty-fairness/rangliste?${qs}` : '/duty-fairness/rangliste'
    api.get(url)
      .then(r => {
        const data: RanglisteResponse = r.data ?? { teams: [], blocks: [] }
        setTeamOptions(data.teams ?? [])
        setBlocks(data.blocks ?? [])
      })
      .catch(err => {
        const status = (err as { response?: { status?: number } })?.response?.status
        setError(status === 403 ? 'Kein Zugriff auf diese Mannschaft.' : 'Rangliste konnte nicht geladen werden.')
      })
      .finally(() => setLoading(false))
  }, [filterTeamIds])

  // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt, kein Ableitungs-Bug (analog DashboardPage)
  useEffect(() => { load() }, [load])
  useLiveUpdates(event => { if (event === 'duties' || event === 'games') load() })

  const activeTeamIds = effectiveTeamIds(filterTeamIds, teamOptions)
  const toggleTeam = (teamId: number) => {
    const nextSelection = toggleTeamId(activeTeamIds, teamId)
    const next = new URLSearchParams(searchParams)
    const value = serializeTeamIds(nextSelection, teamOptions.length)
    if (value === null) next.delete('team')
    else next.set('team', value)
    setSearchParams(next, { replace: true })
  }

  const showEmptyState = !loading && !error && teamOptions.length === 0

  return (
    <div>
      <div className="flex items-center gap-2 mb-4 flex-wrap">
        <h1 className="text-2xl font-bold text-brand-text shrink-0">Dienst-Rangliste</h1>
        {teamOptions.length > 1 && (
          <TeamFilter teams={teamOptions} active={activeTeamIds} onToggle={toggleTeam} compact={compact} />
        )}
      </div>

      {!loading && !error && teamOptions.length > 0 && <Legend />}

      {error && (
        <div className="mb-4 p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger flex items-center gap-2">
          <AlertTriangle className="w-4 h-4 shrink-0" />
          {error}
        </div>
      )}

      {loading ? (
        <div className="space-y-4">
          {[1, 2].map(i => <div key={i} className="h-28 bg-brand-border-subtle rounded-xl animate-pulse" />)}
        </div>
      ) : showEmptyState ? (
        <p className="text-sm text-brand-text-muted py-8 text-center max-w-md mx-auto">
          Die Rangliste zeigt nur Mannschaften, in deren Kader der aktiven Saison du selbst oder eines deiner
          Kinder steht. Aktuell ist das für dich keine Mannschaft.
        </p>
      ) : !error ? (
        <div className="space-y-4">
          {blocks.map(block => <RanglisteBlockCard key={block.teamId} block={block} />)}
        </div>
      ) : null}
    </div>
  )
}
