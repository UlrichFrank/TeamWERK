import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { useNavigate } from 'react-router-dom'
import { MessageSquare } from 'lucide-react'
import { usePersonContact } from '../contexts/PersonContactContext'
import { useAuth } from '../contexts/AuthContext'

interface PersonChipProps {
  userId?: number
  name: string
  photoUrl?: string
}

function toWhatsAppNumber(raw: string): string {
  let digits = raw.replace(/\D/g, '')
  if (digits.startsWith('00')) digits = digits.slice(2)
  else if (digits.startsWith('0')) digits = '49' + digits.slice(1)
  return digits
}

export default function PersonChip({ userId, name, photoUrl }: PersonChipProps) {
  const [open, setOpen] = useState(false)
  const [pos, setPos] = useState({ top: 0, left: 0 })
  const btnRef = useRef<HTMLButtonElement>(null)
  const tooltipRef = useRef<HTMLDivElement>(null)
  const closeTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const { get, fetchContact } = usePersonContact()
  const { user } = useAuth()
  const navigate = useNavigate()

  function scheduleClose() {
    closeTimer.current = setTimeout(() => setOpen(false), 150)
  }

  function cancelClose() {
    if (closeTimer.current) {
      clearTimeout(closeTimer.current)
      closeTimer.current = null
    }
  }

  useEffect(() => {
    if (!open) return
    const handler = (e: MouseEvent) => {
      if (
        btnRef.current && !btnRef.current.contains(e.target as Node) &&
        tooltipRef.current && !tooltipRef.current.contains(e.target as Node)
      ) setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  if (!userId) {
    return (
      <span className="inline-flex items-center rounded-full border border-brand-border-subtle px-2 py-0.5 text-xs text-brand-text-muted">
        {name}
      </span>
    )
  }

  const state = get(userId)
  // Avatar on-demand: fällt der Aufrufer-Prop weg (z. B. Dienstbörse liefert nur
  // noch Namen inline), erscheint das Foto nach dem Kontakt-Fetch beim Öffnen.
  const contact = typeof state === 'object' ? state : undefined
  const chipPhoto = photoUrl ?? contact?.photo_url

  function handleOpen() {
    if (!btnRef.current) return
    const r = btnRef.current.getBoundingClientRect()
    setPos({ top: r.top + window.scrollY, left: r.left + window.scrollX })
    fetchContact(userId!)
    setOpen(true)
  }

  const linkClass = 'underline hover:text-brand-text transition-colors'

  return (
    <>
      <button
        ref={btnRef}
        type="button"
        className="flex items-center gap-1.5 rounded-full bg-brand-border-subtle px-2 py-0.5 text-xs text-brand-text hover:bg-brand-border transition-colors"
        onMouseEnter={() => { cancelClose(); handleOpen() }}
        onMouseLeave={scheduleClose}
        onClick={(e) => { e.stopPropagation(); if (!open) { handleOpen() } else { setOpen(false) } }}
        aria-label={`Details zu ${name}`}
      >
        {chipPhoto && (
          <img src={chipPhoto} alt="" className="w-4 h-4 rounded-full object-cover flex-shrink-0" />
        )}
        {name}
      </button>

      {open && createPortal(
        <div
          ref={tooltipRef}
          style={{ top: pos.top - 8, left: pos.left, transform: 'translateY(-100%)' }}
          className="fixed z-[9999] w-52 bg-white rounded-lg shadow-lg border border-brand-border-subtle p-3 text-xs"
          onMouseEnter={cancelClose}
          onMouseLeave={scheduleClose}
        >
          {state === 'loading' || state === undefined ? (
            <div className="flex items-center gap-2 text-brand-text-muted">
              <svg className="animate-spin w-3 h-3" viewBox="0 0 24 24" fill="none">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
              </svg>
              Lädt…
            </div>
          ) : state === 'error' ? (
            <p className="text-brand-danger italic">Fehler beim Laden</p>
          ) : (
            <>
              {state.photo_url && (
                <img
                  src={state.photo_url}
                  alt={state.name}
                  className="w-10 h-10 rounded-full object-cover mb-2"
                />
              )}
              <p className="font-semibold text-brand-text mb-1.5">{state.name}</p>
              {userId !== user?.id && (
                <button
                  onClick={() => { setOpen(false); navigate(`/chat?openUser=${userId}`) }}
                  className="flex items-center gap-1.5 mb-2 text-brand-text-muted hover:text-brand-text transition-colors"
                >
                  <MessageSquare className="w-3.5 h-3.5" />
                  <span>Nachricht schreiben</span>
                </button>
              )}
              {(state.phones && state.phones.length > 0) || state.address || state.email ? (
                <>
                  {state.phones && state.phones.length > 0 && (
                    <div className="space-y-0.5 mb-1.5">
                      {state.phones.map((p, i) => (
                        <p key={i} className="text-brand-text-muted">
                          <span className="text-brand-text-subtle">{p.label}:</span>{' '}
                          <a href={`tel:${p.number}`} className={linkClass}>{p.number}</a>
                          {state.whatsapp_visible && (
                            <>
                              {' · '}
                              <a
                                href={`https://wa.me/${toWhatsAppNumber(p.number)}`}
                                target="_blank"
                                rel="noreferrer"
                                className={linkClass}
                              >
                                WhatsApp
                              </a>
                            </>
                          )}
                        </p>
                      ))}
                    </div>
                  )}
                  {state.address && (
                    <p className="text-brand-text-muted whitespace-pre-line mb-1">{state.address}</p>
                  )}
                  {state.email && (
                    <p className="text-brand-text-muted">
                      <a href={`mailto:${state.email}`} className={linkClass}>{state.email}</a>
                    </p>
                  )}
                </>
              ) : (
                <p className="text-brand-text-subtle italic">Keine weiteren Infos freigegeben</p>
              )}
            </>
          )}
        </div>,
        document.body
      )}
    </>
  )
}
