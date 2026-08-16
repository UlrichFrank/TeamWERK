// Gemeinsame Typen und Konstanten des Trainingstagebuchs.

export const DIARY_KINDS = [
  { value: 'kraft', label: 'Kraft' },
  { value: 'ausdauer', label: 'Ausdauer' },
  { value: 'athletik', label: 'Athletik' },
  { value: 'technik', label: 'Technik' },
  { value: 'beweglichkeit', label: 'Beweglichkeit' },
  { value: 'reha', label: 'Reha' },
  { value: 'sonstiges', label: 'Sonstiges' },
] as const

export type DiaryKind = (typeof DIARY_KINDS)[number]['value']

// 'none' = nie einer da, 'present' = abrufbar, 'purged' = von der Retention
// gelöscht. Der dritte Zustand verhindert einen Bildabruf, der zwangsläufig
// 410 liefern würde.
export type ProofStatus = 'none' | 'present' | 'purged'

export interface DiaryEntry {
  id: number
  member_id: number
  season_id: number | null
  trained_on: string
  kind: DiaryKind
  kind_custom: string | null
  duration_min: number
  rpe: number
  note: string | null
  proof_status: ProofStatus
  proof_mime: string | null
  proof_purged_at: string | null
  created_at: string
  updated_at: string
}

export interface DiaryMemberSummary {
  member_id: number
  member_name: string
  user_id?: number
  entries: number
  minutes: number
  avg_rpe: number
}

export interface DiaryTeamStats {
  season_id: number
  start_date: string
  end_date: string
  items: DiaryMemberSummary[]
}

// Kompressionsbudget für Nachweisbilder — deutlich enger als im Chat
// (1 MB / 1920 px), weil ein Beleg nur lesbar sein muss, nicht schön.
export const PROOF_TARGET_BYTES = 150 * 1024
export const PROOF_MAX_EDGE = 1280

export const RETENTION_HINT =
  'Nachweise werden 90 Tage nach Saisonende automatisch gelöscht.'

export function kindLabel(entry: Pick<DiaryEntry, 'kind' | 'kind_custom'>): string {
  if (entry.kind === 'sonstiges') return entry.kind_custom || 'Sonstiges'
  return DIARY_KINDS.find(k => k.value === entry.kind)?.label ?? entry.kind
}

// API liefert Datumsfelder als ISO-Timestamp (siehe Gotcha „SQLite DATE-Felder").
export function fmtDate(iso: string): string {
  const parts = iso.slice(0, 10).split('-')
  return parts.length === 3 ? `${parts[2]}.${parts[1]}.${parts[0]}` : iso
}

export function todayISO(): string {
  return new Date().toISOString().slice(0, 10)
}

export function isImageMime(mime: string | null): boolean {
  return !!mime && mime.startsWith('image/')
}
