import { useEffect, type RefObject } from 'react'

/**
 * Schließt ein offenes Dropdown, sobald der Nutzer daneben klickt/tippt oder
 * zu scrollen beginnt.
 *
 * Der Scroll-Teil ist kein Komfort, sondern nötig: die Filter-Dropdowns der
 * Kopfzeile liegen als `absolute`-Overlay über der Liste und „schlucken" dort
 * Touch-/Scroll-Gesten — die Seite wirkt unscrollbar, solange eines offen ist.
 * Nach `touchstart`/Scroll-Beginn ist das Overlay weg und die Geste scrollt
 * normal weiter. `capture: true` fängt das nicht-bubbelnde scroll-Event des
 * `<main>`-Containers (AppShell, `overflow-auto`) mit ab.
 */
export function useDismissOnOutside(
  open: boolean,
  ref: RefObject<HTMLElement | null>,
  close: () => void,
) {
  useEffect(() => {
    if (!open) return
    const closeIfOutside = (e: Event) => {
      if (ref.current && !ref.current.contains(e.target as Node)) close()
    }
    const closeOnScroll = () => close()
    document.addEventListener('mousedown', closeIfOutside)
    document.addEventListener('touchstart', closeIfOutside, { passive: true })
    window.addEventListener('scroll', closeOnScroll, { capture: true, passive: true })
    return () => {
      document.removeEventListener('mousedown', closeIfOutside)
      document.removeEventListener('touchstart', closeIfOutside)
      window.removeEventListener('scroll', closeOnScroll, { capture: true })
    }
    // `close` kommt als stabile Setter-Referenz (useState) aus den Aufrufern.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])
}
