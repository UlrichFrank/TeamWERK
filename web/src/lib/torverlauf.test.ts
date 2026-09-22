import { describe, it, expect } from 'vitest'
import type { EventLine } from './staffeln'
import { torereihe, halbzeitGrenze, halbzeitMinuten, achsen } from './torverlauf'

let seq = 0
function ev(kind: EventLine['kind'], side: EventLine['side'], sec: number, scorer = 'X'): EventLine {
  seq += 1
  return {
    seq, clockTime: '', gameSecond: sec, scoreHome: null, scoreGuest: null,
    kind, side, playerId: null, playerName: scorer, number: null, rawText: '',
  }
}
function tor(side: 'home' | 'guest', sec: number, scorer = 'X') { return ev('goal', side, sec, scorer) }

describe('torereihe', () => {
  it('zaehlt aufeinanderfolgende Tore derselben Mannschaft', () => {
    const t = torereihe([tor('home', 60), tor('home', 120), tor('home', 180), tor('guest', 240)])
    expect(t.map((x) => x.lauf)).toEqual([1, 2, 3, 1])
  })

  it('bestimmt Fuehrung, Gleichstand und Rueckstand aus Sicht des Schuetzen', () => {
    const t = torereihe([tor('home', 60), tor('guest', 120), tor('guest', 180), tor('home', 240)])
    expect(t[0].situation).toBe('fuehrung')      // 1:0, Heim wirft und fuehrt
    expect(t[1].situation).toBe('gleichstand')   // 1:1, Gast wirft zum Ausgleich
    expect(t[2].situation).toBe('fuehrung')      // 1:2, Gast wirft und fuehrt nun
    expect(t[3].situation).toBe('gleichstand')   // 2:2, Heim gleicht aus
  })

  // Der Rueckstand braucht einen eigenen Fall: Heim verkuerzt, liegt aber
  // weiter hinten. Die Situation bezieht sich auf den Stand NACH dem Tor und
  // auf die werfende Mannschaft.
  it('ein verkuerzendes Tor bleibt ein Tor aus dem Rueckstand', () => {
    const t = torereihe([tor('guest', 60), tor('guest', 120), tor('home', 180)])
    expect(t[2].situation).toBe('rueckstand')    // 1:2, Heim wirft, liegt hinten
  })

  it('zaehlt den Spielstand mit, statt ihn aus dem Feld zu lesen', () => {
    const t = torereihe([tor('home', 60), tor('guest', 120), tor('home', 180)])
    expect([t[2].scoreHome, t[2].scoreGuest]).toEqual([2, 1])
  })

  it('Auszeiten, Strafen und verworfene Siebenmeter erzeugen keinen Kreis', () => {
    const t = torereihe([
      tor('home', 60),
      ev('seven_m_miss', 'home', 90),
      ev('two_min', 'guest', 120),
      ev('timeout', 'home', 150),
      ev('warning', 'guest', 160),
    ])
    expect(t).toHaveLength(1)
  })

  it('kennzeichnet Tore aus Siebenmetern', () => {
    const t = torereihe([ev('seven_m_goal', 'home', 60), tor('home', 120)])
    expect(t.map((x) => x.sevenMeter)).toEqual([true, false])
    expect(t[1].scoreHome).toBe(2)
  })

  it('ignoriert Tore ohne zuordenbare Seite', () => {
    expect(torereihe([ev('goal', '', 60)])).toHaveLength(0)
  })
})

describe('halbzeitGrenze', () => {
  it('trennt die Halbzeiten am Halbzeitstand des Kopfes', () => {
    const t = torereihe([tor('home', 60), tor('guest', 120), tor('home', 1317), tor('guest', 1527)])
    // Halbzeitstand 2:1 -> die ersten drei Tore sind die erste Halbzeit.
    expect(halbzeitGrenze(t, 2, 1)).toBe(3)
  })

  it('ohne Halbzeitstand bleibt eine durchgehende Achse', () => {
    const t = torereihe([tor('home', 60), tor('guest', 120)])
    expect(halbzeitGrenze(t, null, null)).toBeNull()
  })

  it('ein torloser Halbzeitstand ergibt eine leere erste Halbzeit', () => {
    const t = torereihe([tor('home', 1600)])
    expect(halbzeitGrenze(t, 0, 0)).toBe(0)
  })

  it('ein im Verlauf nicht auftauchender Halbzeitstand faellt auf eine Achse zurueck', () => {
    const t = torereihe([tor('home', 60), tor('home', 120)])
    expect(halbzeitGrenze(t, 5, 3)).toBeNull()
  })
})

// Die Zahlen stammen aus der eingecheckten Fixture spielbericht_905272:
// Halbzeitstand bei 21:57, erstes Tor der zweiten Halbzeit bei 25:27,
// letztes Ereignis bei 49:06 -> 2x25 Minuten.
const FIXTURE = torereihe([
  tor('home', 99), tor('guest', 1317), tor('home', 1527), tor('guest', 2946),
])
const FIXTURE_GRENZE = 2

describe('halbzeitMinuten', () => {
  it('nutzt die konfigurierte Spieldauer', () => {
    expect(halbzeitMinuten(FIXTURE, FIXTURE_GRENZE, 25)).toBe(25)
  })

  it('verwirft eine zu lange konfigurierte Spieldauer', () => {
    // 30 Minuten legten das erste Tor der zweiten Halbzeit (25:27) vor den
    // Beginn seiner eigenen Achse.
    expect(halbzeitMinuten(FIXTURE, FIXTURE_GRENZE, 30)).toBe(25)
  })

  it('verwirft eine zu kurze konfigurierte Spieldauer', () => {
    // 20 Minuten liessen das letzte Tor der ersten Halbzeit (21:57) herausfallen.
    expect(halbzeitMinuten(FIXTURE, FIXTURE_GRENZE, 20)).toBe(25)
  })

  it('leitet die Spieldauer ohne Regel aus dem Verlauf ab', () => {
    expect(halbzeitMinuten(FIXTURE, FIXTURE_GRENZE, null)).toBe(25)

    const zwanzig = torereihe([tor('home', 60), tor('guest', 1150), tor('home', 1300), tor('guest', 2350)])
    expect(halbzeitMinuten(zwanzig, 2, null)).toBe(20)

    const dreissig = torereihe([tor('home', 60), tor('guest', 1700), tor('home', 1900), tor('guest', 3540)])
    expect(halbzeitMinuten(dreissig, 2, null)).toBe(30)
  })
})

describe('achsen', () => {
  it('verteilt die Tore auf zwei Halbzeiten', () => {
    const a = achsen(FIXTURE, FIXTURE_GRENZE, 25)
    expect(a).toHaveLength(2)
    expect(a[0].tore).toHaveLength(2)
    expect(a[1].tore).toHaveLength(2)
    expect([a[0].vonMinute, a[0].bisMinute]).toEqual([0, 25])
    expect([a[1].vonMinute, a[1].bisMinute]).toEqual([25, 50])
  })

  it('kein Tor liegt ausserhalb seiner Achse', () => {
    for (const konfiguriert of [25, 30, 20, null]) {
      const minuten = halbzeitMinuten(FIXTURE, FIXTURE_GRENZE, konfiguriert)
      for (const hz of achsen(FIXTURE, FIXTURE_GRENZE, minuten)) {
        for (const t of hz.tore) {
          const m = t.gameSecond / 60
          expect(m, `konfiguriert=${konfiguriert}, Halbzeit ${hz.nummer}`).toBeGreaterThanOrEqual(hz.vonMinute)
          expect(m, `konfiguriert=${konfiguriert}, Halbzeit ${hz.nummer}`).toBeLessThanOrEqual(hz.bisMinute)
        }
      }
    }
  })

  it('dehnt die Achse, statt ein spaetes Tor abzuschneiden', () => {
    const spaet = torereihe([tor('home', 60), tor('guest', 1400), tor('home', 1600), tor('guest', 3100)])
    const a = achsen(spaet, 2, 25)
    expect(a[1].bisMinute).toBeGreaterThanOrEqual(3100 / 60)
  })

  it('ohne Halbzeitgrenze bleibt eine Achse', () => {
    const a = achsen(FIXTURE, null, 25)
    expect(a).toHaveLength(1)
    expect(a[0].tore).toHaveLength(4)
  })

  it('die Halbzeitzuordnung haengt nicht an der Spieldauer', () => {
    const verteilung = (min: number) => achsen(FIXTURE, FIXTURE_GRENZE, min).map((h) => h.tore.length)
    expect(verteilung(20)).toEqual(verteilung(25))
    expect(verteilung(30)).toEqual(verteilung(25))
  })
})
