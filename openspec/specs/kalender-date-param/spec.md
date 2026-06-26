# kalender-date-param Specification

## Purpose

Diese Spezifikation beschreibt die Capability `kalender-date-param`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: URL-Param-Navigation
KalenderPage akzeptiert einen optionalen Query-Parameter `?date=YYYY-MM-DD`. Wenn gesetzt, initialisiert die Page `year` und `month` aus dem Datum statt aus `new Date()`.

#### Scenario: Gültiger date-Param
- **WHEN** die KalenderPage mit `?date=2026-06-14` aufgerufen wird
- **THEN** zeigt der Kalender Juni 2026

#### Scenario: Ungültiger date-Param
- **WHEN** die KalenderPage mit `?date=foobar` aufgerufen wird
- **THEN** fällt die Page auf den aktuellen Monat zurück (kein Fehler)

#### Scenario: Kein date-Param
- **WHEN** die KalenderPage ohne Query-Param aufgerufen wird
- **THEN** zeigt der Kalender den aktuellen Monat (unverändertes Verhalten)

### Requirement: Dashboard-Links zu Events
In DashboardPage navigieren Links in der „Nächste Events"-Sektion zu `/kalender?date=YYYY-MM-DD` (Datum des jeweiligen Events), nicht zum bisherigen `g.link`-Wert.

#### Scenario: Klick auf Event im Dashboard
- **WHEN** der Nutzer auf ein Event in „Nächste Events" klickt
- **THEN** öffnet sich der Kalender mit dem Monat des Events
