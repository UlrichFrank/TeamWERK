import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import ChatPollCard, { applyOptimisticSelection } from './ChatPollCard'
import type { Poll } from '../pages/ChatPage'

const mockPut = vi.fn((..._args: unknown[]) => Promise.resolve({ status: 204 }))
vi.mock('../lib/api', () => ({
  api: { put: (...args: unknown[]) => mockPut(...args) },
}))

function makePoll(overrides: Partial<Poll> = {}): Poll {
  return {
    allowMultiple: false,
    closedAt: null,
    voterCount: 4,
    options: [
      { id: 11, label: 'Pizza', count: 3, voted: false, voters: [] },
      { id: 12, label: 'Nudeln', count: 1, voted: false, voters: [] },
    ],
    ...overrides,
  }
}

describe('applyOptimisticSelection', () => {
  test('neue Stimme erhöht count der gewählten Option und voterCount', () => {
    const poll = makePoll()
    const next = applyOptimisticSelection(poll, [11])
    expect(next.options.find((o) => o.id === 11)).toMatchObject({ count: 4, voted: true })
    expect(next.voterCount).toBe(5)
  })

  test('Stimme zurückziehen senkt count und voterCount', () => {
    const poll = makePoll({
      voterCount: 4,
      options: [
        { id: 11, label: 'Pizza', count: 3, voted: true, voters: [] },
        { id: 12, label: 'Nudeln', count: 1, voted: false, voters: [] },
      ],
    })
    const next = applyOptimisticSelection(poll, [])
    expect(next.options.find((o) => o.id === 11)).toMatchObject({ count: 2, voted: false })
    expect(next.voterCount).toBe(3)
  })

  test('Wechsel der Einfachauswahl ändert voterCount nicht (war und bleibt ein Abstimmender)', () => {
    const poll = makePoll({
      voterCount: 4,
      options: [
        { id: 11, label: 'Pizza', count: 3, voted: true, voters: [] },
        { id: 12, label: 'Nudeln', count: 1, voted: false, voters: [] },
      ],
    })
    const next = applyOptimisticSelection(poll, [12])
    expect(next.options.find((o) => o.id === 11)).toMatchObject({ count: 2, voted: false })
    expect(next.options.find((o) => o.id === 12)).toMatchObject({ count: 2, voted: true })
    expect(next.voterCount).toBe(4)
  })
})

describe('ChatPollCard', () => {
  beforeEach(() => {
    mockPut.mockReset()
    mockPut.mockResolvedValue({ status: 204 })
  })

  test('zeigt Zahlen, Balken und die eigene Wahl', () => {
    const poll = makePoll({
      voterCount: 4,
      options: [
        { id: 11, label: 'Pizza', count: 3, voted: true, voters: [] },
        { id: 12, label: 'Nudeln', count: 1, voted: false, voters: [] },
      ],
    })
    render(
      <ChatPollCard poll={poll} question="Pizza oder Nudeln?" messageId={1} isOwn={false} onOpenVotes={() => {}} />,
    )
    expect(screen.getByText('Pizza oder Nudeln?')).toBeInTheDocument()
    expect(screen.getByText('Eine Antwort wählen')).toBeInTheDocument()
    expect(screen.getByText('3')).toBeInTheDocument()
    expect(screen.getByText('1')).toBeInTheDocument()
    expect(screen.getByText(/4 Stimmen · Stimmen anzeigen/)).toBeInTheDocument()
  })

  test('Einfachauswahl-Wechsel sendet {optionIds:[neu]}', async () => {
    const poll = makePoll({
      voterCount: 4,
      options: [
        { id: 11, label: 'Pizza', count: 3, voted: true, voters: [] },
        { id: 12, label: 'Nudeln', count: 1, voted: false, voters: [] },
      ],
    })
    render(
      <ChatPollCard poll={poll} question="Pizza oder Nudeln?" messageId={1} isOwn={false} onOpenVotes={() => {}} />,
    )
    fireEvent.click(screen.getByText('Nudeln'))
    await waitFor(() =>
      expect(mockPut).toHaveBeenCalledWith('/chat/messages/1/poll/vote', { optionIds: [12] }),
    )
  })

  test('Antippen der eigenen Option zieht die Stimme zurück ({optionIds:[]})', async () => {
    const poll = makePoll({
      voterCount: 4,
      options: [
        { id: 11, label: 'Pizza', count: 3, voted: true, voters: [] },
        { id: 12, label: 'Nudeln', count: 1, voted: false, voters: [] },
      ],
    })
    render(
      <ChatPollCard poll={poll} question="Pizza oder Nudeln?" messageId={1} isOwn={false} onOpenVotes={() => {}} />,
    )
    fireEvent.click(screen.getByText('Pizza'))
    await waitFor(() =>
      expect(mockPut).toHaveBeenCalledWith('/chat/messages/1/poll/vote', { optionIds: [] }),
    )
  })

  test('Mehrfachauswahl schaltet die angetippte Option um, ohne andere zu berühren', async () => {
    const poll = makePoll({
      allowMultiple: true,
      voterCount: 1,
      options: [
        { id: 11, label: 'Samstag', count: 1, voted: true, voters: [] },
        { id: 12, label: 'Sonntag', count: 0, voted: false, voters: [] },
      ],
    })
    render(
      <ChatPollCard poll={poll} question="Wann?" messageId={1} isOwn={false} onOpenVotes={() => {}} />,
    )
    fireEvent.click(screen.getByText('Sonntag'))
    await waitFor(() =>
      expect(mockPut).toHaveBeenCalledWith('/chat/messages/1/poll/vote', { optionIds: [11, 12] }),
    )
  })

  test('beendete Umfrage ist nicht klickbar und zeigt „Beendet"', () => {
    const poll = makePoll({ closedAt: '2026-09-10T10:00:00Z' })
    render(
      <ChatPollCard poll={poll} question="Pizza oder Nudeln?" messageId={1} isOwn={false} onOpenVotes={() => {}} />,
    )
    expect(screen.getByText('Beendet')).toBeInTheDocument()
    fireEvent.click(screen.getByText('Pizza'))
    expect(mockPut).not.toHaveBeenCalled()
    expect(screen.getByRole('button', { name: /Pizza/ })).toBeDisabled()
  })

  test('Fehlschlag löst Rollback aus und zeigt eine Fehlermeldung', async () => {
    mockPut.mockRejectedValue({ isAxiosError: true, response: { status: 409, data: { error: 'Umfrage beendet' } } })
    const poll = makePoll()
    render(
      <ChatPollCard poll={poll} question="Pizza oder Nudeln?" messageId={1} isOwn={false} onOpenVotes={() => {}} />,
    )
    fireEvent.click(screen.getByText('Pizza'))
    await waitFor(() => expect(screen.getByText('Umfrage beendet')).toBeInTheDocument())
    // Rollback: die ursprüngliche Zahl (3, nicht die optimistisch erhöhte 4) steht wieder da.
    expect(screen.getByText('3')).toBeInTheDocument()
  })

  test('Stimmen anzeigen ruft onOpenVotes auf', () => {
    const onOpenVotes = vi.fn()
    const poll = makePoll()
    render(
      <ChatPollCard poll={poll} question="Pizza oder Nudeln?" messageId={1} isOwn={false} onOpenVotes={onOpenVotes} />,
    )
    fireEvent.click(screen.getByText(/Stimmen anzeigen/))
    expect(onOpenVotes).toHaveBeenCalled()
  })
})
