import { useCallback, useEffect, useState } from 'react'
import { ChevronDown, ChevronRight, FileText, ImageOff, Info } from 'lucide-react'
import { api } from '../lib/api'
import AuthImage from '../components/AuthImage'
import {
  fmtDate,
  isImageMime,
  kindLabel,
  type DiaryEntry,
  type DiaryTeamStats,
} from '../lib/trainingDiary'
import { useLiveUpdates } from '../hooks/useLiveUpdates'

// Balken relativ zum fleißigsten Mitglied der Mannschaft. Bewusst relativ:
// eine absolute Skala bräuchte eine Soll-Vorgabe, die es nicht gibt.
function MinutesBar({ minutes, max }: { minutes: number; max: number }) {
  const pct = max > 0 ? Math.round((minutes / max) * 100) : 0
  return (
    <div className="h-3 w-full overflow-hidden rounded-full bg-brand-border-subtle">
      {minutes > 0 && <div className="h-full bg-brand-green" style={{ width: `${pct}%` }} />}
    </div>
  )
}

function ProofCell({ entry }: { entry: DiaryEntry }) {
  if (entry.proof_status === 'purged') {
    return (
      <span className="inline-flex items-center gap-1 text-xs text-brand-text-muted italic">
        <ImageOff className="w-4 h-4" /> gelöscht
      </span>
    )
  }
  if (entry.proof_status === 'none') {
    return <span className="text-xs text-brand-text-subtle">—</span>
  }
  if (isImageMime(entry.proof_mime)) {
    return (
      <AuthImage
        url={`/training-diary/${entry.id}/proof`}
        alt="Trainingsnachweis"
        className="max-h-24 rounded border border-brand-border-subtle"
      />
    )
  }
  return (
    <span className="inline-flex items-center gap-1 text-xs text-brand-text">
      <FileText className="w-4 h-4" /> Datei
    </span>
  )
}

// Detailliste eines Mitglieds, lazy nachgeladen beim Aufklappen.
function MemberDetail({ memberId, seasonId }: { memberId: number; seasonId: number }) {
  const [entries, setEntries] = useState<DiaryEntry[] | null>(null)

  useEffect(() => {
    let cancelled = false
    api
      .get<{ items: DiaryEntry[] }>(`/members/${memberId}/training-diary?season=${seasonId}`)
      .then(r => {
        if (!cancelled) setEntries(r.data.items ?? [])
      })
      .catch(() => {
        if (!cancelled) setEntries([])
      })
    return () => {
      cancelled = true
    }
  }, [memberId, seasonId])

  if (entries === null) {
    return <p className="px-4 py-3 text-sm text-brand-text-muted">Lädt …</p>
  }
  if (entries.length === 0) {
    return <p className="px-4 py-3 text-sm text-brand-text-muted">Keine Einträge in dieser Saison.</p>
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full">
        <thead>
          <tr className="border-b border-brand-border-subtle">
            <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Datum</th>
            <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Art</th>
            <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Dauer</th>
            <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">RPE</th>
            <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Nachweis</th>
          </tr>
        </thead>
        <tbody>
          {entries.map(entry => (
            <tr key={entry.id} className="border-b border-brand-border-subtle last:border-0">
              <td className="px-4 py-3 text-sm text-brand-text whitespace-nowrap">{fmtDate(entry.trained_on)}</td>
              <td className="px-4 py-3 text-sm text-brand-text">
                {kindLabel(entry)}
                {entry.note && <span className="block text-xs text-brand-text-muted italic">{entry.note}</span>}
              </td>
              <td className="px-4 py-3 text-sm text-brand-text whitespace-nowrap">{entry.duration_min} min</td>
              <td className="px-4 py-3 text-sm text-brand-text">{entry.rpe}</td>
              <td className="px-4 py-3"><ProofCell entry={entry} /></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export default function TrainingDiaryStatsView({ teamId }: { teamId: number }) {
  const [stats, setStats] = useState<DiaryTeamStats | null>(null)
  const [expanded, setExpanded] = useState<number | null>(null)
  const [error, setError] = useState('')

  const load = useCallback(() => {
    api
      .get<DiaryTeamStats>(`/teams/${teamId}/training-diary-stats`)
      .then(r => setStats(r.data))
      .catch(() => setError('Die Übersicht konnte nicht geladen werden.'))
  }, [teamId])

  useEffect(load, [load])
  useLiveUpdates(event => {
    if (event === 'training-diary-changed') load()
  })

  if (error) {
    return (
      <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
        {error}
      </div>
    )
  }
  if (!stats) return <p className="text-sm text-brand-text-muted">Lädt …</p>

  const maxMinutes = stats.items.reduce((m, i) => Math.max(m, i.minutes), 0)

  return (
    <div className="space-y-4">
      {/* Ohne diesen Hinweis liest sich eine Nullzeile als Faulheit — sie misst
          aber nur, dass nichts eingetragen wurde. */}
      <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
        <span className="inline-flex items-start gap-2">
          <Info className="w-4 h-4 mt-0.5 shrink-0" />
          <span>
            Die Zahlen beruhen auf Selbstauskunft der Spieler und zeigen, wer <em>einträgt</em> —
            nicht zwingend, wer trainiert. Ein Nachweis ist freiwillig.
          </span>
        </span>
      </div>

      {stats.items.length === 0 ? (
        <p className="text-sm text-brand-text-muted">Keine Kadermitglieder in dieser Saison.</p>
      ) : (
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
          {stats.items.map(item => {
            const open = expanded === item.member_id
            return (
              <div key={item.member_id} className="border-b border-brand-border-subtle last:border-0">
                <button
                  type="button"
                  aria-expanded={open}
                  onClick={() => setExpanded(open ? null : item.member_id)}
                  className="w-full px-4 py-3 text-left hover:bg-brand-table-select transition-colors"
                >
                  <div className="flex items-center gap-2">
                    {open ? (
                      <ChevronDown className="w-4 h-4 shrink-0 text-brand-text-muted" />
                    ) : (
                      <ChevronRight className="w-4 h-4 shrink-0 text-brand-text-muted" />
                    )}
                    <span className="flex-1 text-sm font-medium text-brand-text">{item.member_name}</span>
                    <span className="text-sm text-brand-text-muted whitespace-nowrap">
                      {item.entries} Einh. · {item.minutes} min
                      {item.entries > 0 && ` · RPE-Schnitt ${item.avg_rpe.toFixed(1)}`}
                    </span>
                  </div>
                  <div className="mt-2 pl-6">
                    <MinutesBar minutes={item.minutes} max={maxMinutes} />
                  </div>
                </button>
                {open && <MemberDetail memberId={item.member_id} seasonId={stats.season_id} />}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
