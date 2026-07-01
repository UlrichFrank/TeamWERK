## 1. Training-Respond absichern

- [ ] 1.1 In `Respond` (`internal/trainings/handler.go`) den toten `switch claims.Role` durch ownership-/capability-basierte Prüfung ersetzen: eigenes Member immer ok; fremdes `member_id` nur bei Manage-Berechtigung (`hasTeamAccess`) oder `parentHasChild`, sonst 403
- [ ] 1.2 Verhalten bei fehlendem auflösbarem eigenem Member unverändert (400/422)

## 2. Spiel-Respond absichern

- [ ] 2.1 In `RespondToGame` (`internal/games/handler.go`) dieselbe Prüfung umsetzen (Manage-Check über bestehende Team-Access-Logik des games-Package); `UserCanSeeGame`-Gate bleibt
- [ ] 2.2 Toten `case "spieler"/"elternteil"` entfernen

## 3. Tests

- [ ] 3.1 `TestRespond_OwnMember_OK` → 204 (eigene Rückmeldung)
- [ ] 3.2 `TestRespond_OwnChild_OK` → 204 (Eltern für verknüpftes Kind)
- [ ] 3.3 `TestRespond_ForeignMember_Forbidden` → 403 (fremdes, nicht verknüpftes Member; kein DB-Eintrag)
- [ ] 3.4 `TestRespond_TrainerForAnyMember_OK` → 204 (Manage-Berechtigte ausgenommen)
- [ ] 3.5 `TestGameRespond_OwnChild_OK` → 204
- [ ] 3.6 `TestGameRespond_ForeignMember_Forbidden` → 403

## 4. Verifikation

- [ ] 4.1 `go test ./...` + `go vet` + `golangci-lint` + `gofmt` grün; `/verify-change`
- [ ] 4.2 Bestehende RSVP-Tests (eigene/Eltern-Rückmeldung, erw. Kader) bleiben grün
- [ ] 4.3 `openspec validate secure-respond-parent-authz --strict`; nach Abnahme archivieren (Sync der `eltern-rsvp`-MODIFIED-Requirement)
