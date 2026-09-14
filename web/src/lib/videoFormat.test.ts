import { describe, it, expect } from 'vitest'
import { fmtBytes } from './videoFormat'

describe('fmtBytes', () => {
  it('formats null/undefined as a dash (size not known yet)', () => {
    expect(fmtBytes(null)).toBe('–')
    expect(fmtBytes(undefined)).toBe('–')
  })

  it('formats negative values as a dash', () => {
    expect(fmtBytes(-1)).toBe('–')
  })

  it('formats sub-GB sizes in MB', () => {
    expect(fmtBytes(500 * 1024 ** 2)).toBe('500.0 MB')
  })

  it('formats GB-and-above sizes in GB', () => {
    expect(fmtBytes(1.5 * 1024 ** 3)).toBe('1.5 GB')
  })

  it('formats zero as 0.0 MB', () => {
    expect(fmtBytes(0)).toBe('0.0 MB')
  })
})
