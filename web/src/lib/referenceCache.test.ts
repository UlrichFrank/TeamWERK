import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { api, getReference, invalidateReferenceCache, clearReferenceCache, setAccessToken } from './api'

// Token-Helfer: baut ein minimales JWT mit uid-Claim (unsigniert reicht — der
// Cache liest den Payload nur, verifiziert nicht).
function tokenFor(uid: number): string {
  const payload = btoa(JSON.stringify({ uid }))
  return `x.${payload}.y`
}

describe('getReference — TTL-Cache + Single-Flight', () => {
  beforeEach(() => {
    clearReferenceCache()
    setAccessToken(null)
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  test('zweiter Abruf innerhalb der TTL trifft den Cache (nur ein HTTP-Request)', async () => {
    const spy = vi.spyOn(api, 'get').mockResolvedValue({ data: [{ id: 1 }] } as never)

    const first = await getReference<{ id: number }[]>('/teams')
    const second = await getReference<{ id: number }[]>('/teams')

    expect(spy).toHaveBeenCalledTimes(1)
    expect(first).toEqual([{ id: 1 }])
    expect(second).toEqual([{ id: 1 }])
  })

  test('parallele Abrufe teilen sich einen In-Flight-Request (Single-Flight)', async () => {
    let resolveFn: (v: unknown) => void = () => {}
    const spy = vi.spyOn(api, 'get').mockImplementation(
      () => new Promise(res => { resolveFn = res }) as never,
    )

    const p1 = getReference<{ id: number }[]>('/venues')
    const p2 = getReference<{ id: number }[]>('/venues')
    resolveFn({ data: [{ id: 9 }] })
    const [r1, r2] = await Promise.all([p1, p2])

    expect(spy).toHaveBeenCalledTimes(1)
    expect(r1).toEqual([{ id: 9 }])
    expect(r2).toEqual([{ id: 9 }])
  })

  test('Live-Update-Event invalidiert den passenden Cache-Eintrag', async () => {
    const spy = vi.spyOn(api, 'get').mockResolvedValue({ data: ['2025/26'] } as never)

    await getReference<string[]>('/seasons')
    invalidateReferenceCache('seasons')
    await getReference<string[]>('/seasons')

    // Nach Invalidierung muss der nächste Abruf frisch laden.
    expect(spy).toHaveBeenCalledTimes(2)
  })

  test('unpassendes Event lässt den Cache unangetastet', async () => {
    const spy = vi.spyOn(api, 'get').mockResolvedValue({ data: ['2025/26'] } as never)

    await getReference<string[]>('/seasons')
    invalidateReferenceCache('chat') // hört nicht auf /seasons
    await getReference<string[]>('/seasons')

    expect(spy).toHaveBeenCalledTimes(1)
  })

  test('Identitätswechsel (setAccessToken auf andere uid) leert den Cache', async () => {
    const spy = vi.spyOn(api, 'get').mockResolvedValue({ data: [{ id: 1 }] } as never)

    setAccessToken(tokenFor(1))
    await getReference<{ id: number }[]>('/teams')
    // Wechsel auf anderen Nutzer → Cache muss geleert werden.
    setAccessToken(tokenFor(2))
    await getReference<{ id: number }[]>('/teams')

    expect(spy).toHaveBeenCalledTimes(2)
  })

  test('Token-Refresh desselben Nutzers behält den Cache', async () => {
    const spy = vi.spyOn(api, 'get').mockResolvedValue({ data: [{ id: 1 }] } as never)

    setAccessToken(tokenFor(5))
    await getReference<{ id: number }[]>('/teams')
    // Neuer Token, gleiche uid (Refresh) → Cache bleibt.
    setAccessToken(tokenFor(5))
    await getReference<{ id: number }[]>('/teams')

    expect(spy).toHaveBeenCalledTimes(1)
  })

  test('Nicht-Allowlist-Route wird nicht gecacht', async () => {
    const spy = vi.spyOn(api, 'get').mockResolvedValue({ data: { ok: true } } as never)

    await getReference('/members')
    await getReference('/members')

    expect(spy).toHaveBeenCalledTimes(2)
  })

  test('force umgeht den Cache, füllt ihn aber neu', async () => {
    const spy = vi.spyOn(api, 'get').mockResolvedValue({ data: [{ id: 1 }] } as never)

    await getReference('/teams')
    await getReference('/teams', { force: true } as never)
    await getReference('/teams')

    // 1 initial + 1 forced; der dritte trifft den frisch gefüllten Cache.
    expect(spy).toHaveBeenCalledTimes(2)
  })
})
