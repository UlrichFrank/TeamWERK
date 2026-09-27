import { describe, test, expect, afterEach, vi } from 'vitest'
import { render, screen, cleanup, fireEvent } from '@testing-library/react'
import LineupBadge from './LineupBadge'
import LineupCheckbox from './LineupCheckbox'
import { LINEUP_SURFACE, toLineupState } from '../lib/lineup'

afterEach(cleanup)

describe('LineupBadge', () => {
  test.each([
    ['in', 'aufgestellt'],
    ['out', 'nicht aufgestellt'],
    ['open', 'Aufstellung offen'],
  ] as const)('%s → „%s" mit der Fläche des Zustands', (state, label) => {
    render(<LineupBadge state={state} />)
    const el = screen.getByText(label)
    for (const cls of LINEUP_SURFACE[state].split(' ')) expect(el.className).toContain(cls)
    expect(el.querySelector('svg')).toBeNull() // Kennzeichen tragen kein Symbol
  })

  test('„offen" ist innen transparent, nur gestrichelt umrandet', () => {
    render(<LineupBadge state="open" />)
    const el = screen.getByText('Aufstellung offen')
    expect(el.className).toContain('bg-transparent')
    expect(el.className).toContain('border-dashed')
  })

  test('ohne Status rendert nichts', () => {
    const { container } = render(<LineupBadge state={undefined} />)
    expect(container.innerHTML).toBe('')
  })

  test('freier Text für den Kartenkopf', () => {
    render(<LineupBadge state="in" text="5 aufgestellt" />)
    expect(screen.getByText('5 aufgestellt')).toBeInTheDocument()
  })

  test('toLineupState verwirft Unbekanntes', () => {
    expect(toLineupState('in')).toBe('in')
    expect(toLineupState('')).toBeUndefined()
    expect(toLineupState(undefined)).toBeUndefined()
  })
})

describe('LineupCheckbox', () => {
  test('nur lesend: Zustand und Name im aria-label, nicht fokussierbar', () => {
    render(<LineupCheckbox state="out" memberName="Finn Wagner" />)
    const cb = screen.getByRole('checkbox', { name: 'Finn Wagner: nicht aufgestellt' })
    expect(cb).toHaveAttribute('aria-checked', 'false')
    expect(cb).toHaveAttribute('aria-disabled', 'true')
    expect(cb).not.toHaveAttribute('tabindex')
  })

  test('bedienbar: Klick, Leertaste und Enter schalten um', () => {
    const onToggle = vi.fn()
    render(<LineupCheckbox state="open" memberName="Jonas Keller" onToggle={onToggle} />)
    const cb = screen.getByRole('checkbox', { name: 'Jonas Keller: Aufstellung offen' })
    expect(cb).toHaveAttribute('tabindex', '0')
    fireEvent.click(cb)
    fireEvent.keyDown(cb, { key: ' ' })
    fireEvent.keyDown(cb, { key: 'Enter' })
    expect(onToggle).toHaveBeenCalledTimes(3)
    expect(onToggle).toHaveBeenCalledWith(true)
  })

  test('aufgestellt ist angehakt und schaltet auf „nicht aufstellen"', () => {
    const onToggle = vi.fn()
    render(<LineupCheckbox state="in" memberName="Luca Brenner" onToggle={onToggle} />)
    const cb = screen.getByRole('checkbox', { name: 'Luca Brenner: aufgestellt' })
    expect(cb).toHaveAttribute('aria-checked', 'true')
    fireEvent.click(cb)
    expect(onToggle).toHaveBeenCalledWith(false)
  })
})
