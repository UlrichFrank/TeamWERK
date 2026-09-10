import { useEffect, useState } from 'react'
import { Circle, CircleCheck, Square, SquareCheck } from 'lucide-react'
import { api } from '../lib/api'
import { errorMessage } from '../lib/errors'
import type { Poll } from '../pages/ChatPage'

/**
 * applyOptimisticSelection berechnet aus der aktuellen Umfrage und der neuen
 * vollständigen Auswahl (design.md §2 — PUT setzt immer die vollständige
 * Auswahl, kein Toggle) das erwartete Ergebnis, ohne auf die Server-Antwort
 * zu warten. Die `voters`-Namenslisten je Option bleiben unangetastet: die
 * Karte zeigt sie nicht (nur Zahl/Balken/eigene Auswahl), ChatPollVotesModal
 * lädt sie bei Bedarf frisch über GET .../poll — ein hier erfundener Name
 * wäre ohnehin nur geraten, da der Client den eigenen Anzeigenamen nicht
 * kennt.
 */
export function applyOptimisticSelection(poll: Poll, newSelectedIds: number[]): Poll {
  const hadAnySelection = poll.options.some((o) => o.voted)
  const willHaveAnySelection = newSelectedIds.length > 0
  const options = poll.options.map((o) => {
    const willBeVoted = newSelectedIds.includes(o.id)
    if (willBeVoted === o.voted) return o
    return { ...o, voted: willBeVoted, count: o.count + (willBeVoted ? 1 : -1) }
  })
  const voterCount =
    hadAnySelection === willHaveAnySelection
      ? poll.voterCount
      : poll.voterCount + (willHaveAnySelection ? 1 : -1)
  return { ...poll, options, voterCount }
}

interface Props {
  poll: Poll
  question: string
  messageId: number
  isOwn: boolean
  onOpenVotes: () => void
}

// ChatPollCard ersetzt in MessageBubble den Text-Body einer Umfrage-Nachricht
// (design.md §8). Tippen berechnet die neue vollständige Auswahl (Einfach:
// ersetzen bzw. bei eigener Option leeren; Mehrfach: umschalten), optimistisch
// angezeigt bis die Server-Antwort (oder das eigene chat:poll-updated-Event,
// s. ChatPage) die autoritative Umfrage nachliefert. Bei Fehlschlag Rollback
// auf den zuletzt bekannten Stand + Fehlermeldung.
export default function ChatPollCard({ poll, question, messageId, isOwn, onOpenVotes }: Props) {
  const [pending, setPending] = useState<Poll | null>(null)
  const [voting, setVoting] = useState(false)
  const [error, setError] = useState('')

  // Neue autoritative Daten lösen den optimistischen Override wieder ab.
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-getrieben), kein Ableitungs-Bug
    setPending(null)
    setError('')
  }, [poll])

  const displayed = pending ?? poll
  const closed = displayed.closedAt !== null

  const vote = async (optionId: number) => {
    if (closed || voting) return
    const current = displayed.options.filter((o) => o.voted).map((o) => o.id)
    const next = displayed.allowMultiple
      ? current.includes(optionId)
        ? current.filter((id) => id !== optionId)
        : [...current, optionId]
      : current.includes(optionId)
        ? []
        : [optionId]
    setPending(applyOptimisticSelection(displayed, next))
    setVoting(true)
    setError('')
    try {
      await api.put(`/chat/messages/${messageId}/poll/vote`, { optionIds: next })
    } catch (e) {
      setPending(null)
      setError(errorMessage(e, 'Stimme konnte nicht gespeichert werden'))
    } finally {
      setVoting(false)
    }
  }

  return (
    <div className="min-w-[220px] max-w-full">
      <p className="font-medium whitespace-pre-wrap break-words">{question}</p>
      <p className="mt-0.5 text-xs opacity-70">
        {closed
          ? 'Beendet'
          : displayed.allowMultiple
            ? 'Mehrere Antworten möglich'
            : 'Eine Antwort wählen'}
      </p>
      <div className="mt-2 space-y-1.5">
        {displayed.options.map((opt) => {
          const pct =
            displayed.voterCount > 0 ? Math.round((opt.count / displayed.voterCount) * 100) : 0
          const Icon = displayed.allowMultiple
            ? opt.voted
              ? SquareCheck
              : Square
            : opt.voted
              ? CircleCheck
              : Circle
          return (
            <button
              key={opt.id}
              type="button"
              onClick={() => vote(opt.id)}
              disabled={closed}
              className={`w-full text-left rounded-md border border-brand-border-subtle px-2 py-2.5 sm:py-1.5 transition-colors disabled:cursor-default ${
                closed ? '' : 'hover:bg-brand-table-select'
              }`}
            >
              <span className="flex items-center gap-1.5 text-xs">
                <Icon className="w-4 h-4 shrink-0" aria-hidden="true" />
                <span className="flex-1 min-w-0 truncate">{opt.label}</span>
                <span className="shrink-0 font-medium">{opt.count}</span>
              </span>
              <span className="mt-1 block h-1.5 rounded-full overflow-hidden bg-brand-border-subtle">
                <span className="block h-full bg-brand-yellow" style={{ width: `${pct}%` }} />
              </span>
            </button>
          )
        })}
      </div>
      {error && <p className="mt-1.5 text-xs text-brand-danger">{error}</p>}
      <button
        type="button"
        onClick={onOpenVotes}
        className={`mt-2 text-xs underline ${isOwn ? 'text-brand-black/70' : 'text-brand-info'}`}
      >
        {displayed.voterCount} {displayed.voterCount === 1 ? 'Stimme' : 'Stimmen'} · Stimmen
        anzeigen
      </button>
    </div>
  )
}
