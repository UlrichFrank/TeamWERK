import { Home, Plane, Calendar, Dumbbell, X, Check, Pencil, ClipboardList } from 'lucide-react'
import { useEscapeKey } from '../lib/useEscapeKey'

interface Game {
  id: number
  date: string
  time: string
  opponent: string
  event_type: string
  confirmed_count: number
  declined_count: number
  maybe_count: number
}

interface Training {
  id: number
  title: string
  date: string
  start_time: string
  end_time: string
  location: string
  confirmed_count: number
  declined_count: number
  maybe_count: number
}

interface Props {
  type: 'game' | 'training'
  game?: Game
  training?: Training
  onClose: () => void
  onEdit?: () => void
  onDienste?: () => void
}

function formatDate(dateStr: string): string {
  const d = new Date(dateStr.slice(0, 10) + 'T12:00:00')
  return d.toLocaleDateString('de-DE', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })
}

function RsvpRow({ confirmed, declined, maybe }: { confirmed: number; declined: number; maybe: number }) {
  return (
    <div className="pt-3 border-t border-brand-border-subtle flex gap-5 text-sm">
      <span className="flex items-center gap-1 text-green-600">
        <Check className="w-4 h-4" />
        <span className="font-medium">{confirmed}</span>
      </span>
      <span className="flex items-center gap-1 text-brand-danger">
        <X className="w-4 h-4" />
        <span className="font-medium">{declined}</span>
      </span>
      <span className="flex items-center gap-1 text-brand-text-muted">
        <span className="font-medium">{maybe}</span>
        <span className="text-xs">vlt.</span>
      </span>
    </div>
  )
}

export default function EventInfoModal({ type, game, training, onClose, onEdit, onDienste }: Props) {
  useEscapeKey(onClose)

  const eventTypeLabel = game?.event_type === 'heim' ? 'Heimspiel'
    : game?.event_type === 'auswärts' ? 'Auswärtsspiel'
    : 'Event'

  const EventIcon = game?.event_type === 'heim' ? Home
    : game?.event_type === 'auswärts' ? Plane
    : Calendar

  return (
    <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-md">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            {type === 'game'
              ? <EventIcon className="w-5 h-5 text-brand-text-muted" />
              : <Dumbbell className="w-5 h-5 text-brand-green" />}
            <h2 className="text-lg font-bold text-brand-text">
              {type === 'game' ? eventTypeLabel : (training?.title || 'Training')}
            </h2>
          </div>
          <div className="flex items-center gap-1">
            {onEdit && (
              <button onClick={onEdit} className="p-1 rounded hover:bg-brand-border-subtle transition-colors" aria-label="Bearbeiten">
                <Pencil className="w-4 h-4 text-brand-text-muted" />
              </button>
            )}
            {onDienste && (
              <button onClick={onDienste} className="p-1 rounded hover:bg-brand-border-subtle transition-colors" aria-label="Dienste">
                <ClipboardList className="w-4 h-4 text-brand-text-muted" />
              </button>
            )}
            <button onClick={onClose} className="p-1 rounded hover:bg-brand-border-subtle transition-colors" aria-label="Schließen">
              <X className="w-5 h-5 text-brand-text-muted" />
            </button>
          </div>
        </div>

        {type === 'game' && game ? (
          <div className="space-y-2 text-sm">
            {game.opponent && (
              <div className="flex justify-between">
                <span className="text-brand-text-muted">Gegner</span>
                <span className="font-medium text-brand-text">{game.opponent}</span>
              </div>
            )}
            <div className="flex justify-between">
              <span className="text-brand-text-muted">Datum</span>
              <span className="font-medium text-brand-text">{formatDate(game.date)}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-brand-text-muted">Uhrzeit</span>
              <span className="font-medium text-brand-text">{game.time}</span>
            </div>
            <RsvpRow confirmed={game.confirmed_count} declined={game.declined_count} maybe={game.maybe_count} />
          </div>
        ) : training ? (
          <div className="space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-brand-text-muted">Datum</span>
              <span className="font-medium text-brand-text">{formatDate(training.date)}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-brand-text-muted">Uhrzeit</span>
              <span className="font-medium text-brand-text">{training.start_time}–{training.end_time}</span>
            </div>
            {training.location && (
              <div className="flex justify-between">
                <span className="text-brand-text-muted">Ort</span>
                <span className="font-medium text-brand-text">{training.location}</span>
              </div>
            )}
            <RsvpRow confirmed={training.confirmed_count} declined={training.declined_count} maybe={training.maybe_count} />
          </div>
        ) : null}

        <div className="pt-4">
          <button
            onClick={onClose}
            className="w-full border border-brand-border rounded-md px-4 py-2.5 sm:py-2 text-sm text-brand-text-muted hover:text-brand-text hover:bg-brand-border-subtle transition-colors"
          >
            Schließen
          </button>
        </div>
      </div>
    </div>
  )
}
