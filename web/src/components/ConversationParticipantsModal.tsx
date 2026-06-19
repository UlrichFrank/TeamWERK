import { useEffect, useRef, useState } from 'react'
import { X, Pencil, Search, UserMinus, Crown } from 'lucide-react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import { useEscapeKey } from '../lib/useEscapeKey'
import { errorMessage } from '../lib/errors'

interface ConvMember { id: number; name: string }
interface ChatUser { id: number; name: string }

interface Props {
  convId: number
  initialName: string | null
  createdBy: number
  members: ConvMember[]
  onClose: () => void
  onChanged: () => void
}

export default function ConversationParticipantsModal({
  convId, initialName, createdBy, members, onClose, onChanged,
}: Props) {
  const { user } = useAuth()
  const isOwner = user?.id === createdBy
  useEscapeKey(onClose)

  const [editing, setEditing] = useState(false)
  const [name, setName] = useState(initialName ?? '')
  const [draftName, setDraftName] = useState(initialName ?? '')
  const [localMembers, setLocalMembers] = useState<ConvMember[]>(members)
  const [query, setQuery] = useState('')
  const [searchResults, setSearchResults] = useState<ChatUser[]>([])
  const [error, setError] = useState('')

  // Keep state in sync if parent updates (e.g. SSE-triggered reload)
  const lastInitialName = useRef(initialName)
  useEffect(() => {
    if (initialName !== lastInitialName.current) {
      setName(initialName ?? '')
      setDraftName(initialName ?? '')
      lastInitialName.current = initialName
    }
  }, [initialName])
  useEffect(() => { setLocalMembers(members) }, [members])

  useEffect(() => {
    if (!editing) return
    const t = setTimeout(async () => {
      try {
        const r = await api.get('/chat/users', { params: { q: query } })
        const memberIds = new Set(localMembers.map(m => m.id))
        setSearchResults((r.data ?? []).filter((u: ChatUser) => !memberIds.has(u.id)))
      } catch {}
    }, 200)
    return () => clearTimeout(t)
  }, [query, editing, localMembers])

  async function commitName() {
    const trimmed = draftName.trim()
    if (!trimmed || trimmed === name) {
      setDraftName(name)
      return
    }
    const previous = name
    setName(trimmed)
    setError('')
    try {
      await api.put(`/chat/conversations/${convId}`, { name: trimmed })
      onChanged()
    } catch (e) {
      setName(previous)
      setDraftName(previous)
      setError(errorMessage(e, 'Fehler beim Umbenennen'))
    }
  }

  async function removeMember(uid: number) {
    setError('')
    try {
      await api.delete(`/chat/conversations/${convId}/members/${uid}`)
      setLocalMembers(prev => prev.filter(m => m.id !== uid))
      onChanged()
    } catch (e) {
      setError(errorMessage(e, 'Fehler beim Entfernen'))
    }
  }

  async function addMember(u: ChatUser) {
    setError('')
    try {
      await api.post(`/chat/conversations/${convId}/members`, { userId: u.id })
      setLocalMembers(prev => [...prev, { id: u.id, name: u.name }])
      setSearchResults(prev => prev.filter(x => x.id !== u.id))
      onChanged()
    } catch (e) {
      setError(errorMessage(e, 'Fehler beim Hinzufügen'))
    }
  }

  return (
    <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-md max-h-[90vh] flex flex-col">
        <div className="flex items-center justify-between mb-4 shrink-0">
          <h2 className="text-lg font-bold text-brand-text">
            {editing ? 'Teilnehmer bearbeiten' : 'Teilnehmer'}
          </h2>
          <div className="flex items-center gap-1">
            {isOwner && !editing && (
              <button
                onClick={() => setEditing(true)}
                className="p-1 rounded hover:bg-brand-border-subtle transition-colors"
                aria-label="Bearbeiten"
              >
                <Pencil className="w-4 h-4 text-brand-text-muted" />
              </button>
            )}
            <button
              onClick={onClose}
              className="p-1 rounded hover:bg-brand-border-subtle transition-colors"
              aria-label="Schließen"
            >
              <X className="w-5 h-5 text-brand-text-muted" />
            </button>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto space-y-4">
          {editing && (
            <div>
              <label className="block text-xs text-brand-text-muted mb-1">Gruppenname</label>
              <input
                type="text"
                value={draftName}
                onChange={e => setDraftName(e.target.value)}
                onBlur={commitName}
                onKeyDown={e => { if (e.key === 'Enter') { e.currentTarget.blur() } }}
                maxLength={100}
                className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
              />
            </div>
          )}

          <div>
            <p className="text-xs text-brand-text-muted mb-2">
              {editing ? 'Aktuelle Teilnehmer' : `${localMembers.length} Teilnehmer`}
            </p>
            <ul className="border border-brand-border-subtle rounded-md divide-y divide-brand-border-subtle">
              {localMembers.map(m => (
                <li key={m.id} className="flex items-center gap-2 px-3 py-2.5 text-sm">
                  <span className="flex-1 text-brand-text truncate">{m.name}</span>
                  {m.id === createdBy && (
                    <span className="inline-flex items-center gap-1 text-xs bg-brand-yellow/20 text-brand-text rounded-full px-2 py-0.5">
                      <Crown className="w-3 h-3" />
                      Ersteller
                    </span>
                  )}
                  {editing && m.id !== createdBy && (
                    <button
                      onClick={() => removeMember(m.id)}
                      className="p-1.5 rounded-md text-brand-text-muted hover:text-brand-danger hover:bg-brand-danger-light transition-colors"
                      aria-label={`${m.name} entfernen`}
                    >
                      <UserMinus className="w-4 h-4" />
                    </button>
                  )}
                </li>
              ))}
            </ul>
          </div>

          {editing && (
            <div>
              <p className="text-xs text-brand-text-muted mb-2">Hinzufügen</p>
              <div className="relative mb-2">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-brand-text-subtle" />
                <input
                  type="text"
                  placeholder="Person suchen…"
                  value={query}
                  onChange={e => setQuery(e.target.value)}
                  className="w-full border border-brand-border rounded-md pl-9 pr-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                />
              </div>
              <div className="max-h-40 overflow-y-auto border border-brand-border-subtle rounded-md">
                {searchResults.length === 0 ? (
                  <p className="text-brand-text-muted text-sm p-3 text-center">
                    {query ? 'Keine Ergebnisse' : 'Tippen zum Suchen…'}
                  </p>
                ) : (
                  searchResults.map(u => (
                    <button
                      key={u.id}
                      onClick={() => addMember(u)}
                      className="w-full text-left px-3 py-2.5 text-sm text-brand-text hover:bg-brand-table-select transition-colors"
                    >
                      {u.name}
                    </button>
                  ))
                )}
              </div>
            </div>
          )}

          {error && <p className="text-sm text-brand-danger">{error}</p>}
        </div>

        <div className="pt-4 shrink-0">
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
