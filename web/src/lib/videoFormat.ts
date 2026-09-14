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

// fmtBytes formatiert eine Byte-Zahl als GB/MB (1 Nachkommastelle) — für
// Videogröße und Speicherplatz-Anzeige (video-speicherplatz). null/undefined
// (Größe noch nicht bekannt, z.B. 'ready' vor dem Disk-Usage-Backfill) → "–".
export function fmtBytes(bytes?: number | null): string {
  if (bytes == null || bytes < 0) return '–'
  const gb = bytes / 1024 ** 3
  if (gb >= 1) return `${gb.toFixed(1)} GB`
  const mb = bytes / 1024 ** 2
  return `${mb.toFixed(1)} MB`
}
