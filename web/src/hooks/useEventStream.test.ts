import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { act } from 'react'
import { useEventStream, nextBackoffMs } from './useEventStream'

// EventSource-Stub mit Zustandsmaschine: die Tests müssen zwischen „Browser
// reconnectet selbst" (CONNECTING) und „endgültig tot" (CLOSED) unterscheiden
// können — genau diese Unterscheidung trägt die Reconnect-Logik.
class FakeEventSource {
  static instances: FakeEventSource[] = []
  static get last(): FakeEventSource {
    return FakeEventSource.instances[FakeEventSource.instances.length - 1]
  }
  static readonly CONNECTING = 0
  static readonly OPEN = 1
  static readonly CLOSED = 2

  onmessage: ((e: { data: string }) => void) | null = null
  onerror: (() => void) | null = null
  onopen: (() => void) | null = null
  readyState = 0
  url: string
  closed = false

  constructor(url: string) {
    this.url = url
    FakeEventSource.instances.push(this)
  }
  open() {
    this.readyState = FakeEventSource.OPEN
    this.onopen?.()
  }
  emit(data: string) {
    this.onmessage?.({ data })
  }
  /** Fataler Fehler (HTTP-Status) — der Browser gibt auf, wir müssen übernehmen. */
  failFatally() {
    this.readyState = FakeEventSource.CLOSED
    this.onerror?.()
  }
  /** Transportabbruch — der Browser reconnectet selbst. */
  failTransient() {
    this.readyState = FakeEventSource.CONNECTING
    this.onerror?.()
  }
  close() {
    this.closed = true
    this.readyState = FakeEventSource.CLOSED
  }
}

describe('nextBackoffMs', () => {
  test('verdoppelt ab 1s und deckelt bei 30s', () => {
    expect([0, 1, 2, 3, 4, 5, 6, 20].map(nextBackoffMs)).toEqual([
      1000, 2000, 4000, 8000, 16000, 30000, 30000, 30000,
    ])
  })
})

describe('useEventStream', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    FakeEventSource.instances = []
    vi.stubGlobal('EventSource', FakeEventSource as unknown as typeof EventSource)
  })
  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  test('verbindet nicht ohne Identität', () => {
    renderHook(() => useEventStream('/api/chat/events', vi.fn(), null))
    expect(FakeEventSource.instances).toHaveLength(0)
  })

  test('reicht Nachrichten an den Callback durch', () => {
    const cb = vi.fn()
    renderHook(() => useEventStream('/api/chat/events', cb, 1))

    act(() => { FakeEventSource.last.emit('chat:new-message') })

    expect(cb).toHaveBeenCalledWith('chat:new-message')
  })

  test('verbindet nach fatalem Fehler mit wachsender Wartezeit neu', () => {
    renderHook(() => useEventStream('/api/chat/events', vi.fn(), 1))
    expect(FakeEventSource.instances).toHaveLength(1)

    act(() => { FakeEventSource.last.failFatally() })
    // Vor Ablauf der ersten Sekunde passiert nichts.
    act(() => { vi.advanceTimersByTime(999) })
    expect(FakeEventSource.instances).toHaveLength(1)

    act(() => { vi.advanceTimersByTime(1) })
    expect(FakeEventSource.instances).toHaveLength(2)

    // Zweiter Fehlschlag in Folge → 2s.
    act(() => { FakeEventSource.last.failFatally() })
    act(() => { vi.advanceTimersByTime(1000) })
    expect(FakeEventSource.instances).toHaveLength(2)
    act(() => { vi.advanceTimersByTime(1000) })
    expect(FakeEventSource.instances).toHaveLength(3)
  })

  test('erfolgreiche Verbindung setzt die Wartezeit zurück', () => {
    renderHook(() => useEventStream('/api/chat/events', vi.fn(), 1))

    // Zwei Fehlschläge treiben den Backoff hoch …
    act(() => { FakeEventSource.last.failFatally() })
    act(() => { vi.advanceTimersByTime(1000) })
    act(() => { FakeEventSource.last.failFatally() })
    act(() => { vi.advanceTimersByTime(2000) })
    expect(FakeEventSource.instances).toHaveLength(3)

    // … eine offene Verbindung setzt zurück, der nächste Fehler wartet wieder 1s.
    act(() => { FakeEventSource.last.open() })
    act(() => { FakeEventSource.last.failFatally() })
    act(() => { vi.advanceTimersByTime(1000) })
    expect(FakeEventSource.instances).toHaveLength(4)
  })

  test('greift bei transientem Fehler nicht ein — das macht der Browser', () => {
    renderHook(() => useEventStream('/api/chat/events', vi.fn(), 1))

    act(() => { FakeEventSource.last.failTransient() })
    act(() => { vi.advanceTimersByTime(60000) })

    expect(FakeEventSource.instances).toHaveLength(1)
    expect(FakeEventSource.last.closed).toBe(false)
  })

  test('Sichtbarwerden verbindet sofort neu, ohne den Backoff abzuwarten', () => {
    renderHook(() => useEventStream('/api/chat/events', vi.fn(), 1))

    // Backoff hochtreiben, damit „sofort" messbar von „geplant" abweicht.
    act(() => { FakeEventSource.last.failFatally() })
    act(() => { vi.advanceTimersByTime(1000) })
    act(() => { FakeEventSource.last.failFatally() })
    expect(FakeEventSource.instances).toHaveLength(2)

    act(() => {
      vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible')
      document.dispatchEvent(new Event('visibilitychange'))
    })

    expect(FakeEventSource.instances).toHaveLength(3)
  })

  test('Sichtbarwerden lässt eine lebende Verbindung in Ruhe', () => {
    renderHook(() => useEventStream('/api/chat/events', vi.fn(), 1))
    act(() => { FakeEventSource.last.open() })

    act(() => {
      vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible')
      document.dispatchEvent(new Event('visibilitychange'))
    })

    expect(FakeEventSource.instances).toHaveLength(1)
  })

  test('online-Ereignis verbindet einen toten Kanal neu', () => {
    renderHook(() => useEventStream('/api/chat/events', vi.fn(), 1))
    act(() => { FakeEventSource.last.failFatally() })

    act(() => { window.dispatchEvent(new Event('online')) })

    expect(FakeEventSource.instances).toHaveLength(2)
  })

  test('Identitätswechsel schließt den alten Kanal und öffnet einen neuen', () => {
    const { rerender } = renderHook(
      ({ id }: { id: number | null }) => useEventStream('/api/chat/events', vi.fn(), id),
      { initialProps: { id: 1 as number | null } },
    )
    const first = FakeEventSource.last

    rerender({ id: 2 })

    expect(first.closed).toBe(true)
    expect(FakeEventSource.instances).toHaveLength(2)
    expect(FakeEventSource.last).not.toBe(first)
  })

  test('Abmelden schließt den Kanal und verbindet nicht erneut', () => {
    const { rerender } = renderHook(
      ({ id }: { id: number | null }) => useEventStream('/api/chat/events', vi.fn(), id),
      { initialProps: { id: 1 as number | null } },
    )
    const es = FakeEventSource.last

    rerender({ id: null })
    act(() => { vi.advanceTimersByTime(60000) })

    expect(es.closed).toBe(true)
    expect(FakeEventSource.instances).toHaveLength(1)
  })

  test('Unmount räumt Verbindung, Timer und Listener ab', () => {
    const cb = vi.fn()
    const { unmount } = renderHook(() => useEventStream('/api/chat/events', cb, 1))
    const es = FakeEventSource.last

    // Reconnect einplanen, dann unmounten: der Timer darf nicht mehr feuern.
    act(() => { es.failFatally() })
    unmount()
    act(() => { vi.advanceTimersByTime(60000) })
    expect(FakeEventSource.instances).toHaveLength(1)

    // Und die Listener dürfen keinen Kanal mehr aufmachen.
    act(() => { window.dispatchEvent(new Event('online')) })
    expect(FakeEventSource.instances).toHaveLength(1)
    expect(es.closed).toBe(true)
  })
})
