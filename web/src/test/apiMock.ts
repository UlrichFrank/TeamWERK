import MockAdapter from 'axios-mock-adapter'
import { api } from '../lib/api'

let mock: MockAdapter | null = null

export function setupApiMock(): MockAdapter {
  if (mock) mock.restore()
  mock = new MockAdapter(api, { onNoMatch: 'passthrough' })
  mock.onGet('/profile/me').reply(200, {
    id: 1,
    email: 'test@test.local',
    name: 'Test User',
    club_functions: [],
    is_parent: false,
    children: [],
  })
  // Default: alles andere mit leerer Liste beantworten
  mock.onAny().reply(200, [])
  return mock
}

export function resetApiMock() {
  if (mock) {
    mock.reset()
    mock.onGet('/profile/me').reply(200, {
      id: 1,
      email: 'test@test.local',
      name: 'Test User',
      club_functions: [],
      is_parent: false,
      children: [],
    })
    mock.onAny().reply(200, [])
  }
}

export function getApiMock(): MockAdapter {
  if (!mock) return setupApiMock()
  return mock
}
