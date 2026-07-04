import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { act } from 'react'
import { useLiveUpdates } from './useLiveUpdates'

// useAuth: immer eingeloggt, damit die EventSource verbunden wird.
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({ user: { id: 1 } }),
}))

const invalidate = vi.fn()
vi.mock('../lib/api', () => ({
  invalidateReferenceCache: (...args: unknown[]) => invalidate(...args),
}))

// Minimaler EventSource-Stub: merkt sich die letzte Instanz, damit der Test
// onmessage-Frames einspielen kann.
class FakeEventSource {
  static last: FakeEventSource | null = null
  onmessage: ((e: { data: string }) => void) | null = null
  onerror: (() => void) | null = null
  readyState = 0
  url: string
  closed = false
  constructor(url: string) {
    this.url = url
    FakeEventSource.last = this
  }
  emit(data: string) {
    this.onmessage?.({ data })
  }
  close() {
    this.closed = true
  }
}

describe('useLiveUpdates — Coalescing', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    invalidate.mockReset()
    FakeEventSource.last = null
    vi.stubGlobal('EventSource', FakeEventSource as unknown as typeof EventSource)
  })
  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  test('Burst gleicher Events löst genau einen Reload aus', () => {
    const cb = vi.fn()
    renderHook(() => useLiveUpdates(cb))
    const es = FakeEventSource.last!

    act(() => {
      es.emit('duties')
      es.emit('duties')
      es.emit('duties')
    })
    // Vor Fensterablauf: noch kein Callback.
    expect(cb).not.toHaveBeenCalled()

    act(() => { vi.advanceTimersByTime(300) })
    expect(cb).toHaveBeenCalledTimes(1)
    expect(cb).toHaveBeenCalledWith('duties')
    // Cache-Invalidierung passiert pro Event sofort (nicht debounced).
    expect(invalidate).toHaveBeenCalledTimes(3)
  })

  test('Verschiedene Event-Typen im Fenster: je Typ ein Callback', () => {
    const cb = vi.fn()
    renderHook(() => useLiveUpdates(cb))
    const es = FakeEventSource.last!

    act(() => {
      es.emit('games')
      es.emit('trainings')
      es.emit('games')
    })
    act(() => { vi.advanceTimersByTime(300) })

    expect(cb).toHaveBeenCalledTimes(2)
    const calledWith = cb.mock.calls.map(c => c[0]).sort()
    expect(calledWith).toEqual(['games', 'trainings'])
  })

  test('__version:-Event umgeht Coalescing und Callback komplett', () => {
    const cb = vi.fn()
    renderHook(() => useLiveUpdates(cb))
    const es = FakeEventSource.last!

    act(() => { es.emit('__version:abc123') })
    act(() => { vi.advanceTimersByTime(300) })

    expect(cb).not.toHaveBeenCalled()
    expect(invalidate).not.toHaveBeenCalled()
  })

  test('Aufräumen schließt die EventSource und stoppt den Timer', () => {
    const cb = vi.fn()
    const { unmount } = renderHook(() => useLiveUpdates(cb))
    const es = FakeEventSource.last!

    act(() => { es.emit('duties') })
    unmount()
    // Nach Unmount darf der ausstehende Flush nicht mehr feuern.
    act(() => { vi.advanceTimersByTime(300) })

    expect(es.closed).toBe(true)
    expect(cb).not.toHaveBeenCalled()
  })
})
