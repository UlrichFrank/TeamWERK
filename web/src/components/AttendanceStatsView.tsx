import { useEffect, useState } from 'react'
import { Check, MinusCircle, X } from 'lucide-react'
import { api } from '../lib/api'
import { useLiveUpdates } from '../hooks/useLiveUpdates'

interface Counts {
  member_id: number
  member_name: string
  training_present: number
  training_missed: number
  training_excused: number
  game_present: number
  game_missed: number
  game_excused: number
}

type Category = 'present' | 'missed' | 'excused' | 'unknown' | 'cancelled'

interface EventDetail {
  event_type: 'training' | 'game'
  event_id: number
  date: string
  title: string
  category: Category
  reason: string | null
}

interface MemberStats {
  member_id: number
  season_id: number
  start_date: string
  end_date: string
  counts: Counts
  events: EventDetail[]
}

function fmtDate(iso: string) {
  const d = iso.slice(0, 10).split('-')
  return d.length === 3 ? `${d[2]}.${d[1]}.${d[0]}` : iso
}

function quote(present: number, missed: number): string {
  const denom = present + missed
  if (denom === 0) return '–'
  return `${Math.round((present / denom) * 100)}%`
}

// Horizontaler Stacked-Bar grün/gelb/rot für die drei Säulen.
function StackedBar({ present, excused, missed }: { present: number; excused: number; missed: number }) {
  const total = present + excused + missed
  if (total === 0) {
    return <div className="h-3 rounded-full bg-brand-border-subtle" />
  }
  const pct = (n: number) => `${(n / total) * 100}%`
  return (
    <div className="flex h-3 w-full overflow-hidden rounded-full bg-brand-border-subtle">
      {present > 0 && <div className="bg-brand-green" style={{ width: pct(present) }} />}
      {excused > 0 && <div className="bg-brand-yellow" style={{ width: pct(excused) }} />}
      {missed > 0 && <div className="bg-brand-danger" style={{ width: pct(missed) }} />}
    </div>
  )
}

function PillarBlock({ title, present, excused, missed }: { title: string; present: number; excused: number; missed: number }) {
  return (
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
      <div className="flex items-baseline justify-between mb-3">
        <h3 className="font-semibold text-brand-text">{title}</h3>
        <span className="text-sm text-brand-text-muted">Quote {quote(present, missed)}</span>
      </div>
      <StackedBar present={present} excused={excused} missed={missed} />
      <div className="mt-3 flex flex-wrap gap-3 text-sm">
        <span className="inline-flex items-center gap-1 text-brand-text">
          <Check className="w-4 h-4 text-brand-green" /> {present} anwesend
        </span>
        <span className="inline-flex items-center gap-1 text-brand-text">
          <MinusCircle className="w-4 h-4 text-brand-yellow" /> {excused} entschuldigt
        </span>
        <span className="inline-flex items-center gap-1 text-brand-text">
          <X className="w-4 h-4 text-brand-danger" /> {missed} fehlt
        </span>
      </div>
    </div>
  )
}

function CategoryBadge({ category }: { category: Category }) {
  const map: Record<Category, { label: string; cls: string }> = {
    present: { label: 'anwesend', cls: 'bg-brand-green text-white' },
    excused: { label: 'entschuldigt', cls: 'bg-brand-yellow text-brand-black' },
    missed: { label: 'fehlt', cls: 'bg-brand-danger text-white' },
    unknown: { label: '—', cls: 'bg-brand-border-subtle text-brand-text-muted' },
    cancelled: { label: 'abgesagt', cls: 'bg-brand-border-subtle text-brand-text-muted' },
  }
  const { label, cls } = map[category]
  return <span className={`px-2 py-0.5 rounded text-xs font-medium whitespace-nowrap ${cls}`}>{label}</span>
}

function EventTable({ title, events }: { title: string; events: EventDetail[] }) {
  return (
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
      <div className="px-6 py-4 border-b border-brand-border-subtle">
        <h3 className="font-semibold text-brand-text">{title}</h3>
      </div>
      {events.length === 0 ? (
        <p className="px-6 py-4 text-sm text-brand-text-muted">Keine Termine im Saisonzeitraum.</p>
      ) : (
        <table className="w-full">
          <thead>
            <tr className="border-b border-brand-border-subtle">
              <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Datum</th>
              <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Titel</th>
              <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Status</th>
            </tr>
          </thead>
          <tbody>
            {events.map(ev => (
              <tr key={`${ev.event_type}-${ev.event_id}`} className="border-b border-brand-border-subtle last:border-0 hover:bg-brand-table-select transition-colors">
                <td className="px-4 py-3 text-sm text-brand-text whitespace-nowrap">{fmtDate(ev.date)}</td>
                <td className="px-4 py-3 text-sm text-brand-text">
                  {ev.title}
                  {ev.reason && <span className="block text-xs text-brand-text-muted italic">{ev.reason}</span>}
                </td>
                <td className="px-4 py-3"><CategoryBadge category={ev.category} /></td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}

export default function AttendanceStatsView({ memberId }: { memberId: number }) {
  const [stats, setStats] = useState<MemberStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = (silent = false) => {
    if (!silent) setLoading(true)
    api.get(`/members/${memberId}/attendance-stats`)
      .then(r => { setStats(r.data); setError(null) })
      .catch(() => setError('Statistik konnte nicht geladen werden.'))
      .finally(() => { if (!silent) setLoading(false) })
  }

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-/Abhängigkeits-getrieben), kein Ableitungs-Bug
    load()
    // load kapselt memberId, soll nur bei dessen Änderung neu laufen
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [memberId])

  useLiveUpdates((event) => { if (event === 'attendance-changed') load(true) })

  if (loading) return <p className="text-brand-text-muted text-sm p-4">Laden…</p>
  if (error) return <p className="text-brand-danger text-sm p-4">{error}</p>
  if (!stats) return null

  const c = stats.counts
  const trainings = stats.events.filter(e => e.event_type === 'training')
  const games = stats.events.filter(e => e.event_type === 'game')

  return (
    <div className="space-y-4">
      <div className="grid gap-4 sm:grid-cols-2">
        <PillarBlock title="Trainings" present={c.training_present} excused={c.training_excused} missed={c.training_missed} />
        <PillarBlock title="Spiele" present={c.game_present} excused={c.game_excused} missed={c.game_missed} />
      </div>
      <EventTable title="Alle Trainings" events={trainings} />
      <EventTable title="Alle Spiele" events={games} />
    </div>
  )
}
