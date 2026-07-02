## 1. Backend: Endpoint liefert kanonische Kurzform

- [x] 1.1 In `internal/chat/team_groups.go` das `TeamGroup`-Struct anpassen: Feld `DisplayShort string json:"displayShort"` ergänzen, tote Felder `TeamName` und `GroupCount` entfernen.
- [x] 1.2 In `ListTeamGroups` beide Team-Queries (global + `user_accessible_teams`) um die Spalte `COALESCE(<db.TeamDisplayShort("t")>, t.name) AS display_short` erweitern; `internal/db` importieren.
- [x] 1.3 `teamInfo` um `displayShort` erweitern, im `Scan` mitlesen; die Go-seitige `groupCounts`-Map ersatzlos entfernen; `DisplayShort` im Ergebnis setzen.
- [x] 1.4 `go build ./...` + `gofmt` grün.

## 2. Backend: Tests

- [x] 2.1 In `internal/chat/team_groups_test.go` Test ergänzen: zwei männliche B-Jugend-Teams in aktiver Saison, Caller nur auf eines eingetragen → Response-`displayShort` = `"mB2"` (Happy-Path der Invariante).
- [x] 2.2 Test ergänzen: genau ein männliches B-Jugend-Team, Caller sieht es → `displayShort` = `"mB"`.
- [x] 2.3 Bestehende Tests auf Referenzen zu `teamName`/`groupCount` prüfen und anpassen (keine erwartet). `go test ./internal/chat/...` grün.

## 3. Frontend: displayShort konsumieren

- [x] 3.1 In `web/src/pages/ChatPage.tsx` das `TeamGroup`-Interface anpassen: `displayShort: string` ergänzen, `groupCount`/`ageClass`/`gender`/`teamNumber` entfernen, sofern nur für die alte Berechnung genutzt.
- [x] 3.2 `teamGroupShortNames`-Memo ersetzen durch direkten Zugriff auf `tg.displayShort` (Picker-Anzeige Zeile ~1340 und Filter Zeile ~1223).
- [x] 3.3 Sicherstellen, dass `buildTeamShortNames` weiterhin importiert bleibt (Broadcast-Modal nutzt es unverändert) und der Chat-Picker es nicht mehr aufruft.
- [x] 3.4 `pnpm -C web build` + `pnpm -C web lint` grün.

## 4. Verifikation & Abschluss

- [x] 4.1 `/verify-change` ausführen (Build/Test/Lint + Projekt-Invarianten, `openspec validate`).
- [ ] 4.2 Manuell prüfen: als Nutzer mit Zugriff auf nur ein Team einer Mehrfach-Gruppe im Gruppenchat-Picker die Kurzform mit Nummer (`mB2`) sehen.
- [ ] 4.3 Proposal archivieren (`openspec archive chat-team-group-standard-shortname`).
