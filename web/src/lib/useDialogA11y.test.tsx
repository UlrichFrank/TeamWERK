import { useRef, useState } from 'react'
import { describe, test, expect } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { useDialogA11y } from './useDialogA11y'

function Harness({ isOpen }: { isOpen: boolean }) {
  const ref = useRef<HTMLDivElement>(null)
  useDialogA11y(ref, isOpen)
  if (!isOpen) return null
  return (
    <div ref={ref} role="dialog" aria-modal="true" aria-label="Test-Dialog">
      <input aria-label="Erstes Feld" />
      <input aria-label="Mittleres Feld" />
      <button type="button">Letzter Button</button>
    </div>
  )
}

function ToggleHarness() {
  const [open, setOpen] = useState(false)
  return (
    <>
      <button type="button" onClick={() => setOpen(true)}>Öffnen</button>
      <Harness isOpen={open} />
      {open && <button type="button" onClick={() => setOpen(false)}>Schließen</button>}
    </>
  )
}

describe('useDialogA11y', () => {
  test('Fokus liegt initial im Dialog (erstes fokussierbares Element)', () => {
    render(<Harness isOpen={true} />)
    expect(screen.getByLabelText('Erstes Feld')).toHaveFocus()
  })

  test('Tab vom letzten Element springt zyklisch zum ersten', () => {
    render(<Harness isOpen={true} />)
    screen.getByText('Letzter Button').focus()
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Tab' })
    expect(screen.getByLabelText('Erstes Feld')).toHaveFocus()
  })

  test('Shift+Tab vom ersten Element springt zyklisch zum letzten', () => {
    render(<Harness isOpen={true} />)
    screen.getByLabelText('Erstes Feld').focus()
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Tab', shiftKey: true })
    expect(screen.getByText('Letzter Button')).toHaveFocus()
  })

  test('Fokus kehrt beim Schließen zum Auslöser zurück', () => {
    render(<ToggleHarness />)
    const opener = screen.getByText('Öffnen')
    opener.focus()
    fireEvent.click(opener)

    expect(screen.getByLabelText('Erstes Feld')).toHaveFocus()

    fireEvent.click(screen.getByText('Schließen'))

    expect(opener).toHaveFocus()
  })
})
