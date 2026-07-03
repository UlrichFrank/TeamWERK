## ADDED Requirements

### Requirement: Trainer können auf der Termin-Übersicht `/termine` zu-/absagen

Die Termin-Übersicht (`web/src/pages/TerminePage.tsx`, Kartenliste) SHALL die RSVP-Buttons („Zusagen"/„Vielleicht"/„Absagen") einem Trainer für einen Termin anzeigen, wenn er über `kader_trainers` Trainer des jeweiligen Teams ist. Maßgeblich ist ein pro-Termin-Teilnahmesignal: die Buttons erscheinen genau dann, wenn `my_rsvp` für den Termin nicht `null` ist. Die frühere pauschale Ausblendung anhand der `manage_trainings`-Capability (`!isTrainer`) entfällt.

Ein Klick löst `POST /api/training-sessions/{id}/respond` bzw. `POST /api/games/{id}/respond` für das eigene Mitglied aus. Trainer werden dabei weiterhin **nicht** in die Header-Zähler (`confirmed_count`/`declined_count`/`maybe_count`) einbezogen. Die „Zusagen"-Aktion für Trainer SHALL nicht an `rsvp_default_players` (Spieler-Voreinstellung) gekoppelt sein.

#### Scenario: Trainer des Teams sieht Buttons und kann zusagen
- **WHEN** ein Trainer auf `/termine` einen Termin seines Teams sieht (ohne eigene Response, also `my_rsvp='confirmed'` als Trainer-Default)
- **THEN** werden die RSVP-Buttons angezeigt
- **AND** ein Klick auf „Absagen" sendet `POST …/respond` mit `status='declined'`

#### Scenario: Vorstand auf fremdem Team-Termin sieht keine Buttons
- **WHEN** ein Vorstand (kein Trainer/Spieler/Erweiterter dieses Teams) auf `/termine` einen fremden Team-Termin sieht (`my_rsvp=null`)
- **THEN** werden für diesen Termin keine RSVP-Buttons angezeigt

#### Scenario: Trainer bleibt aus Header-Zählern ausgeschlossen
- **WHEN** ein Trainer über `/termine` einen Termin seines Teams zusagt
- **THEN** ändert sich `confirmed_count` des Termins nicht durch die Trainer-Antwort
