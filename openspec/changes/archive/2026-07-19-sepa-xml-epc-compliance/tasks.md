# Implementation Tasks

> **Status-Vermerk (2026-07-16):** Von den drei geplanten EPC-Ergänzungen
> wurde nur **2.1 (`CreDtTm` mit `Z`)** übernommen. **2.2 (`Cdtr/PstlAdr` mit
> nur `<Ctry>DE</Ctry>`)** und **2.3 (`InitgPty/Id` mit `SchmeNm/Prtry=SEPA`)**
> wurden nach genauer Analyse gegen das DK-TVS `pain.008.001.08_GBIC_5.xsd`
> **verworfen**:
>
> - 2.2 ist XSD-invalid unter GBIC_5 — `<TwnNm>` ist [1..1] Pflicht, sobald
>   `<PstlAdr>` überhaupt vorkommt; ein `<PstlAdr>` mit nur `<Ctry>` bricht
>   die Schema-Prüfung. Die passende Härtung (`<PstlAdr>` all-or-nothing im
>   Debtor) wurde stattdessen im Nachfolger-Change umgesetzt.
> - 2.3 wird von der DK ausdrücklich nicht empfohlen — die Gläubiger-ID
>   gehört ausschließlich in `<CdtrSchmeId>` (Anlage 3 V26.11 Kap. 2.2.2.4).
>
> Die eigentliche Ursache (Bank-Reject-Loop ohne mechanische Prüfung) wird
> vom Nachfolger-Change [[sepa-xml-xsd-gate]] adressiert: offizielles
> DK-TVS-XSD eingecheckt, mechanische Validierung im CI-Gate. Dieser Change
> hier kann nach dem `CreDtTm`-Z-Fix archiviert werden.
>
> **Archiviert 2026-07-19:** Kern (2.1 `CreDtTm`-Z) ist im Code (`sepaXml.ts`,
> `creDtTm()`) und auf Prod deployt; 2.2/2.3 wie oben bewusst verworfen. Die
> Delta-Spec (`sepa-beitragslauf`, eine MODIFIED-Requirement) wurde **nicht**
> synchronisiert — die Haupt-Spec trägt bereits die neuere, ablösende Fassung
> aus [[sepa-xml-xsd-gate]]; ein Sync würde sie zurückdrehen. Die verbleibenden
> offenen Tasks sind manuelle Bank-Validierung (deckungsgleich mit dem bereits
> abgehakten Deploy des Nachfolgers).


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
