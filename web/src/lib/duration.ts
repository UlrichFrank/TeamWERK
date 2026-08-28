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

/** "HH:MM" -> Minuten seit Mitternacht, oder null bei unparsbarem Wert. */
export function minutesFromTime(time: string): number | null {
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

/**
 * Löst einen Anker ("start"|"end") + Versatz gegen eine konkrete (oder
 * Beispiel-)Anpfiff-/Spielende-Uhrzeit auf. Frontend-Spiegel von
 * `resolveAnchorTime` in `internal/games/regen.go` — bewusst ohne dessen
 * Fallunterscheidung "gepflegte `games.end_time` vs. Anpfiff + Spieldauer":
 * der Aufrufer entscheidet, was er als `endTime` übergibt (echte Endzeit,
 * Beispielwert, oder als Fallback dieselbe Startzeit).
 * (openspec/changes/dienst-dauer-dynamisch)
 */
export function resolveAnchorClock(
  anchor: 'start' | 'end',
  offsetMinutes: number,
  startTime: string,
  endTime: string,
): string {
  const base = anchor === 'end' ? endTime : startTime
  return addMinutesToTime(base, offsetMinutes)
}

/**
 * Differenz zweier "HH:MM"-Uhrzeiten in Minuten (Ende minus Start). Kann
 * negativ sein — das ist der Signalwert für "Ende liegt vor dem Start". Der
 * Server erzeugt in diesem Fall keinen Slot mehr (dienst-zeitmodus-strikt, der
 * Rückfall auf `hours_value` ist entfallen); im Frontend nutzt das nur noch die
 * Dauer-Vorbelegung im Termin-Dialog, deren Ergebnis der Nutzer sieht und
 * ändern kann. Kein Mitternachtsüberlauf (Tagesraster, wie die Server-Auflösung).
 */
export function clockDiffMinutes(start: string, end: string): number | null {
  const s = minutesFromTime(start)
  const e = minutesFromTime(end)
  if (s === null || e === null) return null
  return e - s
}

/**
 * Meldet eine Anker-/Versatz-Kombination, deren Ende an KEINEM Termin nach dem
 * Start liegen kann. Frontend-Spiegel von `dynamicSpanImpossible` in
 * `internal/duties/handler.go` bzw. `internal/games/handler.go`.
 *
 * Hängen Start und Ende am selben Anker, ist die Dauer exakt die
 * Versatz-Differenz — unabhängig von der Spieldauer, also schon in der Maske
 * entscheidbar. Bei verschiedenen Ankern hängt sie an der Spieldauer des
 * konkreten Termins und wird bewusst NICHT geprüft: „Start bei Anpfiff, Ende
 * 15 min vor Spielende" ist gültig und ergibt bei jedem hinreichend langen
 * Spiel eine positive Dauer.
 * (openspec/changes/dienst-zeitmodus-strikt)
 */
export function dynamicSpanImpossible(
  mode: 'absolut' | 'dynamisch',
  anchor: 'start' | 'end',
  offsetMinutes: number,
  endAnchor: 'start' | 'end',
  endOffsetMinutes: number,
): boolean {
  if (mode !== 'dynamisch') return false
  return anchor === endAnchor && endOffsetMinutes <= offsetMinutes
}

/** Meldungstext für eine unmögliche Spanne — in beiden Masken derselbe Satz. */
export const IMPOSSIBLE_SPAN_MESSAGE =
  'Bei gleichem Start- und End-Anker muss der End-Versatz hinter dem Start-Versatz liegen — sonst endet der Dienst vor seinem Beginn.'
