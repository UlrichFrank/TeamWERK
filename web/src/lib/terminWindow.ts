/**
 * Datumsfenster, mit dem `/termine` Trainings und Spiele lädt.
 *
 * Beide Grenzen sind bewusst großzügig: die Liste filtert danach nur noch
 * clientseitig (Typ, Mannschaft, Text), ein zu enges Fenster fehlt also nicht
 * als leere Trefferliste auf, sondern als lautlos fehlender Termin.
 */

/** Fenster der laufenden Saison (GET /api/seasons/active), reine "YYYY-MM-DD". */
export interface SeasonWindow {
  start_date: string
  end_date: string
}

/** Wie weit die Liste ohne Saisongrenze in jede Richtung reicht. */
const ROLLING_DAYS = 365

export function isoDaysFrom(now: Date, days: number): string {
  return new Date(now.getTime() + days * 24 * 60 * 60 * 1000).toISOString().slice(0, 10)
}

/**
 * Untere Grenze: heute, mit „Vergangene" das frühere von Saisonstart und
 * heute − 365 Tagen (der Saisonstart einer frisch aktivierten Saison liegt
 * hinter diesen 365 Tagen, Deep-Links auf Termine der Vorsaison bleiben
 * auflösbar).
 *
 * Obere Grenze: das **spätere** von Saisonende und heute + 365 Tagen. Das
 * Saisonende allein genügt nicht — `training_sessions` führt die Saison als
 * Spalte, nicht als Datumsbedingung, eine Serie darf also über das formale
 * Saisonende hinauslaufen. Genau das tun die Übungsgruppen-Serien; ihre letzten
 * Termine fehlten dadurch in der Liste, ohne dass etwas darauf hinwies.
 */
export function terminLoadWindow(
  season: SeasonWindow,
  showPast: boolean,
  now: Date = new Date(),
): { from: string; to: string } {
  const today = isoDaysFrom(now, 0)
  const pastFloor = isoDaysFrom(now, -ROLLING_DAYS)
  const futureHorizon = isoDaysFrom(now, ROLLING_DAYS)
  return {
    from: showPast ? (season.start_date < pastFloor ? season.start_date : pastFloor) : today,
    to: season.end_date > futureHorizon ? season.end_date : futureHorizon,
  }
}
