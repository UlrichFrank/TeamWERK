import { useEffect, useId, useRef, useState } from 'react'
import { X } from 'lucide-react'
import { api } from '../lib/api'
import { useEscapeKey } from '../lib/useEscapeKey'
import { useDialogA11y } from '../lib/useDialogA11y'
import { errorMessage } from '../lib/errors'
import type { Poll } from '../pages/ChatPage'

interface Props {
  messageId: number
  onClose: () => void
}

// ChatPollVotesModal zeigt je Option die Namen der Abstimmenden — Umfragen
// sind nicht anonym (spec chat-umfragen, Requirement „Nicht-anonyme
// Ergebnisanzeige"). Lädt frisch über GET .../poll (Vorbild
// MessageReadsModal), unabhängig vom womöglich noch optimistischen Stand der
// Karte, die dieses Modal geöffnet hat.
export default function ChatPollVotesModal({ messageId, onClose }: Props) {
  const titleId = useId()
  const dialogRef = useRef<HTMLDivElement>(null)
  useEscapeKey(onClose)
  useDialogA11y(dialogRef, true)
  const [poll, setPoll] = useState<Poll | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let alive = true
    ;(async () => {
      try {
        const r = await api.get(`/chat/messages/${messageId}/poll`)
        if (alive) setPoll(r.data)
      } catch (e) {
        if (alive) setError(errorMessage(e, 'Fehler beim Laden'))
      } finally {
        if (alive) setLoading(false)
      }
    })()
    return () => {
      alive = false
    }
  }, [messageId])

  return (
    <div
      className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4"
      onClick={onClose}
    >
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-md max-h-[90vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-4 shrink-0">
          <h2 id={titleId} className="text-lg font-bold text-brand-text">Stimmen</h2>
          <button
            onClick={onClose}
            aria-label="Schließen"
            className="p-1 rounded hover:bg-brand-border-subtle transition-colors"
          >
            <X className="w-5 h-5 text-brand-text-muted" />
          </button>
        </div>

        {loading && <p className="text-sm text-brand-text-muted">Lade…</p>}
        {error && <p className="text-sm text-brand-danger">{error}</p>}

        {poll && (
          <div className="flex-1 overflow-y-auto space-y-4">
            {poll.options.map((opt) => (
              <div key={opt.id}>
                <p className="text-sm font-medium text-brand-text mb-1">
                  {opt.label}{' '}
                  <span className="text-brand-text-muted font-normal">({opt.count})</span>
                </p>
                {opt.voters.length === 0 ? (
                  <p className="text-xs text-brand-text-subtle">Noch keine Stimmen</p>
                ) : (
                  <ul className="border border-brand-border-subtle rounded-md divide-y divide-brand-border-subtle">
                    {opt.voters.map((v) => (
                      <li key={v.id} className="px-3 py-2 text-sm text-brand-text">
                        {v.name}
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
