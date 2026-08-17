import { describe, it, expect } from 'vitest'
import { chatUnreadCounts } from './chatUnread'

describe('chatUnreadCounts', () => {
  it('weist Konversationen und Mitteilungen getrennt aus', () => {
    const result = chatUnreadCounts(
      [{ unreadCount: 2 }, { unreadCount: 1 }],
      [
        { isRead: false, isSent: false },
        { isRead: false, isSent: false },
        { isRead: false, isSent: false },
      ],
    )
    expect(result).toEqual({ conversations: 3, broadcasts: 3, total: 6 })
  })

  it('zählt eine eigene Mitteilung nicht mit, auch wenn sie ungelesen ist', () => {
    const result = chatUnreadCounts([], [{ isRead: false, isSent: true }])
    expect(result.broadcasts).toBe(0)
    expect(result.total).toBe(0)
  })

  it('zählt gelesene Mitteilungen nicht mit', () => {
    const result = chatUnreadCounts([], [{ isRead: true, isSent: false }])
    expect(result.broadcasts).toBe(0)
  })

  it('ignoriert Konversationen ohne ungelesene Nachrichten', () => {
    const result = chatUnreadCounts([{ unreadCount: 0 }, { unreadCount: 4 }], [])
    expect(result.conversations).toBe(4)
    expect(result.total).toBe(4)
  })

  it('liefert für leere Listen überall 0', () => {
    expect(chatUnreadCounts([], [])).toEqual({ conversations: 0, broadcasts: 0, total: 0 })
  })

  it('behandelt fehlende Listen wie leere', () => {
    expect(chatUnreadCounts(undefined, null)).toEqual({
      conversations: 0,
      broadcasts: 0,
      total: 0,
    })
  })

  it('total ist immer die Summe beider Anteile', () => {
    const result = chatUnreadCounts(
      [{ unreadCount: 7 }],
      [
        { isRead: false, isSent: false },
        { isRead: true, isSent: false },
        { isRead: false, isSent: true },
      ],
    )
    expect(result.total).toBe(result.conversations + result.broadcasts)
    expect(result.total).toBe(8)
  })
})
