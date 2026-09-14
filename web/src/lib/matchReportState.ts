export type MatchReportState = 'draft' | 'pending_review' | 'publishing' | 'published' | 'publish_failed'

export const MATCH_REPORT_STATE_LABEL: Record<MatchReportState, string> = {
  draft: 'Entwurf',
  pending_review: 'Wartet auf Freigabe',
  publishing: 'Wird veröffentlicht…',
  published: 'Veröffentlicht',
  publish_failed: 'Fehler',
}
