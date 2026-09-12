import { useState } from 'react'
import { describe, test, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import EditModal from '../EditModal'

function ToggleWrapper() {
  const [open, setOpen] = useState(false)
  return (
    <>
      <button onClick={() => setOpen(true)}>Öffnen</button>
      <EditModal isOpen={open} title="Titel" onClose={() => setOpen(false)} onSave={() => {}}>
        <input aria-label="Feld" />
      </EditModal>
    </>
  )
}

describe('EditModal', () => {
  test('trägt role=dialog, aria-modal und aria-labelledby auf den Titel', () => {
    render(
      <EditModal isOpen title="Mitglied bearbeiten" onClose={() => {}} onSave={() => {}}>
        <input aria-label="Name" />
      </EditModal>
    )

    const dialog = screen.getByRole('dialog')
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    const labelledBy = dialog.getAttribute('aria-labelledby')
    expect(labelledBy).toBeTruthy()
    expect(document.getElementById(labelledBy!)).toHaveTextContent('Mitglied bearbeiten')
  })

  test('Fokus liegt initial im Dialog (erstes fokussierbares Element: Schließen-Button)', () => {
    render(
      <EditModal isOpen title="Mitglied bearbeiten" onClose={() => {}} onSave={() => {}}>
        <input aria-label="Name" />
      </EditModal>
    )

    // Der Schließen-Button im Header steht im DOM vor den Formularfeldern und
    // ist damit das erste fokussierbare Element — kein Ableitungsfehler,
    // sondern die generische „erstes Element im Dialog"-Regel aus der Spec.
    expect(screen.getByLabelText('Schließen')).toHaveFocus()
  })

  test('Tab-Trap: vom letzten Element (Speichern) zyklisch zurück zum ersten (Schließen)', () => {
    render(
      <EditModal isOpen title="Mitglied bearbeiten" onClose={() => {}} onSave={() => {}}>
        <input aria-label="Name" />
      </EditModal>
    )

    screen.getByRole('button', { name: 'Speichern' }).focus()
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Tab' })

    expect(screen.getByLabelText('Schließen')).toHaveFocus()
  })

  test('Fokus kehrt beim Schließen zum Auslöser zurück', () => {
    render(<ToggleWrapper />)
    const opener = screen.getByText('Öffnen')
    opener.focus()
    fireEvent.click(opener)
    expect(opener).not.toHaveFocus()
    expect(screen.getByRole('dialog')).toContainElement(document.activeElement as HTMLElement)

    fireEvent.keyDown(window, { key: 'Escape' })
    expect(opener).toHaveFocus()
  })

  test('onSave/onClose funktionieren weiterhin (Happy Path)', () => {
    const onSave = vi.fn()
    const onClose = vi.fn()
    render(
      <EditModal isOpen title="Titel" onClose={onClose} onSave={onSave}>
        <input aria-label="Feld" />
      </EditModal>
    )

    fireEvent.click(screen.getByRole('button', { name: 'Speichern' }))
    expect(onSave).toHaveBeenCalled()

    fireEvent.click(screen.getByRole('button', { name: 'Abbrechen' }))
    expect(onClose).toHaveBeenCalled()
  })
})
