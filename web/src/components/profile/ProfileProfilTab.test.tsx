import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import ProfileProfilTab from './ProfileProfilTab'
import { Member } from '../../pages/ProfilePage'

const mockGet = vi.fn((_path: string, ..._rest: unknown[]) => Promise.resolve({ data: {} }))
const mockPut = vi.fn((..._args: unknown[]) => Promise.resolve({ data: {} }))
const mockPost = vi.fn((..._args: unknown[]) => Promise.resolve({ data: {} }))
const mockDelete = vi.fn((..._args: unknown[]) => Promise.resolve({ data: {} }))
vi.mock('../../lib/api', () => ({
  api: {
    get: (path: string, ...rest: unknown[]) => mockGet(path, ...rest),
    put: (...args: unknown[]) => mockPut(...args),
    post: (...args: unknown[]) => mockPost(...args),
    delete: (...args: unknown[]) => mockDelete(...args),
  },
}))

function makeMember(overrides: Partial<Member> = {}): Member {
  return {
    id: 7, first_name: 'Klara', last_name: 'Muster',
    date_of_birth: '2008-04-15', pass_number: '', position: '', status: 'aktiv',
    ...overrides,
  }
}

const emptyVisibility = {
  phones_visible: false, address_visible: false, photo_visible: false, email_visible: false, whatsapp_visible: false,
}

describe('ProfileProfilTab — Geburtsdatum', () => {
  beforeEach(() => {
    mockGet.mockReset()
    mockPut.mockClear()
    mockPost.mockClear()
    mockDelete.mockClear()
  })

  test('eigenes Profil: Default-Vorbelegung aus own_member.date_of_birth, solange Self-Service-Wert leer ist', async () => {
    mockGet.mockImplementation((path: string) => {
      if (path === '/profile/me') {
        return Promise.resolve({
          data: {
            street: '', zip: '', city: '', date_of_birth: '',
            own_member: { date_of_birth: '2008-04-15' },
            phones: [], visibility: emptyVisibility,
          },
        })
      }
      if (path === '/profile/account') {
        return Promise.resolve({ data: { first_name: 'Klara', last_name: 'Muster' } })
      }
      if (path === '/members/7/change-drafts') {
        return Promise.resolve({ data: { drafts: [] } })
      }
      return Promise.resolve({ data: {} })
    })

    const { container } = render(
      <ProfileProfilTab children={[]} parents={[]} ownMember={makeMember()} />,
    )

    await waitFor(() => {
      const input = container.querySelector('input[type="date"]') as HTMLInputElement
      expect(input).toBeTruthy()
      expect(input.value).toBe('2008-04-15')
    })
  })

  test('eigenes Profil: Speichern sendet date_of_birth an PUT /profile/me und an change-request', async () => {
    mockGet.mockImplementation((path: string) => {
      if (path === '/profile/me') {
        return Promise.resolve({
          data: {
            street: '', zip: '', city: '', date_of_birth: '2008-04-15',
            own_member: { date_of_birth: '2008-04-15' },
            phones: [], visibility: emptyVisibility,
          },
        })
      }
      if (path === '/profile/account') {
        return Promise.resolve({ data: { first_name: 'Klara', last_name: 'Muster' } })
      }
      if (path === '/members/7/change-drafts') {
        return Promise.resolve({ data: { drafts: [] } })
      }
      return Promise.resolve({ data: {} })
    })

    const { container } = render(
      <ProfileProfilTab children={[]} parents={[]} ownMember={makeMember()} />,
    )

    const input = await waitFor(() => {
      const el = container.querySelector('input[type="date"]') as HTMLInputElement
      expect(el.value).toBe('2008-04-15')
      return el
    })
    fireEvent.change(input, { target: { value: '2009-01-01' } })

    const save = screen.getByRole('button', { name: /^Speichern$/ })
    fireEvent.click(save)

    await waitFor(() => {
      expect(mockPut).toHaveBeenCalledWith('/profile/me', expect.objectContaining({ date_of_birth: '2009-01-01' }))
      expect(mockPost).toHaveBeenCalledWith('/members/7/change-request', expect.objectContaining({
        field_name: 'profil',
        new_value: expect.objectContaining({ date_of_birth: '2009-01-01' }),
      }))
    })
  })

  test('eigenes Profil ohne verknüpftes Mitglied: kein Geburtsdatum-Feld', async () => {
    mockGet.mockImplementation((path: string) => {
      if (path === '/profile/me') {
        return Promise.resolve({ data: { street: '', zip: '', city: '', phones: [], visibility: emptyVisibility } })
      }
      if (path === '/profile/account') {
        return Promise.resolve({ data: { first_name: '', last_name: '' } })
      }
      return Promise.resolve({ data: {} })
    })

    const { container } = render(
      <ProfileProfilTab children={[]} parents={[]} ownMember={null} />,
    )

    await waitFor(() => expect(mockGet).toHaveBeenCalled())
    expect(container.querySelector('input[type="date"]')).toBeNull()
    expect(screen.queryByText('Geburtsdatum')).toBeNull()
  })

  test('Kind-Profil mit eigenem Account: Feld sichtbar, Default aus member.date_of_birth', () => {
    const { container } = render(
      <ProfileProfilTab
        mode="child"
        childMemberId="42"
        ownMember={makeMember({ id: 42, date_of_birth: '2016-03-01' })}
        userContact={{
          first_name: 'Max', last_name: 'Muster', street: '', zip: '', city: '', date_of_birth: '',
          recovery_email: '', phones: [], visibility: emptyVisibility,
        }}
        children={[]}
        parents={[]}
      />,
    )

    const input = container.querySelector('input[type="date"]') as HTMLInputElement
    expect(input).toBeTruthy()
    expect(input.value).toBe('2016-03-01')
  })

  test('Kind-Profil ohne eigenen Account: kein Geburtsdatum-Feld', () => {
    const { container } = render(
      <ProfileProfilTab
        mode="child"
        childMemberId="42"
        ownMember={makeMember({ id: 42, date_of_birth: '2016-03-01' })}
        userContact={null}
        children={[]}
        parents={[]}
      />,
    )

    expect(container.querySelector('input[type="date"]')).toBeNull()
    expect(screen.queryByText('Geburtsdatum')).toBeNull()
  })
})
