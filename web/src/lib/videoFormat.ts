// Formatierungs-Helfer für Spielvideos. In einem eigenen Modul, damit Seiten
// neben ihrer Default-Komponente keine zusätzlichen Exporte tragen
// (react-refresh/only-export-components).

export function fmtVideoDate(iso: string): string {
  const d = iso.slice(0, 10)
  const date = new Date(d + 'T12:00:00')
  if (isNaN(date.getTime())) return d
  return date.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric' })
}

export function fmtDuration(sec?: number | null): string {
  if (sec == null || sec <= 0) return '–'
  const m = Math.floor(sec / 60)
  const s = sec % 60
  return `${m}:${s.toString().padStart(2, '0')} min`
}
