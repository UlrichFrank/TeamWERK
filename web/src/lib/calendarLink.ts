/**
 * Ziel-URL „Im Kalender öffnen“ (kalender-sprung): der Monat kommt aus `date`,
 * der Termin aus `focus`. Das Datum wird auf `YYYY-MM-DD` gekürzt, weil die API
 * DATE-Felder als ISO-Timestamp liefert.
 */
export function calendarLink(kind: 'game' | 'training', id: number, date: string): string {
  return `/kalender?date=${date.slice(0, 10)}&focus=${kind}-${id}`
}
