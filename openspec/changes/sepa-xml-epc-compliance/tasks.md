# Implementation Tasks

## 1. Bank-Test der manuell gefixten Datei (Voraussetzung)

- [ ] 1.1 Nutzer lädt `/Users/ulrich/Downloads/beitragslauf_2026-27.fixed.xml` bei der BW-Bank hoch und meldet Ergebnis zurück.
- [ ] 1.2 Wenn die Bank die Datei akzeptiert → weiter mit Phase 2. Wenn abgelehnt → Fehlermeldung notieren, Proposal überarbeiten (keine Blindumsetzung).

## 2. Code-Änderungen in `web/src/lib/sepaXml.ts`

- [ ] 2.1 `creDtTm(input.createdAt)` gibt weiter UTC-Komponenten aus; String-Return um `Z` erweitern (Suffix „Z" an das bestehende `YYYY-MM-DDThh:mm:ss`-Format anhängen).
- [ ] 2.2 Im `<Cdtr>`-Block nach `leaf('Nm', ascii(input.kontoinhaber))` einen `el('PstlAdr', [leaf('Ctry', 'DE')])`-Knoten einfügen.
- [ ] 2.3 Im `<InitgPty>`-Block den `<Othr>`-Kindern nach `leaf('Id', input.glaeubigerId)` einen `el('SchmeNm', [leaf('Prtry', 'SEPA')])`-Knoten hinzufügen (analog zum bereits vorhandenen `<CdtrSchmeId>`-Block).

## 3. Tests in `web/src/lib/sepaXml.test.ts`

- [ ] 3.1 Assertion: `xml` enthält `<CreDtTm>` mit Suffix `Z` (Regex `/\<CreDtTm\>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z\<\/CreDtTm\>/`).
- [ ] 3.2 Assertion: `xml` enthält innerhalb des `<Cdtr>`-Blocks `<PstlAdr><Ctry>DE</Ctry></PstlAdr>`.
- [ ] 3.3 Assertion: `xml` enthält innerhalb des ersten `<Othr>`-Blocks unter `<InitgPty>` sowohl `<Id>` als auch `<SchmeNm><Prtry>SEPA</Prtry></SchmeNm>`.
- [ ] 3.4 Bestehendes Szenario „valides XML-Prolog + Wohlgeformtheit" bleibt grün; neue Assertions dürfen keine Alt-Assertions brechen.

## 4. Manuelle Verifikation (Author)

- [ ] 4.1 `pnpm -C web test -- --run sepaXml` — alle Tests grün.
- [ ] 4.2 Lokal einen Test-Beitragslauf exportieren (Dev-Modus, Dummy-Daten) und die Datei mit `xmllint --schema pain.008.001.08.xsd` gegen das ISO-XSD prüfen.
- [ ] 4.3 Diff gegen `/Users/ulrich/Downloads/beitragslauf_2026-27.fixed.xml` — die drei neuen Ausgaben stimmen strukturell überein.

## 5. Deploy & Post-Deploy-Verifikation (manuell)

- [ ] 5.1 Nach Merge in `main`: `make deploy` (baut Frontend neu, embed.FS, systemctl restart).
- [ ] 5.2 Kassierer erzeugt echten Beitragslauf-Export aus der Produktion und lädt ihn bei der BW-Bank hoch → Upload akzeptiert.
- [ ] 5.3 `openspec archive sepa-xml-epc-compliance`.
