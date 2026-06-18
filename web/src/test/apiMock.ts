import MockAdapter from 'axios-mock-adapter'
import { api } from '../lib/api'

let mock: MockAdapter | null = null

const DEFAULT_ME = {
  id: 1,
  email: 'test@test.local',
  name: 'Test User',
  club_functions: [],
  is_parent: false,
  children: [],
}

export interface MockEntry {
  method?: 'get' | 'post' | 'put' | 'delete' | 'any'
  url: string | RegExp
  data: unknown
}

export function setupApiMock(extra?: MockEntry[]): MockAdapter {
  if (mock) mock.restore()
  mock = new MockAdapter(api, { onNoMatch: 'passthrough' })
  // Extra handlers come BEFORE the catch-all so they take priority.
  for (const { method = 'get', url, data } of extra ?? []) {
    const m = method === 'any' ? mock.onAny(url as any) : mock.onGet(url as any)
    m.reply(200, data)
  }
  mock.onGet('/profile/me').reply(200, DEFAULT_ME)
  mock.onAny().reply(200, [])
  return mock
}

export function resetApiMock() {
  if (mock) {
    mock.reset()
    mock.onGet('/profile/me').reply(200, DEFAULT_ME)
    mock.onAny().reply(200, [])
  }
}

export function getApiMock(): MockAdapter {
  if (!mock) return setupApiMock()
  return mock
}
