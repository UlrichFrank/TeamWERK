import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import GameInfoEditor from '../GameInfoEditor'

let mock: MockAdapter

beforeEach(() => {
  mock = new MockAdapter(api)
})
afterEach(() => {
  mock.restore()
})

const OFFSET = 'Treffen (Min. vor Anwurf)'

describe('GameInfoEditor', () => {
  test('ein Speichern sendet Hinweis und Treffzeit-Abstand', async () => {
    const onMeetingSaved = vi.fn()
    mock.onPut('/games/7/note').reply(200, {})
    mock.onPut('/games/7/meeting').reply(200, { meet_time: '13:30', meet_date: '2026-10-11', meet_offset_minutes: 45, meet_place: 'Parkplatz' })

    render(<GameInfoEditor gameId={7} initialNote="" initialOffset={null} initialPlace="" onMeetingSaved={onMeetingSaved} />)
    expect(screen.getAllByRole('button', { name: 'Speichern' })).toHaveLength(1)
    fireEvent.change(screen.getByLabelText('Hinweis'), { target: { value: 'Trikots mitbringen' } })
    fireEvent.change(screen.getByLabelText(OFFSET), { target: { value: '45' } })
    fireEvent.change(screen.getByLabelText('Treffpunkt'), { target: { value: ' Parkplatz ' } })
    fireEvent.click(screen.getByRole('button', { name: 'Speichern' }))

    await waitFor(() => expect(onMeetingSaved).toHaveBeenCalled())
    expect(JSON.parse(mock.history.put[0].data)).toEqual({ note: 'Trikots mitbringen' })
    expect(JSON.parse(mock.history.put[1].data)).toEqual({ meet_offset_minutes: 45, meet_place: 'Parkplatz' })
  })

  test('nur geänderte Teile werden gesendet', async () => {
    mock.onPut('/games/7/note').reply(200, {})

    render(<GameInfoEditor gameId={7} initialNote="alt" initialOffset={60} initialPlace="Bus" />)
    expect((screen.getByLabelText(OFFSET) as HTMLInputElement).value).toBe('60')
    fireEvent.change(screen.getByLabelText('Hinweis'), { target: { value: 'neu' } })
    fireEvent.click(screen.getByRole('button', { name: 'Speichern' }))

    await waitFor(() => expect(mock.history.put).toHaveLength(1))
    expect(mock.history.put[0].url).toBe('/games/7/note')
  })

  test('leeres Feld entfernt die Treffzeit', async () => {
    const onMeetingSaved = vi.fn()
    mock.onPut('/games/7/meeting').reply(200, { meet_time: null, meet_date: null, meet_offset_minutes: null, meet_place: '' })

    render(<GameInfoEditor gameId={7} initialNote="" initialOffset={60} initialPlace="" onMeetingSaved={onMeetingSaved} />)
    fireEvent.change(screen.getByLabelText(OFFSET), { target: { value: '' } })
    fireEvent.click(screen.getByRole('button', { name: 'Speichern' }))

    await waitFor(() => expect(onMeetingSaved).toHaveBeenCalled())
    expect(JSON.parse(mock.history.put[0].data)).toEqual({ meet_offset_minutes: null, meet_place: '' })
  })

  test('zeigt den übersetzten Fehler des Backends', async () => {
    mock.onPut('/games/7/meeting').reply(400, { error: 'meet_offset_out_of_range' })

    render(<GameInfoEditor gameId={7} initialNote="" initialOffset={null} initialPlace="" />)
    fireEvent.change(screen.getByLabelText(OFFSET), { target: { value: '30' } })
    fireEvent.click(screen.getByRole('button', { name: 'Speichern' }))

    expect(await screen.findByText('Die Treffzeit darf höchstens 12 Stunden vor dem Anwurf liegen')).toBeInTheDocument()
  })

  test('Treffpunkt ohne Treffzeit und ungültiger Abstand sperren Speichern', () => {
    render(<GameInfoEditor gameId={7} initialNote="" initialOffset={null} initialPlace="" />)
    const save = () => screen.getByRole('button', { name: 'Speichern' }) as HTMLButtonElement
    fireEvent.change(screen.getByLabelText('Treffpunkt'), { target: { value: 'Halle' } })
    expect(save().disabled).toBe(true)
    fireEvent.change(screen.getByLabelText(OFFSET), { target: { value: '800' } })
    expect(save().disabled).toBe(true)
    fireEvent.change(screen.getByLabelText(OFFSET), { target: { value: '60' } })
    expect(save().disabled).toBe(false)
  })
})
