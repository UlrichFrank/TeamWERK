## Context

Welle 3. `members.Import` (`internal/members/handler.go:1675-2207`, ~533 LOC, gocognit ~177).
Route `POST /api/members/import` (Multipart; Formfelder `mode`/`preview`/`fields`/`apply_lines`),
Tier `RequireClubFunction("vorstand")`. Bestand: ~24 Tests in `import_test.go` (Member-Number-
Logik, enrich-Ambiguität, Bankdaten-Ignorierung, Whitelist, apply_lines, Status-Mapping).
Ungetestet: BOM, Delimiter-Contract, Dedup, 400-Pfade, not_found, leere Namenszelle. Scope +
Bauplan durch zwei Detail-Recherchen gegen den echten Code verifiziert.

## Goals / Non-Goals

**Goals:**

- Das Ist-Verhalten von `Import` mit HTTP-Charakterisierungstests festnageln (Report-JSON,
  Row-`message`-Strings, DB-Effekte), BEVOR refactored wird.
- `Import` in 6 benannte Einheiten zerlegen, gocognit/gocyclo unter die Gate-Schwelle, **ohne**
  beobachtbares Verhalten zu ändern.
- Nach jedem Extract-Schritt bleibt die Charakterisierungssuite grün (Abnahme-Instanz).

**Non-Goals:**

- Keine Verhaltens-/API-/Schema-/SSE-Änderung, keine neuen Import-Features.
- Keine Änderung der funktionalen Import-Requirements.
- Keine Authz-Tests (Persona-Matrix deckt `vorstand`-only / `kassierer`→403 ab).

## Decisions

**D1 — Charakterisierung ZUERST, dann Refactor (nicht umgekehrt).** Bei funktionserhalt-
kritischem Code ist die Test-Suite das Sicherheitsnetz des Refactors. Phase 1 (Tests) wird
eigenständig grün und gemergt/committed, bevor eine Zeile Produktionscode bewegt wird.

**D2 — Test-Response-Struct erweitern (minimal).** Die vorhandene `importReport`-Test-Struct hat
pro Row nur `line/status/name/changes`. Der Row-`message`-String ist aber der **einzige**
Contract-Träger für Fehlerdetails (Dedup, Pflichtfeld). Ergänze `Message`, `DOB`, `IBANWarning`
(json-Tags `message`/`dob`/`iban_warning`) — additiv, bricht keinen Bestandstest.

**D3 — 400-Antworten sind text/plain, nicht JSON.** Die frühen Fehlerpfade nutzen `http.Error`
(englische Texte: `"missing required column: Vorname"`, `"cannot parse CSV"` …). 400-Tests
prüfen `res.StatusCode` + optional den Body-String, **nicht** `decodeReport`.

**D4 — Inline-Multipart-Helfer für „fehlendes Datei-Feld".** `postImport`/`testutil.PostMultipart`
schreiben immer ein `file`-Feld; der 400-Pfad `missing file` ist nur über einen Multipart-Body
ohne `file`-Feld erreichbar (nur `WriteField("mode","append")`).

**D5 — Extract-Reihenfolge & Signaturen (aus dem Bauplan).**
1→2→3→4→5→6 (Abhängigkeiten: 3→2, 5→{2,4}, 6→{1,2,4}). Signaturen ohne Report-Zugriff; Report-
Nebenwirkungen bleiben im Handler, die Helfer geben diskriminierte Ergebnisse/Structs zurück:
- `parseImportCSV(raw) (*parsedCSV, error)` mit `(*parsedCSV).col`.
- `detectCSVDuplicates(rows, col) csvDupes`.
- `lookupExistingMember(ctx, first,last,dob,mode) lookupResult{outcome, member dbMember, message, dbErr}`.
- `insertNewMember(...) (ibanWarn string, err error)`, `buildMemberUpdate(...) memberUpdate`.
  Der `ibanWarn`-String (dupliziert in Insert- und Update-Pfad) wird **im jeweiligen Helfer
  berechnet und zurückgegeben** (`memberUpdate.ibanWarn` für Stufe 6); der Handler schreibt ihn
  ins `iban_warning`-Row-Feld — vor Stufe 6 festgelegt, damit der Contract nicht driftet.

**D6 — Wörtlich zu erhaltende Contracts (Refactor-Guards).** (a) englische 400-Texte; (b)
deutsche Row-`message`-Strings inkl. Dedup-Zeilennummern (`"Mehrfach in CSV (auch Zeile %d)"` /
`"… (zuerst Zeile %d)"`, 1-basiert inkl. Header); (c) der `substr(COALESCE(date_of_birth,''),1,10)=?`
-Abgleich und der Fall-B-Umschalter (`useDobArg=false`) in `lookupExistingMember`; (d) die
`changes`-Meldungsformate der Vorschau; (e) Status-Fallback `"" → "aktiv"` nur im Insert.

## Risks / Trade-offs

- **Stufe 4 (`lookupExistingMember`) ist die kritischste:** drei voneinander abhängige DB-
  Queries, der `substr`-Gotcha, der 17-Spalten-`Scan` in exakter Reihenfolge, vier Report-
  Ausgänge. Ein Fehler propagiert in Stufe 5/6 (die `dbMember` konsumieren). Mitigation: Stufe 4
  erst nach grüner Charakterisierung; die enrich-Member-Number-Tests als scharfe Abnahme.
- **Delimiter-Detection ist bewusst „nur erste Zeile":** ein Refactor auf einen echten Sniffer
  würde `TestImport_DelimiterDetectedFromFirstLineOnly` brechen — genau der Sinn des Guards.
- **Phasen-Trennung:** Phase 2 (Refactor) ändert Produktionscode und ist deshalb gegenüber
  Phase 1 (reine Tests) ein separater, bestätigungspflichtiger Schritt.
