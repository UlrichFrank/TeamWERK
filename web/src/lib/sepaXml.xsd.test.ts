/// <reference types="node" />
// XSD-Gate: prüft jeden von buildPainXML() erzeugten pain.008-Datei-Inhalt gegen
// das offizielle DK-TVS pain.008.001.08_GBIC_5 (Anlage 3 V3.9 = V26.11,
// gültig ab 05.10.2025).
// Läuft nur, wenn `xmllint` (libxml2) verfügbar ist — auf macOS Standard, auf
// Linux via `apt-get install libxml2-utils`. Ohne xmllint: Test wird geskippt
// (kein Silent-Pass).

import { describe, it, expect } from 'vitest'
import { execFileSync, spawnSync } from 'node:child_process'
import { mkdtempSync, writeFileSync, rmSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { tmpdir } from 'node:os'
import { buildPainXML, type SepaBuildInput } from './sepaXml'

// Vitest CWD ist web/; XSD liegt unter src/lib/__schemas__/.
const XSD = resolve(process.cwd(), 'src/lib/__schemas__/pain.008.001.08_GBIC_5.xsd')

const xmllintAvailable = spawnSync('xmllint', ['--version']).status === 0
const requireXsd = process.env.TEAMWERK_REQUIRE_XSD === '1'

// CI-Guard: In CI ist `TEAMWERK_REQUIRE_XSD=1` gesetzt (siehe ci.yml); fehlt
// xmllint dann, werfen wir laut statt still zu skippen. Verhindert, dass ein
// defektes libxml2 im Runner-Image das gesamte XSD-Gate zur No-Op degradiert.
if (requireXsd && !xmllintAvailable) {
  throw new Error(
    'TEAMWERK_REQUIRE_XSD=1 aber xmllint nicht verfügbar — libxml2-utils installieren',
  )
}

function validate(xml: string): { ok: true } | { ok: false; err: string } {
  // mkdtempSync bewusst vor try — wenn schon das Anlegen des tmp-dir fehlt,
  // gibt es nichts zu räumen. writeFileSync + execFileSync stehen in try,
  // damit auch ein I/O-Fehler beim Schreiben in den cleanup-Pfad läuft.
  const dir = mkdtempSync(join(tmpdir(), 'sepa-xsd-'))
  try {
    const file = join(dir, 'out.xml')
    writeFileSync(file, xml, 'utf8')
    execFileSync('xmllint', ['--noout', '--schema', XSD, file], { stdio: 'pipe' })
    return { ok: true }
  } catch (e) {
    const stderr = (e as { stderr?: Buffer }).stderr?.toString() ?? String(e)
    return { ok: false, err: stderr }
  } finally {
    rmSync(dir, { recursive: true, force: true })
  }
}

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
      { name: 'Max Müller', street: 'Hauptstr. 12', zip: '70182', city: 'Stuttgart',
        iban: 'DE89370400440532013000', betragCent: 9600, mandatRef: '1042',
        mandatDatum: '2026-05-01', memberNumber: '1042' },
    ],
  }
}

describe.skipIf(!xmllintAvailable)('XSD-Gate gegen pain.008.001.08_GBIC_5', () => {
  it('Standard-Fall (Mitglied mit Adresse + Mandatsdatum) validiert', () => {
    const { xml } = buildPainXML(sampleInput())
    const r = validate(xml)
    if (!r.ok) throw new Error(`XSD-Validierung fehlgeschlagen:\n${r.err}`)
    expect(r.ok).toBe(true)
  })

  it('fehlendes Mandatsdatum validiert (DtOfSgntr optional weggelassen)', () => {
    const input = sampleInput()
    input.items[0].mandatDatum = ''
    const { xml } = buildPainXML(input)
    const r = validate(xml)
    if (!r.ok) throw new Error(`XSD-Validierung fehlgeschlagen:\n${r.err}`)
    expect(r.ok).toBe(true)
  })

  it('fehlende Stadt validiert (PstlAdr komplett weggelassen)', () => {
    const input = sampleInput()
    input.items[0].city = ''
    const { xml } = buildPainXML(input)
    const r = validate(xml)
    if (!r.ok) throw new Error(`XSD-Validierung fehlgeschlagen:\n${r.err}`)
    expect(r.ok).toBe(true)
  })

  // Meta-Sanity: Beweist, dass der Validator scharf ist. Ohne diese Tests
  // könnten falsches XSD, offline-nicht-resolvter Import oder ein
  // Silent-Accept dafür sorgen, dass ALLES als valid durchgeht — die anderen
  // Testfälle würden das nicht bemerken.
  it('Meta-Sanity: manipuliertes <FwdgAgt> wird abgelehnt', () => {
    const { xml: good } = buildPainXML(sampleInput())
    // FwdgAgt-Block direkt hinter InitgPty einfügen — bricht GrpHdr-Choice
    // im GBIC_5-Subset.
    const poisoned = good.replace(
      '</InitgPty>',
      '</InitgPty><FwdgAgt><FinInstnId><BICFI>GENODEF1S02</BICFI></FinInstnId></FwdgAgt>',
    )
    const r = validate(poisoned)
    expect(r.ok).toBe(false)
  })

  it('Meta-Sanity: <PstlAdr> ohne <TwnNm> wird abgelehnt', () => {
    const { xml: good } = buildPainXML(sampleInput())
    // TwnNm aus einer bestehenden PstlAdr entfernen — GBIC_5 verlangt TwnNm
    // Pflicht, wenn PstlAdr überhaupt vorkommt.
    const poisoned = good.replace(/<TwnNm>[^<]+<\/TwnNm>/, '')
    const r = validate(poisoned)
    expect(r.ok).toBe(false)
  })

  it('AT-IBAN + Straße ohne Hausnummer validieren (Edge Cases)', () => {
    const input = sampleInput()
    input.items[0].iban = 'AT611904300234573201'
    input.items[0].street = 'Postfach 1234'
    const { xml } = buildPainXML(input)
    const r = validate(xml)
    if (!r.ok) throw new Error(`XSD-Validierung fehlgeschlagen:\n${r.err}`)
    expect(r.ok).toBe(true)
  })

  it('Mehrere Transaktionen validieren', () => {
    const input = sampleInput()
    input.items.push({
      name: 'Erika Schäfer', street: 'Königstr. 5a', zip: '70173', city: 'Stuttgart',
      iban: 'DE02120300000000202051', betragCent: 4800, mandatRef: '1043',
      mandatDatum: '2026-05-15', memberNumber: '1043',
    })
    const { xml } = buildPainXML(input)
    const r = validate(xml)
    if (!r.ok) throw new Error(`XSD-Validierung fehlgeschlagen:\n${r.err}`)
    expect(r.ok).toBe(true)
  })
})

if (!xmllintAvailable) {
  // Sichtbarer Skip-Hinweis, damit fehlendes xmllint auffällt (kein Silent-Pass).
  // eslint-disable-next-line no-console
  console.warn('[sepaXml.xsd.test] xmllint nicht gefunden → XSD-Gate übersprungen. Installation: apt-get install libxml2-utils / brew install libxml2')
}
