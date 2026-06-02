export const CLUB_FUNCTION_OPTIONS = [
  { value: 'spieler', label: 'Spieler' },
  { value: 'trainer', label: 'Trainer' },
  { value: 'vorstand', label: 'Vorstand' },
  { value: 'vorstand_beisitzer', label: 'Vorstands-Beisitzer' },
] as const

export const AUDIENCE_OPTIONS = [
  { value: 'spieler', label: 'Spieler' },
  { value: 'trainer', label: 'Trainer' },
  { value: 'vorstand', label: 'Vorstand' },
  { value: 'vorstand_beisitzer', label: 'Vorstands-Beisitzer' },
  { value: 'eltern', label: 'Eltern' },
] as const

export const AUDIENCE_LABELS: Record<string, string> = Object.fromEntries(
  AUDIENCE_OPTIONS.map(o => [o.value, o.label])
)
