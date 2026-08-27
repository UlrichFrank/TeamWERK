/**
 * Geteilte Helfer für Dienst-Dauern (`hours_value`, REAL in Stunden) und die
 * daraus abgeleitete Zeitspannen-Anzeige.
 *
 * `hoursToDisplay`/`parseHoursInput` sind unverändert aus AdminDutyTypesPage.tsx
 * hierher verschoben (openspec/changes/dienst-dauer, Aufgabe 6.1) — keine
 * Verhaltensänderung, nur ein geteilter Ort, weil ab jetzt auch Vorlagen-Zeilen
 * und Slots eine eigene Dauer tragen.
 */

/** "1.5" (Stunden, REAL) -> "1h 30min". */
export function hoursToDisplay(h: number): string {
  // Guard gegen fehlendes/kaputtes hours_value: der Service Worker liefert
  // /api/* network-first, eine noch gecachte Antwort von VOR dem Deploy trägt
  // das Feld nicht. Ohne den Guard stünde "NaNmin" im Eingabefeld.
  if (!Number.isFinite(h)) return ''
  const totalMins = Math.round(h * 60)
  const hrs = Math.floor(totalMins / 60)
  const mins = totalMins % 60
  if (hrs === 0) return `${mins}min`
  if (mins === 0) return `${hrs}h`
  return `${hrs}h ${mins}min`
}

/** "1h 30min" -> 1.5 (Stunden, REAL). Fällt auf 1 zurück, wenn nichts erkennbar ist. */
export function parseHoursInput(s: string): number {
  const m = s.trim().match(/^(?:(\d+)h\s*)?(?:(\d+)min)?$/)
  if (m && (m[1] || m[2])) return (parseInt(m[1] || '0')) + parseInt(m[2] || '0') / 60
  const n = parseFloat(s)
  return isNaN(n) ? 1 : n
}

function minutesFromTime(time: string): number | null {
  const m = time.match(/^(\d{1,2}):(\d{2})/)
  if (!m) return null
  return parseInt(m[1], 10) * 60 + parseInt(m[2], 10)
}

function wrapMinutes(total: number): number {
  return ((total % 1440) + 1440) % 1440
}

/**
 * Addiert Minuten (auch negativ) auf eine "HH:MM"-Uhrzeit, Mitternachtsüberlauf
 * per Modulo 24h (kein Datumszusatz), Ergebnis immer zweistellig gepaddet —
 * für `<input type="time">`-Werte und API-Payloads. Frontend-Pendant zu
 * `addMinutes` in `internal/games/handler.go`.
 */
export function addMinutesToTime(time: string, minutes: number): string {
  const start = minutesFromTime(time)
  if (start === null) return time
  const total = wrapMinutes(start + Math.round(minutes))
  return `${String(Math.floor(total / 60)).padStart(2, '0')}:${String(total % 60).padStart(2, '0')}`
}

/**
 * Zeitspanne "Start–Ende" (Halbgeviertstrich, ohne "Uhr") aus Startzeit und
 * Dauer eines Dienst-Slots. Mitternachtsüberlauf ohne Datumszusatz
 * ("23:30–00:30", nie "24:30"); die Mitternachtsstunde bleibt zweistellig
 * ("00"), sonst keine führende Null ("8:00" statt "08:00").
 * Trägt der Slot keine Startzeit, bleibt der bisherige Platzhalter '—'
 * erhalten — eine Dauer ohne Startzeitpunkt ergibt keine Spanne.
 * (openspec/changes/dienst-dauer/specs/duties/spec.md)
 */
function formatClock(mins: number): string {
  const h = Math.floor(mins / 60)
  return `${h === 0 ? '00' : h}:${String(mins % 60).padStart(2, '0')}`
}

export function formatTimeSpan(eventTime: string | null | undefined, hours: number): string {
  const start = eventTime ? minutesFromTime(eventTime) : null
  if (start === null) return '—'
  // Fehlt die Dauer (gecachte Alt-Antwort, s. hoursToDisplay), wird der reine
  // Startzeitpunkt gezeigt statt "8:00–NaN:NaN". Eine Uhrzeit ohne Spanne ist
  // immer noch wahr; eine NaN-Spanne wäre es nicht.
  if (!Number.isFinite(hours) || hours <= 0) return formatClock(start)
  const end = wrapMinutes(start + Math.round(hours * 60))
  return `${formatClock(start)}–${formatClock(end)}`
}
