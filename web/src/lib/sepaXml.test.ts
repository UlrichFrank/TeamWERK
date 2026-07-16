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
  const { xml } = buildPainXML(sampleInput())

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

  it('CreDtTm mit UTC-Marker Z (BW-Bank rejectet ohne Zeitzone)', () => {
    expect(xml).toMatch(/<CreDtTm>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z<\/CreDtTm>/)
  })

  it('DK-TVS-Verstöße (GBIC_5, Anlage 3 V26.11): kein FwdgAgt, kein InitgPty/Id', () => {
    // GrpHdr laut DK-TVS nur MsgId/CreDtTm/NbOfTxs/CtrlSum/InitgPty. FwdgAgt und
    // InitgPty/Id lassen BW-Bank per XSD ablehnen — Regression-Guard.
    expect(xml).not.toContain('<FwdgAgt>')
    expect(xml).not.toContain('<OrgId>')
    // InitgPty enthält nur <Nm>, kein <Id>-Kind
    expect(xml).toMatch(/<InitgPty>\s*<Nm>[^<]+<\/Nm>\s*<\/InitgPty>/)
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

describe('DtOfSgntr (Pflichtelement laut GBIC_5-TVS)', () => {
  it('fehlendes Mandatsdatum → Fallback auf 2026-06-01', () => {
    const input = sampleInput()
    input.items[0].mandatDatum = ''
    const { xml } = buildPainXML(input)
    expect(xml).toContain('<DtOfSgntr>2026-06-01</DtOfSgntr>')
    expect(xml).toContain('<MndtId>1042</MndtId>')
  })

  it('mit Mandatsdatum → <DtOfSgntr> mit erfasstem ISODate', () => {
    const { xml } = buildPainXML(sampleInput())
    expect(xml).toContain('<DtOfSgntr>2026-05-01</DtOfSgntr>')
  })
})

describe('DK-TVS-Härtungen (GBIC_5, Anlage 3 V26.11)', () => {
  it('<Nm> auf 70 Zeichen begrenzt (Max140Text_SDD → DK: 70)', () => {
    const input = sampleInput()
    input.items[0].name = 'Alexander-Maximilian ' + 'Freiherr-von-und-zu-Musterstadt-Lichtenberg-Rothenburg'
    const { xml } = buildPainXML(input)
    const m = xml.match(/<Nm>([^<]+)<\/Nm>/g)!
    for (const nm of m) {
      const value = nm.replace(/<\/?Nm>/g, '')
      expect(value.length).toBeLessThanOrEqual(70)
    }
  })

  it('Truncation liefert Warning an den Aufrufer (nicht stumm)', () => {
    const input = sampleInput()
    const longName = 'A'.repeat(120)
    input.items[0].name = longName
    const { warnings } = buildPainXML(input)
    const debtor = warnings.find(w => w.location === 'debtor-name')
    expect(debtor).toBeDefined()
    expect(debtor!.memberNumber).toBe('1042')
    expect(debtor!.maxLen).toBe(70)
    expect(debtor!.original.length).toBe(120)
    expect(debtor!.truncated.length).toBe(70)
  })

  it('keine Truncation → keine Warnings', () => {
    const { warnings } = buildPainXML(sampleInput())
    expect(warnings).toEqual([])
  })

  it('<Ustrd> auf 140 Zeichen begrenzt (Max140Text)', () => {
    const { xml } = buildPainXML(sampleInput())
    const ustrd = xml.match(/<Ustrd>([^<]+)<\/Ustrd>/)![1]
    expect(ustrd.length).toBeLessThanOrEqual(140)
  })

  it('fehlende Stadt → gesamter <PstlAdr>-Block weggelassen (TwnNm ist Pflicht)', () => {
    const input = sampleInput()
    input.items[0].city = ''
    const { xml } = buildPainXML(input)
    expect(xml).not.toContain('<PstlAdr>')
    // Dbtr enthält nur <Nm>, kein PstlAdr
    expect(xml).toMatch(/<Dbtr>\s*<Nm>[^<]+<\/Nm>\s*<\/Dbtr>/)
  })

  it('Stadt vorhanden → <PstlAdr> enthält immer <TwnNm> und <Ctry>', () => {
    const { xml } = buildPainXML(sampleInput())
    expect(xml).toContain('<TwnNm>Stuttgart</TwnNm>')
    expect(xml).toContain('<Ctry>DE</Ctry>')
  })
})

describe('Edge Cases (Regression-Guards für Bestandsdaten)', () => {
  it('Betrag <= 0 wirft (kein stiller Math.abs mehr)', () => {
    const input = sampleInput()
    input.items[0].betragCent = 0
    expect(() => buildPainXML(input)).toThrow(/muss positiv sein/)

    const input2 = sampleInput()
    input2.items[0].betragCent = -1000
    expect(() => buildPainXML(input2)).toThrow(/muss positiv sein/)
  })

  it('parseStreet: Hausnummer mit Buchstabe ("Silberburgstr. 155 A")', () => {
    const input = sampleInput()
    input.items[0].street = 'Silberburgstr. 155 A'
    const { xml } = buildPainXML(input)
    expect(xml).toContain('<StrtNm>Silberburgstr.</StrtNm>')
    expect(xml).toContain('<BldgNb>155 A</BldgNb>')
  })

  it('parseStreet: keine erkennbare Hausnummer → BldgNb weggelassen', () => {
    const input = sampleInput()
    input.items[0].street = 'Postfach'
    const { xml } = buildPainXML(input)
    expect(xml).toContain('<StrtNm>Postfach</StrtNm>')
    expect(xml).not.toContain('<BldgNb>')
  })

  it('AT/CH-IBANs werden 1:1 durchgereicht (keine DE-Sonderbehandlung)', () => {
    const input = sampleInput()
    input.items[0].iban = 'AT611904300234573201'
    const { xml } = buildPainXML(input)
    expect(xml).toContain('<IBAN>AT611904300234573201</IBAN>')
  })

  it('Sehr lange memberNumber → EndToEndId bleibt XSD-konform kurz genug', () => {
    const input = sampleInput()
    input.items[0].memberNumber = 'M-123456789' // 11 Zeichen
    input.items[0].mandatRef = 'M-123456789'
    const { xml } = buildPainXML(input)
    const m = xml.match(/<EndToEndId>([^<]+)<\/EndToEndId>/)!
    // Max35Text: TW-{memberNumber}-{saisonStamp} = TW-M-123456789-2026-27 = 22 Zeichen
    expect(m[1].length).toBeLessThanOrEqual(35)
  })
})

describe('nextBusinessDay', () => {
  it('verschiebt Samstag/Sonntag auf Montag', () => {
    expect(nextBusinessDay('2026-07-04')).toBe('2026-07-06') // Sa → Mo
    expect(nextBusinessDay('2026-07-05')).toBe('2026-07-06') // So → Mo
    expect(nextBusinessDay('2026-07-01')).toBe('2026-07-01') // Mi unverändert
  })
})
