## Why

BW-Bank (LBBW) lehnt die per `sepaXml.ts` erzeugte `pain.008.001.08`-Datei mit „XML Dokument ist nicht gültig" ab, obwohl sie gegen das offizielle ISO-XSD sauber validiert. Ursache sind drei EPC-SEPA-Rulebook- bzw. bankseitige Zusatzregeln, die das Basis-XSD nicht erzwingt, viele deutsche Banken beim Upload aber prüfen. Solange der Beitragslauf-Export bei der Hausbank nicht durchgeht, ist der jährliche SEPA-Einzug blockiert.

## What Changes

Der clientseitige Builder in `web/src/lib/sepaXml.ts` erzeugt zusätzlich:

- **CreDtTm mit UTC-Zeitzone**: `<CreDtTm>YYYY-MM-DDThh:mm:ssZ</CreDtTm>` statt bisher ohne Zonenmarker.
- **Cdtr mit Postal-Adresse**: `<Cdtr>` bekommt `<PstlAdr><Ctry>DE</Ctry></PstlAdr>` nach dem `<Nm>`.
- **InitgPty Gläubiger-ID mit SchmeNm**: `<InitgPty>/<Id>/<OrgId>/<Othr>` bekommt zusätzlich `<SchmeNm><Prtry>SEPA</Prtry></SchmeNm>` (analog zum bereits vorhandenen `<CdtrSchmeId>`-Block).

Kein Umbau am Datenmodell, keine neuen Felder in DB oder API, keine Änderung an der Auth-/Krypto-Grenze. Das bestehende XSD-Validitäts-Szenario bleibt erfüllt; die neuen Ausgaben werden zusätzlich abgesichert.

## Capabilities

### New Capabilities
(keine)

### Modified Capabilities
- `sepa-beitragslauf`: Requirement „SEPA-XML-Export (pain.008.001.08), immer RCUR — clientseitig erzeugt" wird um drei konkrete EPC-Rulebook-Konformitäten ergänzt (CreDtTm-Zone, Cdtr-PstlAdr, InitgPty-Othr-SchmeNm).

## Impact

- **Code**: `web/src/lib/sepaXml.ts` (3 Ergänzungen im Node-Baum).
- **Tests**: `web/src/lib/sepaXml.test.ts` bekommt Assertions für die drei neuen Ausgaben.
- **Betrieb**: Nach Deploy erzeugt der Beitragslauf-Export unverändert eine `.xml`, jetzt aber BW-Bank-akzeptiert. Kein Migrations- oder Backfill-Bedarf.
- **Nicht betroffen**: Backend-Routen, DB-Schema, Zero-Knowledge-Krypto, Auth-Tiers.
