// Clientseitiger pain.008.001.08-Builder (Port von internal/beitragslauf/xml.go).
// Erzeugt die SEPA-Lastschrift-Datei im Browser des Kassierers — der Server sieht nie
// Klartext-IBANs. Genau ein PmtInf-Block, alle Lastschriften RCUR.
//
// Die Element-Struktur spiegelt 1:1 die Go-Strukturen (document/cstmr/grpHdr/pmtInf/txInf).
// Byte-Identität zur Go-Ausgabe ist nicht erforderlich (Spec verlangt XSD-Validität +
// fachliche Parität), aber Baum und Inhalte sind identisch.

export interface SepaItem {
  name: string
  street: string
  zip: string
  city: string
  iban: string
  betragCent: number
  mandatRef: string // = member_number
  mandatDatum: string // YYYY-MM-DD
  memberNumber: string
}

export interface SepaBuildInput {
  saisonKurz: string // z. B. "2026/27"
  clubName: string
  glaeubigerId: string
  clubIban: string
  bic: string
  kontoinhaber: string
  faelligkeit: string // YYYY-MM-DD (01.07. der Saison)
  createdAt: Date
  items: SepaItem[]
}

// Truncation-Ereignis: dem Aufrufer (UI) mitteilen, dass ein Wert wegen
// DK-TVS-Längenlimit gekürzt wurde. Bank-relevant vor allem für Debtor-Nm
// (Identifikation beim Zahler) — daher nicht stumm, sondern zurückgegeben.
export interface SepaBuildWarning {
  location: 'debtor-name' | 'creditor-name' | 'initiator-name' | 'remittance-info'
  memberNumber: string // '' für vereins-weite Felder (creditor-name, initiator-name)
  original: string
  truncated: string
  maxLen: number
}

export interface SepaBuildResult {
  xml: string
  warnings: SepaBuildWarning[]
}

const PAIN_NS = 'urn:iso:std:iso:20022:tech:xsd:pain.008.001.08'

// Fallback-Signatur-Datum für Bestandsmandate ohne erfasstes Datum. GBIC_5-TVS
// verlangt <DtOfSgntr> als Pflichtelement in <MndtRltdInf> — Weglassen ist
// XSD-invalid, ein leerer Wert ebenso. Bewusst konservatives Datum.
//
// TRADE-OFF (dokumentiert nach Code-Review 2026-07-16): Rechtlich heikel —
// die Bank bekommt ein Signaturdatum, das nicht dem tatsächlichen Mandatstag
// entspricht. Bei einer Rücklastschrift-Reklamation („Ich habe nie
// unterschrieben") ist der Verein beweispflichtig; ein Fallback-Datum ist
// keine solide Position. Sobald alle Bestandsmandate ihr echtes Signaturdatum
// tragen, sollte diese Konstante (und die zugehörige Fallback-Logik unten)
// wieder entfallen. Bis dahin ist der Fallback die pragmatische Wahl, damit
// der Beitragslauf überhaupt läuft.
const DEFAULT_MANDAT_DATUM = '2026-06-01'

// --- Mini-XML-Knotenbaum (jedes Element auf eigener Zeile, 2-Space-Indent wie Go) ---

type Node = { tag: string; attrs?: Record<string, string>; text?: string; children?: Node[] }

function el(tag: string, children: Node[]): Node {
  return { tag, children }
}
function leaf(tag: string, text: string, attrs?: Record<string, string>): Node {
  return { tag, text, attrs }
}

function escapeText(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}
function escapeAttr(s: string): string {
  return escapeText(s).replace(/"/g, '&quot;')
}

function serialize(node: Node, indent: string): string {
  const attrs = node.attrs
    ? Object.entries(node.attrs)
        .map(([k, v]) => ` ${k}="${escapeAttr(v)}"`)
        .join('')
    : ''
  if (node.children && node.children.length > 0) {
    const inner = node.children.map(c => serialize(c, indent + '  ')).join('\n')
    return `${indent}<${node.tag}${attrs}>\n${inner}\n${indent}</${node.tag}>`
  }
  // Blatt: Text (auch leerer Text → <Tag></Tag>, wie Go bei chardata)
  return `${indent}<${node.tag}${attrs}>${escapeText(node.text ?? '')}</${node.tag}>`
}

// --- Helfer (Port aus xml.go) ---

function euro(cent: number): string {
  // Kein Math.abs — bei einem geleakten Vorzeichen (Bug in der Preview- oder
  // Import-Logik) würde die Bank sonst still einen positiven Betrag einziehen.
  // Aufrufer sind verpflichtet, positive Beträge zu übergeben; buildPainXML
  // wirft bei <= 0.
  return `${Math.floor(cent / 100)}.${String(cent % 100).padStart(2, '0')}`
}

export function saisonStamp(s: string): string {
  return s.replace(/\//g, '-')
}

// abrechnungsjahr liefert nur das Beitrags-/Abrechnungsjahr (2-stellig, Startjahr)
// aus einem Saison-Label wie "2026/27" → "26" bzw. "2026/2027" → "26".
// Im Verwendungszweck soll nur das Beitragsjahr stehen, nicht die Saison-Spanne.
export function abrechnungsjahr(s: string): string {
  const first = s.split('/')[0].trim()
  return first.length === 4 && /^\d+$/.test(first) ? first.slice(2) : first
}

const STREET_RE = /^(.+?)\s+(\d+\s*[a-zA-Z]?)$/

function parseStreet(street: string): { strtNm: string; bldgNb: string } {
  const s = street.trim()
  const m = STREET_RE.exec(s)
  if (m) return { strtNm: m[1].trim(), bldgNb: m[2].trim() }
  return { strtNm: s, bldgNb: '' }
}

// ascii ersetzt Umlaute und entfernt sonstige Nicht-ASCII-Zeichen (für IDs).
export function ascii(s: string): string {
  const replaced = s
    .replace(/ä/g, 'ae').replace(/ö/g, 'oe').replace(/ü/g, 'ue')
    .replace(/Ä/g, 'Ae').replace(/Ö/g, 'Oe').replace(/Ü/g, 'Ue')
    .replace(/ß/g, 'ss')
  return replaced.replace(/[^\x20-\x7E]/g, '')
}

// DK-TVS pain.008.001.08_GBIC_5: <Nm> in Party-Elementen (Dbtr/Cdtr/InitgPty)
// ist auf 70 Zeichen limitiert (Max140Text_SDD mit DK-Einschränkung, siehe
// Anlage 3 V3.9 (= V26.11 in interner DK-Zählung), Kap. 2.2.2.5/2.2.2.6).
// Ustrd (Verwendungszweck) ist auf 140 Zeichen limitiert (Max140Text).
// Längerer Wert → XSD-Reject bei der Bank. Truncation-Events werden im
// warnings-Kollektor gesammelt, damit die UI vor dem Download bestätigen
// kann — bei Debtor-Nm ist die Kürzung bank-relevant für die
// Identifikation beim Zahler.
function truncate(
  s: string,
  maxLen: number,
  loc: SepaBuildWarning['location'],
  memberNumber: string,
  warnings: SepaBuildWarning[],
): string {
  const asciiFull = ascii(s)
  if (asciiFull.length <= maxLen) return asciiFull
  const cut = asciiFull.slice(0, maxLen)
  warnings.push({ location: loc, memberNumber, original: asciiFull, truncated: cut, maxLen })
  return cut
}

// nextBusinessDay verschiebt Sa/So auf den folgenden Werktag (für ReqdColltnDt).
export function nextBusinessDay(dateYmd: string): string {
  const d = new Date(dateYmd + 'T12:00:00Z')
  const day = d.getUTCDay() // 0=So, 6=Sa
  if (day === 6) d.setUTCDate(d.getUTCDate() + 2)
  else if (day === 0) d.setUTCDate(d.getUTCDate() + 1)
  return d.toISOString().slice(0, 10)
}

function pad2(n: number): string {
  return String(n).padStart(2, '0')
}

// CreDtTm im Format YYYY-MM-DDTHH:MM:SSZ (UTC-Marker Pflicht für BW-Bank; ohne Z
// interpretieren strenge XSD-Validatoren die Zeit als lokal → Reject. Entspricht
// dem DK-Beispiel in Anlage 3 V26.11, S. 126: „2023-11-21T09:30:47.000Z".)
function creDtTm(d: Date): string {
  return (
    `${d.getUTCFullYear()}-${pad2(d.getUTCMonth() + 1)}-${pad2(d.getUTCDate())}` +
    `T${pad2(d.getUTCHours())}:${pad2(d.getUTCMinutes())}:${pad2(d.getUTCSeconds())}Z`
  )
}

function stamp14(d: Date): string {
  return (
    `${d.getUTCFullYear()}${pad2(d.getUTCMonth() + 1)}${pad2(d.getUTCDate())}` +
    `${pad2(d.getUTCHours())}${pad2(d.getUTCMinutes())}${pad2(d.getUTCSeconds())}`
  )
}

// buildPainXML erzeugt das pain.008.001.08-Dokument.
// Liefert das XML plus eine Liste an Truncation-Warnungen (DK-TVS-Längenlimits
// wurden für einzelne Werte überschritten und gekürzt). Aufrufer sollten die
// Warnungen dem Nutzer sichtbar machen (SEPA-Debtor-Namen können bank-relevant
// sein, stumme Kürzung wäre gefährlich).
export function buildPainXML(input: SepaBuildInput): SepaBuildResult {
  const warnings: SepaBuildWarning[] = []
  // Guard: 0/negative Beträge sind semantisch keine Lastschriften und würden
  // XSD-technisch entweder als "0.00" oder als negativer Wert (XSD-Reject)
  // enden. Explizit werfen ist besser als still zu verstümmeln.
  for (const it of input.items) {
    if (!Number.isFinite(it.betragCent) || it.betragCent <= 0) {
      throw new Error(
        `SEPA: Betrag für Mitglied ${it.memberNumber || '(ohne Nummer)'} ist ${it.betragCent} — muss positiv sein`,
      )
    }
  }
  let sumCent = 0
  const txNodes: Node[] = input.items.map(it => {
    sumCent += it.betragCent
    const { strtNm, bldgNb } = parseStreet(it.street)
    // PstlAdr nur emittieren, wenn TwnNm (city) vorhanden ist — im DK-TVS
    // GBIC_5 sind TwnNm [1..1] und Ctry [1..1] Pflicht, sobald PstlAdr überhaupt
    // vorkommt (Anlage 3 V26.11 Kap. 2.2.2.10.1 S. 164). Ein PstlAdr nur mit
    // <Ctry> ohne TwnNm ist XSD-invalid.
    const dbtrChildren: Node[] = [
      leaf('Nm', truncate(it.name, 70, 'debtor-name', it.memberNumber, warnings)),
    ]
    if (it.city) {
      const pstlAdr: Node[] = []
      if (strtNm) pstlAdr.push(leaf('StrtNm', ascii(strtNm)))
      if (bldgNb) pstlAdr.push(leaf('BldgNb', ascii(bldgNb)))
      if (it.zip) pstlAdr.push(leaf('PstCd', ascii(it.zip)))
      pstlAdr.push(leaf('TwnNm', ascii(it.city)))
      pstlAdr.push(leaf('Ctry', 'DE'))
      dbtrChildren.push(el('PstlAdr', pstlAdr))
    }
    // <DtOfSgntr> ist im GBIC_5-TVS Pflicht in <MndtRltdInf>. Fehlt bei einem
    // Mitglied das Mandatsdatum (Altbestand vor systematischer Erfassung),
    // fällt der Builder auf DEFAULT_MANDAT_DATUM zurück — siehe Trade-off-
    // Kommentar bei der Konstante oben. Neue Mandate sollen ihr echtes
    // Signaturdatum tragen; der Fallback soll perspektivisch entfallen.
    const mandatDatum = it.mandatDatum || DEFAULT_MANDAT_DATUM
    const mndtChildren: Node[] = [
      leaf('MndtId', ascii(it.mandatRef)),
      leaf('DtOfSgntr', mandatDatum),
    ]
    return el('DrctDbtTxInf', [
      el('PmtId', [leaf('EndToEndId', ascii(`TW-${it.memberNumber}-${saisonStamp(input.saisonKurz)}`))]),
      leaf('InstdAmt', euro(it.betragCent), { Ccy: 'EUR' }),
      el('DrctDbtTx', [
        el('MndtRltdInf', mndtChildren),
      ]),
      el('DbtrAgt', [el('FinInstnId', [el('Othr', [leaf('Id', 'NOTPROVIDED')])])]),
      el('Dbtr', dbtrChildren),
      el('DbtrAcct', [el('Id', [leaf('IBAN', it.iban)])]),
      el('RmtInf', [
        leaf('Ustrd', truncate(
          `Mitgliedsbeitrag Team Stuttgart ${abrechnungsjahr(input.saisonKurz)} - Mitglied ${it.memberNumber}`,
          140,
          'remittance-info',
          it.memberNumber,
          warnings,
        )),
      ]),
    ])
  })

  // GrpHdr laut DK-TVS pain.008.001.08_GBIC_5 (Anlage 3 V26.11, Kap. 2.2.2.3):
  // nur MsgId, CreDtTm, NbOfTxs, CtrlSum, InitgPty. FwdgAgt ist nicht Teil des
  // DK-Subsets — mit FwdgAgt lehnt die BW-Bank per XSD ab. InitgPty/Id ist laut DK
  // ausdrücklich nicht zu belegen; die Gläubiger-ID steckt in CdtrSchmeId.
  const grpHdr = el('GrpHdr', [
    leaf('MsgId', ascii(`TW-${saisonStamp(input.saisonKurz)}-${stamp14(input.createdAt)}`)),
    leaf('CreDtTm', creDtTm(input.createdAt)),
    leaf('NbOfTxs', String(txNodes.length)),
    leaf('CtrlSum', euro(sumCent)),
    el('InitgPty', [leaf('Nm', truncate(input.clubName, 70, 'initiator-name', '', warnings))]),
  ])

  const pmtInf = el('PmtInf', [
    leaf('PmtInfId', ascii(`TW-${saisonStamp(input.saisonKurz)}-RCUR`)),
    leaf('PmtMtd', 'DD'),
    leaf('BtchBookg', 'true'),
    leaf('NbOfTxs', String(txNodes.length)),
    leaf('CtrlSum', euro(sumCent)),
    el('PmtTpInf', [
      el('SvcLvl', [leaf('Cd', 'SEPA')]),
      el('LclInstrm', [leaf('Cd', 'CORE')]),
      leaf('SeqTp', 'RCUR'),
    ]),
    leaf('ReqdColltnDt', nextBusinessDay(input.faelligkeit)),
    el('Cdtr', [leaf('Nm', truncate(input.kontoinhaber, 70, 'creditor-name', '', warnings))]),
    el('CdtrAcct', [el('Id', [leaf('IBAN', input.clubIban)])]),
    el('CdtrAgt', [el('FinInstnId', [leaf('BICFI', input.bic)])]),
    leaf('ChrgBr', 'SLEV'),
    el('CdtrSchmeId', [
      el('Id', [
        el('PrvtId', [
          el('Othr', [leaf('Id', input.glaeubigerId), el('SchmeNm', [leaf('Prtry', 'SEPA')])]),
        ]),
      ]),
    ]),
    ...txNodes,
  ])

  const doc = el('Document', [el('CstmrDrctDbtInitn', [grpHdr, pmtInf])])
  doc.attrs = { xmlns: PAIN_NS }
  const xml = '<?xml version="1.0" encoding="UTF-8"?>\n' + serialize(doc, '') + '\n'
  return { xml, warnings }
}
