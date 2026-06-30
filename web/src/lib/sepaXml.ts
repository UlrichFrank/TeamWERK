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

const PAIN_NS = 'urn:iso:std:iso:20022:tech:xsd:pain.008.001.08'

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
  const c = Math.abs(cent)
  return `${Math.floor(c / 100)}.${String(c % 100).padStart(2, '0')}`
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

// CreDtTm im Format YYYY-MM-DDTHH:MM:SS (UTC), wie Go in.CreatedAt.UTC().
function creDtTm(d: Date): string {
  return (
    `${d.getUTCFullYear()}-${pad2(d.getUTCMonth() + 1)}-${pad2(d.getUTCDate())}` +
    `T${pad2(d.getUTCHours())}:${pad2(d.getUTCMinutes())}:${pad2(d.getUTCSeconds())}`
  )
}

function stamp14(d: Date): string {
  return (
    `${d.getUTCFullYear()}${pad2(d.getUTCMonth() + 1)}${pad2(d.getUTCDate())}` +
    `${pad2(d.getUTCHours())}${pad2(d.getUTCMinutes())}${pad2(d.getUTCSeconds())}`
  )
}

// buildPainXML erzeugt das pain.008.001.08-Dokument als String.
export function buildPainXML(input: SepaBuildInput): string {
  let sumCent = 0
  const txNodes: Node[] = input.items.map(it => {
    sumCent += it.betragCent
    const { strtNm, bldgNb } = parseStreet(it.street)
    const pstlAdr: Node[] = []
    if (strtNm) pstlAdr.push(leaf('StrtNm', ascii(strtNm)))
    if (bldgNb) pstlAdr.push(leaf('BldgNb', ascii(bldgNb)))
    if (it.zip) pstlAdr.push(leaf('PstCd', ascii(it.zip)))
    if (it.city) pstlAdr.push(leaf('TwnNm', ascii(it.city)))
    pstlAdr.push(leaf('Ctry', 'DE'))
    return el('DrctDbtTxInf', [
      el('PmtId', [leaf('EndToEndId', ascii(`TW-${it.memberNumber}-${saisonStamp(input.saisonKurz)}`))]),
      leaf('InstdAmt', euro(it.betragCent), { Ccy: 'EUR' }),
      el('DrctDbtTx', [
        el('MndtRltdInf', [leaf('MndtId', ascii(it.mandatRef)), leaf('DtOfSgntr', it.mandatDatum)]),
      ]),
      el('DbtrAgt', [el('FinInstnId', [el('Othr', [leaf('Id', 'NOTPROVIDED')])])]),
      el('Dbtr', [leaf('Nm', ascii(it.name)), el('PstlAdr', pstlAdr)]),
      el('DbtrAcct', [el('Id', [leaf('IBAN', it.iban)])]),
      el('RmtInf', [
        leaf('Ustrd', ascii(`Mitgliedsbeitrag Team Stuttgart ${abrechnungsjahr(input.saisonKurz)} - Mitglied ${it.memberNumber}`)),
      ]),
    ])
  })

  const grpHdr = el('GrpHdr', [
    leaf('MsgId', ascii(`TW-${saisonStamp(input.saisonKurz)}-${stamp14(input.createdAt)}`)),
    leaf('CreDtTm', creDtTm(input.createdAt)),
    leaf('NbOfTxs', String(txNodes.length)),
    leaf('CtrlSum', euro(sumCent)),
    el('InitgPty', [
      leaf('Nm', ascii(input.clubName)),
      el('Id', [el('OrgId', [el('Othr', [leaf('Id', input.glaeubigerId)])])]),
    ]),
    el('FwdgAgt', [el('FinInstnId', [leaf('BICFI', input.bic)])]),
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
    el('Cdtr', [leaf('Nm', ascii(input.kontoinhaber))]),
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
  return '<?xml version="1.0" encoding="UTF-8"?>\n' + serialize(doc, '') + '\n'
}
