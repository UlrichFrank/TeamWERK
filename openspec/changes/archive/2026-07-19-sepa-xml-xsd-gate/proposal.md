## Why

Bisher prüfen die Tests in `web/src/lib/sepaXml.test.ts` den erzeugten
`pain.008.001.08`-XML nur per Substring-/Regex-Assertions. XSD-Konformität — die
eigentliche Wahrheit, an der Banken beim Upload rejecten — wird nirgends
mechanisch validiert. Konsequenz: Regressionen bemerkt der Kassierer erst am
Bank-Portal-Fehler. Der Vorgänger-Change [[sepa-xml-epc-compliance]] fixte drei
konkrete BW-Bank-Rejects empirisch; ohne mechanische Prüfung ist das nicht
haltbar — die Regeln driften mit jeder DK-Anlage-3-Version.

Parallel zeigt die Analyse des DK-TVS `pain.008.001.08_GBIC_5` (Anlage 3
V26.11, gültig ab 05.10.2025) drei Edge-Case-Bruchpunkte im Generator:

- **`<Nm>` unbegrenzt**: DK-TVS limitiert auf 70 Zeichen (`Max140Text_SDD`),
  längere Mitgliedsnamen produzieren XSD-Reject.
- **`<Ustrd>` unbegrenzt**: DK-TVS limitiert auf 140 Zeichen.
- **`<PstlAdr>` mit nur `<Ctry>`**: Wenn `city` fehlt, entsteht ein
  `<PstlAdr><Ctry>DE</Ctry></PstlAdr>` ohne `<TwnNm>` — GBIC_5 fordert
  `<TwnNm>` [1..1] Pflicht, sobald `<PstlAdr>` überhaupt vorkommt.
- **`<DtOfSgntr>` weggelassen bei fehlendem Mandatsdatum** (Commit `4363a87`):
  ist unter GBIC_5 kein XSD-valides Verhalten — DtOfSgntr ist Pflichtelement in
  `MndtRltdInf`. Der Fix tauschte einen XSD-Verstoß gegen einen anderen. Für
  Altbestand ohne erfasstes Mandatsdatum braucht es einen sinnvollen Fallback.

## What Changes

- Das offizielle DK-TVS-XSD **`pain.008.001.08_GBIC_5.xsd`** wird ins Repo
  eingecheckt (`web/src/lib/__schemas__/`, mit DK-Copyright-Header in der Datei
  + Herkunfts-Doku in `README.md`).
- Ein neuer Vitest-Test **`sepaXml.xsd.test.ts`** shell-outet `xmllint`
  (libxml2) und validiert jede von `buildPainXML()` erzeugte Datei mechanisch
  gegen das XSD. Testfälle: Standard-Fall, fehlendes Mandatsdatum, fehlende
  Stadt, Multi-Transaktion.
- `web/src/lib/sepaXml.ts` bekommt vier DK-TVS-Härtungen:
  - `<Nm>`-Truncation auf 70 Zeichen (neuer Helfer `nm70()`) für Debtor,
    Creditor und Initiating Party.
  - `<Ustrd>`-Truncation auf 140 Zeichen (neuer Helfer `ustrd140()`).
  - `<PstlAdr>` wird nur emittiert, wenn `city` gesetzt ist (all-or-nothing:
    entweder komplett mit `<TwnNm>` + `<Ctry>`, oder gar nicht).
  - `<DtOfSgntr>` fällt bei fehlendem Mandatsdatum auf eine bewusst gewählte
    Konstante (`2026-06-01`) zurück statt weggelassen zu werden. Neu erfasste
    Mandate tragen weiter das echte Signatur-Datum.
- **CI-Integration** in `.github/workflows/ci.yml`: `libxml2-utils` (Debian-
  Paket, enthält `xmllint`) wird vor `pnpm -C web test` installiert, damit
  das XSD-Gate im PR-/Push-Gate wirklich läuft (kein Silent-Skip in CI).
- Doku-Ergänzung in `docs/agent/06-gotchas.md` (SEPA-Beitragslauf-Absatz)
  bzw. `docs/agent/07-testing.md`, damit künftige Änderungen am Generator die
  XSD-Anforderung kennen und `libxml2-utils` als Voraussetzung dokumentiert
  ist.

Kein Umbau am Datenmodell, keine neuen API-Routen, keine Änderung am
Zero-Knowledge-Krypto-Modell. Kein Rebuild bestehender Beitragsläufe nötig —
der Generator läuft nur bei neuen Exporten.

## Capabilities

### New Capabilities
(keine)

### Modified Capabilities
- `sepa-beitragslauf`: Requirement „SEPA-XML-Export (pain.008.001.08), immer
  RCUR — clientseitig erzeugt" bekommt eine explizite **DK-TVS-XSD-Konformität**
  als Muss-Kriterium (nicht mehr nur „gegen das ISO-XSD sauber") und vier
  konkrete Härtungen (Nm-70, Ustrd-140, PstlAdr all-or-nothing,
  DtOfSgntr-Fallback).

## Impact

- **Code**: `web/src/lib/sepaXml.ts` (4 Härtungen), `web/src/lib/sepaXml.test.ts`
  (angepasste + neue Regression-Tests), `web/src/lib/sepaXml.xsd.test.ts`
  (neu), `web/src/lib/__schemas__/pain.008.001.08_GBIC_5.xsd` (137K, DK-Datei
  mit Original-Copyright-Header), `web/src/lib/__schemas__/README.md` (neu).
- **CI**: `.github/workflows/ci.yml` bekommt einen `apt-get install -y
  libxml2-utils` Schritt vor `pnpm -C web test`.
- **Betrieb**: Kein Deploy-relevanter Effekt, kein Migrations- oder
  Backfill-Bedarf. Nach Merge sind neue Beitragsläufe strenger validiert; die
  bereits laufende BW-Bank-Kompatibilität bleibt erhalten (Härtungen sind
  strikter, nicht lockerer).
- **Nicht betroffen**: Backend-Routen, DB-Schema, Zero-Knowledge-Krypto, Auth-
  Tiers, Deploy-Pipeline.

## Verhältnis zum Vorgänger [[sepa-xml-epc-compliance]]

Der Vorgänger-Change fügte drei EPC-Regel-basierte Ergänzungen ein (`CreDtTm`
mit `Z`, `Cdtr/PstlAdr/Ctry=DE`, `InitgPty/Id` mit `SchmeNm/Prtry=SEPA`). Von
den drei ist **nur `CreDtTm` mit `Z` tatsächlich umgesetzt** — die anderen
beiden wurden nach genauer DK-TVS-Analyse verworfen:

- Ein `<Cdtr><PstlAdr><Ctry>DE</Ctry></PstlAdr>` **ohne** `<TwnNm>` ist im
  GBIC_5-XSD invalid (TwnNm ist [1..1] Pflicht) — die Bestandsdaten für
  Vereins-Kontoinhaber-Stadt fehlen aktuell.
- Ein `<InitgPty><Id>` wird von der DK ausdrücklich **nicht** empfohlen; die
  Gläubiger-ID gehört ausschließlich in `<CdtrSchmeId>`.

Diese Erkenntnis rechtfertigt genau den XSD-Gate hier: solche Detailregeln
zuverlässig zu erkennen, statt sie per Bank-Rejection-Loop empirisch zu
suchen. Der Vorgänger-Change kann parallel archiviert werden — er ist mit dem
`CreDtTm-Z`-Teilfix funktional erledigt; die beiden verworfenen Punkte werden
in dessen Archiv-Notiz vermerkt.
