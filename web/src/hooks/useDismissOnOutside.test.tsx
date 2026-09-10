import { describe, it, expect, vi } from 'vitest'
import { render, fireEvent } from '@testing-library/react'
import { useRef } from 'react'
import { useDismissOnOutside } from './useDismissOnOutside'

function Harness({ close }: { close: () => void }) {
  const ref = useRef<HTMLDivElement>(null)
  useDismissOnOutside(true, ref, close)
  return (
    <div>
      <div ref={ref}>
        <div data-testid="list" style={{ overflowY: 'auto', maxHeight: 50 }} />
      </div>
      <div data-testid="outside" />
    </div>
  )
}

describe('useDismissOnOutside', () => {
  it('schliesst beim Scrollen der Seite', () => {
    const close = vi.fn()
    const { getByTestId } = render(<Harness close={close} />)
    fireEvent.scroll(getByTestId('outside'))
    expect(close).toHaveBeenCalled()
  })

  // Regression: Mit `capture` kam auch das scroll-Event der Liste im Dropdown
  // an und schloss es beim ersten Tick — lange Mannschaftslisten waren nicht
  // scrollbar.
  it('bleibt offen beim Scrollen innerhalb des Dropdowns', () => {
    const close = vi.fn()
    const { getByTestId } = render(<Harness close={close} />)
    fireEvent.scroll(getByTestId('list'))
    expect(close).not.toHaveBeenCalled()
  })

  it('schliesst bei Klick ausserhalb, nicht bei Klick innerhalb', () => {
    const close = vi.fn()
    const { getByTestId } = render(<Harness close={close} />)
    fireEvent.mouseDown(getByTestId('list'))
    expect(close).not.toHaveBeenCalled()
    fireEvent.mouseDown(getByTestId('outside'))
    expect(close).toHaveBeenCalled()
  })
})
