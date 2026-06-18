const longDateFormatter = new Intl.DateTimeFormat('de-DE', {
  weekday: 'long',
  day: 'numeric',
  month: 'long',
  year: 'numeric',
})

function dayKey(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function diffDays(later: Date, earlier: Date): number {
  const a = new Date(later.getFullYear(), later.getMonth(), later.getDate())
  const b = new Date(earlier.getFullYear(), earlier.getMonth(), earlier.getDate())
  return Math.round((a.getTime() - b.getTime()) / 86_400_000)
}

export function daySeparatorLabel(date: Date, now: Date): string {
  const d = diffDays(now, date)
  if (d === 0) return 'Heute'
  if (d === 1) return 'Gestern'
  return longDateFormatter.format(date)
}

export function shouldRenderSeparator(prevSentAt: string | null, currentSentAt: string): boolean {
  if (prevSentAt === null) return true
  return dayKey(new Date(prevSentAt)) !== dayKey(new Date(currentSentAt))
}
