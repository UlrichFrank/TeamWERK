// formatBetrag wandelt Cent in einen deutschen Euro-String: 9600 → "96,00 €".
export function formatBetrag(cent: number): string {
  return (cent / 100).toLocaleString('de-DE', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) + ' €'
}

// formatIBAN gruppiert in Viererblöcke: DE89370400440532013000 → "DE89 3704 ..."
export function formatIBAN(iban: string): string {
  return (iban || '').replace(/\s/g, '').replace(/(.{4})/g, '$1 ').trim()
}
