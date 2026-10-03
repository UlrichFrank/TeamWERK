import { describe, test, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import MeetingPointLine from '../MeetingPointLine'

describe('MeetingPointLine', () => {
  test('zeigt Treffzeit und Ort', () => {
    render(<MeetingPointLine gameDate="2026-10-11" meetTime="13:30" meetDate="2026-10-11" meetPlace="Parkplatz" />)
    expect(screen.getByText('Treffen 13:30 · Parkplatz')).toBeInTheDocument()
  })

  test('ohne Ort nur die Uhrzeit, mit „Uhr" auf Wunsch', () => {
    render(<MeetingPointLine gameDate="2026-10-11" meetTime="13:30" meetDate="2026-10-11" meetPlace="" withUhr />)
    expect(screen.getByText('Treffen 13:30 Uhr')).toBeInTheDocument()
  })

  test('markiert ein Treffen am Vortag', () => {
    render(<MeetingPointLine gameDate="2026-10-11T00:00:00Z" meetTime="23:00" meetDate="2026-10-10" withUhr />)
    expect(screen.getByText('Treffen 23:00 Uhr (Vortag)')).toBeInTheDocument()
  })

  test('rendert nichts ohne Treffzeit', () => {
    const { container } = render(<MeetingPointLine gameDate="2026-10-11" meetTime={null} meetDate={null} meetPlace="" />)
    expect(container).toBeEmptyDOMElement()
  })
})
