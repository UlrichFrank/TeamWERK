import { useEffect, useRef, useState } from 'react'
import { MessageSquare, Megaphone, X } from 'lucide-react'
import { api } from '../lib/api'
import { errorMessage } from '../lib/errors'
import { relativeTime } from '../lib/relativeTime'
import { useEscapeKey } from '../lib/useEscapeKey'
import { highlight } from '../lib/chatSearchHighlight'

/**
 * Ein Suchtreffer über Chat-Nachrichten oder Mitteilungen (`GET /api/chat/search`).
 * Siehe openspec/changes/chat-message-search/design.md Entscheidung 1/5.
 */
export interface SearchHit {
  kind: 'message' | 'broadcast'
  id: number
  conversationId?: number
  conversationName: string
  senderName: string
  snippet: string
  sentAt: string
}

interface Props {
  onClose: () => void
  onSelect: (hit: SearchHit) => void
}

const MIN_QUERY_LENGTH = 2
const DEBOUNCE_MS = 300
const PAGE_SIZE = 50

/**
 * Modal zur Volltextsuche über Chat-Nachrichten und Mitteilungen
 * (openspec/changes/chat-message-search, Task 4.2). Entscheidet selbst nicht,
 * was ein Klick auf einen Treffer bewirkt — das bleibt beim Aufrufer
 * (`onSelect`, Sprung-Logik lebt in ChatPage/design.md Entscheidung 7).
 */
export default function ChatSearchModal({ onClose, onSelect }: Props) {
  useEscapeKey(onClose)

  const [query, setQuery] = useState('')
  const [items, setItems] = useState<SearchHit[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [searched, setSearched] = useState(false)

  // Race-Schutz: nur die Antwort auf die zuletzt gestartete Anfrage darf noch
  // State setzen — eine schnell getippte, überholte Anfrage kann sonst nach
  // der aktuellen zurückkommen und die Liste mit veralteten Treffern überschreiben.
  const requestSeq = useRef(0)

  const trimmedQuery = query.trim()

  // Reset unterhalb der Mindestlänge passiert im onChange-Handler (nicht hier):
  // synchrones setState im Effekt würde einen Kaskaden-Render auslösen
  // (react-hooks/set-state-in-effect). Der Effekt plant nur die Anfrage.
  const onQueryChange = (value: string) => {
    setQuery(value)
    if (value.trim().length < MIN_QUERY_LENGTH) {
      requestSeq.current++ // laufende Anfrage entwerten
      setItems([])
      setTotal(0)
      setError('')
      setSearched(false)
    }
  }

  useEffect(() => {
    if (trimmedQuery.length < MIN_QUERY_LENGTH) return

    const seq = ++requestSeq.current
    const timer = setTimeout(() => {
      setLoading(true)
      setError('')
      api
        .get('/chat/search', { params: { q: trimmedQuery, limit: PAGE_SIZE, offset: 0 } })
        .then((res) => {
          if (seq !== requestSeq.current) return
          setItems(res.data.items)
          setTotal(res.data.total)
        })
        .catch((e) => {
          if (seq !== requestSeq.current) return
          setError(errorMessage(e, 'Suche fehlgeschlagen'))
          setItems([])
          setTotal(0)
        })
        .finally(() => {
          if (seq !== requestSeq.current) return
          setLoading(false)
          setSearched(true)
        })
    }, DEBOUNCE_MS)

    return () => clearTimeout(timer)
  }, [trimmedQuery])

  return (
    <div
      className="fixed inset-0 z-50 bg-brand-black/40 flex items-start justify-center p-4 sm:pt-20"
      onClick={onClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label="Nachrichten durchsuchen"
        className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-lg max-h-[80vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center gap-2 shrink-0">
          <input
            type="text"
            autoFocus
            value={query}
            onChange={(e) => onQueryChange(e.target.value)}
            placeholder="Nachrichten und Mitteilungen durchsuchen…"
            aria-label="Suchbegriff"
            className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
          />
          <button
            type="button"
            onClick={onClose}
            aria-label="Schließen"
            className="p-1.5 rounded-md text-brand-text-muted hover:bg-brand-table-select hover:text-brand-text transition-colors shrink-0"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="mt-4 flex-1 overflow-y-auto min-h-0">
          {trimmedQuery.length < MIN_QUERY_LENGTH && (
            <p className="text-sm text-brand-text-muted">Mindestens 2 Zeichen eingeben</p>
          )}

          {trimmedQuery.length >= MIN_QUERY_LENGTH && loading && (
            <div role="status" className="text-sm text-brand-text-muted">
              Suche…
            </div>
          )}

          {trimmedQuery.length >= MIN_QUERY_LENGTH && !loading && error && (
            <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
              {error}
            </p>
          )}

          {trimmedQuery.length >= MIN_QUERY_LENGTH && !loading && !error && searched && items.length === 0 && (
            <p className="text-sm text-brand-text-muted">Keine Treffer für „{trimmedQuery}“</p>
          )}

          {trimmedQuery.length >= MIN_QUERY_LENGTH && !loading && !error && items.length > 0 && (
            <>
              <ul>
                {items.map((hit) => (
                  <li key={`${hit.kind}-${hit.id}`}>
                    <button
                      type="button"
                      onClick={() => onSelect(hit)}
                      className="w-full text-left px-3 py-2.5 sm:py-2 rounded-md hover:bg-brand-table-select transition-colors"
                    >
                      <div className="flex items-center gap-2 text-xs text-brand-text-muted">
                        {hit.kind === 'broadcast' ? (
                          <Megaphone className="w-4 h-4 shrink-0" />
                        ) : (
                          <MessageSquare className="w-4 h-4 shrink-0" />
                        )}
                        <span className="font-medium text-brand-text">{hit.conversationName}</span>
                        <span>{hit.senderName}</span>
                        <span className="ml-auto shrink-0">{relativeTime(hit.sentAt)}</span>
                      </div>
                      <div className="mt-0.5 text-sm text-brand-text">
                        {highlight(hit.snippet, trimmedQuery)}
                      </div>
                    </button>
                  </li>
                ))}
              </ul>
              {total > items.length && (
                <p className="mt-2 px-3 text-xs text-brand-text-muted">
                  {total - items.length} weitere Treffer – Suchbegriff verfeinern
                </p>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  )
}
