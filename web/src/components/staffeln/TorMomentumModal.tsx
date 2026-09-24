import { useEffect, useId, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { X } from 'lucide-react'
import { fetchReport } from '../../lib/staffeln'
import type { MatrixGame, ReportDetail } from '../../lib/staffeln'
import { useEscapeKey } from '../../lib/useEscapeKey'
import { useDialogA11y } from '../../lib/useDialogA11y'
import TorMomentum from './TorMomentum'

/**
 * Der Ablauf einer Begegnung als Tor-Momentum.
 *
 * Lädt den Bericht erst beim Öffnen und über die vorhandene Route — Lauf,
 * Situation und Halbzeitgrenze sind Ableitungen über eine bereits gelieferte
 * Ereignisliste, dafür eine eigene Route zu bauen hieße, dieselben Zeilen ein
 * zweites Mal zu serialisieren (design.md §1).
 */
export default function TorMomentumModal({
  game, halfDurationMinutes, onClose,
}: {
  game: MatrixGame
  halfDurationMinutes: number | null
  onClose: () => void
}) {
  const titleId = useId()
  const dialogRef = useRef<HTMLDivElement>(null)
  useEscapeKey(onClose)
  useDialogA11y(dialogRef, true)
  const [report, setReport] = useState<ReportDetail | null>(null)
  const [state, setState] = useState<'laden' | 'da' | 'fehler'>('laden')

  useEffect(() => {
    let alive = true
    fetchReport(game.bwhvGameId)
      .then((r) => { if (alive) { setReport(r); setState('da') } })
      .catch(() => { if (alive) setState('fehler') })
    return () => { alive = false }
  }, [game.bwhvGameId])

  return createPortal(
    <div
      className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4"
      onClick={onClose}
    >
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow transform-gpu p-6 w-full max-w-3xl max-h-[90vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-4 mb-4">
          <div className="min-w-0">
            <h2 id={titleId} className="text-lg font-bold text-brand-text truncate">
              {game.homeTeam} <span className="text-brand-text-subtle">–</span> {game.guestTeam}
            </h2>
            <p className="text-sm text-brand-text-muted">
              {game.homeGoals !== null && `${game.homeGoals}:${game.guestGoals}`}
              {report?.homeGoalsHt !== null && report?.homeGoalsHt !== undefined && (
                <span className="text-brand-text-subtle">
                  {' '}({report.homeGoalsHt}:{report.guestGoalsHt})
                </span>
              )}
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label="Schließen"
            className="text-brand-text-muted hover:text-brand-text transition-colors shrink-0"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {state === 'laden' && <p className="text-sm text-brand-text-muted">Lade Spielverlauf…</p>}
        {state === 'fehler' && (
          <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
            Der Spielverlauf konnte nicht geladen werden.
          </div>
        )}
        {state === 'da' && report && (
          <TorMomentum
            events={report.events}
            homeTeam={report.homeTeam}
            guestTeam={report.guestTeam}
            homeGoalsHt={report.homeGoalsHt}
            guestGoalsHt={report.guestGoalsHt}
            halfDurationMinutes={halfDurationMinutes}
          />
        )}
      </div>
    </div>,
    document.body,
  )
}
