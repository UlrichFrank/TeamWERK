import { RefObject, useEffect, useRef } from 'react'

/**
 * Selektor für „fokussierbar" innerhalb eines Dialogs (openspec/changes/
 * betriebshaertung-welle-2, Decision 8 / specs/dialog-accessibility).
 */
const FOCUSABLE_SELECTOR =
  '[autofocus], input, select, textarea, button:not([disabled]), [href], [tabindex]:not([tabindex="-1"])'

function focusableElements(container: HTMLElement): HTMLElement[] {
  return Array.from(container.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR))
}

/**
 * Zugänglichkeits-Grundverhalten für modale Dialoge: merkt beim Öffnen den
 * Auslöser (`document.activeElement`), setzt den Fokus auf das erste
 * fokussierbare Element im Dialog (Fallback: der Container selbst,
 * `tabIndex=-1`), hält Tab/Shift+Tab zyklisch innerhalb des Dialogs
 * (Fokus-Trap) und gibt den Fokus beim Schließen/Unmount an den Auslöser
 * zurück (falls der noch im DOM steht). Escape bleibt bewusst bei
 * `useEscapeKey` — nicht duplizieren, dieser Hook kümmert sich nur um Fokus.
 *
 * **Muster bei bedingt gerenderten Modals** (`if (!isOpen) return null`):
 * der Hook wird trotzdem — wie jeder Hook — unbedingt vor dem frühen
 * `return` aufgerufen; `ref.current` ist dann `null`, solange der Dialog
 * nicht gerendert ist, und der Effekt greift erst, sobald der Container
 * tatsächlich im DOM steht (`isOpen=true` im gerenderten Zustand).
 */
export function useDialogA11y(ref: RefObject<HTMLElement | null>, isOpen: boolean) {
  const triggerRef = useRef<HTMLElement | null>(null)

  useEffect(() => {
    if (!isOpen) return
    const container = ref.current
    if (!container) return

    triggerRef.current = document.activeElement as HTMLElement | null

    // Ein Feld mit `autofocus` (React-Prop `autoFocus`) hat sich zu diesem
    // Zeitpunkt bereits selbst fokussiert — React wendet das synchron in der
    // Commit-Phase an, deutlich vor diesem (passiven) Effekt. Ein solches Feld
    // hat Vorrang vor dem generischen „erstes fokussierbares Element" (genau
    // dafür steht `[autofocus]` an erster Stelle im Selektor): wir greifen nur
    // ein, wenn noch NICHTS innerhalb des Dialogs den Fokus trägt.
    if (!container.contains(document.activeElement)) {
      const initial = focusableElements(container)[0]
      if (initial) {
        initial.focus()
      } else {
        container.tabIndex = -1
        container.focus()
      }
    }

    function onKeyDown(e: KeyboardEvent) {
      if (e.key !== 'Tab') return
      const items = focusableElements(container!)
      if (items.length === 0) {
        e.preventDefault()
        return
      }
      const first = items[0]
      const last = items[items.length - 1]
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault()
        last.focus()
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault()
        first.focus()
      }
    }

    container.addEventListener('keydown', onKeyDown)

    return () => {
      container.removeEventListener('keydown', onKeyDown)
      const trigger = triggerRef.current
      if (trigger && document.contains(trigger)) {
        trigger.focus()
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- ref ist ein stabiles RefObject, nur isOpen soll den Effekt erneut auslösen
  }, [isOpen])
}
