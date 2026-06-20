## Context

Die Migration der `.golangci.yml` auf v2-Schema hat die Linter-Ausführung wieder ermöglicht (siehe vorausgegangener `chore(lint)`-Commit, der die v1-`linters.enable`-Liste durch die v2-Defaults ersetzt hat). Der erste Lauf seither meldet 23 Issues — ein Mix aus echter toter Funktionalität (`unused`), stilistischen Inkonsistenzen (`ST1005`), mechanischen Refactoring-Vorschlägen (`QF1001`/`QF1003`) und einem Konfigurationsmangel (Drittanbieter-Go-Code unter `web/node_modules` wird mit gescannt).

Bislang gibt es keine Spezifikation, die den lint-clean-State als Invariante festhält — der CLAUDE.md-Standard erwähnt zwar `make lint` als Teil des `pre-push`-Gates, aber im Repo war der Zustand nicht erreichbar.

## Goals / Non-Goals

**Goals:**
- Nach Abschluss: `golangci-lint run ./...` liefert 0 Issues
- Drittanbieter-Code unter `node_modules` wird nicht mehr gescannt
- Eine neue Capability `lint-clean-baseline` macht „lint grün" zur prüfbaren Invariante
- Keine Verhaltensänderung in irgendeiner Geschäftslogik

**Non-Goals:**
- Keine neuen Linter aktivieren (z. B. `gocritic`, `revive`) — das ist ein eigener Change
- Kein Test-Refactoring abseits des erforderlichen Mit-Angleichens an entkapitalisierte Errors
- Keine Generic-Refactorings oder Performance-Optimierungen am angefassten Code

## Decisions

### Tote Funktionen wirklich löschen, nicht über `_ = foo` ruhigstellen

`corsMiddleware` und `spaHandler` in `cmd/teamwerk/main.go` sind Reste aus einer früheren Iteration, in der das Frontend separat gemountet wurde. Heute liefert die App das SPA via `embed.FS` und CORS-Header werden durch Nginx gesetzt. Reine Code-Pflege — löschen.

Die unused-Test-Helpers `directConvExists`, `claimsWithFn` waren wahrscheinlich Hilfsfunktionen für Tests, die später entfernt wurden, ohne den Helper mit-zu-entfernen.

`effectiveEventDuration` und `findTemplateForGame` in `internal/games/handler.go` müssen vor Löschung **mit `grep` projekt-weit** gegengeprüft werden (auch in Tests + `internal/games/regen.go` mit demselben Domain-Wissen). Sind sie wirklich tot, werden sie gelöscht; falls doch noch referenziert, ist das `unused`-Finding ein Fehlalarm (sehr unwahrscheinlich bei `staticcheck`).

### ST1005-Strings: zentral entkapitalisieren

Go-Konvention: Fehler-Strings beginnen klein, weil sie in Wrap-Ketten (`fmt.Errorf("foo: %w", err)`) als Mittelteil auftauchen. Beispiel: `"Vorlage nicht gefunden"` → `"vorlage nicht gefunden"`. Pure Stilkorrektur, keine semantische Änderung.

**Risiko:** Tests, die die Error-Message wörtlich vergleichen, müssen mit-aktualisiert werden. Vor jedem Patch `grep` auf die Originalmessage im Test-Code.

### QF1003 (tagged switch) und QF1001 (De Morgan)

Mechanische Refactorings. Beispiel für QF1003:

```go
// vorher
if scanErr == sql.ErrNoRows { ... }
if scanErr == nil { ... }

// nachher
switch scanErr {
case sql.ErrNoRows: ...
case nil: ...
}
```

QF1001 wandelt `!(a && b)` in `!a || !b`. Beide reduzieren die kognitive Last beim Lesen, ohne Verhalten zu ändern.

### node_modules-Exclude statt einzelner Issue-Suppressions

In v2 unter `linters.exclusions.paths` reichen `node_modules` und `web/node_modules` als Regex-Anker, um die Drittanbieter-Go-Pakete (z. B. `flatted/golang`) auszuschließen. Das ist sauberer als `nolint:govet`-Direktiven in Dateien, die wir nicht besitzen.

### Capability-Modellierung

`lint-clean-baseline` als eigene Capability ist absichtlich klein und prüf-orientiert — die Requirement-Sprache nutzt SHALL, um den Maintenance-Zustand klar machbar zu halten. Damit kann zukünftig jedes Proposal, das Lint-Pflege beeinflusst (neue Linter, weitere Excludes), diese Capability gezielt erweitern.

## Risks / Trade-offs

- **Löschen von Production-Funktionen** (`corsMiddleware`, `spaHandler`): wenn doch noch jemand reflektiert / via Tag ruft, würde Build zur Compile-Zeit nicht fehlschlagen, aber zur Laufzeit (panic on nil). → Mitigation: vor Löschung `grep -rn "corsMiddleware\|spaHandler" cmd internal web/src` — wenn Treffer nur am Definitionsort, sicher.
- **Test-Update-Welle**: wenn 5+ Tests Error-Strings vergleichen, wächst der Diff. → Realistisch: max. 2-3 betroffen, Repo nutzt überwiegend `errors.Is` statt String-Compare.
- **Pre-existing Bugs**: ein paar `unused`-Findings können tatsächlich Reste eines unvollendeten Features sein. → Beim Löschen kurz im Commit-Hinweis vermerken, falls offensichtlich.

## Migration Plan

Kein Schema-Change. Reine Code-Pflege. Reihenfolge (jeweils eigener Commit per Konvention):

1. **Lint-Config**: `node_modules`-Exclude — sofortiger Effekt: `govet`-Issues weg
2. **Unused entfernen**: 6 Funktionen gelöscht — Build muss durchlaufen
3. **ST1005 entkapitalisieren**: 8 Stellen — Tests evtl. mit-angleichen
4. **QF1003 tagged switches**: 4 Stellen — `go test` als Regression-Check
5. **QF1001 De Morgan**: 3 Stellen — `go test` als Regression-Check
6. **Verifikation**: `golangci-lint run ./...` muss 0 Issues + `go test -race ./...` grün
