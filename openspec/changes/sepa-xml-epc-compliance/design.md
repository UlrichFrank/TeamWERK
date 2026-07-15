## Context

Die per `web/src/lib/sepaXml.ts` erzeugte `pain.008.001.08`-Datei validiert gegen das offizielle ISO-20022-XSD (lokal mit `xmllint --schema` verifiziert), wird vom Online-Upload der BW-Bank (LBBW) aber generisch mit „XML Dokument ist nicht gültig" abgewiesen. Ursache ist das **EPC-SEPA-Rulebook** bzw. bankseitige Zusatzprüfungen, die drei Erweiterungen erwarten, die das Basis-XSD als optional deklariert.

Der Beitragslauf ist der jährliche Haupt-Cashflow des Vereins — der Export muss verlässlich bei der Hausbank durchgehen. Der Fix für leere `<DtOfSgntr>`-Elemente wurde bereits in Commit `<vorherige Änderung>` gemergt; hier geht es um die verbleibenden bankseitigen Konformitätsregeln.

Der Builder ist **client-seitig** (Zero-Knowledge Modell B): Die Datei wird im Browser des Kassierers aus entschlüsselten Blobs gebaut, der Server sieht sie nie. Änderungen betreffen daher ausschließlich TypeScript.

## Goals / Non-Goals

**Goals:**
- BW-Bank akzeptiert die pain.008.001.08-Datei aus dem Beitragslauf ohne weitere Handanpassungen.
- Datei bleibt XSD-valid gegen das offizielle ISO-Schema.
- Fachliche Invarianten (ein `PmtInf`, `SeqTp=RCUR`, `ReqdColltnDt`, Beträge, Verwendungszweck) unverändert.

**Non-Goals:**
- Keine Umstellung auf pain.008.001.02 (User hat bestätigt: BW-Bank verlangt .08).
- Keine Änderungen am Server-Endpoint `POST /api/fee-run/export-data`, an Datenmodell oder Auth.
- Keine bankspezifische Verzweigung — die drei Ergänzungen sind EPC-konform und werden von allen SEPA-erreichbaren Banken akzeptiert.
- Keine kryptographische Signatur der XML (nicht vom EPC-Rulebook gefordert).

## Decisions

**1. Feste UTC-Zeitzone `Z` in `<CreDtTm>`, keine lokale Zone.**
Der Node-Builder emittiert `2026-07-15T17:18:03Z` statt bisher `2026-07-15T17:18:03`. Alternative wäre `+01:00`/`+02:00` je Sommer-/Winterzeit — verworfen, weil `Z` (UTC) simpler und für Banken unmissverständlich ist. `creDtTm(input.createdAt)` in `sepaXml.ts` liefert bereits UTC-Komponenten aus dem Date-Objekt; nur der String-Suffix fehlt.

**2. `<Cdtr>` bekommt minimale `<PstlAdr><Ctry>DE</Ctry></PstlAdr>`, keine Straße/PLZ.**
Der Verein ist der Rechnungssteller; der Countrycode `DE` reicht für die BW-Bank-Validierung. Volladresse (Straße, PLZ, Ort) wäre optional möglich, aber ohne Nutzen — der Verein ist über die Gläubiger-ID eindeutig identifiziert. Kein neues Feld im `SepaBuildInput` nötig — `Ctry='DE'` ist im deutschen SEPA-Kontext hardkodierbar (die Vereins-Adresse liegt zudem nicht im Krypto-Blob; müsste erst neu strukturiert werden).

**3. `<SchmeNm><Prtry>SEPA</Prtry></SchmeNm>` unter `<InitgPty>/<Id>/<OrgId>/<Othr>`.**
Der bereits vorhandene `<CdtrSchmeId>`-Block trägt diese Annotation schon (siehe `sepaXml.ts` L192). Wir spiegeln sie 1:1 in den `InitgPty`-Block der Gläubiger-ID.

**4. Kein neuer Test gegen ein bundled XSD.**
Die Validierung erfolgt weiter über String-Assertions in `sepaXml.test.ts`. Ein echtes XSD-Round-Trip wäre wertvoll, aber (a) das XSD ist nicht im Repo (Lizenzfrage bei ISO 20022), (b) `xmllint` als Test-Dependency würde CI-Setup verkomplizieren. Manuelle Verifikation lokal mit `xmllint --schema` reicht als Absicherung.

## Risks / Trade-offs

- **Risiko: BW-Bank akzeptiert die drei Fixes trotzdem nicht** → Mitigation: Nutzer testet vorab die manuell gefixte Datei `/Users/ulrich/Downloads/beitragslauf_2026-27.fixed.xml`. Erst wenn die durchgeht, wird der Code-Fix gemergt und deployed.
- **Risiko: Eine andere Bank weist das erweiterte Format ab** → sehr unwahrscheinlich (die drei Ergänzungen sind EPC-Standard). Falls doch, ist ein bankspezifischer Zweig weiterhin möglich, aber nicht Teil dieser Change.
- **Trade-off: Hardcodiertes `Ctry=DE`** → Solange die App auf Vereine in Deutschland beschränkt ist (siehe Betrieb: LBBW/Baden-Württemberg), kein Problem. Wenn irgendwann ein AT-/CH-Verein die App nutzt, muss `Ctry` konfigurierbar werden — dann separater Change.
