# SEPA-XSD-Schemas

DK-TVS (Technical Validation Subset) für pain.008.001.08 (SEPA-Basislastschrift).
Wird von `sepaXml.xsd.test.ts` genutzt, um jede Änderung an `sepaXml.ts` gegen
das echte Bank-Schema zu prüfen (via `xmllint`).

## Datei

- **`pain.008.001.08_GBIC_5.xsd`** — DK-TVS für SEPA-Basislastschriften (CORE)
  auf Basis pain.008.001.08.
  - **Quelle:** ebics.de → Datenformate → Ergänzende Dokumente,
    `DK-TVS_SEPA_GBIC_5zzglISO_Originale.zip`.
  - **Herausgeber:** Die Deutsche Kreditwirtschaft (DK), 01.04.2025.
  - **Gültig ab:** 05.10.2025 (Anlage 3 V3.9).
  - **Verifiziert:** validiert das offizielle DK-Beispiel `pain.008.001.08.xml`
    aus `XML-Beispiele_SEPA.zip`.

## Aktualisierung bei neuer DK-TVS-Version (GBIC_6+)

1. Neue Anlage-3-Version auf https://www.ebics.de/de/datenformate prüfen; das
   passende Schemas-Bundle (üblicherweise
   `DK-TVS_SEPA_GBIC_<n>zzglISO_Originale.zip` in „Ergänzende Dokumente")
   herunterladen.
2. `pain.008.001.XX_GBIC_<n>.xsd` aus dem ZIP extrahieren und diese Datei hier
   überschreiben (Dateiname mit ändern, falls Version wechselt).
3. Als Sanity-Check die offizielle DK-Beispieldatei
   (`XML-Beispiele_SEPA.zip → pain.008.001.XX.xml`) gegen das neue XSD
   validieren:
   ```
   xmllint --noout --schema pain.008.001.XX_GBIC_<n>.xsd pain.008.001.XX.xml
   ```
   Muss „validates" zurückgeben.
4. Dateipfad in `web/src/lib/sepaXml.xsd.test.ts` (`const XSD = …`) anpassen.
5. Falls ISO-Version wechselt (z. B. auf `pain.008.001.10`): auch den
   `PAIN_NS`-Konstanten-String in `web/src/lib/sepaXml.ts` mit ändern und die
   ADDED-Requirement `Unterstützte SEPA-XML-Schema-Version` in
   `openspec/specs/sepa-beitragslauf/spec.md` fortschreiben.
6. `pnpm -C web exec vitest run src/lib/sepaXml` — alle Tests grün.

Nicht vergessen: Reine Datei-Ersetzung ohne Test-Lauf bringt kein Signal —
das Gate greift erst, wenn `sepaXml.xsd.test.ts` gegen das neue XSD grün ist.

## Voraussetzung

`xmllint` (Teil von libxml2). Auf macOS vorinstalliert; auf CI/Linux via
`apt-get install libxml2-utils`. Fehlt es, wird der Test übersprungen (mit
sichtbarem Skip-Hinweis).
