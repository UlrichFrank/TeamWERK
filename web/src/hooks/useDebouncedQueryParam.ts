import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'

/**
 * Hält einen Textfilter lokal und schreibt ihn verzögert in die URL.
 *
 * Zwei Gründe für die Entkopplung (design.md §9): die History-API verträgt kein
 * Schreiben pro Tastenanschlag, und `useSearchParams` löst bei jedem Schreiben
 * einen Router-Re-Render aus, den die Liste beim Tippen nicht braucht. Die
 * Filterung selbst wirkt sofort aus dem lokalen State.
 *
 * `replace: true` ist die bestehende Konvention der Filter-Seiten — ohne sie
 * legte jedes getippte Zeichen einen History-Eintrag an.
 *
 * @returns [wert, setWert] — der Wert ist sofort aktuell, die URL zieht nach.
 */
export function useDebouncedQueryParam(key: string, delayMs = 250): [string, (v: string) => void] {
  const [searchParams, setSearchParams] = useSearchParams()
  const fromUrl = searchParams.get(key) ?? ''
  const [value, setValue] = useState(fromUrl)

  // Externe URL-Wechsel (Back/Forward, Deep-Link) übernehmen — aber nicht das
  // eigene, verzögerte Zurückschreiben. Das ist der von React empfohlene
  // "Adjusting state during render"-Fall: die Anpassung gehört in den
  // Render-Körper, nicht in einen Effekt (ein Effekt liefe erst nach einem
  // Commit mit dem alten Wert und erzeugte einen sichtbaren Zwischenzustand).
  // `prevUrl` merkt sich, welchen URL-Wert wir zuletzt übernommen haben —
  // ohne ihn wäre nicht unterscheidbar, ob sich die URL geändert hat oder
  // der Nutzer gerade tippt.
  const [prevUrl, setPrevUrl] = useState(fromUrl)
  if (fromUrl !== prevUrl) {
    setPrevUrl(fromUrl)
    setValue(fromUrl)
  }

  useEffect(() => {
    if (value === fromUrl) return
    const timer = setTimeout(() => {
      setSearchParams(prev => {
        const next = new URLSearchParams(prev)
        if (value === '') next.delete(key)
        else next.set(key, value)
        return next
      }, { replace: true })
    }, delayMs)
    return () => clearTimeout(timer)
    // fromUrl bewusst nicht in den Deps: der Effekt soll auf Tippen reagieren,
    // nicht auf das eigene Zurückschreiben.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value, key, delayMs])

  return [value, setValue]
}
