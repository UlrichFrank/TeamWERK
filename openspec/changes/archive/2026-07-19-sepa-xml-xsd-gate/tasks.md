# Implementation Tasks

## 1. XSD-Datei ins Repo

- [x] 1.1 `pain.008.001.08_GBIC_5.xsd` aus dem offiziellen DK-Bundle
  `DK-TVS_SEPA_GBIC_5zzglISO_Originale.zip` (ebics.de → Datenformate →
  Ergänzende Dokumente) nach `web/src/lib/__schemas__/` extrahieren.
  DK-Copyright-Header in der XSD-Datei belassen.
- [x] 1.2 `web/src/lib/__schemas__/README.md` anlegen (Herkunft, Version,
  Aktualisierungspfad, Voraussetzung `xmllint`).

## 2. Härtungen im Generator `web/src/lib/sepaXml.ts`

- [x] 2.1 Helfer `nm70(s: string): string` (ASCII + Truncation 70). Anwenden
  auf `Dbtr/Nm`, `Cdtr/Nm`, `InitgPty/Nm`.
- [x] 2.2 Helfer `ustrd140(s: string): string` (ASCII + Truncation 140).
  Anwenden auf `RmtInf/Ustrd`.
- [x] 2.3 `PstlAdr` im Dbtr nur emittieren, wenn `city` gesetzt ist
  (all-or-nothing — TwnNm/Ctry Pflicht laut GBIC_5).
- [x] 2.4 Konstante `DEFAULT_MANDAT_DATUM = '2026-06-01'` definieren.
  `DtOfSgntr` bei fehlendem `it.mandatDatum` auf diesen Wert setzen
  (statt Element wegzulassen). Kommentar erklärt Rationale.

## 3. XSD-Test-Gate `web/src/lib/sepaXml.xsd.test.ts`

- [x] 3.1 Neue Vitest-Datei anlegen. `spawnSync('xmllint', ['--version'])`
  prüft Verfügbarkeit; wenn fehlt: `describe.skipIf` + sichtbare
  `console.warn`-Meldung (kein Silent-Pass).
- [x] 3.2 `validate(xml)`-Helfer: schreibt XML in `mkdtempSync`-Temp-Datei,
  ruft `execFileSync('xmllint', ['--noout', '--schema', XSD, file])`, gibt
  Exit-Code + stderr zurück.
- [x] 3.3 Vier Testfälle: Standard-Fall, fehlendes Mandatsdatum (→ Fallback),
  fehlende Stadt (→ kein PstlAdr), Multi-Transaktion.

## 4. Anpassung bestehender Substring-Tests `web/src/lib/sepaXml.test.ts`

- [x] 4.1 Ersetze den `describe('DtOfSgntr weglassen, …')`-Block durch neuen
  Block „`DtOfSgntr (Pflichtelement laut GBIC_5-TVS)`" mit Assertion, dass
  fehlendes `mandatDatum` auf `2026-06-01` fällt.
- [x] 4.2 Neue Regression-Tests für Nm-70, Ustrd-140, PstlAdr-all-or-nothing
  im `describe('DK-TVS-Härtungen …')`-Block.
- [x] 4.3 `pnpm -C web exec vitest run src/lib/sepaXml` — alle Tests grün
  (17/17 Stand 2026-07-16).

## 5. CI-Integration

- [x] 5.1 In `.github/workflows/ci.yml` vor `- run: pnpm -C web test` einen
  Schritt `- run: sudo apt-get update && sudo apt-get install -y libxml2-utils`
  einfügen.
- [ ] 5.2 CI-Lauf auf dem PR verifizieren: `xmllint --version` geloggt,
  `sepaXml.xsd.test.ts` nicht skipped (4/4 Tests laufen, nicht 0).
- [x] 5.3 `.githooks/pre-push` prüft nur lokal `xmllint`-Verfügbarkeit (falls
  fehlt: Warnung, kein Hard-Fail — Entwickler ohne libxml2 sollen weiter
  commiten können, das CI-Gate ist die verbindliche Quelle).

## 6. Dokumentation

- [x] 6.1 `docs/agent/06-gotchas.md` — im SEPA-Beitragslauf-Absatz einen
  Hinweis auf das XSD-Gate ergänzen („Änderungen an `sepaXml.ts` werden
  mechanisch gegen `pain.008.001.08_GBIC_5.xsd` validiert; für lokale
  Test-Läufe libxml2 installiert haben").
- [x] 6.2 `docs/agent/07-testing.md` oder `08-verification.md` — Absatz zu
  Schema-Validierung als weiteren Harness-Baustein aufnehmen (analog zu
  `broadcast_test.go`, `arch_test.go`).
- [x] 6.3 In `web/src/lib/__schemas__/README.md` den Prozess zum Ersetzen bei
  neuer DK-Anlage-3-Version dokumentieren (bereits vorhanden — bitte
  querlesen und ggf. schärfen).

## 7. Vorgänger-Change `sepa-xml-epc-compliance` abschließen

- [x] 7.1 Notiz in `openspec/changes/sepa-xml-epc-compliance/tasks.md`
  ergänzen: Tasks 2.2 (Cdtr-PstlAdr) und 2.3 (InitgPty-Id-SchmeNm) wurden
  nach DK-TVS-Analyse verworfen; nur 2.1 (CreDtTm-Z) implementiert. Weiteres
  im Nachfolger `sepa-xml-xsd-gate` (mechanische Verifikation).
- [ ] 7.2 `openspec archive sepa-xml-epc-compliance` — mit dem obigen Vermerk
  im Archiv-Ordner.

## 8. Merge & Post-Deploy

- [ ] 8.1 PR öffnen, CI grün, Review.
- [ ] 8.2 Merge nach `main`. `make deploy`.
- [ ] 8.3 Kassierer erzeugt echten Beitragslauf-Export, lädt bei BW-Bank hoch,
  bestätigt Akzeptanz.
- [ ] 8.4 `openspec archive sepa-xml-xsd-gate`.
