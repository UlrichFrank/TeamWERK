import { isAxiosError } from 'axios'

/**
 * Übersetzung der maschinenlesbaren Fehlercodes, die das Backend seit der
 * httpx-Migration (`internal/httpx`) als `{"error": "<code>"}` liefert.
 *
 * Zweistufig: oben die generischen Codes aus `httpx`, unten die fachlichen aus
 * `games`/`trainings`/`kader`. Codes, die hier fehlen, werden unverändert
 * durchgereicht — Komponenten mit eigener, genauerer Zuordnung
 * (DutyBulkRegenModal, GameDayHostPicker, H4AImportModal) prüfen ohnehin vorher
 * selbst und rufen `errorMessage` nur als Fallback.
 */
const CODE_MESSAGES: Record<string, string> = {
  // Generisch (httpx)
  internal: 'Interner Fehler, bitte später erneut versuchen',
  invalid_id: 'Ungültige ID',
  invalid_body: 'Ungültige Anfrage',
  not_found: 'Nicht gefunden',
  forbidden: 'Keine Berechtigung',
  conflict: 'Der Datensatz existiert bereits',
  validation: 'Eingabe ungültig',
  unauthorized: 'Nicht angemeldet',

  // Fachlich (games / trainings / kader)
  no_active_season: 'Keine aktive Saison eingestellt',
  no_member_record: 'Dieser Account ist keinem Mitglied zugeordnet',
  attendance_window: 'Anwesenheit lässt sich erst ab dem Termintag erfassen',
  rsvp_locked_absence: 'Die Rückmeldung ist durch eine Abwesenheit gesperrt',
  series_unavailable: 'Für diese Terminserie abgemeldet',
  note_too_long: 'Die Notiz ist zu lang',
  kader_not_found: 'Der Kader existiert nicht (mehr)',
  unknown_member: 'Das Mitglied existiert nicht (mehr)',
  target_season_not_found: 'Die Ziel-Saison existiert nicht (mehr)',
  season_id_required: 'Bitte eine Saison wählen',
  season_ids_required: 'Bitte Quell- und Ziel-Saison wählen',
  kader_ids_required: 'Bitte mindestens einen Kader wählen',
  team_ids_required: 'Bitte mindestens eine Mannschaft wählen',
  member_id_required: 'Bitte ein Mitglied wählen',
  date_required: 'Bitte ein Datum angeben',
  time_required: 'Bitte eine Uhrzeit angeben',
  from_date_required: 'Bitte ein Startdatum angeben',
  invalid_from_date: 'Ungültiges Startdatum',
  invalid_valid_from: 'Ungültiges Datum „gültig ab"',
  invalid_valid_until: 'Ungültiges Datum „gültig bis"',
  invalid_rsvp_default: 'Ungültige Rückmelde-Voreinstellung',
  invalid_rsvp_default_players: 'Ungültige Rückmelde-Voreinstellung für Spieler',
  invalid_rsvp_default_extended: 'Ungültige Rückmelde-Voreinstellung für den erweiterten Kader',
  invalid_template_id: 'Ungültige Vorlage',
  invalid_template_type: 'Ungültiger Vorlagentyp',
  invalid_event_type: 'Ungültige Terminart',
  invalid_duty_type_id: 'Ungültiger Diensttyp',
}

/**
 * Liefert eine menschenlesbare Fehlermeldung aus einem unbekannten Catch-Wert
 * (axios-Fehler oder generisch). Ersetzt das frühere `catch (e: any)` +
 * `e.response?.data`-Muster typsicher.
 *
 * Bevorzugt wird `data.error` — das Backend antwortet dort mit einem
 * maschinenlesbaren Code, der über CODE_MESSAGES übersetzt wird. Unbekannte
 * Codes und Plain-Text-Bodies (noch nicht migrierte Domänen) bleiben
 * unverändert.
 */
export function errorMessage(e: unknown, fallback = 'Ein Fehler ist aufgetreten'): string {
  if (isAxiosError(e)) {
    const data = e.response?.data
    if (data && typeof data === 'object') {
      const obj = data as { error?: string; message?: string }
      if (obj.error) return CODE_MESSAGES[obj.error] ?? obj.error
      if (obj.message) return obj.message
    }
    if (typeof data === 'string' && data) return data
    return e.message || fallback
  }
  if (e instanceof Error) return e.message || fallback
  return fallback
}

/** Übersetzt einen Backend-Fehlercode, sofern bekannt; sonst undefined. */
export function errorCodeMessage(code: string | undefined): string | undefined {
  return code ? CODE_MESSAGES[code] : undefined
}

/** HTTP-Status eines axios-Fehlers, sonst undefined. */
export function errorStatus(e: unknown): number | undefined {
  return isAxiosError(e) ? e.response?.status : undefined
}

/** Response-`data` eines axios-Fehlers als locker typisiertes Objekt, sonst undefined. */
export function errorData<T = Record<string, unknown>>(e: unknown): T | undefined {
  return isAxiosError(e) ? (e.response?.data as T | undefined) : undefined
}
