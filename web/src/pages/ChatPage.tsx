import { useState, useEffect, useRef, useCallback } from 'react'
import { Send, Plus, LogOut, MessageSquare, Megaphone, X, Search, Users, UserPlus, Trash2 } from 'lucide-react'
import { api } from '../lib/api'
import { useAuth, hasFunction } from '../contexts/AuthContext'
import { useChatEvents } from '../hooks/useChatEvents'

interface ConvMember { id: number; name: string }
interface LastMessage { body: string; sentAt: string }
interface Conversation {
  id: number
  type: 'direct' | 'group'
  name: string | null
  createdBy: number
  unreadCount: number
  lastMessage: LastMessage | null
  members: ConvMember[]
}
interface Message {
  id: number
  senderId: number
  senderName: string
  body: string
  sentAt: string
}
interface Broadcast {
  id: number
  senderName: string
  body: string
  sentAt: string
  isRead: boolean
  isSent: boolean
}
interface ChatUser { id: number; name: string }

type Tab = 'chats' | 'broadcasts'

export default function ChatPage() {
  const { user } = useAuth()
  const [tab, setTab] = useState<Tab>('chats')
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [broadcasts, setBroadcasts] = useState<Broadcast[]>([])
  const [activeConv, setActiveConv] = useState<Conversation | null>(null)
  const [messages, setMessages] = useState<Message[]>([])
  const [msgInput, setMsgInput] = useState('')
  const [sending, setSending] = useState(false)
  const [showNewModal, setShowNewModal] = useState(false)
  const [showBroadcastModal, setShowBroadcastModal] = useState(false)
  const [showAddMember, setShowAddMember] = useState(false)
  const [activeBroadcast, setActiveBroadcast] = useState<Broadcast | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const isMobile = window.innerWidth < 640
  const [mobileShowChat, setMobileShowChat] = useState(false)

  const canBroadcast = user && (user.role === 'admin' || hasFunction(user, 'vorstand') || hasFunction(user, 'trainer') || hasFunction(user, 'sportliche_leitung'))

  const loadConversations = useCallback(async () => {
    try {
      const r = await api.get('/chat/conversations')
      setConversations(r.data ?? [])
    } catch {}
  }, [])

  const reloadActiveConv = useCallback(async (convId: number) => {
    try {
      const r = await api.get('/chat/conversations')
      const updated = (r.data ?? []).find((c: Conversation) => c.id === convId)
      if (updated) setActiveConv(updated)
      setConversations(r.data ?? [])
    } catch {}
  }, [])

  const loadBroadcasts = useCallback(async () => {
    try {
      const r = await api.get('/chat/broadcasts')
      setBroadcasts(r.data ?? [])
    } catch {}
  }, [])

  useEffect(() => {
    loadConversations()
    loadBroadcasts()
  }, [loadConversations, loadBroadcasts])

  useChatEvents((event) => {
    if (event.startsWith('chat:new-message')) {
      loadConversations()
      const parts = event.split(':')
      const convId = parseInt(parts[2])
      if (activeConv?.id === convId) loadMessages(convId)
    }
    if (event === 'chat:new-broadcast') loadBroadcasts()
  })

  const loadMessages = async (convId: number) => {
    try {
      const r = await api.get(`/chat/conversations/${convId}/messages`)
      setMessages(r.data ?? [])
      await api.post(`/chat/conversations/${convId}/read`)
      loadConversations()
    } catch {}
  }

  const openConversation = async (conv: Conversation) => {
    setActiveConv(conv)
    setMobileShowChat(true)
    await loadMessages(conv.id)
    setTimeout(() => messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' }), 50)
  }

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const sendMessage = async () => {
    if (!activeConv || !msgInput.trim() || sending) return
    setSending(true)
    try {
      await api.post(`/chat/conversations/${activeConv.id}/messages`, { body: msgInput.trim() })
      setMsgInput('')
      await loadMessages(activeConv.id)
    } catch {} finally {
      setSending(false)
    }
  }

  const leaveGroup = async () => {
    if (!activeConv || activeConv.type !== 'group') return
    if (!confirm('Gruppe verlassen?')) return
    await api.delete(`/chat/conversations/${activeConv.id}/members/me`)
    setActiveConv(null)
    setMobileShowChat(false)
    loadConversations()
  }

  const deleteConversation = async (conv: Conversation) => {
    if (!confirm('Gespräch löschen?')) return
    await api.delete(`/chat/conversations/${conv.id}`).catch(() => {})
    if (activeConv?.id === conv.id) {
      setActiveConv(null)
      setMobileShowChat(false)
    }
    loadConversations()
  }

  const deleteBroadcast = async (bc: Broadcast) => {
    if (!confirm('Mitteilung löschen?')) return
    await api.delete(`/chat/broadcasts/${bc.id}`).catch(() => {})
    if (activeBroadcast?.id === bc.id) {
      setActiveBroadcast(null)
      setMobileShowChat(false)
    }
    loadBroadcasts()
  }

  const openBroadcast = async (bc: Broadcast) => {
    setActiveBroadcast(bc)
    setMobileShowChat(true)
    if (!bc.isRead && !bc.isSent) {
      await api.post(`/chat/broadcasts/${bc.id}/read`).catch(() => {})
      loadBroadcasts()
    }
  }

  const convName = (conv: Conversation) => {
    if (conv.name) return conv.name
    const others = conv.members.filter(m => m.id !== user?.id)
    return others.map(m => m.name).join(', ') || 'Konversation'
  }

  const totalUnread = conversations.reduce((s, c) => s + c.unreadCount, 0)
    + broadcasts.filter(b => !b.isRead && !b.isSent).length

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold text-brand-text flex items-center gap-2">
          Nachrichten
          {totalUnread > 0 && (
            <span className="bg-brand-yellow text-brand-black text-xs font-bold rounded-full px-2 py-0.5">{totalUnread}</span>
          )}
        </h1>
      </div>

      <div className="flex flex-1 min-h-0 gap-4">
        {/* Left panel: list */}
        <div className={`${isMobile && mobileShowChat ? 'hidden' : 'flex'} flex-col w-full sm:w-72 bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden`}>
          {/* Tabs */}
          <div className="flex border-b border-brand-border-subtle">
            <button
              onClick={() => setTab('chats')}
              className={`flex-1 py-3 text-sm font-medium flex items-center justify-center gap-1.5 transition-colors ${tab === 'chats' ? 'text-brand-text border-b-2 border-brand-yellow' : 'text-brand-text-muted hover:text-brand-text'}`}
            >
              <MessageSquare className="w-4 h-4" />
              Chats
            </button>
            <button
              onClick={() => setTab('broadcasts')}
              className={`flex-1 py-3 text-sm font-medium flex items-center justify-center gap-1.5 transition-colors ${tab === 'broadcasts' ? 'text-brand-text border-b-2 border-brand-yellow' : 'text-brand-text-muted hover:text-brand-text'}`}
            >
              <Megaphone className="w-4 h-4" />
              Mitteilungen
              {broadcasts.filter(b => !b.isRead && !b.isSent).length > 0 && (
                <span className="bg-brand-yellow text-brand-black text-xs font-bold rounded-full px-1.5">{broadcasts.filter(b => !b.isRead && !b.isSent).length}</span>
              )}
            </button>
          </div>

          {tab === 'chats' && (
            <>
              <div className="p-3 border-b border-brand-border-subtle">
                <button
                  onClick={() => setShowNewModal(true)}
                  className="w-full bg-brand-yellow text-brand-black rounded-md px-3 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors flex items-center justify-center gap-1.5"
                >
                  <Plus className="w-4 h-4" />
                  Neues Gespräch
                </button>
              </div>
              <div className="flex-1 overflow-y-auto">
                {conversations.length === 0 && (
                  <p className="text-brand-text-muted text-sm p-4 text-center">Noch keine Gespräche</p>
                )}
                {conversations.map(conv => (
                  <div
                    key={conv.id}
                    className={`flex items-center border-b border-brand-border-subtle hover:bg-brand-table-select transition-colors ${activeConv?.id === conv.id ? 'bg-brand-table-select' : ''}`}
                  >
                    <button
                      onClick={() => openConversation(conv)}
                      className="flex-1 min-w-0 text-left px-4 py-3"
                    >
                      <div className="flex items-center justify-between gap-2">
                        <span className="text-sm font-medium text-brand-text truncate">{convName(conv)}</span>
                        {conv.unreadCount > 0 && (
                          <span className="bg-brand-yellow text-brand-black text-xs font-bold rounded-full px-1.5 shrink-0">{conv.unreadCount}</span>
                        )}
                      </div>
                      {conv.lastMessage && (
                        <p className="text-xs text-brand-text-muted truncate mt-0.5">{conv.lastMessage.body}</p>
                      )}
                    </button>
                    <button
                      onClick={() => deleteConversation(conv)}
                      className="shrink-0 px-3 py-3 text-brand-text-subtle hover:text-brand-danger transition-colors"
                      aria-label="Gespräch löschen"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                ))}
              </div>
            </>
          )}

          {tab === 'broadcasts' && (
            <>
              {canBroadcast && (
                <div className="p-3 border-b border-brand-border-subtle">
                  <button
                    onClick={() => setShowBroadcastModal(true)}
                    className="w-full bg-brand-yellow text-brand-black rounded-md px-3 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors flex items-center justify-center gap-1.5"
                  >
                    <Megaphone className="w-4 h-4" />
                    Mitteilung senden
                  </button>
                </div>
              )}
              <div className="flex-1 overflow-y-auto">
                {broadcasts.length === 0 && (
                  <p className="text-brand-text-muted text-sm p-4 text-center">Keine Mitteilungen</p>
                )}
                {broadcasts.map(bc => (
                  <div
                    key={bc.id}
                    className={`flex items-center border-b border-brand-border-subtle hover:bg-brand-table-select transition-colors ${activeBroadcast?.id === bc.id ? 'bg-brand-table-select' : ''}`}
                  >
                    <button
                      onClick={() => openBroadcast(bc)}
                      className="flex-1 min-w-0 text-left px-4 py-3"
                    >
                      <div className="flex items-center justify-between gap-2">
                        <span className={`text-sm truncate ${!bc.isRead && !bc.isSent ? 'font-semibold text-brand-text' : 'font-medium text-brand-text-muted'}`}>
                          {bc.isSent ? 'Gesendet' : bc.senderName}
                        </span>
                        {!bc.isRead && !bc.isSent && (
                          <span className="w-2 h-2 rounded-full bg-brand-yellow shrink-0" />
                        )}
                      </div>
                      <p className="text-xs text-brand-text-muted truncate mt-0.5">{bc.body}</p>
                    </button>
                    <button
                      onClick={() => deleteBroadcast(bc)}
                      className="shrink-0 px-3 py-3 text-brand-text-subtle hover:text-brand-danger transition-colors"
                      aria-label="Mitteilung löschen"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                ))}
              </div>
            </>
          )}
        </div>

        {/* Right panel: active chat or broadcast */}
        <div className={`${isMobile && !mobileShowChat ? 'hidden' : 'flex'} flex-col flex-1 min-w-0 bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden`}>
          {activeConv && tab === 'chats' && (
            <>
              <div className="flex items-center justify-between px-4 py-3 border-b border-brand-border-subtle">
                <div className="flex items-center gap-2 min-w-0">
                  {isMobile && (
                    <button onClick={() => setMobileShowChat(false)} className="text-brand-text-muted hover:text-brand-text mr-1" aria-label="Zurück">
                      <X className="w-5 h-5" />
                    </button>
                  )}
                  <span className="font-semibold text-brand-text truncate">{convName(activeConv)}</span>
                  {activeConv.type === 'group' && (
                    <span className="text-xs text-brand-text-subtle shrink-0">
                      <Users className="w-3.5 h-3.5 inline mr-0.5" />{activeConv.members.length}
                    </span>
                  )}
                </div>
                {activeConv.type === 'group' && (
                  <div className="flex items-center gap-2 shrink-0">
                    {activeConv.createdBy === user?.id && (
                      <button
                        onClick={() => setShowAddMember(true)}
                        className="text-brand-text-muted hover:text-brand-text transition-colors"
                        aria-label="Mitglied hinzufügen"
                      >
                        <UserPlus className="w-4 h-4" />
                      </button>
                    )}
                    <button
                      onClick={leaveGroup}
                      className="text-brand-text-muted hover:text-brand-danger transition-colors"
                      aria-label="Gruppe verlassen"
                    >
                      <LogOut className="w-4 h-4" />
                    </button>
                  </div>
                )}
              </div>
              <div className="flex-1 overflow-y-auto p-4 flex flex-col gap-2">
                {messages.map(msg => {
                  const isOwn = msg.senderId === user?.id
                  return (
                    <div key={msg.id} className={`flex flex-col ${isOwn ? 'items-end' : 'items-start'}`}>
                      {!isOwn && <span className="text-xs text-brand-text-muted mb-0.5">{msg.senderName}</span>}
                      <div className={`max-w-xs sm:max-w-sm rounded-xl px-3 py-2 text-sm ${isOwn ? 'bg-brand-yellow text-brand-black' : 'bg-white border border-brand-border text-brand-text'}`}>
                        {msg.body}
                      </div>
                      <span className="text-xs text-brand-text-subtle mt-0.5">
                        {new Date(msg.sentAt).toLocaleTimeString('de-DE', { hour: '2-digit', minute: '2-digit' })}
                      </span>
                    </div>
                  )
                })}
                <div ref={messagesEndRef} />
              </div>
              <div className="px-4 py-3 border-t border-brand-border-subtle flex gap-2">
                <input
                  type="text"
                  value={msgInput}
                  onChange={e => setMsgInput(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && !e.shiftKey && sendMessage()}
                  placeholder="Nachricht schreiben…"
                  maxLength={2000}
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                />
                <button
                  onClick={sendMessage}
                  disabled={!msgInput.trim() || sending}
                  className="bg-brand-yellow text-brand-black rounded-md px-3 py-2 hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                  aria-label="Senden"
                >
                  <Send className="w-4 h-4" />
                </button>
              </div>
            </>
          )}

          {activeBroadcast && tab === 'broadcasts' && (
            <div className="flex-1 overflow-y-auto p-6">
              <div className="flex items-center gap-2 mb-1">
                {isMobile && (
                  <button onClick={() => { setActiveBroadcast(null); setMobileShowChat(false) }} className="text-brand-text-muted hover:text-brand-text mr-1" aria-label="Zurück">
                    <X className="w-5 h-5" />
                  </button>
                )}
                <span className="font-semibold text-brand-text">{activeBroadcast.isSent ? 'Gesendet von mir' : activeBroadcast.senderName}</span>
              </div>
              <p className="text-xs text-brand-text-muted mb-4">
                {new Date(activeBroadcast.sentAt).toLocaleString('de-DE')}
              </p>
              <p className="text-sm text-brand-text whitespace-pre-wrap">{activeBroadcast.body}</p>
            </div>
          )}

          {!activeConv && !activeBroadcast && (
            <div className="flex-1 flex items-center justify-center text-brand-text-muted text-sm">
              Gespräch oder Mitteilung auswählen
            </div>
          )}
        </div>
      </div>

      {showNewModal && (
        <NewConversationModal onClose={() => setShowNewModal(false)} onCreated={(conv) => {
          setShowNewModal(false)
          loadConversations()
          openConversation(conv)
          setTab('chats')
        }} />
      )}

      {showBroadcastModal && (
        <BroadcastModal
          onClose={() => setShowBroadcastModal(false)}
          onSent={() => { setShowBroadcastModal(false); loadBroadcasts() }}
          isAdmin={user?.role === 'admin' || hasFunction(user!, 'vorstand')}
        />
      )}

      {showAddMember && activeConv && (
        <AddMemberModal
          convId={activeConv.id}
          existingMemberIds={activeConv.members.map(m => m.id)}
          onClose={() => setShowAddMember(false)}
          onAdded={() => { setShowAddMember(false); reloadActiveConv(activeConv.id) }}
        />
      )}
    </div>
  )
}

// --- New Conversation Modal ---
function NewConversationModal({ onClose, onCreated }: { onClose: () => void; onCreated: (conv: Conversation) => void }) {
  const [type, setType] = useState<'direct' | 'group'>('direct')
  const [query, setQuery] = useState('')
  const [users, setUsers] = useState<ChatUser[]>([])
  const [selected, setSelected] = useState<ChatUser[]>([])
  const [groupName, setGroupName] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    const t = setTimeout(async () => {
      try {
        const r = await api.get('/chat/users', { params: { q: query } })
        setUsers(r.data ?? [])
      } catch {}
    }, 200)
    return () => clearTimeout(t)
  }, [query])

  const toggleUser = (u: ChatUser) => {
    if (type === 'direct') {
      setSelected([u])
    } else {
      setSelected(prev => prev.find(p => p.id === u.id) ? prev.filter(p => p.id !== u.id) : [...prev, u])
    }
  }

  const submit = async () => {
    if (selected.length === 0) return
    setLoading(true)
    setError('')
    try {
      const payload = type === 'direct'
        ? { type, userId: selected[0].id }
        : { type, name: groupName, memberIds: selected.map(u => u.id) }
      const r = await api.post('/chat/conversations', payload)
      onCreated(r.data)
    } catch (e: any) {
      setError(e.response?.data || 'Fehler beim Erstellen')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-md">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold text-brand-text">Neues Gespräch</h2>
          <button onClick={onClose} aria-label="Schließen"><X className="w-5 h-5 text-brand-text-muted" /></button>
        </div>

        <div className="flex gap-2 mb-4">
          {(['direct', 'group'] as const).map(t => (
            <button
              key={t}
              onClick={() => { setType(t); setSelected([]) }}
              className={`flex-1 py-2 rounded-md text-sm font-medium transition-colors ${type === t ? 'bg-brand-yellow text-brand-black' : 'bg-brand-surface-card text-brand-text-muted hover:text-brand-text'}`}
            >
              {t === 'direct' ? 'Direkt' : 'Gruppe'}
            </button>
          ))}
        </div>

        {type === 'group' && (
          <input
            type="text"
            placeholder="Gruppenname"
            value={groupName}
            onChange={e => setGroupName(e.target.value)}
            className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow mb-3"
          />
        )}

        <div className="relative mb-3">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-brand-text-subtle" />
          <input
            type="text"
            placeholder="Person suchen…"
            value={query}
            onChange={e => setQuery(e.target.value)}
            className="w-full border border-brand-border rounded-md pl-9 pr-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
          />
        </div>

        {selected.length > 0 && (
          <div className="flex flex-wrap gap-1.5 mb-3">
            {selected.map(u => (
              <span key={u.id} className="flex items-center gap-1 bg-brand-yellow/20 text-brand-text text-xs rounded-full px-2 py-0.5">
                {u.name}
                <button onClick={() => setSelected(prev => prev.filter(p => p.id !== u.id))} aria-label="Entfernen"><X className="w-3 h-3" /></button>
              </span>
            ))}
          </div>
        )}

        <div className="max-h-48 overflow-y-auto border border-brand-border-subtle rounded-md mb-4">
          {users.map(u => (
            <button
              key={u.id}
              onClick={() => toggleUser(u)}
              className={`w-full text-left px-3 py-2 text-sm hover:bg-brand-table-select transition-colors ${selected.find(s => s.id === u.id) ? 'bg-brand-yellow/10 font-medium' : 'text-brand-text'}`}
            >
              {u.name}
            </button>
          ))}
          {users.length === 0 && <p className="text-brand-text-muted text-sm p-3 text-center">Keine Ergebnisse</p>}
        </div>

        {error && <p className="text-brand-danger text-sm mb-3">{error}</p>}

        <button
          onClick={submit}
          disabled={loading || selected.length === 0 || (type === 'group' && !groupName.trim())}
          className="w-full bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {loading ? 'Erstelle…' : 'Gespräch starten'}
        </button>
      </div>
    </div>
  )
}

// --- Broadcast Modal ---
function BroadcastModal({ onClose, onSent, isAdmin }: { onClose: () => void; onSent: () => void; isAdmin: boolean }) {
  const [body, setBody] = useState('')
  const [targetType, setTargetType] = useState<'all' | 'team' | 'role'>('all')
  const [teams, setTeams] = useState<{ id: number; name: string }[]>([])
  const [targetId, setTargetId] = useState(0)
  const [targetRole, setTargetRole] = useState('spieler')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    api.get('/teams').then(r => setTeams(r.data ?? [])).catch(() => {})
  }, [])

  const submit = async () => {
    if (!body.trim()) return
    setLoading(true)
    setError('')
    try {
      await api.post('/chat/broadcasts', { body: body.trim(), targetType, targetId, targetRole })
      onSent()
    } catch (e: any) {
      setError(e.response?.data || 'Fehler beim Senden')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-md">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold text-brand-text">Mitteilung senden</h2>
          <button onClick={onClose} aria-label="Schließen"><X className="w-5 h-5 text-brand-text-muted" /></button>
        </div>

        <label className="block text-sm font-medium text-brand-text mb-1">Zielgruppe</label>
        <select
          value={targetType}
          onChange={e => setTargetType(e.target.value as any)}
          className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow mb-3"
        >
          {isAdmin && <option value="all">Alle Mitglieder</option>}
          <option value="team">Team</option>
          {isAdmin && <option value="role">Rolle</option>}
        </select>

        {targetType === 'team' && (
          <select
            value={targetId}
            onChange={e => setTargetId(Number(e.target.value))}
            className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow mb-3"
          >
            <option value={0}>Team wählen…</option>
            {teams.map(t => <option key={t.id} value={t.id}>{t.name}</option>)}
          </select>
        )}

        {targetType === 'role' && (
          <select
            value={targetRole}
            onChange={e => setTargetRole(e.target.value)}
            className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow mb-3"
          >
            <option value="spieler">Spieler</option>
            <option value="elternteil">Elternteile</option>
            <option value="trainer">Trainer</option>
          </select>
        )}

        <label className="block text-sm font-medium text-brand-text mb-1">Nachricht</label>
        <textarea
          value={body}
          onChange={e => setBody(e.target.value)}
          maxLength={2000}
          rows={5}
          placeholder="Deine Mitteilung…"
          className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow resize-none mb-1"
        />
        <p className="text-xs text-brand-text-subtle text-right mb-3">{body.length}/2000</p>

        {error && <p className="text-brand-danger text-sm mb-3">{error}</p>}

        <button
          onClick={submit}
          disabled={loading || !body.trim() || (targetType === 'team' && !targetId)}
          className="w-full bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {loading ? 'Sende…' : 'Mitteilung senden'}
        </button>
      </div>
    </div>
  )
}

// --- Add Member Modal ---
function AddMemberModal({ convId, existingMemberIds, onClose, onAdded }: {
  convId: number
  existingMemberIds: number[]
  onClose: () => void
  onAdded: () => void
}) {
  const [query, setQuery] = useState('')
  const [users, setUsers] = useState<ChatUser[]>([])
  const [selected, setSelected] = useState<ChatUser | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    const t = setTimeout(async () => {
      try {
        const r = await api.get('/chat/users', { params: { q: query } })
        setUsers((r.data ?? []).filter((u: ChatUser) => !existingMemberIds.includes(u.id)))
      } catch {}
    }, 200)
    return () => clearTimeout(t)
  }, [query, existingMemberIds])

  const submit = async () => {
    if (!selected) return
    setLoading(true)
    setError('')
    try {
      await api.post(`/chat/conversations/${convId}/members`, { userId: selected.id })
      onAdded()
    } catch (e: any) {
      setError(e.response?.data || 'Fehler beim Hinzufügen')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-md">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold text-brand-text">Mitglied hinzufügen</h2>
          <button onClick={onClose} aria-label="Schließen"><X className="w-5 h-5 text-brand-text-muted" /></button>
        </div>

        <div className="relative mb-3">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-brand-text-subtle" />
          <input
            type="text"
            placeholder="Person suchen…"
            value={query}
            onChange={e => setQuery(e.target.value)}
            className="w-full border border-brand-border rounded-md pl-9 pr-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
          />
        </div>

        <div className="max-h-48 overflow-y-auto border border-brand-border-subtle rounded-md mb-4">
          {users.map(u => (
            <button
              key={u.id}
              onClick={() => setSelected(u)}
              className={`w-full text-left px-3 py-2 text-sm hover:bg-brand-table-select transition-colors ${selected?.id === u.id ? 'bg-brand-yellow/10 font-medium' : 'text-brand-text'}`}
            >
              {u.name}
            </button>
          ))}
          {users.length === 0 && <p className="text-brand-text-muted text-sm p-3 text-center">Keine Ergebnisse</p>}
        </div>

        {error && <p className="text-brand-danger text-sm mb-3">{error}</p>}

        <button
          onClick={submit}
          disabled={loading || !selected}
          className="w-full bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {loading ? 'Hinzufügen…' : selected ? `${selected.name} hinzufügen` : 'Person auswählen'}
        </button>
      </div>
    </div>
  )
}
