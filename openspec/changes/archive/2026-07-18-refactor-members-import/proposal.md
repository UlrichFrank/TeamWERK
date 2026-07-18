## Why

Welle 3 der `test-coverage-roadmap` — der funktionserhalt-kritische Fall. `members.Import`
(`internal/members/handler.go:1675`, ~533 LOC, gocognit ~177) ist die komplexeste Funktion des
Codebase und der Kern des Mitglieder-Imports (append/update/enrich/preview × dryRun × selected).
Sie ist heute nur punktuell getestet: BOM-Handling, CSV-interne Dedup, die 400-Fehlerpfade und
der not_found-Pfad haben **keinen** Test; Delimiter und Spalten-Aliase sind nur **implizit**
(über Happy-Path-Bestandstests) abgedeckt, aber nie als expliziter Contract festgehalten.
Kritischer: die `changes[]`-Report-Reihenfolge und der frühe enrich-Mehrdeutigkeits-Ausgang —
beide entstehen in den riskantesten Extract-Stufen — haben **null** Absicherung. Ein Refactor dieser Funktion ohne
Sicherheitsnetz wäre fahrlässig. `test-strategy` schreibt hier ausdrücklich „erst Extract-Method-
Refactoring, dann Tests" **nicht** vor, sondern das Gegenteil für Sicherheit: **erst
Charakterisierungstests, dann Refactor** — die Tests sind die Abnahme-Instanz jedes Schritts.

## What Changes

**Zweiphasig, Verhalten bleibt byte-genau gleich.**

**Phase 1 — Charakterisierung (16 HTTP-Tests, `internal/members/import_test.go`):** BOM-Strip,
Delimiter-Detection (`,`/`;`, „nur erste Zeile"), Column-Aliase, CSV-interne Dedup inkl.
Zeilennummern-Meldungen, die 400-Pfade (fehlende Pflichtspalte, kaputte CSV, leere Datei,
fehlendes Datei-Feld), leere Namenszelle → Row-Error (kein 400), enrich not_found. Nagelt das
**Ist-Verhalten** fest (Report-JSON + Row-`message`-Strings + DB-Effekte). Erfordert eine
minimale Erweiterung der Test-Response-Struct (`Message`/`DOB`/`IBANWarning`-Felder — der Row-
`message`-String ist der einzige Contract-Träger für Fehlerdetails).

**Phase 2 — 6-Stufen-Extract (behavior-preserving, `internal/members/handler.go`):**
1. `normalize*` (Gender/Status/Beitragsfrei/Sepa) Closures → Top-Level-Funktionen.
2. `parseImportCSV` (BOM + Delimiter + Header/Alias + ReadAll) → `parsedCSV` + `col`-Methode.
3. `detectCSVDuplicates` → `csvDupes`.
4. `lookupExistingMember` (DB-Lookup + Ambiguität, **kritischste Stufe**) → diskriminiertes
   `lookupResult` ohne Report-Zugriff.
5. `insertNewMember` (created-Branch).
6. `buildMemberUpdate` (update/enrich-Branch, größte Feldbreite).

Ziel: `Import` fällt von gocognit ~177 unter die Gate-Schwelle (`metrics/thresholds.yml`:
gocognit 35 / gocyclo 12). Nach **jedem** Extract-Schritt muss die Charakterisierungssuite grün
bleiben.

## Capabilities

### New Capabilities

- `members-import-refactor`: dokumentiert (a) dass das Import-Verhalten durch HTTP-
  Charakterisierungstests festgenagelt ist (Report-JSON, Fehler-Meldungen, DB-Effekte), und
  (b) dass `Import` in benannte Einheiten unterhalb der Komplexitäts-Schwelle zerlegt ist,
  ohne beobachtbares Verhalten zu ändern.

### Modified Capabilities

_(keine — die funktionalen Import-Capabilities [`csv-import`, `member-csv-import-*`,
`members-csv-enrich-mode`] behalten ihre Requirements unverändert; dieser Change ändert kein
beobachtbares Verhalten, sondern sichert und strukturiert es.)_

## Impact

- **Tests:** `internal/members/import_test.go` (16 neue Charakterisierungstests + Struct-
  Erweiterung + ein Inline-Multipart-Helfer für „fehlendes Datei-Feld").
- **Code:** `internal/members/handler.go` — `Import` wird in 6 Schritten in Top-Level-/Methoden-
  Helfer zerlegt. Kein Verhaltens-, API-, Schema- oder SSE-Change. Exakte Fehlermeldungen
  (englische 400-Texte, deutsche Row-`message`-Strings) und der `substr(date_of_birth,1,10)`-
  Gotcha bleiben wörtlich erhalten.
- **Metriken:** `make metrics-gate` — `Import` unter gocognit 35 / gocyclo 12; neue Helfer dürfen
  keine `dupl`-Regression erzeugen.
- **Reihenfolge-Sicherheit:** Phase 1 wird zuerst gemergt/grün, Phase 2 Schritt für Schritt mit
  grüner Suite nach jedem Schritt; `make test-race` vor Merge.
