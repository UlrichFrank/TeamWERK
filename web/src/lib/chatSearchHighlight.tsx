import type { ReactNode } from 'react'

/** Sonderzeichen für den regulären Ausdruck escapen, bevor `q` als Suchmuster dient. */
function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/**
 * Hebt alle case-insensitiven Vorkommen von `query` in `text` per `<mark>` hervor.
 * Exportiert für den Poison-/Einzeltest unabhängig vom gemounteten Modal.
 */
export function highlight(text: string, query: string): ReactNode[] {
  const trimmed = query.trim()
  if (!trimmed) return [text]

  const re = new RegExp(`(${escapeRegExp(trimmed)})`, 'gi')
  const parts = text.split(re)
  // Ungerade Indizes sind die Fundstellen (Klammer-Capture-Group von `split`).
  return parts.map((part, i) =>
    i % 2 === 1 ? (
      <mark key={i} className="bg-brand-yellow/60 text-brand-black rounded-sm px-0.5">
        {part}
      </mark>
    ) : (
      part
    ),
  )
}
