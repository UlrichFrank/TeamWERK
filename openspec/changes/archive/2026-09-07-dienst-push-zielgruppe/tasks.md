## 1. Empfängerbestimmung an die Sichtbarkeitsregel angleichen

- [x] 1.1 `internal/duties/handler.go`: `eligibleDutyUsers` → `eligibleDutyRecipients(ctx, teamIDs []int, audiences []string)`; Team-Quelle über `player_memberships` **und** `trainer_memberships` **und** Eltern via `family_links`, jeweils mit `seasons.is_active = 1`
- [x] 1.2 Audience-Filter ergänzen (Vereinsfunktion des Empfängers; `eltern` team-gescoped wie im Board), NULL/leer = keine Einschränkung
- [x] 1.3 `CreateSlot`: wirksame Zielgruppe auflösen (`audiences` des Slots, sonst Vorbelegung des Diensttyps) und an die Empfängerbestimmung übergeben

## 2. Tests

- [x] 2.1 `TestCreateSlot_ZielgruppeEltern_BenachrichtigtNurEltern`
- [x] 2.2 `TestCreateSlot_ZielgruppeTrainer_BenachrichtigtTrainerDesKaders`
- [x] 2.3 `TestCreateSlot_ZielgruppeAusDiensttyp_WirdAngewendet`
- [x] 2.4 `TestCreateSlot_OhneZielgruppe_BenachrichtigtDenGanzenKader`
- [x] 2.5 `TestCreateSlot_AltePlayerMembership_WirdNichtBenachrichtigt`
- [x] 2.6 Bestandstests aus `createslot_multiteam_test.go` bleiben grün

## 3. Abschluss

- [x] 3.1 `make test` + `golangci-lint` grün
- [x] 3.2 `openspec validate dienst-push-zielgruppe --strict`
