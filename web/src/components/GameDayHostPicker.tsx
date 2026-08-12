import { useCallback, useEffect, useState } from 'react'
import { AlertTriangle, X } from 'lucide-react'
import { api } from '../lib/api'
import { errorData, errorMessage } from '../lib/errors'
import { useEscapeKey } from '../lib/useEscapeKey'
import { useLiveUpdates } from '../hooks/useLiveUpdates'

// Tages-Ausrichter im Kalender (heimspieltag-ausrichter, design.md Decision 9/10).
//
// Der Ausrichter ist eine Eigenschaft des SPIELTAGS, sitzt in der Oberfläche aber
// zwischen lauter Termin-Feldern — im Detail-Modal eines einzelnen Heimspiels und
// im Termin-Wizard. Beides ist dieselbe UX-Falle: eine Änderung wirkt auf alle
// Termine des Tages, inklusive bestehender mit laufenden Zusagen. Die Gegenmittel
// stecken deshalb hier in der Komponente und nicht in den Aufrufern:
//
//   1. Die Beschriftung nennt immer den Tag ("Ausrichter am 14.09.") und sagt
//      ausdrücklich, dass der Wert für alle Termine des Tages gilt.
//   2. Jede Änderung läuft über den serverseitigen Dry-Run und zeigt die Bilanz,
//      bevor irgendetwas geschrieben wird.
//
// Es gibt bewusst KEINEN Client-Nachbau der Regen-Logik: created/deleted/
// assignments_lost kommen aus POST /api/game-days/host/preview, das denselben
// Codepfad wie das Anwenden nutzt und nur mit Rollback statt Commit endet.

export interface Ausrichter {
  id: number
  name: string
  aktiv: boolean
  is_default: boolean
  sort_order: number
}

export interface GameDayHostBalance {
  created: number
  deleted: number
  assignments_kept: number
  assignments_lost: number
  slots_before: number
  slots_after: number
  assignments_before: number
  assignments_after: number
}

export interface GameDayHost {
  date: string
  ausrichter_id: number
  ausrichter_name: string
  is_explicit: boolean
  balance?: GameDayHostBalance
  applied?: boolean
}

const SELECT = 'w-full border border-brand-border rounded-md px-3 py-2.5 sm:py-2 text-sm text-brand-text bg-white focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow disabled:opacity-40'
const BTN_PRIMARY = 'bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed'
const BTN_SECONDARY = 'px-4 py-2.5 sm:py-2 border border-brand-border rounded-md text-sm text-brand-text-muted hover:text-brand-text hover:bg-brand-border-subtle transition-colors disabled:opacity-40 disabled:cursor-not-allowed'

/** "2026-09-14" → "14.09." — die Kurzform für die tagesbezogene Beschriftung. */
export function formatDayShort(date: string): string {
  const iso = date.slice(0, 10)
  if (iso.length < 10) return iso
  return `${iso.slice(8, 10)}.${iso.slice(5, 7)}.`
}

export function gameDayHostLabel(date: string): string {
  return `Ausrichter am ${formatDayShort(date)}`
}

export const GAME_DAY_HOST_HINT = 'Gilt für alle Termine dieses Tages.'

/** Lädt die Ausrichter-Liste (Authenticated-Tier, nur aktive Einträge). */
export function useAusrichterOptions(enabled = true): Ausrichter[] {
  const [options, setOptions] = useState<Ausrichter[]>([])
  useEffect(() => {
    if (!enabled) return
    let cancelled = false
    api.get<{ items?: Ausrichter[] }>('/ausrichter')
      .then(r => { if (!cancelled) setOptions(r.data?.items ?? []) })
      .catch(() => { if (!cancelled) setOptions([]) })
    return () => { cancelled = true }
  }, [enabled])
  return options
}

export async function fetchGameDayHost(date: string): Promise<GameDayHost> {
  const r = await api.get<GameDayHost>(`/game-days/${date.slice(0, 10)}/host`)
  return r.data
}

export async function previewGameDayHost(date: string, ausrichterId: number): Promise<GameDayHost> {
  const r = await api.post<GameDayHost>('/game-days/host/preview', { date: date.slice(0, 10), ausrichter_id: ausrichterId })
  return r.data
}

export async function applyGameDayHost(date: string, ausrichterId: number): Promise<GameDayHost> {
  const r = await api.post<GameDayHost>('/game-days/host/apply', { date: date.slice(0, 10), ausrichter_id: ausrichterId })
  return r.data
}

export function describeHostError(err: unknown): string {
  const code = errorData<{ error?: string }>(err)?.error
  if (code === 'unknown_ausrichter') return 'Der gewählte Ausrichter existiert nicht (mehr).'
  if (code === 'inactive_ausrichter') return 'Der gewählte Ausrichter ist deaktiviert.'
  if (code === 'no_active_season') return 'Keine aktive Saison eingestellt.'
  if (code === 'no_default_ausrichter') return 'Es ist kein Standard-Ausrichter gesetzt (Einstellungen → Heimspieltage).'
  return errorMessage(err, 'Der Ausrichter konnte nicht geändert werden.')
}

/**
 * Reines Auswahlfeld — tagesbezogen beschriftet. Wird sowohl vom Detail-Modal
 * (sofortiges Schreiben über die Vorschau) als auch vom Wizard (Wert wird erst
 * beim Anlegen angewandt) genutzt.
 */
export function GameDayHostSelect({ id, date, value, options, disabled, onChange }: {
  id: string
  date: string
  value: number | null
  options: Ausrichter[]
  disabled?: boolean
  onChange: (v: number) => void
}) {
  return (
    <div>
      <label htmlFor={id} className="block text-sm font-medium text-brand-text-muted mb-1">
        {gameDayHostLabel(date)}
      </label>
      <select
        id={id}
        className={SELECT}
        value={value ?? ''}
        disabled={disabled}
        onChange={e => { if (e.target.value !== '') onChange(Number(e.target.value)) }}
      >
        {value == null && <option value="">Auswählen…</option>}
        {options.map(o => (
          <option key={o.id} value={o.id}>{o.name}{o.is_default ? ' (Standard)' : ''}</option>
        ))}
      </select>
      <p className="text-xs text-brand-text-subtle mt-1">{GAME_DAY_HOST_HINT}</p>
    </div>
  )
}

/**
 * Bestätigungsdialog mit der Bilanz aus dem Dry-Run. Der wichtige Wert ist
 * assignments_lost: entfallende Dienste sind ärgerlich, verlorene Zusagen
 * betreffen namentlich Personen, die zugesagt hatten.
 */
export function GameDayHostPreviewDialog({ preview, targetName, busy, error, onCancel, onConfirm }: {
  preview: GameDayHost
  targetName: string
  busy?: boolean
  error?: string | null
  onCancel: () => void
  onConfirm: () => void
}) {
  useEscapeKey(busy ? null : onCancel)
  const b = preview.balance
  const destructive = (b?.assignments_lost ?? 0) > 0
  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center p-4">
      <div className="fixed inset-0 bg-brand-black/50" onClick={busy ? undefined : onCancel} />
      <div className="relative bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-md">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold text-brand-text">Ausrichter wechseln?</h2>
          <button onClick={onCancel} disabled={busy} aria-label="Schließen" className="text-brand-text-muted hover:text-brand-text transition-colors">
            <X className="w-5 h-5" />
          </button>
        </div>

        <p className="text-sm text-brand-text mb-3">
          Am {formatDayShort(preview.date)} richtet künftig <strong className="font-semibold">{targetName}</strong> aus.
          Das gilt für alle Termine dieses Tages.
        </p>

        <ul className="text-sm text-brand-text-muted space-y-1 mb-3">
          <li>{b?.created ?? 0} Dienste kommen hinzu</li>
          <li>{b?.deleted ?? 0} Dienste entfallen</li>
          <li>{b?.assignments_kept ?? 0} Zuweisungen bleiben erhalten</li>
        </ul>

        {destructive && (
          <div className="flex items-start gap-2 p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger mb-3">
            <AlertTriangle className="w-4 h-4 mt-0.5 shrink-0" />
            <span>{b?.assignments_lost} Zuweisung{b?.assignments_lost === 1 ? '' : 'en'} gehen verloren — die Betroffenen werden benachrichtigt.</span>
          </div>
        )}

        {error && (
          <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger mb-3">{error}</div>
        )}

        <div className="flex gap-2 justify-end">
          <button onClick={onCancel} disabled={busy} className={BTN_SECONDARY}>Abbrechen</button>
          <button onClick={onConfirm} disabled={busy} className={BTN_PRIMARY}>
            {busy ? 'Wird übernommen…' : 'Übernehmen'}
          </button>
        </div>
      </div>
    </div>
  )
}

/**
 * Ausrichter-Abschnitt für das Termin-Detail-Modal: lädt den geltenden Wert,
 * zeigt an, ob er explizit gesetzt oder vom Standard geerbt ist, und schreibt
 * eine Änderung erst nach der Vorschau.
 */
export default function GameDayHostSection({ date, canEdit, onApplied }: {
  date: string
  canEdit: boolean
  onApplied?: () => void
}) {
  const options = useAusrichterOptions()
  const [host, setHost] = useState<GameDayHost | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [pending, setPending] = useState<{ id: number; preview: GameDayHost } | null>(null)
  const [busy, setBusy] = useState(false)
  const [dialogError, setDialogError] = useState<string | null>(null)

  const reload = useCallback(() => {
    fetchGameDayHost(date)
      .then(h => { setHost(h); setLoadError(null) })
      .catch(err => { setHost(null); setLoadError(describeHostError(err)) })
  }, [date])

  useEffect(() => { reload() }, [reload])

  // Der Tageswert ist gemeinsamer Zustand: ein anderer Vorstand kann ihn im
  // Kalender oder im Massenlauf umstellen, während dieses Modal offen ist.
  // Apply broadcastet "duties"+"games", der Ausrichter-Wechsel im Massenlauf
  // ebenso; "settings-changed" deckt Umbenennungen der Liste ab.
  useLiveUpdates(event => {
    if (event === 'games' || event === 'duties' || event === 'settings-changed') reload()
  })

  async function choose(id: number) {
    if (!host || id === host.ausrichter_id) return
    setBusy(true)
    setDialogError(null)
    try {
      const preview = await previewGameDayHost(date, id)
      setPending({ id, preview })
    } catch (err) {
      setLoadError(describeHostError(err))
    } finally {
      setBusy(false)
    }
  }

  async function confirm() {
    if (!pending) return
    setBusy(true)
    setDialogError(null)
    try {
      const applied = await applyGameDayHost(date, pending.id)
      setHost(applied)
      setPending(null)
      onApplied?.()
    } catch (err) {
      setDialogError(describeHostError(err))
    } finally {
      setBusy(false)
    }
  }

  if (loadError) {
    return <p className="text-xs text-brand-danger">{loadError}</p>
  }
  if (!host) return null

  const targetName = options.find(o => o.id === pending?.id)?.name ?? ''

  return (
    <div className="pt-3 border-t border-brand-border-subtle">
      {canEdit ? (
        <GameDayHostSelect
          id="game-day-host"
          date={date}
          value={host.ausrichter_id}
          options={options}
          disabled={busy}
          onChange={choose}
        />
      ) : (
        <div className="flex justify-between">
          <span className="text-brand-text-muted">{gameDayHostLabel(date)}</span>
          <span className="font-medium text-brand-text">{host.ausrichter_name}</span>
        </div>
      )}
      <p className="text-xs text-brand-text-subtle mt-1">
        {host.is_explicit ? 'Für diesen Tag festgelegt.' : 'Vom Standard geerbt.'}
      </p>

      {pending && (
        <GameDayHostPreviewDialog
          preview={pending.preview}
          targetName={targetName}
          busy={busy}
          error={dialogError}
          onCancel={() => { setPending(null); setDialogError(null) }}
          onConfirm={confirm}
        />
      )}
    </div>
  )
}
