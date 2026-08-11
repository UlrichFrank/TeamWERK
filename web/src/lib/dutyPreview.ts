/**
 * URL für die Dienst-Vorschau in Schritt 4 des Anlege-Wizards (KalenderPage).
 *
 * `team_ids` ist Pflicht für ein ehrliches Ergebnis: ohne die Teams zeigt der
 * Server auch Einträge, die auf andere Kaderteams eingeschränkt sind und real
 * gar keinen Slot erzeugen (er filtert bewusst nicht, wenn er die Teams nicht
 * kennt — siehe PreviewSlots in internal/games/handler.go).
 *
 * Liegt hier statt in KalenderPage.tsx, weil eine Nicht-Komponente im
 * Komponenten-Modul Fast Refresh aushebelt (react-refresh/only-export-components)
 * — und weil die Parameter-Logik so ohne Durchklicken des 4-Schritt-Wizards
 * testbar ist.
 */
export function buildPreviewUrl(opts: {
  templateId: number
  eventType: string
  time: string
  date: string
  endTime: string
  teamIds: number[]
}): string {
  const params = new URLSearchParams({ time: opts.time })
  if (opts.eventType === 'heim') params.set('date', opts.date)
  if (opts.eventType === 'generisch') params.set('end_time', opts.endTime)
  if (opts.teamIds.length > 0) params.set('team_ids', opts.teamIds.join(','))
  return `/duty-templates/${opts.templateId}/preview?${params.toString()}`
}
