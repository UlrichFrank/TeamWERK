export const CLUB_FUNCTION_OPTIONS = [
  { value: 'spieler', label: 'Spieler' },
  { value: 'trainer', label: 'Trainer' },
  { value: 'sportliche_leitung', label: 'Sportliche Leitung' },
  { value: 'vorstand', label: 'Vorstand' },
  { value: 'vorstand_beisitzer', label: 'Vorstands-Beisitzer' },
  { value: 'kassierer', label: 'Kassierer' },
  { value: 'medien', label: 'Medien' },
] as const

export const AUDIENCE_OPTIONS = [
  { value: 'spieler', label: 'Spieler' },
  { value: 'trainer', label: 'Trainer' },
  { value: 'sportliche_leitung', label: 'Sportliche Leitung' },
  { value: 'vorstand', label: 'Vorstand' },
  { value: 'vorstand_beisitzer', label: 'Vorstands-Beisitzer' },
  { value: 'kassierer', label: 'Kassierer' },
  { value: 'medien', label: 'Medien' },
  { value: 'eltern', label: 'Eltern' },
] as const

export const AUDIENCE_LABELS: Record<string, string> = Object.fromEntries(
  AUDIENCE_OPTIONS.map(o => [o.value, o.label])
)
