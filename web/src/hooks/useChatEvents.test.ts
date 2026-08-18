import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { act } from 'react'
import { useChatEvents } from './useChatEvents'

let currentUser: { id: number } | null = { id: 7 }
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({ user: currentUser }),
}))

class FakeEventSource {
  static instances: FakeEventSource[] = []
  static get last(): FakeEventSource {
    return FakeEventSource.instances[FakeEventSource.instances.length - 1]
  }
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
  close() {
    this.closed = true
    this.readyState = 2
  }
}

describe('useChatEvents', () => {
  beforeEach(() => {
    currentUser = { id: 7 }
    FakeEventSource.instances = []
    vi.stubGlobal('EventSource', FakeEventSource as unknown as typeof EventSource)
  })
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  // Regressionsschutz: der Access Token gehört nicht in die URL — der Server
  // wertet ihn nicht aus, Logs speichern ihn trotzdem.
  test('öffnet die Route ohne Query-Parameter', () => {
    renderHook(() => useChatEvents(vi.fn()))

    expect(FakeEventSource.last.url).toBe('/api/chat/events')
    expect(FakeEventSource.last.url).not.toContain('token')
  })

  test('reicht Chat-Ereignisse durch', () => {
    const cb = vi.fn()
    renderHook(() => useChatEvents(cb))

    act(() => { FakeEventSource.last.onmessage?.({ data: 'chat:new-broadcast' }) })

    expect(cb).toHaveBeenCalledWith('chat:new-broadcast')
  })

  test('verbindet ohne angemeldeten Nutzer nicht', () => {
    currentUser = null
    renderHook(() => useChatEvents(vi.fn()))

    expect(FakeEventSource.instances).toHaveLength(0)
  })

  test('baut bei Identitätswechsel neu auf', () => {
    const { rerender } = renderHook(() => useChatEvents(vi.fn()))
    const first = FakeEventSource.last

    currentUser = { id: 42 }
    rerender()

    expect(first.closed).toBe(true)
    expect(FakeEventSource.instances).toHaveLength(2)
  })
})
