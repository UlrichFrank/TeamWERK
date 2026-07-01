## 1. Auth-Helper

- [x] 1.1 In `internal/auth/tokens.go`: Methode `func (c *Claims) CanOverrideRSVPCutoff() bool` (admin || vorstand || trainer || sportliche_leitung).
- [x] 1.2 Unit-Test in `internal/auth/claims_test.go`: alle 6 Rollen-Kombinationen (admin / vorstand / trainer / sL / kassierer / spieler) + jeweils mit/ohne weitere Funktionen.

## 2. Backend — Trainings

- [x] 2.1 In `internal/trainings/handler.go` Konstante `const TrainingRSVPCutoff = 2 * time.Hour` und Helper `trainingLocksAt(dateISO, startTimeHHMM string) (time.Time, error)` mit `time.LoadLocation("Europe/Berlin")`.
- [x] 2.2 `Handler`-Struct erweitern um `now func() time.Time` (Default `time.Now` in `NewHandler`); für Tests injizierbar.
- [x] 2.3 In `Respond` (Handler) nach dem Absence-Check eine Cutoff-Prüfung einfügen: wenn `!claims.CanOverrideRSVPCutoff() && h.now().After(locksAt)`, 422 mit `{error:"rsvp_locked", message:"Training kann nur bis 2 Stunden vor Beginn umgesagt werden.", locks_at:<UTC RFC3339>}`.
- [x] 2.4 In `GET /api/training-sessions` und `GET /api/training-sessions/{id}` `rsvp_locks_at` ins JSON aufnehmen (RFC3339 UTC); Berechnung im Handler aus `date` + `start_time` mit demselben Helper.

## 3. Backend — Games

- [x] 3.1 In `internal/games/handler.go` Konstante `const GameRSVPCutoff = 18 * time.Hour` und Helper `gameLocksAt(dateISO, timeHHMM string) (time.Time, error)` mit Europe/Berlin.
- [x] 3.2 `Handler`-Struct erweitern um `now func() time.Time` (Default `time.Now`).
- [x] 3.3 In `RespondToGame` Cutoff-Prüfung nach Absence-Check: bei Sperre 422 mit dem game-spezifischen Fehlertext und `locks_at`.
- [x] 3.4 `rsvp_locks_at` in den Responses von `GET /api/games`, `GET /api/games/my` und `GET /api/games/{id}` ergänzen.

## 4. Backend — Tests

- [x] 4.1 Test-Helper `testutil.FrozenTime` o. ä., um `h.now` in Handler-Tests zu setzen (oder Test setzt direkt das Feld). _(via `(*Handler).SetNow`)_
- [x] 4.2 `internal/trainings/handler_test.go`: pro Cutoff-Scenario aus `specs/training-rsvp/spec.md` einen Test (Spieler vor Cutoff → 204, Spieler nach Cutoff → 422, Spieler erstmals nach Cutoff → 422, Eltern nach Cutoff → 422, Trainer nach Cutoff → 204, sportliche_leitung nach Cutoff → 204, Vorstand nach Beginn → 204, Admin nach Cutoff → 204, Kassierer nach Cutoff → 422, Absence-Lock 403 vor Cutoff-Check).
- [x] 4.3 `internal/trainings/handler_test.go`: DST-Test (Session in Sommer- und Winterzeit → `rsvp_locks_at` korrekt in UTC).
- [x] 4.4 `internal/trainings/handler_test.go`: Listing-Responses enthalten `rsvp_locks_at` (Liste + Detail).
- [x] 4.5 `internal/games/handler_test.go`: analoge Cutoff-Scenarios für `RespondToGame`. _(neue Datei `internal/games/cutoff_test.go`)_
- [x] 4.6 `internal/games/handler_test.go`: Listing-Responses (`/api/games`, `/api/games/my`, `/api/games/{id}`) enthalten `rsvp_locks_at`.

## 5. Frontend — gemeinsame UI-Komponente

- [x] 5.1 In `web/src/components/` Komponente `RsvpButtons` (falls noch nicht vorhanden zentralisiert) bzw. die bestehende Stelle in `Termine.tsx` so anpassen, dass sie ein Prop `rsvpLocksAt: string | null` und `canOverride: boolean` akzeptiert. _(inline gelöst in `TerminePage.tsx` über lokale `cutoffLocked`-Variable und `RsvpLockNotice`)_
- [x] 5.2 Vor Cutoff: subtiler Hinweis-Text „Bis HH:MM Uhr änderbar" unter den Buttons (Format `Intl.DateTimeFormat('de-DE', { hour:'2-digit', minute:'2-digit' })`).
- [x] 5.3 Nach Cutoff (und `!canOverride`): RSVP-Buttons `disabled`, Hinweis-Text „Änderungen nur noch beim Trainer möglich" in `text-brand-text-muted`.
- [x] 5.4 Wenn `canOverride === true` (admin/vorstand/trainer/sL): keinerlei Sperre, kein Hinweis-Text. _(`!canOverrideRsvpCutoff &&`-Guard vor Notice + Button-Disable)_

## 6. Frontend — Integration

- [x] 6.1 `web/src/pages/TerminePage.tsx` (Liste): `rsvp_locks_at` aus API-Response lesen, `canOverride` aus `hasCapability('manage_games')` (deckt admin/vorstand/trainer/sportliche_leitung).
- [x] 6.2 Game-Detail-Page: nicht erforderlich — `TermineDetailPage.tsx` ist die Trainer-Sicht (Anwesenheit/Lineup), enthält keine Spieler-RSVP-Buttons.
- [x] 6.3 Training-Detail-Page: dito, siehe 6.2 — Spieler-RSVP wird ausschließlich in der Liste auf `/termine` abgegeben.
- [x] 6.4 Fehler-Handling: 422-Response mit `error="rsvp_locked"` → `extractRsvpError` zeigt `message` aus dem Body statt generischem „Fehler beim Speichern".

## 7. Verification

- [x] 7.1 `make test` grün (inkl. neue Tests aus 4.x).
- [x] 7.2 `make lint` grün.
- [x] 7.3 `pnpm build` und `pnpm lint` grün (0 errors, bestehende 52 Warnings unverändert).
- [ ] 7.4 Lokal verifizieren: zwei Trainings/Spiele anlegen — eines >2h/>18h in der Zukunft, eines <2h/<18h —, als Spieler einloggen, prüfen dass Buttons im erwarteten Zustand sind und 422-Pfad eine sinnvolle Meldung anzeigt. _(manuelle Verifikation — vom Benutzer durchzuführen)_
- [x] 7.5 `openspec validate rsvp-cutoff --strict` grün.
