# kalender-spiel-rsvp Specification

## Purpose

Diese Spezifikation beschreibt die Capability `kalender-spiel-rsvp`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Spiel-Kachel zeigt RSVP-Zähler

Spiel-Kacheln (heim, auswärts, generisch) im Monatskalender SHALL `confirmed_count` (Zusagen) und `declined_count` (Absagen) in der Uhrzeitzeile anzeigen, sobald die Kachelbreite ≥ 80 px beträgt.

#### Scenario: RSVP-Zähler sichtbar bei ausreichender Kachelbreite
- **WHEN** eine Spiel-Kachel im Kalender ≥ 80 px breit ist
- **THEN** zeigt die Uhrzeitzeile ein grünes ✓-Icon mit `confirmed_count` und ein rotes ✗-Icon mit `declined_count`

#### Scenario: RSVP-Zähler versteckt auf schmalen Kacheln
- **WHEN** eine Spiel-Kachel < 80 px breit ist (z. B. Mobile)
- **THEN** sind Zu-/Absage-Zähler nicht sichtbar; nur die Uhrzeit bleibt

#### Scenario: rsvp_opt_out-Spiel
- **WHEN** ein Spiel `rsvp_opt_out = 1` hat
- **THEN** werden die vom Backend berechneten Zähler unverändert angezeigt (kein Sonderfall im Frontend)

### Requirement: Dienst-Punkt in Teamname-Zeile

Der Dienst-Punkt einer Spiel-Kachel SHALL rechts in der Teamname-Zeile (erste Zeile) erscheinen, nicht in der Uhrzeitzeile.

#### Scenario: Dienst-Punkt bei vorhandenen Slots
- **WHEN** ein Spiel `slot_count > 0` hat und die Kachel ≥ 80 px breit ist
- **THEN** erscheint ein farbiger Punkt am rechten Rand der Teamname-Zeile (grün/gelb/rot nach Füllgrad)

#### Scenario: Kein Dienst-Punkt ohne Slots
- **WHEN** ein Spiel `slot_count = 0` hat
- **THEN** ist kein Dienst-Punkt sichtbar

#### Scenario: Dienst-Punkt auf schmalen Kacheln
- **WHEN** die Kachel < 80 px breit ist
- **THEN** ist der Dienst-Punkt nicht sichtbar
