# Tasks

## 1. Foundation (`internal/notify`)

- [x] 1.1 `notify.KaderTrainers(db, kaderIDs ...int) []int` (Nutzerkonten aus `kader_trainers` → `members.user_id`, dedupliziert, leere Eingabe → nil, Query-Fehler per `slog.Error`); verifiziert durch Test mit zwei Kadern, gemeinsamem Trainer (einmal) und Trainer ohne `user_id` (fehlt); `internal/arch` bleibt grün
- [x] 1.2 `notify.RSVPChangeBody(member, subject, when, from, to, reason)` + Status-Labels (Zusage/Absage/Vielleicht), Grund über `TrimReason`, leerer Grund → kein „Grund:"-Satz; verifiziert durch Tabellentest

## 2. Spiele (`internal/games`)

- [x] 2.1 In `RespondToGame` den Absence-Lookup auf `absence_id, status` erweitern und vorherigen Status (oder „keiner") festhalten; verifiziert durch unveränderte bestehende RSVP-Tests
- [x] 2.2 Nach dem Upsert: Bedingungen aus der Spec prüfen (vorheriger Status ≠ neuer, `0 < Beginn − h.now() ≤ 7×24h`, Ziel-Mitglied nicht Trainer der Spiel-Kader in der Saison des Spiels), Empfänger = `KaderTrainers` minus `claims.UserID`, `notify.SendAsync(..., "operativ", ...)` mit URL `/termine?focus=game-<id>`; Fenster als benannte Konstante
- [x] 2.3 Tests (Datei-DB, `notify.Send` abgefangen, `h.now` fixiert): `TestRespondToGame_AenderungImFenster_BenachrichtigtTrainer`, `…_ErsteAntwort_KeineMeldung`, `…_GleicherStatus_KeineMeldung`, `…_AchtTageVorher_KeineMeldung`, `…_GenauSiebenTage_Meldung`, `…_TrainerEigeneAntwort_KeineMeldung`, `…_TrainerFuerSpieler_AusloeserNichtEmpfaenger`, `…_Elternteil_NameDesKindes`, `…_AbsenceLock_KeineMeldung` (403); alle mit `go test ./internal/games/...` grün

## 3. Trainings (`internal/trainings`)

- [x] 3.1 In `Respond` den Absence-Lookup auf `absence_id, status` erweitern; verifiziert durch unveränderte bestehende RSVP-Tests
- [x] 3.2 Nach dem Upsert dieselbe Logik mit Kader = `training_sessions.kader_id`, URL `/termine?focus=training-<id>`, Betreff aus Kader-/Serienname
- [x] 3.3 Tests analog 2.3 (`TestRespond_…`), zusätzlich `TestRespond_Uebungsgruppe_BenachrichtigtTrainer` und `TestRespond_SeriesUnavailable_KeineMeldung`; `go test ./internal/trainings/...` grün

## 4. Frontend

- [x] 4.1 Beschreibung der Kategorie `operativ` in `web/src/components/profile/ProfileMiscTab.tsx` ändern auf „Meldungen zu deinen Vereinsaufgaben, z. B. Anwesenheiten nachtragen, Spielberichte freigeben oder kurzfristige Umentscheidungen deiner Spieler.“ (Label „Vereinsaufgaben“ bleibt); verifiziert durch `pnpm -C web test` und Sichtprüfung im Profil-Tab „Sonstiges“

## 5. Abschluss

- [x] 5.1 Gotcha-Absatz in `docs/agent/06-gotchas.md` (Kategorie `operativ`, Auslöser gefiltert, Abwesenheiten bewusst außen vor) ergänzen
- [x] 5.2 `/verify-change` bzw. `make test lint` und `openspec validate rsvp-aenderung-trainer-push` grün
