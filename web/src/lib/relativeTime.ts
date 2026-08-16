// Grob-relative deutsche Zeitangabe ("vor 5 Min.", "gestern", "vor 3 Monaten").
// Sekundengenauigkeit ist hier nicht nötig — die Grenzen sind bewusst grob
// (Monate à 30 Tage), das reicht für Aktivitäts-/Log-Anzeigen.

// SQLite liefert DATETIME-Spalten als "2026-08-16 15:44:12" — UTC, aber ohne
// Zonenkennung. `new Date(...)` liest so einen String als ORTSZEIT und
// verschiebt ihn damit um den lokalen Offset: ein gerade entstandener Eintrag
// erschiene im Sommer als "vor 2 Std.". Deshalb wird ein String ohne
// Zonenkennung explizit als UTC interpretiert.
//
// Strings, die bereits eine Zone tragen (RFC3339 aus Go-`time.Time`, z. B.
// "2026-08-16T15:44:12Z" oder "...+02:00"), bleiben unangetastet.
const HAS_TIMEZONE = /(?:Z|[+-]\d{2}:?\d{2})$/i

export function parseServerTime(raw: string): Date {
  const s = raw.trim()
  if (HAS_TIMEZONE.test(s)) return new Date(s)
  return new Date(s.replace(' ', 'T') + 'Z')
}

export function relativeTime(iso: string): string {
  const diff = Date.now() - parseServerTime(iso).getTime()
  const m = Math.floor(diff / 60000)
  if (m < 2) return 'gerade eben'
  if (m < 60) return `vor ${m} Min.`
  const h = Math.floor(m / 60)
  if (h < 24) return `vor ${h} Std.`
  const d = Math.floor(h / 24)
  if (d === 1) return 'gestern'
  if (d < 30) return `vor ${d} Tagen`
  const mo = Math.floor(d / 30)
  return mo === 1 ? 'vor 1 Monat' : `vor ${mo} Monaten`
}
