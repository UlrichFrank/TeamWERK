import { describe, test, expect, vi, beforeAll, afterEach } from 'vitest'
import { screen, act, fireEvent } from '@testing-library/react'
import ChatPage from '../ChatPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../hooks/useChatEvents', () => ({ useChatEvents: vi.fn() }))

// Mobile-Layout: ChatPage entscheidet `isMobile` über window.innerWidth < 640.
beforeAll(() => {
  Element.prototype.scrollIntoView = vi.fn()
  Element.prototype.scrollTo = vi.fn() as unknown as Element['scrollTo']
  if (typeof globalThis.ResizeObserver === 'undefined') {
    globalThis.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    } as unknown as typeof ResizeObserver
  }
  Object.defineProperty(window, 'innerWidth', { value: 400, writable: true, configurable: true })
})
afterEach(() => vi.restoreAllMocks())

const CONV = {
  id: 7, type: 'group' as const, name: 'Mannschaft', createdBy: 99,
  unreadCount: 0, lastMessage: null,
  members: [{ id: 1, name: 'Ich' }, { id: 2, name: 'Andere' }],
}

const BODY = 'Wer hat denn hochgeladen? Ging der Upload durch?'
const MSG = {
  id: 1, senderId: 1, senderName: 'Ich', preview: BODY, truncated: false, editedAt: null,
  sentAt: '2026-06-28T10:00:00Z', replyToId: 5,
  replyToBody: 'Kann es sein, dass die Videohochladefunktion mit großen Dateien Probleme hat?',
  replyToSenderName: 'Florian', deletedAt: null, isSystem: false,
  reactions: [], readCount: 0, readTotal: 1, read: false,
}

async function openOverlay() {
  renderAsPersona(<ChatPage />, 'spieler', {
    mocks: [
      { url: '/chat/conversations', data: [CONV] },
      { url: '/chat/broadcasts', data: [] },
      { url: /\/chat\/conversations\/7\/messages/, data: [MSG] },
      { method: 'any', url: /\/chat\/conversations\/7\/read/, data: {} },
    ],
  })
  await flushAsync()
  fireEvent.click(screen.getByText('Mannschaft'))
  await flushAsync()

  // Long-Press auf die Nachricht: touchstart + 500 ms, Finger bleibt liegen.
  const bubbleText = screen.getByText(BODY)
  fireEvent.touchStart(bubbleText, { touches: [{ clientX: 100, clientY: 300 }] })
  await act(async () => {
    await new Promise((r) => setTimeout(r, 550))
  })
  expect(screen.getByText('Antworten')).toBeInTheDocument()
  return screen.getByTestId('mobile-overlay-bubble')
}

// iOS wertet seine Long-Press-Textauswahl gegen das Element unter dem Finger im
// Moment der Geste — das ist nach 500 ms die frisch gemountete Overlay-Blase. Trägt
// sie sofort `select-text`, wird ein Wort markiert und die native Leiste liegt über
// unserem Menü (ScreenRecording 2026-09-12). Selektierbar darf sie erst werden, wenn
// der öffnende Finger weg ist; ein zweiter Long-Press auf den Text markiert dann.
describe('ChatPage — Mobile-Overlay nach Long-Press', () => {
  test('Overlay-Blase ist während des öffnenden Touchs nicht selektierbar, danach schon', async () => {
    const bubble = await openOverlay()

    expect(bubble.className).toContain('select-none')
    expect(bubble.className).toContain('[-webkit-touch-callout:none]')
    expect(bubble.className).not.toContain('select-text')

    // Finger geht hoch → Textauswahl per zweitem Long-Press wieder möglich.
    await act(async () => {
      fireEvent.touchEnd(window)
    })
    expect(bubble.className).toContain('select-text')
    expect(bubble.className).not.toContain('select-none')
  })

  test('ein neuer Touch auf der Blase gibt die Selektion ebenfalls frei (Finger war beim Mount schon weg)', async () => {
    const bubble = await openOverlay()
    expect(bubble.className).toContain('select-none')

    await act(async () => {
      fireEvent.touchStart(bubble, { touches: [{ clientX: 100, clientY: 300 }] })
    })
    expect(bubble.className).toContain('select-text')
  })

  // Die Blase ist ein Flex-Item mit self-end; ohne max-width ist ihr automatischer
  // Mindestwert die min-content-Breite der nowrap-Zitatzeile (`truncate`, bis 60 Zeichen)
  // — sie schiebt sich über die Spalte hinaus und wird links abgeschnitten.
  test('Overlay-Blase ist auf die Spaltenbreite gedeckelt (Zitat mit truncate)', async () => {
    const bubble = await openOverlay()
    expect(bubble.className).toContain('max-w-full')
    expect(bubble.className).toContain('min-w-0')
    // Das Zitat ist da und weiterhin als truncate-Zeile gerendert.
    const quote = screen.getAllByText(/Videohochladefunktion/).find((el) =>
      el.className.includes('truncate') && bubble.contains(el),
    )
    expect(quote).toBeTruthy()
  })
})
