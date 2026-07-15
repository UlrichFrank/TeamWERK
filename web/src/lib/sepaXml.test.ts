import { describe, it, expect } from 'vitest'
import { buildPainXML, nextBusinessDay, type SepaBuildInput } from './sepaXml'

// Eingabe gespiegelt aus internal/beitragslauf/xml_test.go (sampleInput).
function sampleInput(): SepaBuildInput {
  return {
    saisonKurz: '2026/27',
    clubName: 'Team Stuttgart',
    glaeubigerId: 'DE98ZZZ09999999999',
    clubIban: 'DE89370400440532013000',
    bic: 'GENODEF1S02',
    kontoinhaber: 'Team Stuttgart e.V.',
    faelligkeit: '2026-07-01',
    createdAt: new Date(Date.UTC(2026, 5, 18, 10, 0, 0)),
    items: [
      {
        name: 'Max Müller', street: 'Hauptstr. 12', zip: '70182', city: 'Stuttgart',
        iban: 'DE89370400440532013000', betragCent: 9600, mandatRef: '1042',
        mandatDatum: '2026-05-01', memberNumber: '1042',
      },
    ],
  }
}

describe('buildPainXML (Parität zu internal/beitragslauf/xml.go)', () => {
  const xml = buildPainXML(sampleInput())

  it('genau ein PmtInf-Block, ausschließlich RCUR', () => {
    expect((xml.match(/<PmtInf>/g) || []).length).toBe(1)
    expect((xml.match(/<SeqTp>RCUR<\/SeqTp>/g) || []).length).toBe(1)
    expect(xml).not.toContain('FRST')
  })

  it('Namespace + Betrags-/Summenformat', () => {
    expect(xml).toContain('urn:iso:std:iso:20022:tech:xsd:pain.008.001.08')
    expect(xml).toContain('<InstdAmt Ccy="EUR">96.00</InstdAmt>')
    expect(xml).toContain('<CtrlSum>96.00</CtrlSum>')
  })

  it('Gläubiger-ID, IBANs, Verwendungszweck, ASCII-Umlaut in Name', () => {
    expect(xml).toContain('<IBAN>DE89370400440532013000</IBAN>')
    expect(xml).toContain('<Id>DE98ZZZ09999999999</Id>')
    expect(xml).toContain('Mitgliedsbeitrag Team Stuttgart 26 - Mitglied 1042')
    expect(xml).toContain('<EndToEndId>TW-1042-2026-27</EndToEndId>')
    // Umlaut im Dbtr-Namen bleibt erhalten (kein ASCII-Zwang für Nm/Ustrd? -> Go ascii()t Nm)
    expect(xml).toContain('<Nm>Max Mueller</Nm>')
    // Straße/Hausnummer getrennt
    expect(xml).toContain('<StrtNm>Hauptstr.</StrtNm>')
    expect(xml).toContain('<BldgNb>12</BldgNb>')
    expect(xml).toContain('<ReqdColltnDt>2026-07-01</ReqdColltnDt>')
  })

  it('valides XML-Prolog + Wohlgeformtheit (ausgeglichene Document-Tags)', () => {
    expect(xml.startsWith('<?xml version="1.0" encoding="UTF-8"?>\n<Document')).toBe(true)
    expect((xml.match(/<Document/g) || []).length).toBe(1)
    expect((xml.match(/<\/Document>/g) || []).length).toBe(1)
  })
})

describe('DtOfSgntr weglassen, wenn kein Mandatsdatum vorliegt', () => {
  it('fehlendes Mandatsdatum → kein leeres <DtOfSgntr>-Element (XSD-Verstoß bei Banken)', () => {
    const input = sampleInput()
    input.items[0].mandatDatum = ''
    const xml = buildPainXML(input)
    expect(xml).not.toContain('<DtOfSgntr>')
    expect(xml).toContain('<MndtId>1042</MndtId>')
  })

  it('mit Mandatsdatum → <DtOfSgntr> mit ISODate', () => {
    const xml = buildPainXML(sampleInput())
    expect(xml).toContain('<DtOfSgntr>2026-05-01</DtOfSgntr>')
  })
})

describe('nextBusinessDay', () => {
  it('verschiebt Samstag/Sonntag auf Montag', () => {
    expect(nextBusinessDay('2026-07-04')).toBe('2026-07-06') // Sa → Mo
    expect(nextBusinessDay('2026-07-05')).toBe('2026-07-06') // So → Mo
    expect(nextBusinessDay('2026-07-01')).toBe('2026-07-01') // Mi unverändert
  })
})
