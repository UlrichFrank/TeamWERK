import { useEffect, useState } from 'react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import DutySlotList, { BoardSlot } from '../components/DutySlotList'

interface BoardGroup {
  game_id: number | null
  date: string | null
  event_time: string | null
  opponent: string | null
  event_type: string | null
  team_name: string
  label: string | null
  past: boolean
  slots: BoardSlot[]
}

export interface ProxyChild {
  user_id: number
  member_id: number
  name: string
}

const WEEKDAYS = ['So', 'Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa']

function formatDate(iso: string): string {
  const d = new Date(iso.slice(0, 10) + 'T12:00:00')
  return `${WEEKDAYS[d.getDay()]} ${String(d.getDate()).padStart(2, '0')}.${String(d.getMonth() + 1).padStart(2, '0')}.`
}

export default function DutyPage() {
  const { user } = useAuth()
  const isAdminOrTrainer = user?.role === 'admin' || user?.role === 'trainer'

  const [groups, setGroups] = useState<BoardGroup[]>([])
  const [showPast, setShowPast] = useState(false)
  const [viewMine, setViewMine] = useState(false)
  const [proxyChildren, setProxyChildren] = useState<ProxyChild[]>([])

  const load = () => {
    const url = viewMine ? '/duty-board?view=mine' : '/duty-board'
    api.get(url).then(r => setGroups(r.data ?? []))
  }

  useEffect(() => { load() }, [viewMine])
  useLiveUpdates((event) => { if (event === 'duties') load() })

  useEffect(() => {
    api.get('/family/proxy-accounts')
      .then(r => setProxyChildren(r.data ?? []))
      .catch(() => setProxyChildren([]))
  }, [])

  const visible = groups.filter(g => showPast || !g.past)

  return (
    <div>
      <div className="flex items-center justify-between mb-4 flex-wrap gap-2">
        <h1 className="text-2xl font-bold">Dienste</h1>
        <div className="flex items-center gap-3 flex-wrap">
          {isAdminOrTrainer && (
            <div className="flex rounded-lg border border-brand-border-subtle overflow-hidden text-xs">
              <button
                onClick={() => setViewMine(false)}
                className={`px-3 py-1.5 ${!viewMine ? 'bg-brand-yellow text-brand-black font-medium' : 'text-brand-text-muted hover:bg-brand-border-subtle'}`}
              >
                Alle Dienste
              </button>
              <button
                onClick={() => setViewMine(true)}
                className={`px-3 py-1.5 border-l border-brand-border-subtle ${viewMine ? 'bg-brand-yellow text-brand-black font-medium' : 'text-brand-text-muted hover:bg-brand-border-subtle'}`}
              >
                Meine Dienste
              </button>
            </div>
          )}
          <button
            onClick={() => setShowPast(p => !p)}
            className="text-xs text-brand-text-muted hover:text-brand-blue transition-colors"
          >
            {showPast ? 'Vergangene ausblenden' : 'Vergangene einblenden'}
          </button>
        </div>
      </div>

      {visible.length === 0 && (
        <p className="text-brand-text-muted">
          {groups.length === 0
            ? 'Keine Dienste für deine Mannschaften.'
            : 'Keine aktuellen Dienste. Vergangene können oben eingeblendet werden.'}
        </p>
      )}

      <div className="space-y-4">
        {visible.map((g, i) => (
          <div
            key={i}
            className={`bg-brand-surface-card rounded-xl shadow border-t-4 overflow-hidden ${g.past ? 'border-brand-border opacity-70' : 'border-brand-yellow'}`}
          >
            <div className="px-4 py-3 bg-brand-surface-card border-b border-brand-border-subtle flex items-center justify-between">
              <div>
                {g.game_id ? (
                  <span className="font-semibold text-sm text-brand-text">
                    {g.date ? formatDate(g.date) : ''}
                    {g.event_time ? ` · ${g.event_time} Uhr` : ''}
                    {g.opponent ? ` · ${g.event_type === 'generisch' ? g.opponent : `Team vs ${g.opponent}`}` : ''}
                  </span>
                ) : (
                  <span className="font-semibold text-sm text-brand-text">{g.label}</span>
                )}
              </div>
              <span className="text-xs text-brand-text-muted font-medium">{g.team_name}</span>
            </div>

            <DutySlotList
              slots={g.slots}
              isPast={g.past}
              canEdit={isAdminOrTrainer}
              onReload={load}
              proxyChildren={proxyChildren}
            />
          </div>
        ))}
      </div>
    </div>
  )
}
