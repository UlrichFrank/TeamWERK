import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Check, X, HelpCircle, Dumbbell } from 'lucide-react'
import { api } from '../lib/api'
import { useAuth, hasFunction } from '../contexts/AuthContext'

const WEEKDAYS = ['So', 'Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa']

function fmtDate(iso: string) {
  const d = new Date(iso.slice(0, 10) + 'T12:00:00')
  return `${WEEKDAYS[d.getDay()]} ${String(d.getDate()).padStart(2, '0')}.${String(d.getMonth() + 1).padStart(2, '0')}.${d.getFullYear()}`
}

interface Session {
  id: number
  series_id: number | null
  team_id: number
  season_id: number
  date: string
  start_time: string
  end_time: string
  location: string
  note: string
  status: 'active' | 'cancelled'
  cancel_reason: string
  confirmed_count: number
  declined_count: number
  maybe_count: number
  my_rsvp: string | null
}

export default function TrainingsPage() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const isTrainer = user?.role === 'admin' || hasFunction(user, 'trainer')

  const [sessions, setSessions] = useState<Session[]>([])
  const [showPast, setShowPast] = useState(false)
  const [loading, setLoading] = useState(true)
  const [rsvpLoading, setRsvpLoading] = useState<number | null>(null)

  const today = new Date().toISOString().slice(0, 10)
  const from = showPast
    ? new Date(Date.now() - 90 * 24 * 60 * 60 * 1000).toISOString().slice(0, 10)
    : today
  const to = new Date(Date.now() + 180 * 24 * 60 * 60 * 1000).toISOString().slice(0, 10)

  const load = () => {
    setLoading(true)
    api.get(`/training-sessions?from=${from}&to=${to}`)
      .then(r => setSessions(r.data ?? []))
      .finally(() => setLoading(false))
  }

  useEffect(() => { load() }, [showPast])

  const respond = async (sessionId: number, status: string, e: React.MouseEvent) => {
    e.stopPropagation()
    setRsvpLoading(sessionId)
    try {
      await api.post(`/training-sessions/${sessionId}/respond`, { status })
      setSessions(prev => prev.map(s =>
        s.id === sessionId ? { ...s, my_rsvp: status } : s
      ))
    } finally {
      setRsvpLoading(null)
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6 flex-wrap gap-2">
        <h1 className="text-2xl font-bold text-brand-text">Trainings</h1>
        <div className="flex items-center gap-2">
          <label className="flex items-center gap-2 text-sm text-brand-text-muted cursor-pointer select-none">
            <input
              type="checkbox"
              checked={showPast}
              onChange={e => setShowPast(e.target.checked)}
              className="rounded border-brand-border"
            />
            Vergangene anzeigen
          </label>
          {isTrainer && (
            <button
              onClick={() => navigate('/admin/trainings')}
              className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
            >
              Verwalten
            </button>
          )}
        </div>
      </div>

      {loading ? (
        <p className="text-brand-text-muted text-sm">Laden…</p>
      ) : sessions.length === 0 ? (
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-8 text-center">
          <Dumbbell className="w-10 h-10 mx-auto mb-3 text-brand-text-subtle" />
          <p className="text-brand-text-muted">Keine Trainingstermine vorhanden.</p>
        </div>
      ) : (
        <div className="space-y-3">
          {sessions.map(s => (
            <div
              key={s.id}
              onClick={() => navigate(`/trainings/${s.id}`)}
              className={`bg-brand-surface-card rounded-xl shadow border-t-4 p-4 cursor-pointer hover:shadow-md transition-shadow ${
                s.status === 'cancelled' ? 'border-brand-border opacity-60' : 'border-brand-yellow'
              }`}
            >
              <div className="flex items-start justify-between gap-4 flex-wrap">
                <div className="flex items-start gap-3 min-w-0">
                  <Dumbbell className="w-5 h-5 mt-0.5 text-brand-text-muted shrink-0" />
                  <div className="min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className={`font-semibold text-brand-text ${s.status === 'cancelled' ? 'line-through' : ''}`}>
                        {fmtDate(s.date)}
                      </span>
                      <span className="text-brand-text-muted text-sm">{s.start_time} – {s.end_time}</span>
                      {s.status === 'cancelled' && (
                        <span className="bg-brand-danger-light text-brand-danger text-xs font-medium px-2 py-0.5 rounded-full">
                          Abgesagt
                        </span>
                      )}
                    </div>
                    {s.location && (
                      <p className="text-sm text-brand-text-muted mt-0.5">{s.location}</p>
                    )}
                    {s.status === 'cancelled' && s.cancel_reason && (
                      <p className="text-sm text-brand-danger mt-0.5">{s.cancel_reason}</p>
                    )}
                  </div>
                </div>

                {s.status === 'active' && (
                  <div className="flex items-center gap-1 shrink-0">
                    <span className="text-xs text-brand-text-muted bg-white border border-brand-border-subtle rounded px-2 py-1 flex items-center gap-1">
                      <Check className="w-3 h-3 text-green-600" />{s.confirmed_count}
                    </span>
                    <span className="text-xs text-brand-text-muted bg-white border border-brand-border-subtle rounded px-2 py-1 flex items-center gap-1">
                      <X className="w-3 h-3 text-brand-danger" />{s.declined_count}
                    </span>
                    <span className="text-xs text-brand-text-muted bg-white border border-brand-border-subtle rounded px-2 py-1 flex items-center gap-1">
                      <HelpCircle className="w-3 h-3 text-brand-text-subtle" />{s.maybe_count}
                    </span>
                  </div>
                )}
              </div>

              {s.status === 'active' && !isTrainer && (
                <div className="mt-3 flex gap-2" onClick={e => e.stopPropagation()}>
                  <RsvpButton
                    label="Zusagen"
                    icon={<Check className="w-4 h-4" />}
                    active={s.my_rsvp === 'confirmed'}
                    activeClass="bg-green-600 text-white border-green-600"
                    disabled={rsvpLoading === s.id}
                    onClick={e => respond(s.id, s.my_rsvp === 'confirmed' ? 'maybe' : 'confirmed', e)}
                  />
                  <RsvpButton
                    label="Vielleicht"
                    icon={<HelpCircle className="w-4 h-4" />}
                    active={s.my_rsvp === 'maybe'}
                    activeClass="bg-brand-yellow text-brand-black border-brand-yellow"
                    disabled={rsvpLoading === s.id}
                    onClick={e => respond(s.id, 'maybe', e)}
                  />
                  <RsvpButton
                    label="Absagen"
                    icon={<X className="w-4 h-4" />}
                    active={s.my_rsvp === 'declined'}
                    activeClass="bg-brand-danger text-white border-brand-danger"
                    disabled={rsvpLoading === s.id}
                    onClick={e => respond(s.id, 'declined', e)}
                  />
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function RsvpButton({ label, icon, active, activeClass, disabled, onClick }: {
  label: string
  icon: React.ReactNode
  active: boolean
  activeClass: string
  disabled: boolean
  onClick: (e: React.MouseEvent) => void
}) {
  return (
    <button
      disabled={disabled}
      onClick={onClick}
      className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 sm:py-1 text-xs font-medium border transition-colors disabled:opacity-40 disabled:cursor-not-allowed ${
        active
          ? activeClass
          : 'bg-white border-brand-border text-brand-text-muted hover:border-brand-text hover:text-brand-text'
      }`}
    >
      {icon}
      <span className="hidden sm:inline">{label}</span>
    </button>
  )
}
