import { describe, test, expect, vi, beforeAll } from 'vitest'
import { screen, fireEvent, act } from '@testing-library/react'
import ChatPage from '../ChatPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../hooks/useChatEvents', () => ({ useChatEvents: vi.fn() }))

// Layout-Mocks für den WindowedRows-Container (Muster aus
// ChatPage.openAtUnread.test.tsx / ChatPage.deepLink.test.tsx): der
// Öffnungs-Anker landet hier immer im 'bottom'-Fall (unreadCount=0) und
// treibt smoothScrollToBottom — ein rAF-Loop über scrollHeight/scrollTop/
// clientHeight des [data-windowed-scroll]-Containers. Ohne diese Mocks
// hängt der Loop an echten jsdom-Timern statt in einem Tick durchzulaufen.
const VIEWPORT = 300
const CONTENT_HEIGHT = 5000
const scrollBox = { value: 0 }

beforeAll(() => {
  Element.prototype.scrollIntoView = vi.fn()
  Element.prototype.scrollTo = function (this: HTMLElement, arg: unknown) {
    const opts = typeof arg === 'object' && arg !== null ? (arg as { top?: number }) : null
    if (opts && typeof opts.top === 'number') this.scrollTop = opts.top
  } as unknown as Element['scrollTo']
  // Synchroner rAF-Mock: der Custom-Smooth-Loop in ChatPage schließt sich
  // damit in einem Tick statt über ~16 Frames verteilt.
  let rafTime = 0
  globalThis.requestAnimationFrame = (cb: FrameRequestCallback) => {
    rafTime += 400
    cb(rafTime)
    return 0
  }
  if (typeof globalThis.ResizeObserver === 'undefined') {
    globalThis.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    } as unknown as typeof ResizeObserver
  }
  Object.defineProperty(HTMLElement.prototype, 'clientHeight', {
    configurable: true,
    get() {
      return this.hasAttribute('data-windowed-scroll') ? VIEWPORT : 0
    },
  })
  Object.defineProperty(HTMLElement.prototype, 'scrollHeight', {
    configurable: true,
    get() {
      return this.hasAttribute('data-windowed-scroll') ? CONTENT_HEIGHT : 0
    },
  })
  Object.defineProperty(HTMLElement.prototype, 'scrollTop', {
    configurable: true,
    get() {
      return this.hasAttribute('data-windowed-scroll') ? scrollBox.value : 0
    },
    set(v: number) {
      if (this.hasAttribute('data-windowed-scroll')) scrollBox.value = v
    },
  })
})

function makeMessages(label: string, count: number) {
  return Array.from({ length: count }, (_, i) => ({
    id: i + 1,
    senderId: 42,
    senderName: 'Andere',
    preview: `Nachricht-${label}-${i + 1}`,
    truncated: false,
    sentAt: '2026-06-28T10:00:00Z',
    replyToId: null,
    replyToBody: null,
    replyToSenderName: null,
    editedAt: null,
    deletedAt: null,
    isSystem: false,
    reactions: [],
  }))
}

const CONV_A = {
  id: 7,
  type: 'group' as const,
  name: 'Team Rot',
  createdBy: 99,
  unreadCount: 0,
  lastMessage: null,
  members: [
    { id: 1, name: 'Ich' },
    { id: 42, name: 'Andere' },
  ],
}

const CONV_B = {
  id: 8,
  type: 'group' as const,
  name: 'Team Blau',
  createdBy: 99,
  unreadCount: 0,
  lastMessage: null,
  members: [
    { id: 1, name: 'Ich' },
    { id: 42, name: 'Andere' },
  ],
}

const MESSAGES_A_URL = /\/chat\/conversations\/7\/messages/
const MESSAGES_B_URL = /\/chat\/conversations\/8\/messages/
const READ_URL = /\/chat\/conversations\/\d+\/read/

// Kontrollierbares Promise für den Nachrichten-Fetch von Konversation B: der
// Test löst/verwirft es selbst, statt sich auf echtes Timing zu verlassen —
// macht das Rennen aus proposal.md deterministisch reproduzierbar (kein
// WebKit, keine Frame-Analyse nötig).
function deferred<T>() {
  let resolveFn!: (value: T) => void
  let rejectFn!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolveFn = res
    rejectFn = rej
  })
  return {
    promise,
    resolve: (value: T) => resolveFn(value),
    reject: (reason?: unknown) => rejectFn(reason),
  }
}

// Öffnet Konversation A (Nachrichten sichtbar), überschreibt danach den Mock
// für Konversation B mit einem noch NICHT aufgelösten Promise und klickt B
// an. Der Rückgabewert erlaubt es dem jeweiligen Test, das Promise gezielt
// aufzulösen oder zu verwerfen und den Ausgang zu prüfen.
async function switchFromAToB() {
  const messagesA = makeMessages('A', 3)
  renderAsPersona(<ChatPage />, 'spieler', {
    mocks: [
      { url: '/chat/conversations', data: [CONV_A, CONV_B] },
      { url: '/chat/broadcasts', data: [] },
      { url: MESSAGES_A_URL, data: messagesA },
      // Platzhalter — wird unten vor dem Klick auf B überschrieben. Muss
      // trotzdem hier registriert sein: axios-mock-adapter matcht in
      // Registrierungsreihenfolge, und die spätere Überschreibung ersetzt
      // denselben Handler-Platz (VOR dem Catch-all aus setupApiMock) nur,
      // wenn schon einer mit demselben Regex-Quelltext existiert.
      { url: MESSAGES_B_URL, data: [] },
      { method: 'any', url: READ_URL, data: {} },
    ],
  })
  await flushAsync()

  fireEvent.click(screen.getByText('Team Rot'))
  await flushAsync()
  expect(screen.getByText('Nachricht-A-1')).toBeInTheDocument()

  const bFetch = deferred<unknown>()
  getApiMock()
    .onGet(MESSAGES_B_URL)
    .reply(() => bFetch.promise.then((data) => [200, data]))

  fireEvent.click(screen.getByText('Team Blau'))
  await flushAsync()

  return bFetch
}

describe('ChatPage — Konversationswechsel zeigt keinen Fremdinhalt', () => {
  test('Wechsel zeigt keine Nachrichten der vorigen Konversation', async () => {
    const bFetch = await switchFromAToB()

    // Header zeigt bereits B. Der Sidebar-Eintrag trägt denselben Text —
    // Selektor auf die Header-spezifische Klasse (font-semibold statt
    // font-medium) macht die Prüfung eindeutig.
    expect(
      screen.getByText('Team Blau', { selector: '.font-semibold' }),
    ).toBeInTheDocument()

    // Kern-Invariante: keine Nachricht aus A ist sichtbar, obwohl der Fetch
    // für B noch aussteht (das Promise wurde bewusst nicht aufgelöst).
    expect(screen.queryByText('Nachricht-A-1')).toBeNull()
    expect(screen.queryByText(/^Nachricht-A-/)).toBeNull()

    // Aufräumen, damit kein offenes Promise über den Test hinaus hängt.
    bFetch.resolve([])
  })

  test('Ladezustand während des Fetches: role=status sichtbar, verschwindet nach Auflösen mit Bs Nachrichten', async () => {
    const bFetch = await switchFromAToB()

    // Genau ein Ladehinweis ist sichtbar — Ungelesen-Chip und "Ältere laden"
    // sind im Ladezweig bewusst ausgeblendet (siehe Kommentar am
    // loadingMessages-Render-Zweig in ChatPage.tsx).
    expect(screen.getByRole('status')).toBeInTheDocument()

    const messagesB = makeMessages('B', 2)
    bFetch.resolve(messagesB)
    await flushAsync()

    expect(screen.queryByRole('status')).toBeNull()
    expect(screen.getByText('Nachricht-B-1')).toBeInTheDocument()
    expect(screen.getByText('Nachricht-B-2')).toBeInTheDocument()
  })

  test('Abruf schlägt fehl: Ladezustand endet, keine Nachrichten aus A sichtbar', async () => {
    const bFetch = await switchFromAToB()

    expect(screen.getByRole('status')).toBeInTheDocument()

    bFetch.reject(new Error('Netzwerkfehler'))
    await flushAsync()

    expect(screen.queryByRole('status')).toBeNull()
    expect(screen.queryByText(/^Nachricht-A-/)).toBeNull()
    // Der fehlgeschlagene Fetch darf auch keine B-Nachrichten hinterlassen —
    // die Liste bleibt leer (loadMessages bricht vor setMessages(msgs) ab).
    expect(screen.queryByText(/^Nachricht-B-/)).toBeNull()
  })

  // Kehrseite des Leerens: die absichtlich leere Liste darf den Scroll-Anker
  // nicht kosten. Ohne Guard rief der [messages]-Layout-Effekt (und der
  // MutationObserver des Watchers) applyAnchor auf der leeren Box auf, das
  // armierte über scheduleAnchorSettle das 600-ms-Settle, und weil eine leere
  // Box keine ladenden Bilder hat, gab check() den Anker frei — noch bevor die
  // Nachrichten da waren. Beim ERSTEN Öffnen räumt niemand diesen Timer ab
  // (das Cleanup des Watcher-Effekts läuft nur, wenn sich activeConv?.id
  // ändert), also traf es jeden Abruf > 600 ms.
  test('Anker überlebt einen Fetch länger als das Settle-Timeout', async () => {
    const convUnread = { ...CONV_A, unreadCount: 3 }
    const messagesA = makeMessages('A', 20)
    const scrollIntoViewSpy = Element.prototype.scrollIntoView as unknown as {
      mock: { calls: Array<unknown[]> }
    }

    // Nur die für das Settle relevanten Uhren fälschen (window.setTimeout /
    // clearTimeout / Date.now) — requestAnimationFrame bleibt der synchrone
    // Mock aus beforeAll. shouldAdvanceTime, damit das setTimeout(0) in
    // flushAsync weiterhin von selbst feuert.
    vi.useFakeTimers({
      shouldAdvanceTime: true,
      toFake: ['setTimeout', 'clearTimeout', 'Date'],
    })
    try {
      renderAsPersona(<ChatPage />, 'spieler', {
        mocks: [
          { url: '/chat/conversations', data: [convUnread] },
          { url: '/chat/broadcasts', data: [] },
          // Platzhalter mit demselben RegExp-OBJEKT, das unten überschrieben
          // wird (siehe Kommentar in switchFromAToB).
          { url: MESSAGES_A_URL, data: messagesA },
          { method: 'any', url: READ_URL, data: {} },
        ],
      })
      await flushAsync()

      const aFetch = deferred<unknown>()
      getApiMock()
        .onGet(MESSAGES_A_URL)
        .reply(() => aFetch.promise.then((data) => [200, data]))

      scrollIntoViewSpy.mock.calls.length = 0
      fireEvent.click(screen.getByText('Team Rot'))
      await flushAsync()

      // Leerphase: der Abruf steht noch aus.
      expect(screen.getByRole('status')).toBeInTheDocument()

      // Deutlich über die 600 ms des Settle hinaus vorspulen. Ohne Guard gibt
      // releaseAnchor() hier den 'divider'-Anker frei.
      await act(async () => {
        vi.advanceTimersByTime(2000)
      })

      aFetch.resolve(messagesA)
      await flushAsync()

      // Divider ist gerendert …
      expect(screen.getByText('3 ungelesene Nachrichten')).toBeInTheDocument()
      // … und der Anker hat gewirkt: Positionierung auf den Divider
      // (scrollIntoView mit block:'start'), nicht der freigegebene Sticky-Fall.
      const hasBlockStart = scrollIntoViewSpy.mock.calls.some((args) => {
        const opts = args[0] as { block?: string } | undefined
        return opts?.block === 'start'
      })
      expect(hasBlockStart).toBe(true)
    } finally {
      vi.useRealTimers()
    }
  })
})
