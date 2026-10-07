/**
 * Treffzeit eines Spiels (spiel-treffpunkt). Der Server speichert nur den
 * Abstand zum Anwurf und liefert die daraus berechnete Uhrzeit mit — das
 * Frontend rechnet nie selbst.
 */
export interface MeetingFields {
  meet_time: string | null
  meet_date: string | null
  meet_offset_minutes?: number | null
  meet_place: string
}

/** true, wenn das Treffen vor dem Spieltag liegt (nach einer Anwurf-Verlegung). */
export function meetingIsPreviousDay(gameDate: string, meetDate: string | null | undefined): boolean {
  if (!meetDate) return false
  return meetDate.slice(0, 10) < gameDate.slice(0, 10)
}

