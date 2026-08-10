import { useAuth } from '../contexts/AuthContext'

/**
 * Gemeinsame Zusatzfelder für alle Lösch-Bestätigungen von Terminen und Diensten
 * (Spiel, Trainingseinheit, Trainingsserie, Dienst-Slot).
 *
 * - „Grund (optional)" reist im Body des DELETE-Requests mit und landet
 *   ausschließlich im Benachrichtigungstext. Der Server trimmt und kürzt still
 *   auf 200 Zeichen — hier wird bewusst nicht validiert oder blockiert.
 * - Das Häkchen „Ohne Benachrichtigung löschen" hängt an der Capability
 *   `suppress_event_notification` (Vorstand + Admin) und NICHT an
 *   `manage_games`/`manage_trainings`: `manage_games` hat der Trainer ebenfalls,
 *   `manage_trainings` schließt den reinen Vorstand aus (siehe design.md §4).
 */

const CAP_SUPPRESS_EVENT_NOTIFICATION = 'suppress_event_notification'

const REASON_INPUT = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'

/**
 * Body des DELETE-Requests. `reason` wird immer mitgesendet — der leere String
 * ist serverseitig gleichbedeutend mit „kein Grund".
 */
// eslint-disable-next-line react-refresh/only-export-components -- reine Hilfsfunktion neben der Komponente; betrifft nur Dev-HMR
export function deletionPayload(reason: string, silent: boolean): { reason: string; silent: boolean } {
  return { reason: reason.trim(), silent }
}

interface Props {
  reason: string
  onReasonChange: (value: string) => void
  silent: boolean
  onSilentChange: (value: boolean) => void
  /** Eindeutiges Präfix für die Feld-IDs — mehrere Dialoge können gleichzeitig im DOM stehen. */
  idPrefix: string
}

export default function DeleteReasonFields({
  reason, onReasonChange, silent, onSilentChange, idPrefix,
}: Props) {
  const { hasCapability } = useAuth()
  const reasonId = `${idPrefix}-reason`
  const silentId = `${idPrefix}-silent`

  return (
    <div className="mb-4 text-left">
      <label htmlFor={reasonId} className="block text-sm font-medium text-brand-text-muted mb-1">
        Grund <span className="text-brand-text-subtle font-normal">(optional)</span>
      </label>
      <input
        id={reasonId}
        type="text"
        value={reason}
        onChange={e => onReasonChange(e.target.value)}
        placeholder="z. B. Halle gesperrt"
        className={REASON_INPUT}
      />
      {hasCapability(CAP_SUPPRESS_EVENT_NOTIFICATION) && (
        <label htmlFor={silentId} className="flex items-center gap-2 mt-2 py-2.5 sm:py-1.5 cursor-pointer">
          <input
            id={silentId}
            type="checkbox"
            checked={silent}
            onChange={e => onSilentChange(e.target.checked)}
            className="accent-brand-yellow"
          />
          <span className="text-sm text-brand-text">Ohne Benachrichtigung löschen</span>
        </label>
      )}
    </div>
  )
}
