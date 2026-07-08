import { describe, it, expect } from 'vitest'
import { conversationTimeLabel, daySeparatorLabel, shouldRenderSeparator } from './chatDateFormat'

describe('daySeparatorLabel', () => {
  it('returns "Heute" for same calendar day', () => {
    const now = new Date(2026, 5, 18, 14, 0)
    const date = new Date(2026, 5, 18, 9, 30)
    expect(daySeparatorLabel(date, now)).toBe('Heute')
  })

  it('returns "Gestern" for the previous calendar day', () => {
    const now = new Date(2026, 5, 18, 14, 0)
    const date = new Date(2026, 5, 17, 23, 50)
    expect(daySeparatorLabel(date, now)).toBe('Gestern')
  })

  it('returns weekday and full date for two days ago', () => {
    const now = new Date(2026, 5, 18)
    const date = new Date(2026, 5, 16, 12, 0)
    expect(daySeparatorLabel(date, now)).toBe('Dienstag, 16. Juni 2026')
  })

  it('returns weekday and full date including year for older dates', () => {
    const now = new Date(2026, 5, 18)
    const date = new Date(2025, 11, 24, 18, 0)
    expect(daySeparatorLabel(date, now)).toBe('Mittwoch, 24. Dezember 2025')
  })

  it('returns "Gestern" across midnight even when distance is under 1 hour', () => {
    const now = new Date(2026, 5, 18, 0, 30)
    const date = new Date(2026, 5, 17, 23, 30)
    expect(daySeparatorLabel(date, now)).toBe('Gestern')
  })

  it('handles DST spring transition correctly (calendar days, not 24h)', () => {
    // 2026 DST in Europe: 29.03.2026 02:00 → 03:00. The day after is 30.03.
    // Date 28.03 → 30.03 should be 2 calendar days regardless of DST.
    const now = new Date(2026, 2, 30, 10, 0)
    const date = new Date(2026, 2, 28, 22, 0)
    expect(daySeparatorLabel(date, now)).toBe('Samstag, 28. März 2026')
  })
})

describe('conversationTimeLabel', () => {
  it('returns the time for activity today', () => {
    const now = new Date(2026, 5, 18, 14, 0)
    const date = new Date(2026, 5, 18, 9, 30)
    expect(conversationTimeLabel(date, now)).toBe('09:30')
  })

  it('returns "Gestern" for the previous calendar day', () => {
    const now = new Date(2026, 5, 18, 14, 0)
    const date = new Date(2026, 5, 17, 23, 50)
    expect(conversationTimeLabel(date, now)).toBe('Gestern')
  })

  it('returns the weekday for 3 calendar days ago', () => {
    // 2026-06-15 is a Monday
    const now = new Date(2026, 5, 18, 14, 0)
    const date = new Date(2026, 5, 15, 12, 0)
    expect(conversationTimeLabel(date, now)).toBe('Montag')
  })

  it('returns the weekday at the 6-day edge', () => {
    // 2026-06-12 is a Friday, still within the < 7 window
    const now = new Date(2026, 5, 18, 14, 0)
    const date = new Date(2026, 5, 12, 8, 0)
    expect(conversationTimeLabel(date, now)).toBe('Freitag')
  })

  it('returns the numeric date at exactly 7 calendar days (no weekday collision)', () => {
    const now = new Date(2026, 5, 18, 14, 0)
    const date = new Date(2026, 5, 11, 8, 0)
    expect(conversationTimeLabel(date, now)).toBe('11.06.26')
  })

  it('returns the numeric date for older activity', () => {
    const now = new Date(2026, 5, 18, 14, 0)
    const date = new Date(2025, 11, 24, 18, 0)
    expect(conversationTimeLabel(date, now)).toBe('24.12.25')
  })

  it('uses "Gestern" across midnight even under one hour apart', () => {
    const now = new Date(2026, 5, 18, 0, 30)
    const date = new Date(2026, 5, 17, 23, 30)
    expect(conversationTimeLabel(date, now)).toBe('Gestern')
  })
})

describe('shouldRenderSeparator', () => {
  it('returns true when there is no previous message', () => {
    expect(shouldRenderSeparator(null, '2026-06-18T09:00:00Z')).toBe(true)
  })

  it('returns false when previous and current are on the same local day', () => {
    const prev = new Date(2026, 5, 18, 9, 0).toISOString()
    const curr = new Date(2026, 5, 18, 22, 30).toISOString()
    expect(shouldRenderSeparator(prev, curr)).toBe(false)
  })

  it('returns true when previous and current span a day change', () => {
    const prev = new Date(2026, 5, 17, 23, 55).toISOString()
    const curr = new Date(2026, 5, 18, 0, 5).toISOString()
    expect(shouldRenderSeparator(prev, curr)).toBe(true)
  })
})
