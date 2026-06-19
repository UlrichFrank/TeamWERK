import { isAxiosError } from 'axios'

/**
 * Liefert eine menschenlesbare Fehlermeldung aus einem unbekannten Catch-Wert
 * (axios-Fehler oder generisch). Ersetzt das frühere `catch (e: any)` +
 * `e.response?.data`-Muster typsicher.
 */
export function errorMessage(e: unknown, fallback = 'Ein Fehler ist aufgetreten'): string {
  if (isAxiosError(e)) {
    const data = e.response?.data
    if (typeof data === 'string' && data) return data
    if (data && typeof data === 'object') {
      const obj = data as { error?: string; message?: string }
      if (obj.error) return obj.error
      if (obj.message) return obj.message
    }
    return e.message || fallback
  }
  if (e instanceof Error) return e.message || fallback
  return fallback
}

/** HTTP-Status eines axios-Fehlers, sonst undefined. */
export function errorStatus(e: unknown): number | undefined {
  return isAxiosError(e) ? e.response?.status : undefined
}

/** Response-`data` eines axios-Fehlers als locker typisiertes Objekt, sonst undefined. */
export function errorData<T = Record<string, unknown>>(e: unknown): T | undefined {
  return isAxiosError(e) ? (e.response?.data as T | undefined) : undefined
}
