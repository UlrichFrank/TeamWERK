import { createContext, useCallback, useContext } from 'react'
import { useNavigate } from 'react-router-dom'

/**
 * History-Index, bei dem die App-Sitzung begann — gesetzt von `AppShell`.
 *
 * `window.history.length` taugt für „Gibt es ein Zurück?" nicht: es zählt auch
 * Einträge vor dem App-Start (fremde Seiten, der Login, der Tab-Vorlauf beim
 * Öffnen per Push/Deep-Link). Ein `navigate(-1)` verlässt dann die App, und
 * übrig bleibt eine leere Seite. React Router führt in `history.state.idx`
 * einen eigenen Zähler — nur der ist verlässlich.
 */
export const HistoryBaseContext = createContext(0)

export function historyIdx(): number {
  const idx = (window.history.state as { idx?: unknown } | null)?.idx
  return typeof idx === 'number' ? idx : 0
}

/**
 * Zurück innerhalb der App, sonst Ersatz-Navigation auf `fallbackPath`
 * (ersetzend, damit der Deep-Link-Eintrag nicht im Verlauf stehen bleibt).
 */
export function useGoBack(fallbackPath: string): () => void {
  const base = useContext(HistoryBaseContext)
  const navigate = useNavigate()
  return useCallback(() => {
    if (historyIdx() > base) navigate(-1)
    else navigate(fallbackPath, { replace: true })
  }, [base, navigate, fallbackPath])
}
