export function formatDuration(minutes: number): string {
  const total = Math.abs(Math.round(minutes))
  const days = Math.floor(total / 1440)
  const hours = Math.floor((total % 1440) / 60)
  const mins = total % 60
  const parts: string[] = []
  if (days) parts.push(`${days}d`)
  if (hours) parts.push(`${hours}h`)
  if (mins) parts.push(`${mins}min`)
  return parts.length ? parts.join(' ') : '0'
}

export function parseDuration(s: string): number {
  const clean = s.trim().replace(/^[+-]/, '').trim()
  const days = parseInt(clean.match(/(\d+)\s*d/)?.[1] ?? '0')
  const hours = parseInt(clean.match(/(\d+)\s*h/)?.[1] ?? '0')
  const mins = parseInt(clean.match(/(\d+)\s*min/)?.[1] ?? '0')
  if (days || hours || mins) return days * 1440 + hours * 60 + mins
  const n = parseInt(clean)
  return isNaN(n) ? 0 : Math.abs(n)
}

export function formatOffset(minutes: number): string {
  if (minutes === 0) return '0'
  return (minutes < 0 ? '-' : '+') + formatDuration(minutes)
}

export function parseOffset(s: string): number {
  const trimmed = s.trim()
  if (!trimmed || trimmed === '0') return 0
  const negative = trimmed.startsWith('-')
  const abs = parseDuration(trimmed)
  return negative ? -abs : abs
}
