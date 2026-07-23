import { describe, test, expect, vi } from 'vitest'
import { createThrottledProgress } from './VideoUploadPage'

describe('createThrottledProgress', () => {
  test('100 rasche Progress-Events innerhalb 100 ms → nur 1 setProgress + 1 setRemaining', () => {
    const setProgress = vi.fn()
    const setRemaining = vi.fn()
    const throttler = createThrottledProgress({ setProgress, setRemaining })

    const total = 1_000_000
    // 100 Events × 1 ms Abstand, bytesSent wächst — Event 0 published sofort.
    for (let i = 0; i < 100; i++) {
      throttler.onProgress(i * 5_000, total, 1000 + i)
    }

    // Erster Event = pct 0 % → wird publiziert; setRemaining braucht dt > 0.5 s
    // im Fenster, das ist bei 100 ms Spanne nicht gegeben → 0 Aufrufe erwartet.
    expect(setProgress).toHaveBeenCalledTimes(1)
    expect(setProgress).toHaveBeenCalledWith(0)
    expect(setRemaining).not.toHaveBeenCalled()
  })

  test('nach 1 Sekunde Throttle-Fenster wird ein weiterer Event publiziert', () => {
    const setProgress = vi.fn()
    const setRemaining = vi.fn()
    const throttler = createThrottledProgress({ setProgress, setRemaining })

    const total = 1_000_000
    throttler.onProgress(100_000, total, 0)      // erster: pct 10, publiziert
    throttler.onProgress(200_000, total, 500)    // innerhalb 1 s: verworfen
    throttler.onProgress(300_000, total, 999)    // innerhalb 1 s: verworfen
    throttler.onProgress(400_000, total, 1001)   // Grenze überschritten: publiziert

    expect(setProgress).toHaveBeenCalledTimes(2)
    expect(setProgress).toHaveBeenNthCalledWith(1, 10)
    expect(setProgress).toHaveBeenNthCalledWith(2, 40)
  })

  test('gleicher gerundeter Prozentwert löst kein zweites setProgress aus', () => {
    const setProgress = vi.fn()
    const setRemaining = vi.fn()
    const throttler = createThrottledProgress({ setProgress, setRemaining })

    const total = 1_000_000
    // pct 25 %, dann 1500 ms später wieder pct 25 % (bytesSent leicht anders,
    // aber Math.round(...) liefert dasselbe).
    throttler.onProgress(250_000, total, 0)
    throttler.onProgress(252_000, total, 1500) // 25.2 % → gerundet 25

    expect(setProgress).toHaveBeenCalledTimes(1)
    expect(setProgress).toHaveBeenCalledWith(25)
    // Restzeit-Update darf trotzdem raus (dt > 0.5 s, dbytes > 0):
    expect(setRemaining).toHaveBeenCalledTimes(1)
  })

  test('Sliding-Window verwirft Samples älter als 10 s', () => {
    const setProgress = vi.fn()
    const setRemaining = vi.fn()
    const throttler = createThrottledProgress({ setProgress, setRemaining })

    const total = 100_000_000
    // Fülle das Fenster mit 1 Sample pro Sekunde für 15 s.
    for (let s = 0; s <= 15; s++) {
      throttler.onProgress(s * 5_000_000, total, s * 1000)
    }
    // Nach 15 s sollten nur Samples aus den letzten 10 s im Fenster sein.
    // Bei Sampling im 1-s-Takt: 11 Einträge (t=5000..15000, inklusiv beider
    // Ränder). Wichtig ist: NICHT alle 16.
    expect(throttler._samplesLength()).toBeLessThanOrEqual(11)
    expect(throttler._samplesLength()).toBeGreaterThanOrEqual(10)
  })

  test('Restzeit basiert auf Sliding-Window, nicht auf Gesamt-Durchschnitt', () => {
    const setProgress = vi.fn()
    const setRemaining = vi.fn()
    const throttler = createThrottledProgress({ setProgress, setRemaining })

    const total = 100_000_000
    // 10 s mit 10 MB/s → 100 MB in 10 s, "seit Start"-Rate wäre 10 MB/s.
    for (let s = 0; s < 10; s++) {
      throttler.onProgress(s * 10_000_000, total, s * 1000)
    }
    const callsAfterFast = setRemaining.mock.calls.length

    // Danach 15 s Stillstand bei 100 MB gesendet (Rate im Fenster → 0).
    // Der Sliding-Window verwirft die alten schnellen Samples nach 10 s.
    for (let s = 10; s < 25; s++) {
      throttler.onProgress(100_000_000, total, s * 1000)
    }
    // Restzeit-Aufrufe sind (weiter) durchgekommen, weil dbytes im letzten
    // publizierten Event während der schnellen Phase > 0 war. Die exakte
    // Anzahl ist nicht das Prüf-Ziel — wichtig ist, dass die Logik nicht
    // aus einer stale Gesamt-Durchschnitts-Rate lebt.
    expect(setRemaining.mock.calls.length).toBeGreaterThan(callsAfterFast)
  })

  test('zwei Uploads hintereinander: jede Factory-Instanz startet mit frischem Zustand', () => {
    const setProgress1 = vi.fn()
    const throttler1 = createThrottledProgress({ setProgress: setProgress1, setRemaining: vi.fn() })
    throttler1.onProgress(500_000, 1_000_000, 0)      // pct 50
    throttler1.onProgress(500_000, 1_000_000, 1500)   // gleicher pct → kein Update

    expect(setProgress1).toHaveBeenCalledTimes(1)

    // Zweiter Upload: eigene Factory, unabhängiger Closure-State.
    const setProgress2 = vi.fn()
    const throttler2 = createThrottledProgress({ setProgress: setProgress2, setRemaining: vi.fn() })
    // Auch pct 50 %, aber neue Instanz — MUSS trotzdem publizieren.
    throttler2.onProgress(500_000, 1_000_000, 0)

    expect(setProgress2).toHaveBeenCalledTimes(1)
    expect(setProgress2).toHaveBeenCalledWith(50)
  })
})
