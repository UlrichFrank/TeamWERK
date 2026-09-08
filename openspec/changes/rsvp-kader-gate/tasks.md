## 1. Helfer

- [ ] 1.1 `internal/trainings/handler.go` — `isKaderParticipant(ctx, kaderID, memberID)
      (bool, error)` mit den drei Zweigen `kader_members`, `kader_extended_members`,
      `kader_trainers` gegen `kader_id`. Ein Query, kein N+1
- [ ] 1.2 Prüfen, ob `ListSessions`' `am_i_participant` denselben Helfer nutzen kann, ohne
      die Listen-Query in eine Schleife zu zerlegen. Wenn nicht: Kommentar an **beiden**
      Stellen, der auf die jeweils andere zeigt (`design.md — Entscheidung 1`)

## 2. Gate in Respond

- [ ] 2.1 Selbst-Zweig: nach dem Auflösen von `ownMemberID` die Zugehörigkeit gegen
      `training_sessions.kader_id` prüfen → 403
- [ ] 2.2 Fremd-Zweig: dieselbe Prüfung für das **Ziel**-Mitglied, zusätzlich zur
      bestehenden Eltern-/Staff-Prüfung (`design.md — Entscheidung 3`)
- [ ] 2.3 Reihenfolge: die Kaderprüfung läuft **vor** Absence-Lock, Serien-Abmeldung und
      RSVP-Cutoff — ein Fremder soll kein 403 „für diese Terminserie abgemeldet" bekommen,
      das über die Existenz einer Abmeldung Auskunft gibt
- [ ] 2.4 Kein `Broadcast`-Wechsel, keine Signaturänderung — die Route bleibt sonst, wie
      sie ist

## 3. Tests

- [ ] 3.1 Alle Tests aus `proposal.md — Test-Anforderungen`
- [ ] 3.2 `TestRespond_CreatesRSVP` und `TestRespond_UpdatesExistingRSVP` bekommen die
      Kader-Zugehörigkeit, die sie fachlich meinen; ihre Assertions bleiben unverändert
- [ ] 3.3 Bestehende `am_i_participant`-Tests müssen grün bleiben — sie sind der
      Regressionsschutz für Task 1.2
- [ ] 3.4 `make test`, `golangci-lint`, `openspec validate rsvp-kader-gate --strict`
