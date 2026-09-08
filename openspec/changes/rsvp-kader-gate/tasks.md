## 1. Helfer

- [x] 1.1 `internal/trainings/handler.go` — `isKaderParticipant(ctx, kaderID, memberID)
      (bool, error)` mit den drei Zweigen `kader_members`, `kader_extended_members`,
      `kader_trainers` gegen `kader_id`. Ein Query, kein N+1
- [x] 1.2 Prüfen, ob `ListSessions`' `am_i_participant` denselben Helfer nutzen kann, ohne
      die Listen-Query in eine Schleife zu zerlegen. Wenn nicht: Kommentar an **beiden**
      Stellen, der auf die jeweils andere zeigt (`design.md — Entscheidung 1`)
      → nicht möglich ohne N+1 über die Seite; Kommentare stehen an allen drei Fundstellen
      (`isKaderParticipant`, `ListSessions`, `GetSession`), abgesichert durch
      `TestRespond_AnzeigeUndAntwortrechtStimmenUeberein`

## 2. Gate in Respond

- [x] 2.1 Selbst-Zweig: nach dem Auflösen von `ownMemberID` die Zugehörigkeit gegen
      `training_sessions.kader_id` prüfen → 403
- [x] 2.2 Fremd-Zweig: dieselbe Prüfung für das **Ziel**-Mitglied, zusätzlich zur
      bestehenden Eltern-/Staff-Prüfung (`design.md — Entscheidung 3`)
- [x] 2.3 Reihenfolge: die Kaderprüfung läuft **vor** Absence-Lock, Serien-Abmeldung und
      RSVP-Cutoff — ein Fremder soll kein 403 „für diese Terminserie abgemeldet" bekommen,
      das über die Existenz einer Abmeldung Auskunft gibt
- [x] 2.4 Kein `Broadcast`-Wechsel, keine Signaturänderung — die Route bleibt sonst, wie
      sie ist

## 3. Sichtbarkeit angleichen (Fund aus 1.2, `design.md — Entscheidung 4`)

- [x] 3.1 `ListSessions`, Sichtbarkeitsfilter: der `kader_trainers`-Zweig gilt unabhängig
      von der Vereinsfunktion `trainer` — sonst darf ein eingetragener Kader-Trainer den
      Termin verwalten und beantworten, sieht ihn aber nicht
- [x] 3.2 Spec-Delta um das Requirement „Termin-Sichtbarkeit folgt der Kader-Eintragung,
      nicht der Vereinsfunktion" ergänzt; die Gegenbehauptung im Risiken-Absatz von
      `design.md` korrigiert

## 4. Tests

- [x] 4.1 Alle Tests aus `proposal.md — Test-Anforderungen`
- [x] 4.2 `TestRespond_CreatesRSVP` und `TestRespond_UpdatesExistingRSVP` bekommen die
      Kader-Zugehörigkeit, die sie fachlich meinen; ihre Assertions bleiben unverändert
      (Helfer `joinKaderOfSession`)
- [x] 4.3 Bestehende `am_i_participant`-Tests müssen grün bleiben — sie sind der
      Regressionsschutz für Task 1.2
- [x] 4.4 `TestListSessions_KaderTrainerOhneVereinsfunktion` als benannter
      Regressionsschutz für Task 3.1
- [x] 4.5 `make test` (Go: alle Pakete grün; Vitest: 138 Dateien / 1064 Tests grün),
      `golangci-lint run ./internal/trainings/...` (0 issues),
      `openspec validate rsvp-kader-gate --strict` (valid)
