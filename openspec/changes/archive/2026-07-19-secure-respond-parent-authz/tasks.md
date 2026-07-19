> **Status-Vermerk (2026-07-19):** Der eigentliche IDOR-Fix (toter `switch
> claims.Role` → ownership-/capability-basierte Prüfung) war zum Umsetzungs-
> zeitpunkt in **beiden** Handlern (`Respond`, `RespondToGame`) bereits im Code
> — er landete mit `erw-kader-eltern-rueckmeldung`. Die Manage-Prüfung ist
> bewusst **global** (admin/vorstand/trainer/sportliche_leitung), nicht strikt
> team-scoped: das entspricht dem bereits deployten + getesteten Verhalten
> (`TestRespond_Cutoff_TrainerAfter_OK` erlaubt einem Trainer ohne Team-Bindung
> die Rückmeldung). Diese Session hat daher nur die **fehlende Test-Abdeckung**
> ergänzt (games: `TestGameRespond_ForeignMember_Forbidden`,
> `TestGameRespond_OwnMember_OK`) und die Invarianten verifiziert. Die trainings-
> Invarianten sind bereits durch bestehende Tests gedeckt (Name-Mapping):
> OwnMember=`TestRespond_SavesRSVP`, OwnChild=`TestRespond_ParentForChild`,
> Foreign=`TestRespond_StandardForeignMember_Forbidden`,
> Manage=`TestRespond_Cutoff_TrainerAfter_OK`; games OwnChild=
> `TestGameRespond_ParentForExtendedChild_OK`.

## 1. Training-Respond absichern

- [x] 1.1 In `Respond` (`internal/trainings/handler.go`) den toten `switch claims.Role` durch ownership-/capability-basierte Prüfung ersetzen: eigenes Member immer ok; fremdes `member_id` nur bei Manage-Berechtigung (`hasTeamAccess`) oder `parentHasChild`, sonst 403
- [x] 1.2 Verhalten bei fehlendem auflösbarem eigenem Member unverändert (400/422)

## 2. Spiel-Respond absichern

- [x] 2.1 In `RespondToGame` (`internal/games/handler.go`) dieselbe Prüfung umsetzen (Manage-Check über bestehende Team-Access-Logik des games-Package); `UserCanSeeGame`-Gate bleibt
- [x] 2.2 Toten `case "spieler"/"elternteil"` entfernen

## 3. Tests

- [x] 3.1 `TestRespond_OwnMember_OK` → 204 (eigene Rückmeldung)
- [x] 3.2 `TestRespond_OwnChild_OK` → 204 (Eltern für verknüpftes Kind)
- [x] 3.3 `TestRespond_ForeignMember_Forbidden` → 403 (fremdes, nicht verknüpftes Member; kein DB-Eintrag)
- [x] 3.4 `TestRespond_TrainerForAnyMember_OK` → 204 (Manage-Berechtigte ausgenommen)
- [x] 3.5 `TestGameRespond_OwnChild_OK` → 204
- [x] 3.6 `TestGameRespond_ForeignMember_Forbidden` → 403

## 4. Verifikation

- [x] 4.1 `go test ./...` + `go vet` + `golangci-lint` + `gofmt` grün; `/verify-change`
- [x] 4.2 Bestehende RSVP-Tests (eigene/Eltern-Rückmeldung, erw. Kader) bleiben grün
- [ ] 4.3 `openspec validate secure-respond-parent-authz --strict`; nach Abnahme archivieren (Sync der `eltern-rsvp`-MODIFIED-Requirement)
