## 1. Backend — Cutoff 18h → 2h

- [ ] 1.1 `internal/games/handler.go`: Konstante `GameRSVPCutoff` von `18 * time.Hour` auf `2 * time.Hour` reduzieren
- [ ] 1.2 Fehler-Message in Game-Zweig von `writeRSVPLocked` auf „Spiel kann nur bis 2 Stunden vor Beginn umgesagt werden." anpassen
- [ ] 1.3 Bestehende Cutoff-Tests in `internal/games/cutoff_test.go` und `handler_test.go` von 18h/12h/48h auf 2h/30min/3h umstellen; Assertions auf `rsvp_locks_at = start − 2h`

## 2. Backend — am_i_participant Feld

- [ ] 2.1 `gameListItem` und `gameDetail` in `internal/games/handler.go` um `AmIParticipant bool `json:"am_i_participant"`` erweitern
- [ ] 2.2 In `ListMyGames` (und `ListGames`, falls es dort auch relevant ist) das Feld aus `inRegularKader==1 || inExtendedKader==1 || inTrainerKader==1` befüllen
- [ ] 2.3 In `GetGame` (Detail) analog befüllen — dort ggf. drei EXISTS-Subqueries ergänzen falls noch nicht vorhanden
- [ ] 2.4 `sessionListItem` und `sessionDetail` in `internal/trainings/handler.go` um `AmIParticipant bool `json:"am_i_participant"`` erweitern
- [ ] 2.5 In `ListSessions`-SQL drei EXISTS-Subqueries für den Aufrufer ergänzen (regular/extended/trainer-Kader), Ergebnis in Scan aufnehmen, Feld befüllen
- [ ] 2.6 In `GetSession` analog
- [ ] 2.7 Für Eltern: unverändert (Kind-Zeilen kommen bereits kader-basiert via `attachChildrenRSVPToGames`/`…ToSessions`)

## 3. Backend — Tests

- [ ] 3.1 `internal/games/handler_test.go`: Neuer Test „Spieler im Kader mit default=none sieht `am_i_participant=true` und `my_rsvp=null`"
- [ ] 3.2 `internal/games/handler_test.go`: Neuer Test „Nicht-Kader-Nutzer sieht `am_i_participant=false`"
- [ ] 3.3 `internal/trainings/handler_test.go`: Analoge zwei Tests
- [ ] 3.4 Cutoff-Tests aus Task 1.3 grün

## 4. Frontend — Sichtbarkeit an am_i_participant koppeln

- [ ] 4.1 `web/src/pages/TerminePage.tsx`: Typen `TrainingSession` und `GameSummary` um `am_i_participant: boolean` erweitern
- [ ] 4.2 Outer condition und `showOwn` an beiden Stellen (Training ~520/522, Game ~626/628): `s.am_i_participant` statt `s.my_rsvp !== null`
- [ ] 4.3 `web/src/pages/TermineDetailPage.tsx` analog prüfen und anpassen
- [ ] 4.4 Vitest-Suite in `web/src/pages/TerminePage.test.tsx` (und ggf. Detail-Test) grün halten / erweitern um „Buttons sichtbar bei am_i_participant=true und my_rsvp=null"

## 5. Frontend — Cutoff-Texte

- [ ] 5.1 `grep -rn "18 Stunden\|18h" web/src` — wenn Text vorkommt, für Games auf „2 Stunden" umstellen
- [ ] 5.2 `RsvpLockNotice`-Text prüfen (nutzt evtl. dynamisch das API-`locks_at`, dann keine Textänderung nötig)

## 6. Verifikation

- [ ] 6.1 `/verify-change` durchlaufen (go vet/test/lint, pnpm build/test/lint, openspec validate)
- [ ] 6.2 `openspec validate rsvp-teilnehmer-sichtbarkeit --strict`
- [ ] 6.3 Manuell mit lokalem Backend (falls im Zeitrahmen): `/termine` als Spieler-Account öffnen und prüfen dass Buttons erscheinen
