## 1. Backend — Trainer-Einträge beim Speichern überspringen

- [x] 1.1 `internal/trainings/handler.go` `SaveAttendances`: bei erkanntem Trainer-only-Member `continue` statt `http.Error(…, 400)`; Kommentar aktualisieren
- [x] 1.2 `internal/games/handler.go` `SaveAttendances`: dito (`continue` statt 400)
- [x] 1.3 Grenzfall festlegen (design.md): trainer-only/leeres Paket → 204 mit 0 Writes (No-op) — durch `continue` automatisch erfüllt

## 2. Backend — Regressionstests

- [x] 2.1 `internal/trainings/handler_test.go` `TestSaveAttendances_TrainerInBatch_Skipped`: Trainer **und** Spieler im Paket → 204; Spieler persistiert, keine `training_attendances`-Zeile für den Trainer. Bestandstest `TestSaveAttendances_TrainerRejected` → `_TrainerSkipped` (204) angepasst.
- [x] 2.2 `internal/games/attendance_test.go` `TestSaveGameAttendances_TrainerInBatch_Skipped`: analog → 204; Spieler persistiert, keine `game_attendances`-Zeile. Bestandstest `_TrainerRejected` → `_TrainerSkipped` (204) angepasst.
- [x] 2.3 Fehlerfall weiter abgedeckt (Bestandstests grün): Nicht-Trainer/Fremd-Team → 403; Zukunfts-Spiel → 422. 187 Tests grün in beiden Packages.

## 3. Frontend — Trainer aus dem Speicher-Paket filtern

- [x] 3.1 `web/src/pages/TermineDetailPage.tsx` `toggleAttendance`: `ids` aus `attendances.filter(a => !a.is_trainer)` bzw. `participants.filter(p => !p.is_trainer)` bilden
- [~] 3.2 Manuell verifizieren: Toggle eines Spielers in einem Team mit Trainer → Haken bleibt, kein Fehler-Banner. (Automatisiert abgedeckt: Build + 541 Web-Tests grün; Browser-Smoke-Test durch Nutzer empfohlen.)

## 4. Abschluss

- [x] 4.1 Verifikation grün: go test games+trainings (187), go vet, arch-test (4), web build, web test (541), web lint (exit 0)
- [x] 4.2 Commit(s) je Task, Conventional Commits (`fix(games,trainings): …`, `fix(termine): …`)
- [x] 4.3 Proposal archiviert
