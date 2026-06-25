// formatBetrag wandelt Cent in einen deutschen Euro-String: 9600 → "96,00 €".
export function formatBetrag(cent: number): string {
  return (cent / 100).toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) + ' €'
}

// formatIBAN gruppiert in Viererblöcke: DE89370400440532013000 → "DE89 3704 ..."
export function formatIBAN(iban: string): string {
  return (iban || '').replace(/\s/g, '').replace(/(.{4})/g, '$1 ').trim()
}

// --- IBAN-Validierung (Port von internal/sepa/iban.go für den clientseitigen Fee-Run) ---

// Erwartete IBAN-Gesamtlänge je ISO-Ländercode (relevant: DE/AT/CH; weitere SEPA-Länder
// der Vollständigkeit halber). Muss mit ibanLength in internal/sepa/iban.go übereinstimmen.
const IBAN_LENGTH: Record<string, number> = {
  DE: 22, AT: 20, CH: 21, LI: 21,
  FR: 27, IT: 27, ES: 24, NL: 18,
  BE: 16, LU: 20, DK: 18, PL: 28,
}

// normalizeIBAN entfernt Leerzeichen und wandelt in Großbuchstaben um.
export function normalizeIBAN(s: string): string {
  return (s || '').trim().replace(/\s/g, '').toUpperCase()
}

// isValidIBAN prüft Ländercode-spezifische Länge, erlaubte Zeichen und die
// Mod-97-Prüfsumme (ISO 13616 / ISO 7064).
export function isValidIBAN(input: string): boolean {
  const iban = normalizeIBAN(input)
  if (iban.length < 5) return false
  const country = iban.slice(0, 2)
  if (!isUpperAlpha(country[0]) || !isUpperAlpha(country[1])) return false
  const want = IBAN_LENGTH[country]
  if (want !== undefined && iban.length !== want) return false
  if (!/^[A-Z0-9]+$/.test(iban)) return false
  return mod97(iban) === 1
}

function isUpperAlpha(c: string): boolean {
  return c >= 'A' && c <= 'Z'
}

// mod97 setzt die ersten vier Zeichen ans Ende, ersetzt Buchstaben durch Zahlen
// (A=10 … Z=35) und berechnet den Rest modulo 97 stückweise.
function mod97(iban: string): number {
  const rearranged = iban.slice(4) + iban.slice(0, 4)
  let remainder = 0
  for (const ch of rearranged) {
    let val: number
    if (ch >= '0' && ch <= '9') {
      val = ch.charCodeAt(0) - 48
    } else if (ch >= 'A' && ch <= 'Z') {
      val = ch.charCodeAt(0) - 65 + 10
    } else {
      return -1
    }
    remainder = val >= 10 ? (remainder * 100 + val) % 97 : (remainder * 10 + val) % 97
  }
  return remainder
}
