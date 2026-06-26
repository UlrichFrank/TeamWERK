# rsvp-event-config Specification

## Purpose

Diese Spezifikation beschreibt die Capability `rsvp-event-config`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: rsvp-opt-out-flag
Jeder Termin (training_session, game) MUST ein `rsvp_opt_out`-Flag besitzen (INTEGER 0/1).
Bei `rsvp_opt_out = 1` gilt ein **regulärer Kader-Spieler** (Eintrag in `kader_members` für das
Team und die Saison des Termins) ohne Response-Eintrag als "confirmed".
Das Flag MUST beim Anlegen einer Session von der zugehörigen training_series kopiert werden.
Das Flag MUST nach dem Anlegen für berechtigte Nutzer änderbar sein (siehe Requirement
`rsvp-config-edit-ui`).

Erweiterte Kader-Mitglieder (`kader_extended_members`) sind von der Opt-Out-Implicit-Confirmed-
Logik ausgeschlossen: sie müssen explizit zusagen.

#### Scenario: Regulärer Kader-Spieler ohne Eintrag bei Opt-Out-Termin
- **WHEN** ein training_session oder game hat `rsvp_opt_out = 1` und ein regulärer Kader-Spieler hat keinen Eintrag in der Response-Tabelle
- **THEN** gibt `my_rsvp` den Wert `"confirmed"` zurück

#### Scenario: Erweiterter Kader-Spieler bei Opt-Out-Termin
- **WHEN** ein game hat `rsvp_opt_out = 1` und ein Spieler ist nur in `kader_extended_members` (nicht in `kader_members`) und hat keine Response
- **THEN** bleibt sein `rsvp_status` `null` bzw. seine `my_rsvp` `null` — er muss aktiv zusagen

#### Scenario: confirmed_count bei Opt-Out
- **WHEN** ein Termin hat `rsvp_opt_out = 1`
- **THEN** ist `confirmed_count` gleich der Anzahl explizit bestätigter Einträge plus der Anzahl regulärer Kader-Mitglieder ohne Response-Eintrag
- **THEN** wird `confirmed_count` einheitlich von ALLEN Endpoints (`ListGames`, `ListMyGames`, `GetGame`, `GetSession`) so berechnet — keine endpoint-spezifische Speziallogik

#### Scenario: declined_count und maybe_count unverändert
- **WHEN** ein Termin hat `rsvp_opt_out = 1`
- **THEN** zählen `declined_count` und `maybe_count` **nur** explizite Responses — Opt-Out bedeutet implizite Zusage, nie implizite Absage

#### Scenario: Zusagen-Button vorausgewählt
- **WHEN** `my_rsvp = "confirmed"` (implizit oder explizit)
- **THEN** zeigt die TerminePage den Zusagen-Button als aktiv/ausgewählt

#### Scenario: Session erbt Flag von Serie
- **WHEN** eine neue training_session aus einer training_series erstellt wird
- **THEN** werden `rsvp_opt_out` und `rsvp_require_reason` von der Serie kopiert

#### Scenario: Flag nach Anlegen änderbar
- **WHEN** ein berechtigter Nutzer (admin, trainer, sportliche_leitung, vorstand) ein bestehendes game oder eine bestehende training_session bearbeitet
- **THEN** KÖNNEN `rsvp_opt_out` und `rsvp_require_reason` geändert werden; der neue Wert wird persistiert und beeinflusst alle künftigen Response-Auswertungen (z.B. `confirmed_count`, `my_rsvp`-Default)


### Requirement: rsvp-participants-opt-out
Der Endpoint `GET /api/games/{id}/participants` MUST bei `rsvp_opt_out = 1` für reguläre
Kader-Mitglieder (`is_extended = 0`) ohne expliziten Response-Eintrag das Feld `rsvp_status`
auf `"confirmed"` setzen, damit Frontend-Konsumenten ihren impliziten Zusage-Status sehen,
ohne selbst rechnen zu müssen.

#### Scenario: Kader-Member ohne Response bei Opt-Out
- **WHEN** ein game hat `rsvp_opt_out = 1` und ein Kader-Member hat keinen Eintrag in `game_responses`
- **THEN** liefert `GetParticipants` für diesen Member `rsvp_status = "confirmed"` und `is_extended = false`

#### Scenario: Kader-Member mit expliziter Absage bei Opt-Out
- **WHEN** ein game hat `rsvp_opt_out = 1` und ein Kader-Member hat `game_responses.status='declined'`
- **THEN** liefert `GetParticipants` für diesen Member `rsvp_status = "declined"` — die explizite Absage gewinnt gegenüber dem impliziten Opt-Out

#### Scenario: Extended-Member ohne Response bei Opt-Out
- **WHEN** ein game hat `rsvp_opt_out = 1` und ein Extended-Kader-Member hat keinen Eintrag in `game_responses`
- **THEN** liefert `GetParticipants` für diesen Member `rsvp_status = null` und `is_extended = true`

#### Scenario: Opt-Out=0 — kein Impliziter-Status-Override
- **WHEN** ein game hat `rsvp_opt_out = 0`
- **THEN** zeigt `GetParticipants` `rsvp_status` exakt wie in `game_responses` (oder `null` falls kein Eintrag)

### Requirement: rsvp-getgame-counts
Der Endpoint `GET /api/games/{id}` MUST `confirmed_count`, `declined_count` und `maybe_count`
im Game-Response-Objekt liefern. Die Werte folgen den selben Regeln wie in
`confirmed_count bei Opt-Out` und `declined_count und maybe_count unverändert`.

#### Scenario: Counts im Game-Response
- **WHEN** ein Client `GET /api/games/{id}` aufruft
- **THEN** enthält das `game`-Objekt im Response die Felder `confirmed_count`, `declined_count`, `maybe_count` (alle Integer)
