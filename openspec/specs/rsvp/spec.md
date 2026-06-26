# rsvp Specification

## Purpose

Diese Spezifikation beschreibt die Capability `rsvp`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: rsvp-modal-behavior
Das RSVP-Modal für Absage/Vielleicht MUSS nur erscheinen wenn der Termin
`rsvp_require_reason = 1` hat. Bei `rsvp_require_reason = 0` wird die RSVP
direkt ohne Modal gesendet.

#### Scenario: Kein Modal bei rsvp_require_reason = 0
- **WHEN** `rsvp_require_reason = 0` und Spieler klickt Absagen oder Vielleicht
- **THEN** wird POST /api/.../respond sofort aufgerufen mit leerem reason-Feld

#### Scenario: Modal bei rsvp_require_reason = 1
- **WHEN** `rsvp_require_reason = 1` und Spieler klickt Absagen oder Vielleicht
- **THEN** öffnet sich das Begründungs-Modal vor dem API-Call
