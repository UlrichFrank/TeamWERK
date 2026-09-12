## Purpose

Hintergrundarbeit (Benachrichtigungen, Backfills) darf den Serverprozess nie beenden.

## ADDED Requirements

### Requirement: Panics in Hintergrundarbeit werden abgefangen

Jede vom Domänencode gestartete Goroutine MUST über einen Wrapper laufen, der Panics abfängt, protokolliert und als Metrik zählt. Ein Architektur-Test MUST nackte Goroutine-Starts außerhalb einer begründeten Allowlist ablehnen. Nach der HTTP-Antwort gestartete Arbeit MUST einen vom Request unabhängigen Context verwenden.

#### Scenario: Panic im Versand beendet den Prozess nicht
- **WHEN** eine Benachrichtigungs-Goroutine panict
- **THEN** läuft der Server weiter, die Metrik `teamwerk_background_panics_total` steigt um eins und eine Log-Zeile nennt den Job

#### Scenario: Passwort-Reset-Token überlebt die Antwort
- **WHEN** `POST /api/auth/request-reset` mit 204 antwortet
- **THEN** existiert der Reset-Token anschließend in der Datenbank
