# api-routes Specification

## Purpose

Diese Spezifikation beschreibt die Capability `api-routes`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Teams-Endpoint ist rollenabhängig

`GET /api/teams` SHALL dasselbe Ergebnis liefern wie bisher, aber abhängig von der Rolle des aufrufenden Users:
- Users mit Vereinsfunktion `vorstand` oder Rolle `admin` erhalten alle Teams inklusive inaktiver, ohne Kader-Filterung.
- Alle anderen authentifizierten Users erhalten nur Teams mit aktivem Kader, gefiltert nach ihrer Zugehörigkeit (Trainer: nur eigene Teams; andere: alle Teams mit aktivem Kader).

#### Scenario: Vorstand ruft Teams ab
- **WHEN** ein User mit Vereinsfunktion `vorstand` `GET /api/teams` aufruft
- **THEN** liefert der Endpoint alle Teams inklusive inaktiver zurück

#### Scenario: Trainer ruft Teams ab
- **WHEN** ein User mit Vereinsfunktion `trainer` `GET /api/teams` aufruft
- **THEN** liefert der Endpoint nur Teams zurück, in deren Kader der Trainer der aktiven Saison eingetragen ist

#### Scenario: Spieler ruft Teams ab
- **WHEN** ein User mit Rolle `spieler` `GET /api/teams` aufruft
- **THEN** liefert der Endpoint alle Teams zurück, die einen aktiven Kader haben

### Requirement: Keine /admin-Präfix-Routen

Die API SHALL keine Routen unter `/api/` mehr exponieren. Alle bisherigen `/api/*`-Routen MÜSSEN unter ihrem kanonischen Ressource-Pfad `/api/{ressource}` erreichbar sein. Zugriffskontrolle erfolgt ausschließlich über Middleware-Gruppen.

#### Scenario: Vorstand ruft Vereinskonfiguration ab
- **WHEN** ein User mit Vereinsfunktion `vorstand` `GET /api/club` aufruft
- **THEN** liefert der Endpoint die Vereinskonfiguration zurück (bisher: `GET /api/club`)

#### Scenario: Unautorisierter Zugriff auf privilegierte Route
- **WHEN** ein User ohne ausreichende Berechtigung eine privilegierte Route aufruft (z.B. `GET /api/club`)
- **THEN** antwortet der Server mit HTTP 403
