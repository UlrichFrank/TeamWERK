/**
 * Chat-Alben (chat-mehrere-bilder): Mehrfachauswahl in Chat-Eingabe und
 * Mitteilungs-Dialog, Grenze 10, einzelnes Entfernen, sequenzieller Upload und
 * ein einziger POST mit mediaIds in Auswahlreihenfolge.
 */
import { describe, test, expect, vi, beforeAll, beforeEach } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import ChatPage from '../ChatPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../hooks/useChatEvents', () => ({ useChatEvents: vi.fn() }))
// jsdom hat kein Canvas — die Verkleinerung gibt die Datei unverändert zurück.
vi.mock('../../lib/imageCompress', () => ({
  compressImage: vi.fn(async (file: File) => ({ blob: file, fileName: file.name })),
}))

let urlSeq = 0
const revoked: string[] = []

beforeAll(() => {
  Element.prototype.scrollIntoView = vi.fn()
  Element.prototype.scrollTo = function (this: HTMLElement, arg: unknown) {
    const opts = typeof arg === 'object' && arg !== null ? (arg as { top?: number }) : null
    if (opts && typeof opts.top === 'number') this.scrollTop = opts.top
  } as unknown as Element['scrollTo']
})

beforeEach(() => {
  localStorage.clear()
  urlSeq = 0
  revoked.length = 0
  ;(URL as unknown as { createObjectURL: () => string }).createObjectURL = () => `blob:preview-${++urlSeq}`
  ;(URL as unknown as { revokeObjectURL: (u: string) => void }).revokeObjectURL = (u: string) => {
    revoked.push(u)
  }
})

const CONV = {
  id: 7,
  type: 'group' as const,
  name: 'Mannschaft',
  createdBy: 99,
  unreadCount: 0,
  lastMessage: null,
  members: [{ id: 1, name: 'Ich' }, { id: 2, name: 'Andere' }],
}
const CONV2 = { ...CONV, id: 8, name: 'Vorstand' }

function files(n: number, prefix = 'foto'): File[] {
  return Array.from({ length: n }, (_, i) => new File(['x'], `${prefix}-${i + 1}.jpg`, { type: 'image/jpeg' }))
}

interface UploadLog {
  order: string[]
  maxInFlight: number
  messagePosts: Record<string, unknown>[]
}

/**
 * Öffnet die Konversation und installiert die Upload-/Sende-Handler. Uploads
 * antworten verzögert, damit Parallelität messbar wird; failAt lässt den
 * k-ten Upload (1-basiert) mit 500 scheitern.
 */
async function openConversation(failAt?: number): Promise<UploadLog> {
  renderAsPersona(<ChatPage />, 'spieler', {
    mocks: [
      { url: '/chat/conversations', data: [CONV, CONV2] },
      { url: '/chat/broadcasts', data: [] },
      { url: /\/chat\/conversations\/\d+\/messages/, data: [] },
    ],
  })
  await flushAsync()
  fireEvent.click(screen.getByText('Mannschaft'))
  await flushAsync()

  const log: UploadLog = { order: [], maxInFlight: 0, messagePosts: [] }
  let inFlight = 0
  let nextID = 100
  const mock = getApiMock()
  mock.reset()
  mock.onGet('/chat/conversations').reply(200, [CONV, CONV2])
  mock.onPost('/media/upload').reply(async (config) => {
    const file = (config.data as FormData).get('image') as File
    inFlight++
    log.maxInFlight = Math.max(log.maxInFlight, inFlight)
    log.order.push(file.name)
    await new Promise((r) => setTimeout(r, 5))
    inFlight--
    if (failAt !== undefined && log.order.length === failAt) return [500, {}]
    return [201, { mediaId: nextID++, url: '/media/x' }]
  })
  mock.onPost(/\/chat\/conversations\/\d+\/messages/).reply((config) => {
    log.messagePosts.push(JSON.parse(config.data as string))
    return [201, { id: 1 }]
  })
  mock.onAny().reply(200, [])
  return log
}

function pick(list: File[], testId = 'chat-image-input') {
  fireEvent.change(screen.getByTestId(testId), { target: { files: list } })
}

async function settle(rounds = 8) {
  for (let i = 0; i < rounds; i++) await flushAsync()
}

describe('ChatPage — Album auswählen', () => {
  test('Mehrfachauswahl zeigt eine Miniatur je Bild', async () => {
    await openConversation()
    pick(files(3))
    await flushAsync()
    expect(screen.getAllByAltText(/^Vorschau \d+$/)).toHaveLength(3)
  })

  test('einzelnes Entfernen lässt die übrigen in Reihenfolge stehen', async () => {
    await openConversation()
    pick(files(3))
    await flushAsync()
    fireEvent.click(screen.getByRole('button', { name: 'Bild 2 entfernen' }))
    await flushAsync()
    const thumbs = screen.getAllByAltText(/^Vorschau \d+$/) as HTMLImageElement[]
    expect(thumbs.map((t) => t.getAttribute('src'))).toEqual(['blob:preview-1', 'blob:preview-3'])
    expect(revoked).toContain('blob:preview-2')
  })

  test('12 gewählt → 10 übernommen und sichtbarer Hinweis', async () => {
    await openConversation()
    pick(files(12))
    await flushAsync()
    expect(screen.getAllByAltText(/^Vorschau \d+$/)).toHaveLength(10)
    expect(screen.getByText('Höchstens 10 Bilder je Nachricht')).toBeInTheDocument()
  })

  test('Einfügen aus der Zwischenablage übernimmt alle Bilder', async () => {
    await openConversation()
    const input = screen.getByPlaceholderText('Nachricht schreiben…')
    fireEvent.paste(input, {
      clipboardData: { files: [...files(2), new File(['t'], 'notiz.txt', { type: 'text/plain' })] },
    })
    await flushAsync()
    expect(screen.getAllByAltText(/^Vorschau \d+$/)).toHaveLength(2)
  })

  test('Konversationswechsel verwirft die Auswahl und gibt alle Vorschau-URLs frei', async () => {
    await openConversation()
    pick(files(2))
    await flushAsync()
    fireEvent.click(screen.getByText('Vorstand'))
    await flushAsync()
    expect(screen.queryAllByAltText(/^Vorschau \d+$/)).toHaveLength(0)
    expect(revoked).toEqual(expect.arrayContaining(['blob:preview-1', 'blob:preview-2']))
  })
})

describe('ChatPage — Album senden', () => {
  test('3 Uploads strikt nacheinander, dann ein POST mit mediaIds in Auswahlreihenfolge', async () => {
    const log = await openConversation()
    pick(files(3))
    await flushAsync()
    fireEvent.click(screen.getByRole('button', { name: 'Senden' }))
    await flushAsync()
    expect(screen.getByText(/^Bild [12]\/3$/)).toBeInTheDocument()
    await waitFor(() => expect(log.messagePosts).toHaveLength(1))

    expect(log.order).toEqual(['foto-1.jpg', 'foto-2.jpg', 'foto-3.jpg'])
    expect(log.maxInFlight).toBe(1)
    expect(log.messagePosts).toHaveLength(1)
    expect(log.messagePosts[0].mediaIds).toEqual([100, 101, 102])
    expect(log.messagePosts[0]).not.toHaveProperty('mediaId')
    expect(screen.queryAllByAltText(/^Vorschau \d+$/)).toHaveLength(0)
  })

  test('zweiter Upload scheitert → kein Message-POST, Auswahl bleibt', async () => {
    const log = await openConversation(2)
    pick(files(3))
    await flushAsync()
    fireEvent.click(screen.getByRole('button', { name: 'Senden' }))
    await waitFor(() => expect(screen.getByRole('button', { name: 'Senden' })).not.toHaveTextContent(/Bild/))
    await settle()

    expect(log.order).toEqual(['foto-1.jpg', 'foto-2.jpg'])
    expect(log.messagePosts).toHaveLength(0)
    expect(screen.getAllByAltText(/^Vorschau \d+$/)).toHaveLength(3)
  })

  test('reiner Text sendet kein mediaIds-Feld', async () => {
    const log = await openConversation()
    fireEvent.change(screen.getByPlaceholderText('Nachricht schreiben…'), { target: { value: 'Hallo' } })
    fireEvent.click(screen.getByRole('button', { name: 'Senden' }))
    await settle()
    expect(log.messagePosts).toHaveLength(1)
    expect(log.messagePosts[0]).not.toHaveProperty('mediaIds')
  })
})

describe('Mitteilungs-Dialog — Album', () => {
  test('Mehrfachauswahl, sequenzieller Upload und mediaIds im Request', async () => {
    renderAsPersona(<ChatPage />, 'vorstand')
    await flushAsync()

    const mock = getApiMock()
    mock.reset()
    const order: string[] = []
    let inFlight = 0
    let maxInFlight = 0
    let nextID = 200
    const sent: Record<string, unknown>[] = []
    mock.onGet('/chat/broadcast-targets').reply(200, [{ kind: 'users', teamId: null, label: 'Alle Nutzer', count: 9 }])
    mock.onPost('/media/upload').reply(async (config) => {
      inFlight++
      maxInFlight = Math.max(maxInFlight, inFlight)
      order.push(((config.data as FormData).get('image') as File).name)
      await new Promise((r) => setTimeout(r, 5))
      inFlight--
      return [201, { mediaId: nextID++ }]
    })
    mock.onPost('/chat/broadcasts').reply((config) => {
      sent.push(JSON.parse(config.data as string))
      return [201, { id: 1, recipients: 9 }]
    })
    mock.onAny().reply(200, [])

    fireEvent.click(screen.getByText('Mitteilungen'))
    await flushAsync()
    fireEvent.click(screen.getByText('Mitteilung senden'))
    await flushAsync()

    fireEvent.click(screen.getByLabelText('Alle Nutzer', { exact: false }))
    pick(files(4, 'bc'), 'broadcast-image-input')
    await flushAsync()
    expect(screen.getAllByAltText(/^Vorschau \d+$/)).toHaveLength(4)
    fireEvent.click(screen.getByRole('button', { name: 'Bild 1 entfernen' }))
    await flushAsync()

    const buttons = screen.getAllByText('Mitteilung senden')
    fireEvent.click(buttons[buttons.length - 1])
    await waitFor(() => expect(sent).toHaveLength(1))

    expect(order).toEqual(['bc-2.jpg', 'bc-3.jpg', 'bc-4.jpg'])
    expect(maxInFlight).toBe(1)
    expect(sent).toHaveLength(1)
    expect(sent[0].mediaIds).toEqual([200, 201, 202])
    expect(sent[0]).not.toHaveProperty('mediaId')
  })

  test('mehr als 10 Bilder → 10 übernommen und Hinweis im Dialog', async () => {
    renderAsPersona(<ChatPage />, 'vorstand')
    await flushAsync()
    const mock = getApiMock()
    mock.reset()
    mock.onGet('/chat/broadcast-targets').reply(200, [{ kind: 'users', teamId: null, label: 'Alle Nutzer', count: 9 }])
    mock.onAny().reply(200, [])
    fireEvent.click(screen.getByText('Mitteilungen'))
    await flushAsync()
    fireEvent.click(screen.getByText('Mitteilung senden'))
    await flushAsync()

    pick(files(11, 'bc'), 'broadcast-image-input')
    await flushAsync()
    expect(screen.getAllByAltText(/^Vorschau \d+$/)).toHaveLength(10)
    expect(screen.getByText('Höchstens 10 Bilder je Nachricht')).toBeInTheDocument()
  })
})
