import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, act } from '@testing-library/react'
import ChatSearchModal, { type SearchHit } from '../ChatSearchModal'
import { highlight } from '../../lib/chatSearchHighlight'
import { api } from '../../lib/api'

vi.mock('../../lib/api', () => ({ api: { get: vi.fn() } }))

const HIT: SearchHit = {
  kind: 'message',
  id: 1,
  conversationId: 7,
  conversationName: 'Mannschaft C',
  senderName: 'Anna',
  snippet: 'Wo ist die Hallenadresse für Samstag?',
  sentAt: '2026-09-10 15:44:12',
}

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
  vi.clearAllMocks()
})

describe('ChatSearchModal', () => {
  test('unter 2 Zeichen: kein Request, Hinweis sichtbar', async () => {
    render(<ChatSearchModal onClose={() => {}} onSelect={() => {}} />)

    fireEvent.change(screen.getByLabelText('Suchbegriff'), { target: { value: 'a' } })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(500)
    })

    expect(api.get).not.toHaveBeenCalled()
    expect(screen.getByText('Mindestens 2 Zeichen eingeben')).toBeInTheDocument()
  })

  test('ab 2 Zeichen nach 300ms: genau ein Request, Treffer werden gerendert', async () => {
    vi.mocked(api.get).mockResolvedValue({ data: { items: [HIT], total: 1 } })

    render(<ChatSearchModal onClose={() => {}} onSelect={() => {}} />)

    fireEvent.change(screen.getByLabelText('Suchbegriff'), { target: { value: 'hall' } })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })

    expect(api.get).toHaveBeenCalledTimes(1)
    const [url, config] = vi.mocked(api.get).mock.calls[0]
    expect(url).toBe('/chat/search')
    expect(config?.params).toMatchObject({ q: 'hall', limit: 50, offset: 0 })

    expect(screen.getByText('Mannschaft C')).toBeInTheDocument()
    expect(screen.getByText('Anna')).toBeInTheDocument()
    // Snippet enthält den hervorgehobenen Treffer als <mark>.
    const mark = document.querySelector('mark')
    expect(mark).not.toBeNull()
    expect(mark?.textContent?.toLowerCase()).toBe('hall')
  })

  test('Klick auf Treffer ruft onSelect mit dem Hit auf', async () => {
    vi.mocked(api.get).mockResolvedValue({ data: { items: [HIT], total: 1 } })
    const onSelect = vi.fn()

    render(<ChatSearchModal onClose={() => {}} onSelect={onSelect} />)

    fireEvent.change(screen.getByLabelText('Suchbegriff'), { target: { value: 'hall' } })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })

    fireEvent.click(screen.getByText('Mannschaft C').closest('button')!)
    expect(onSelect).toHaveBeenCalledWith(HIT)
  })

  test('Fehler vom Mock zeigt Fehler-Alert, kein Absturz', async () => {
    vi.mocked(api.get).mockRejectedValue(new Error('Netzwerkfehler'))

    render(<ChatSearchModal onClose={() => {}} onSelect={() => {}} />)

    fireEvent.change(screen.getByLabelText('Suchbegriff'), { target: { value: 'hall' } })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300)
    })

    expect(screen.getByText('Netzwerkfehler')).toBeInTheDocument()
  })

  test('Escape schließt das Modal', () => {
    const onClose = vi.fn()
    render(<ChatSearchModal onClose={onClose} onSelect={() => {}} />)

    fireEvent.keyDown(window, { key: 'Escape' })

    expect(onClose).toHaveBeenCalled()
  })

  test('highlight hebt Treffer case-insensitiv hervor', () => {
    const nodes = highlight('Die Hallenadresse', 'hallen')
    render(<>{nodes}</>)

    const mark = document.querySelector('mark')
    expect(mark).not.toBeNull()
    expect(mark?.textContent).toBe('Hallen')
  })
})
