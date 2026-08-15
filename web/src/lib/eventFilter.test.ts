import { describe, it, expect } from 'vitest'
import { parseQuery, matchesQuery, normalize } from './eventFilter'

// Kleine Hilfe: filtert eine Liste von {text, dates}-Objekten wie es die
// Seiten-Adapter tun, und gibt die Labels der Treffer zurück.
type Item = { label: string; text: string[]; dates?: string[] }
const apply = (q: string, items: Item[]): string[] => {
  const tokens = parseQuery(q)
  return items.filter(i => matchesQuery(tokens, i.text, i.dates ?? [])).map(i => i.label)
}

describe('normalize', () => {
  it('senkt Groß-/Kleinschreibung und strippt Diakritika', () => {
    expect(normalize('Göppingen')).toBe('goppingen')
    expect(normalize('MÄRZ')).toBe('marz')
  })
})

describe('leerer Ausdruck ist ein No-Op', () => {
  const items: Item[] = [
    { label: 'a', text: ['Ludwigsburg'], dates: ['2026-09-14T00:00:00Z'] },
    { label: 'b', text: ['Göppingen'], dates: ['2026-10-01T00:00:00Z'] },
  ]

  // Invariante 9: der Filter darf bei leerem Feld nichts wegschneiden.
  it.each(['', '   ', '\t \n'])('q=%j liefert alle Elemente', q => {
    expect(parseQuery(q)).toEqual([])
    expect(apply(q, items)).toEqual(['a', 'b'])
  })
})

describe('Tokens sind konjunktiv', () => {
  const items: Item[] = [
    { label: 'sept-ludwigsburg', text: ['Ludwigsburg'], dates: ['2026-09-14T00:00:00Z'] },
    { label: 'sept-goppingen', text: ['Göppingen'], dates: ['2026-09-20T00:00:00Z'] },
    { label: 'okt-ludwigsburg', text: ['Ludwigsburg'], dates: ['2026-10-05T00:00:00Z'] },
  ]

  // Invariante 4: "sept ludwigsburg" ist der Schnitt, nicht die Vereinigung.
  it('liefert nur den Schnitt beider Tokens', () => {
    expect(apply('sept ludwigsburg', items)).toEqual(['sept-ludwigsburg'])
  })

  it('ein einzelnes Token liefert seine volle Menge', () => {
    expect(apply('sept', items)).toEqual(['sept-ludwigsburg', 'sept-goppingen'])
    expect(apply('ludwigsburg', items)).toEqual(['sept-ludwigsburg', 'okt-ludwigsburg'])
  })
})

describe('Interpretationen sind disjunktiv', () => {
  // Invariante 2 — der Kern des Modells: "mar" ist ein Monatspräfix UND ein
  // Freitext-Präfix. Bricht dieser Test, hat jemand die Interpretationen
  // exklusiv statt ODER-verknüpft ausgewertet.
  it('mar trifft März UND die Markthalle', () => {
    const items: Item[] = [
      { label: 'maerz-termin', text: ['Ludwigsburg'], dates: ['2026-03-07T00:00:00Z'] },
      { label: 'markthalle', text: ['Markthalle'], dates: ['2026-11-02T00:00:00Z'] },
      { label: 'weder-noch', text: ['Sporthalle'], dates: ['2026-05-02T00:00:00Z'] },
    ]
    expect(apply('mar', items)).toEqual(['maerz-termin', 'markthalle'])
  })

  it('ein Datums-Token matcht auch als Freitext', () => {
    const items: Item[] = [
      { label: 'im-datum', text: ['Halle'], dates: ['2026-09-14T00:00:00Z'] },
      { label: 'im-text', text: ['Turnier 14.09. Vorrunde'], dates: ['2026-12-01T00:00:00Z'] },
    ]
    expect(apply('14.09.', items)).toEqual(['im-datum', 'im-text'])
  })
})

describe('Datumsauswertung', () => {
  const items: Item[] = [
    { label: '2026', text: ['Halle'], dates: ['2026-09-14T00:00:00Z'] },
    { label: '2027', text: ['Halle'], dates: ['2027-09-14T00:00:00Z'] },
    { label: 'anderer-tag', text: ['Halle'], dates: ['2026-09-15T00:00:00Z'] },
  ]

  // Invariante 3: die Saison läuft über den Jahreswechsel — ein geratenes Jahr
  // würde die Hälfte still verwerfen.
  it('jahreslos trifft beide Jahrgänge', () => {
    expect(apply('14.09.', items)).toEqual(['2026', '2027'])
  })

  it('mit Jahr trifft exakt', () => {
    expect(apply('14.09.2026', items)).toEqual(['2026'])
  })

  it('akzeptiert einstellige Schreibweise ohne Endpunkt', () => {
    expect(apply('14.9', items)).toEqual(['2026', '2027'])
  })

  it('ignoriert die Zeitzone des ISO-Timestamps', () => {
    // 00:00:00Z würde bei new Date() in westlichen Zonen auf den Vortag
    // rutschen; der Vergleich läuft deshalb über slice(0, 10).
    expect(apply('14.09.', [{ label: 'x', text: [], dates: ['2026-09-14T00:00:00Z'] }])).toEqual(['x'])
  })

  it('verwirft unmögliche Datumsangaben als Datum', () => {
    const t = parseQuery('32.13.')
    expect(t[0].date).toBeUndefined()
  })

  it('liest 14.092026 nicht als Datum', () => {
    expect(parseQuery('14.092026')[0].date).toBeUndefined()
  })
})

describe('Monatsnamen', () => {
  const items: Item[] = [
    { label: 'juni', text: ['Halle'], dates: ['2026-06-10T00:00:00Z'] },
    { label: 'juli', text: ['Halle'], dates: ['2026-07-10T00:00:00Z'] },
    { label: 'maerz', text: ['Halle'], dates: ['2026-03-10T00:00:00Z'] },
  ]

  // Invariante 5: erst ab drei Zeichen wird die Menge beim Weitertippen
  // monoton kleiner.
  it('ju ist kein Monat', () => {
    expect(parseQuery('ju')[0].month).toBeUndefined()
    expect(apply('ju', items)).toEqual([])
  })

  it('jun trifft Juni und nicht Juli', () => {
    expect(apply('jun', items)).toEqual(['juni'])
  })

  it('juli trifft Juli und nicht Juni', () => {
    expect(apply('juli', items)).toEqual(['juli'])
  })

  it('mar trifft März nach Diakritika-Strip', () => {
    expect(apply('mar', items)).toEqual(['maerz'])
  })

  it('Monatsnamen matchen jahresübergreifend', () => {
    const zweiJahre: Item[] = [
      { label: 'sep26', text: [], dates: ['2026-09-01T00:00:00Z'] },
      { label: 'sep27', text: [], dates: ['2027-09-01T00:00:00Z'] },
      { label: 'okt26', text: [], dates: ['2026-10-01T00:00:00Z'] },
    ]
    expect(apply('september', zweiJahre)).toEqual(['sep26', 'sep27'])
  })
})

describe('Normalisierung im Match', () => {
  const items: Item[] = [{ label: 'g', text: ['Göppingen'], dates: [] }]

  // Invariante 6: Diakritika werden gestrippt, aber nicht transliteriert.
  it('goppingen findet Göppingen', () => {
    expect(apply('goppingen', items)).toEqual(['g'])
  })

  it('goeppingen findet Göppingen nicht', () => {
    expect(apply('goeppingen', items)).toEqual([])
  })

  it('Großschreibung ist egal', () => {
    expect(apply('GÖPPINGEN', items)).toEqual(['g'])
  })
})

describe('Feldbehandlung', () => {
  it('überspringt leere und fehlende Felder ohne zu werfen', () => {
    const tokens = parseQuery('halle')
    expect(matchesQuery(tokens, ['', null, undefined, 'Sporthalle'], [])).toBe(true)
    expect(matchesQuery(tokens, [null, undefined], [])).toBe(false)
  })

  it('matcht über mehrere Felder hinweg', () => {
    const items: Item[] = [
      { label: 'via-gegner', text: ['Ludwigsburg', 'Sporthalle'], dates: [] },
      { label: 'via-ort', text: ['Kornwestheim', 'Ludwigsburger Halle'], dates: [] },
    ]
    expect(apply('ludwigsburg', items)).toEqual(['via-gegner', 'via-ort'])
  })

  it('dates ist optional', () => {
    expect(matchesQuery(parseQuery('halle'), ['Sporthalle'])).toBe(true)
  })
})
