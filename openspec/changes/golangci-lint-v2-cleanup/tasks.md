## 1. Lint-Config: node_modules ausschließen

- [ ] 1.1 `.golangci.yml` → `linters.exclusions.paths` ergänzen um `node_modules` und `web/node_modules`
- [ ] 1.2 Verifikation: `golangci-lint run ./...` zeigt keine `flatted`-Findings mehr (2× govet)

## 2. Toten Code entfernen (6× unused)

- [ ] 2.1 `cmd/teamwerk/main.go`: vor Löschung `grep -rn "corsMiddleware\|spaHandler" .` — wenn keine Aufrufer, Funktionen löschen + ungenutzte Imports aufräumen
- [ ] 2.2 `internal/chat/handler_test.go`: `directConvExists` löschen
- [ ] 2.3 `internal/files/handler_test.go`: `claimsWithFn` löschen
- [ ] 2.4 `internal/games/handler.go`: vor Löschung `grep -rn "effectiveEventDuration\|findTemplateForGame" .` — wenn nur am Definitionsort, löschen
- [ ] 2.5 `go build ./...` durchläuft nach Löschungen

## 3. ST1005: Error-Strings entkapitalisieren (8 Stellen)

- [ ] 3.1 `internal/config/handler.go:50, 60` — beide Strings kleinschreiben
- [ ] 3.2 `internal/games/handler.go:159, 177, 180` — kleinschreiben
- [ ] 3.3 `internal/games/regen.go:540, 558, 561` — kleinschreiben
- [ ] 3.4 Pro Änderung: `grep -rn "<exakte alte Meldung>" .` — falls Tests die Message vergleichen, mit-aktualisieren
- [ ] 3.5 `go test -race ./...` durchläuft

## 4. QF1003: tagged switches (4 Stellen)

- [ ] 4.1 `internal/carpooling/handler.go:372` — `if`-Kette auf `switch scanErr` umstellen
- [ ] 4.2 `internal/carpooling/handler.go:390` — analog
- [ ] 4.3 `internal/carpooling/handler.go:471` — `switch typ` statt `if typ == ...`
- [ ] 4.4 `internal/members/welcome_email.go:79` — `switch gender` statt `if gender == "m" ... else if gender == "w"`
- [ ] 4.5 `go test -race ./internal/carpooling ./internal/members` grün

## 5. QF1001: De-Morgan-Vereinfachungen (3 Stellen)

- [ ] 5.1 `internal/carpooling/handler_test.go:393, 394` — Boolean-Ausdrücke vereinfachen
- [ ] 5.2 `internal/games/handler.go:271` — analog
- [ ] 5.3 `go test -race ./internal/carpooling ./internal/games` grün

## 6. Verifikation

- [ ] 6.1 `golangci-lint run ./...` → `0 issues.`
- [ ] 6.2 `go test -race ./...` → alle Tests grün
- [ ] 6.3 `go vet ./...` ohne Findings
- [ ] 6.4 `openspec validate golangci-lint-v2-cleanup` grün

## 7. Archivierung

- [ ] 7.1 Proposal nach Merge in `main` archivieren (`openspec archive golangci-lint-v2-cleanup`)
