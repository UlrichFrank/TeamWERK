import { useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { Link, useNavigate } from 'react-router-dom'
import { BookOpen, MessageCircle } from 'lucide-react'
import { api } from '../lib/api'
import { formatTimeSpan } from '../lib/duration'
import { relativeTime } from '../lib/relativeTime'
import { useAuth } from '../contexts/AuthContext'
import { useEscapeKey } from '../lib/useEscapeKey'
import { useDialogA11y } from '../lib/useDialogA11y'
import { useWindowedList } from '../hooks/useWindowedList'
import WindowedTableBody from './WindowedTableBody'
import PersonChip from './PersonChip'
import AushilfeBadge from './AushilfeBadge'
import ActionMenu from './ActionMenu'
import { AUDIENCE_LABELS } from '../lib/constants'
import type { ProxyChild } from '../pages/DutyPage'
import { BTN_PRIMARY, BTN_DANGER } from '../lib/buttonStyles'

// Bewusst schlank: Board liefert nur Namen inline; Avatar/Kontakt lädt
// PersonChip on-demand über GET /api/users/{id}/contact (Sichtbarkeitsregeln
// wie bisher serverseitig via *_visible).
export interface PublicAssignee {
  user_id: number
  name: string
  // Eingetragener hilft aus dem erweiterten Kader aus (dienste-erweiterter-kader).
  aushilfe?: boolean
}

export interface BoardSlot {
  id: number
  duty_type: string
  duty_type_id: number
  has_instruction: boolean
  event_time: string
  hours_value: number
  slots_total: number
  vacancies: number
  claimed_by_me: boolean
  role_desc?: string
  audiences?: string[] | null
  assignees?: PublicAssignee[]
  comment_count: number
  my_assignment_id?: number
}

interface AssignmentComment {
  user_id: number
  user_name: string
  body: string
  created_at: string
}


interface DutySlotListProps {
  slots: BoardSlot[]
  isPast: boolean
  canEdit: boolean
  onReload: () => void
  onEdit?: (slotId: number) => void
  proxyChildren?: ProxyChild[]
  hideClaimActions?: boolean
  /**
   * Wird vor der Navigation zur Anleitungsseite aufgerufen, damit der Aufrufer
   * (DutyPage) den Fokus-Marker (`focus=slot-<id>`) auf der aktuellen /dienste-URL
   * hinterlegen kann — Voraussetzung dafür, dass „Zurück" später zu dieser Zeile
   * zurückscrollt (siehe openspec/changes/zurueck-position-wiederherstellen).
   * Optional, weil DutySlotList auch außerhalb von /dienste eingebunden wird
   * (SpieltagDetailModal) — dort bleibt das Verhalten unverändert (kein Fokus-Marker).
   */
  onFocusSlot?: (slotId: number) => void
}

export default function DutySlotList({ slots, isPast, canEdit, onReload, onEdit, proxyChildren = [], hideClaimActions = false, onFocusSlot }: DutySlotListProps) {
  const { user } = useAuth()
  const navigate = useNavigate()
  // Windowing der Slot-Zeilen: bei sehr vielen Slots nur sichtbare im DOM.
  // Scroll-Quelle ist die Seite (Duty-Board scrollt als Ganzes).
  const { containerRef: slotContainerRef, start: slotStart, end: slotEnd, padTop: slotPadTop, padBottom: slotPadBottom } =
    useWindowedList({ count: slots.length, estimatedRowHeight: 52, scroll: 'window' })
  const [claimDialog, setClaimDialog] = useState<{ slotId: number; selectedUserId: number | null } | null>(null)
  const [claimLoading, setClaimLoading] = useState(false)
  const [noInstructionOpen, setNoInstructionOpen] = useState(false)
  // Kommentar-Modal (dienst-kommentare): assignmentId ist nur gesetzt, wenn der
  // Betrachter selbst für diesen Slot eingetragen ist — sonst reiner Lesemodus.
  const [commentModal, setCommentModal] = useState<{ slotId: number; assignmentId?: number } | null>(null)
  const [commentModalLoading, setCommentModalLoading] = useState(false)
  const [comments, setComments] = useState<AssignmentComment[]>([])
  const [myCommentDraft, setMyCommentDraft] = useState('')
  const [commentSaving, setCommentSaving] = useState(false)
  // Drei echte modale Dialoge in dieser Liste (dialog-accessibility): je eigener
  // Ref, damit Fokus-Trap und -Rückgabe pro Dialog greifen. Löschen ist bewusst
  // kein Dialog mehr hier — siehe „Anleitung ins Aktionsmenü" unten.
  const noInstructionRef = useRef<HTMLDivElement>(null)
  const claimDialogRef = useRef<HTMLDivElement>(null)
  const commentModalRef = useRef<HTMLDivElement>(null)
  useDialogA11y(noInstructionRef, noInstructionOpen)
  useDialogA11y(claimDialogRef, claimDialog !== null)
  useDialogA11y(commentModalRef, commentModal !== null)

  useEscapeKey(
    claimDialog !== null ? () => setClaimDialog(null)
      : noInstructionOpen ? () => setNoInstructionOpen(false)
      : commentModal !== null ? () => setCommentModal(null)
      : null,
  )

  const openCommentModal = async (slot: BoardSlot) => {
    setCommentModal({ slotId: slot.id, assignmentId: slot.claimed_by_me ? slot.my_assignment_id : undefined })
    setCommentModalLoading(true)
    setMyCommentDraft('')
    try {
      const res = await api.get<AssignmentComment[]>(`/duty-slots/${slot.id}/comments`)
      setComments(res.data)
      const mine = user && res.data.find(c => c.user_id === user.id)
      setMyCommentDraft(mine?.body ?? '')
    } catch {
      setComments([])
    } finally {
      setCommentModalLoading(false)
    }
  }

  const saveMyComment = async () => {
    if (!commentModal?.assignmentId || !myCommentDraft.trim()) return
    setCommentSaving(true)
    try {
      await api.put(`/duty-assignments/${commentModal.assignmentId}/comment`, { body: myCommentDraft.trim() })
      setCommentModal(null)
      onReload()
    } catch {
      alert('Kommentar speichern fehlgeschlagen.')
    } finally {
      setCommentSaving(false)
    }
  }

  const deleteMyComment = async () => {
    if (!commentModal?.assignmentId) return
    setCommentSaving(true)
    try {
      await api.delete(`/duty-assignments/${commentModal.assignmentId}/comment`)
      setCommentModal(null)
      onReload()
    } catch {
      alert('Kommentar löschen fehlgeschlagen.')
    } finally {
      setCommentSaving(false)
    }
  }

  const claimForUser = async (slotId: number, userId: number) => {
    setClaimLoading(true)
    try {
      await api.post(`/duty-board/${slotId}/claim`, { user_id: userId })
      setClaimDialog(null)
      onReload()
    } catch {
      alert('Dieser Dienst ist bereits vergeben oder du hast ihn bereits.')
    } finally {
      setClaimLoading(false)
    }
  }

  const handleClaimClick = (slotId: number) => {
    if (proxyChildren.length > 0 && user) {
      setClaimDialog({ slotId, selectedUserId: user.id ?? null })
    } else if (user) {
      claimForUser(slotId, user.id)
    }
  }

  const claim = async (id: number) => {
    handleClaimClick(id)
  }

  const unclaim = async (id: number) => {
    try {
      await api.delete(`/duty-board/${id}/claim`)
      onReload()
    } catch {
      alert('Austragen fehlgeschlagen.')
    }
  }

  return (
    <>
      <div ref={slotContainerRef}>
      <table className="w-full text-sm table-fixed">
        <colgroup>
          {/* Mobile enger (nur Uhrzeit-Ziffern), Desktop etwas großzügiger. */}
          <col className="w-22 sm:w-[6rem]" />
          <col />
          <col style={{ width: '35%' }} />
          {/* Mobile zeigt nur das Punkte-Menü (schmal) statt der Desktop-Buttons —
              schmalere Spalte hier zieht Spalte 3 (rechtsbündig) direkt ans Menü heran. */}
          <col className="w-11 sm:w-[9.5rem]" />
        </colgroup>
        <WindowedTableBody
          items={slots}
          start={slotStart}
          end={slotEnd}
          padTop={slotPadTop}
          padBottom={slotPadBottom}
          colSpan={4}
          className="divide-y divide-brand-border-subtle"
          renderRow={s => (
              <tr key={s.id} id={`duty-slot-${s.id}`}>
                <td className="pl-4 pr-1 sm:px-4 py-2.5 text-brand-text-muted whitespace-nowrap">{formatTimeSpan(s.event_time, s.hours_value)}</td>
                <td className="pl-4 pr-4 sm:px-4 py-2.5 font-medium text-brand-text">
                  <span className="inline-flex items-center gap-1.5">
                    {s.duty_type}
                    {s.has_instruction ? (
                      <Link
                        to={`/dienste/anleitung/${s.duty_type_id}`}
                        aria-label="Anleitung ansehen"
                        onClick={e => { e.stopPropagation(); onFocusSlot?.(s.id) }}
                        className="text-brand-text-muted hover:text-brand-text"
                      >
                        <BookOpen className="w-4 h-4" />
                      </Link>
                    ) : (
                      <button
                        type="button"
                        aria-label="Keine Anleitung vorhanden"
                        onClick={e => { e.stopPropagation(); setNoInstructionOpen(true) }}
                        className="relative inline-flex items-center justify-center text-brand-text-subtle hover:text-brand-text-muted"
                      >
                        <BookOpen className="w-4 h-4 opacity-60" />
                        <span
                          aria-hidden
                          className="pointer-events-none absolute left-0 right-0 top-1/2 -translate-y-1/2 h-px bg-current rotate-45"
                        />
                      </button>
                    )}
                    {s.comment_count > 0 && (
                      <button
                        type="button"
                        aria-label={`${s.comment_count} Kommentar${s.comment_count === 1 ? '' : 'e'} ansehen`}
                        onClick={e => { e.stopPropagation(); openCommentModal(s) }}
                        className="text-brand-text-muted hover:text-brand-text"
                      >
                        <MessageCircle className="w-4 h-4" />
                      </button>
                    )}
                  </span>
                  {s.role_desc ? <span className="text-brand-text-subtle font-normal"> · {s.role_desc}</span> : null}
                </td>
                <td className="px-4 py-2.5 text-brand-text-muted text-right">
                  <div className="flex flex-col items-end gap-1.5">
                    {s.vacancies > 0 && <div><span className="text-xs">{s.vacancies} frei</span></div>}
                    {s.audiences && s.audiences.length > 0 && (
                      <div className="flex flex-wrap justify-end gap-1">
                        {s.audiences.map(a => (
                          <span key={a} className="text-xs bg-brand-info/10 text-brand-text px-1.5 py-0.5 rounded">
                            {AUDIENCE_LABELS[a] ?? a}
                          </span>
                        ))}
                      </div>
                    )}
                    {s.assignees && s.assignees.length > 0 && (
                      <div className="flex flex-wrap justify-end gap-1">
                        {s.assignees.map((a, i) => (
                          <span key={i} className="inline-flex items-center gap-1">
                            <PersonChip userId={a.user_id} name={a.name} />
                            {a.aushilfe && <AushilfeBadge />}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                </td>
                <td className="px-4 py-2.5 text-right">
                  {/* Desktop: Eintragen/Austragen bleiben direkt sichtbare
                      Primärbuttons; alles Weitere (Bearbeiten/Anleitung/
                      Kommentieren) wandert ins ⋮-Menü — bisher nur mobile. */}
                  <div className="hidden sm:flex items-center justify-end gap-2">
                    {!hideClaimActions && s.claimed_by_me && !isPast && (
                      <button onClick={() => unclaim(s.id)} className="text-xs bg-brand-danger text-white font-medium px-2 py-1 rounded hover:bg-brand-danger/90 transition-colors">
                        Austragen
                      </button>
                    )}
                    {!hideClaimActions && !s.claimed_by_me && s.vacancies > 0 && !isPast && (
                      <button onClick={() => claim(s.id)} className="text-xs bg-brand-yellow text-brand-black font-medium px-2 py-1 rounded hover:bg-brand-black hover:text-brand-yellow transition-colors">
                        Eintragen
                      </button>
                    )}
                    {(() => {
                      const menuActions = [
                        ...(canEdit && onEdit ? [{ label: 'Bearbeiten', onClick: () => onEdit(s.id) }] : []),
                        ...(s.has_instruction ? [{ label: 'Anleitung', onClick: () => { onFocusSlot?.(s.id); navigate(`/dienste/anleitung/${s.duty_type_id}`) } }] : []),
                        ...(s.claimed_by_me && s.my_assignment_id ? [{ label: 'Kommentieren', onClick: () => openCommentModal(s) }] : []),
                      ]
                      return menuActions.length > 0 ? <ActionMenu actions={menuActions} /> : null
                    })()}
                  </div>
                  {/* Mobile ActionMenu. Löschen ist hier bewusst kein Eintrag —
                      Dienste werden ausschließlich über das Kalender-Modal
                      (Bearbeiten-Dialog) gelöscht, nie versehentlich aus dieser
                      Liste heraus. */}
                  <div className="sm:hidden">
                    {(() => {
                      const actions = [
                        ...(!hideClaimActions && !s.claimed_by_me && s.vacancies > 0 && !isPast ? [{ label: 'Eintragen', onClick: () => claim(s.id) }] : []),
                        ...(!hideClaimActions && s.claimed_by_me && !isPast ? [{ label: 'Austragen', onClick: () => unclaim(s.id), variant: 'danger' as const }] : []),
                        ...(canEdit && onEdit ? [{ label: 'Bearbeiten', onClick: () => onEdit(s.id) }] : []),
                        ...(s.has_instruction ? [{ label: 'Anleitung', onClick: () => { onFocusSlot?.(s.id); navigate(`/dienste/anleitung/${s.duty_type_id}`) } }] : []),
                        ...(s.claimed_by_me && s.my_assignment_id ? [{ label: 'Kommentieren', onClick: () => openCommentModal(s) }] : []),
                      ]
                      return actions.length > 0 ? <ActionMenu actions={actions} /> : null
                    })()}
                  </div>
                </td>
              </tr>
          )}
        />
      </table>
      </div>

      {noInstructionOpen && createPortal(
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div
            ref={noInstructionRef}
            role="dialog"
            aria-modal="true"
            aria-labelledby="duty-no-instruction-title"
            className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow transform-gpu p-6 max-w-sm w-full mx-4"
          >
            <h2 id="duty-no-instruction-title" className="text-lg font-bold mb-2 text-brand-text">Keine Anleitung</h2>
            <p className="text-sm text-brand-text-muted mb-4">
              Für diesen Dienst gibt es noch keine Anleitung.
            </p>
            <div className="flex justify-end">
              <button
                onClick={() => setNoInstructionOpen(false)}
                className="text-sm px-4 py-2 rounded bg-brand-yellow text-brand-black font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
              >
                OK
              </button>
            </div>
          </div>
        </div>,
        document.body,
      )}

      {claimDialog !== null && user && createPortal(
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div
            ref={claimDialogRef}
            role="dialog"
            aria-modal="true"
            aria-labelledby="duty-claim-dialog-title"
            className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow transform-gpu p-6 max-w-sm w-full mx-4"
          >
            <h2 id="duty-claim-dialog-title" className="text-lg font-bold mb-3 text-brand-text">Dienst übernehmen für…</h2>
            <div className="space-y-2 mb-4">
              <label className="flex items-center gap-3 p-2.5 rounded-lg border border-brand-border-subtle cursor-pointer hover:bg-brand-surface-card transition-colors">
                <input
                  type="radio"
                  name="claim-for"
                  value={user.id}
                  checked={claimDialog.selectedUserId === user.id}
                  onChange={() => setClaimDialog(d => d ? { ...d, selectedUserId: user.id } : d)}
                  className="accent-brand-yellow"
                />
                <span className="text-sm font-medium text-brand-text">Mich selbst</span>
              </label>
              {proxyChildren.map(child => (
                <label key={child.user_id} className="flex items-center gap-3 p-2.5 rounded-lg border border-brand-border-subtle cursor-pointer hover:bg-brand-surface-card transition-colors">
                  <input
                    type="radio"
                    name="claim-for"
                    value={child.user_id}
                    checked={claimDialog.selectedUserId === child.user_id}
                    onChange={() => setClaimDialog(d => d ? { ...d, selectedUserId: child.user_id } : d)}
                    className="accent-brand-yellow"
                  />
                  <span className="text-sm text-brand-text">{child.name}</span>
                </label>
              ))}
            </div>
            <div className="flex gap-2">
              <button
                disabled={claimDialog.selectedUserId === null || claimLoading}
                onClick={() => claimDialog.selectedUserId !== null && claimForUser(claimDialog.slotId, claimDialog.selectedUserId)}
                className={`${BTN_PRIMARY} flex-1`}
              >
                {claimLoading ? 'Eintragen…' : 'Eintragen'}
              </button>
              <button
                onClick={() => setClaimDialog(null)}
                className="flex-1 px-4 py-2 text-sm border border-brand-border rounded-md text-brand-text-muted hover:text-brand-text hover:border-brand-text-muted transition-colors"
              >
                Abbrechen
              </button>
            </div>
          </div>
        </div>,
        document.body,
      )}

      {commentModal !== null && createPortal(
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div
            ref={commentModalRef}
            role="dialog"
            aria-modal="true"
            aria-labelledby="duty-comment-modal-title"
            className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow transform-gpu p-6 max-w-sm w-full mx-4"
          >
            <h2 id="duty-comment-modal-title" className="text-lg font-bold mb-3 text-brand-text">Kommentare</h2>
            {commentModalLoading ? (
              <p className="text-sm text-brand-text-muted mb-4">Lädt…</p>
            ) : (
              <div className="space-y-3 mb-4 max-h-64 overflow-y-auto">
                {comments.length === 0 && (
                  <p className="text-sm text-brand-text-muted">Noch keine Kommentare.</p>
                )}
                {comments.map((c, i) => (
                  <div key={i} className="text-sm">
                    <div className="flex items-baseline justify-between gap-2">
                      <span className="font-medium text-brand-text">{c.user_name}</span>
                      <span className="text-xs text-brand-text-subtle whitespace-nowrap">{relativeTime(c.created_at)}</span>
                    </div>
                    <p className="text-brand-text-muted whitespace-pre-wrap">{c.body}</p>
                  </div>
                ))}
              </div>
            )}
            {commentModal.assignmentId && (
              <div className="border-t border-brand-border-subtle pt-3 mb-4">
                <label htmlFor="duty-comment-draft" className="block text-xs text-brand-text-muted mb-1">
                  Dein Kommentar
                </label>
                <textarea
                  id="duty-comment-draft"
                  value={myCommentDraft}
                  onChange={e => setMyCommentDraft(e.target.value)}
                  maxLength={280}
                  rows={3}
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                  placeholder="z.B. Marmorkuchen"
                />
              </div>
            )}
            <div className="flex gap-2">
              {commentModal.assignmentId && (
                <button
                  disabled={commentSaving || !myCommentDraft.trim()}
                  onClick={saveMyComment}
                  className={`${BTN_PRIMARY} flex-1`}
                >
                  {commentSaving ? 'Speichern…' : 'Speichern'}
                </button>
              )}
              {commentModal.assignmentId && comments.some(c => c.user_id === user?.id) && (
                <button
                  disabled={commentSaving}
                  onClick={deleteMyComment}
                  className={BTN_DANGER}
                >
                  Löschen
                </button>
              )}
              <button
                onClick={() => setCommentModal(null)}
                className="flex-1 px-4 py-2 text-sm border border-brand-border rounded-md text-brand-text-muted hover:text-brand-text hover:border-brand-text-muted transition-colors"
              >
                Schließen
              </button>
            </div>
          </div>
        </div>,
        document.body,
      )}
    </>
  )
}
