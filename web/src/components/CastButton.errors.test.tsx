import { describe, test, expect, afterEach, beforeEach, vi } from 'vitest'
import { render, screen, cleanup, fireEvent, waitFor } from '@testing-library/react'
import CastButton, { MSG_LOAD_FAILED, MSG_START_FAILED, MSG_UNAVAILABLE } from './CastButton'
import { isCastAvailable, loadCastSDK, startCastSession } from '../lib/cast'

// Fehlerpfade nach dem Klick. Früher verschwand der Button bei JEDEM
// Fehlschlag kommentarlos — ein CSP-Block des SDK sah damit aus wie ein
// Browser ohne Cast. Invariante: nach einem Fehlschlag steht immer ein
// sichtbarer Hinweis, und nur 'unavailable' entfernt den Button endgültig.

vi.mock('../lib/cast', () => ({
  isCastAvailable: vi.fn(),
  loadCastSDK: vi.fn(),
  startCastSession: vi.fn(),
}))

const URL = 'https://example.test/api/videos/1/hls/master.m3u8?st=abc'

function clickCast() {
  fireEvent.click(screen.getByRole('button', { name: /Chromecast/i }))
}

beforeEach(() => {
  vi.mocked(isCastAvailable).mockReturnValue(false)
  vi.spyOn(console, 'warn').mockImplementation(() => {})
})

afterEach(() => {
  cleanup()
  vi.resetAllMocks()
})

describe('CastButton Fehlerpfade', () => {
  test('SDK nicht ladbar: Button bleibt, Hinweis erscheint, erneuter Klick lädt erneut', async () => {
    vi.mocked(loadCastSDK).mockResolvedValue('load_failed')
    render(<CastButton masterURL={URL} />)

    clickCast()
    expect(await screen.findByText(MSG_LOAD_FAILED)).toBeTruthy()
    expect(screen.getByRole('button', { name: /Chromecast/i })).toBeTruthy()
    expect(startCastSession).not.toHaveBeenCalled()

    await waitFor(() =>
      expect((screen.getByRole('button', { name: /Chromecast/i }) as HTMLButtonElement).disabled).toBe(false),
    )
    clickCast()
    await waitFor(() => expect(loadCastSDK).toHaveBeenCalledTimes(2))
  })

  test('Browser ohne Cast: Button weicht einem sichtbaren Hinweis', async () => {
    vi.mocked(loadCastSDK).mockResolvedValue('unavailable')
    render(<CastButton masterURL={URL} />)

    clickCast()
    expect(await screen.findByText(MSG_UNAVAILABLE)).toBeTruthy()
    expect(screen.queryByRole('button', { name: /Chromecast/i })).toBeNull()
  })

  test('Geräte-Dialog abgebrochen (ErrorCode-String "cancel"): kein Hinweis', async () => {
    vi.mocked(loadCastSDK).mockResolvedValue('available')
    vi.mocked(startCastSession).mockRejectedValue('cancel')
    render(<CastButton masterURL={URL} />)

    clickCast()
    await waitFor(() => expect(startCastSession).toHaveBeenCalledWith(URL))
    await waitFor(() =>
      expect((screen.getByRole('button', { name: /Chromecast/i }) as HTMLButtonElement).disabled).toBe(false),
    )
    expect(screen.queryByText(MSG_START_FAILED)).toBeNull()
  })

  test('Session scheitert (ErrorCode-String): Hinweis erscheint', async () => {
    vi.mocked(loadCastSDK).mockResolvedValue('available')
    vi.mocked(startCastSession).mockRejectedValue('receiver_unavailable')
    render(<CastButton masterURL={URL} />)

    clickCast()
    expect(await screen.findByText(MSG_START_FAILED)).toBeTruthy()
  })

  test('SDK bereits geladen: kein erneuter Ladeversuch', async () => {
    vi.mocked(isCastAvailable).mockReturnValue(true)
    vi.mocked(startCastSession).mockResolvedValue()
    render(<CastButton masterURL={URL} />)

    clickCast()
    await waitFor(() => expect(startCastSession).toHaveBeenCalledWith(URL))
    expect(loadCastSDK).not.toHaveBeenCalled()
  })
})
