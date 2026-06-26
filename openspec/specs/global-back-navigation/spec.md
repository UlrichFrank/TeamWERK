# global-back-navigation Specification

## Purpose

Diese Spezifikation beschreibt die Capability `global-back-navigation`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Globaler Zurück-Button in AppShell
AppShell SHALL einen generischen „← Zurück"-Button anzeigen, sobald der React-Router-interne History-Index größer als 0 ist. Der Button soll auf allen eingeloggten Seiten einheitlich positioniert sein und `navigate(-1)` auslösen.

#### Scenario: Button erscheint nach erster Navigation
- **WHEN** der Nutzer von einer Seite auf eine andere navigiert (egal ob via Sidebar, Link oder programmatisch)
- **THEN** erscheint der „← Zurück"-Button sichtbar in AppShell

#### Scenario: Button fehlt beim ersten Seitenaufruf
- **WHEN** der Nutzer die App direkt über eine URL aufruft (kein vorheriger App-interner Navigate-Call)
- **THEN** ist kein Zurück-Button sichtbar

#### Scenario: Zurück-Klick navigiert zur Vorseite
- **WHEN** der Nutzer den „← Zurück"-Button klickt
- **THEN** wird `navigate(-1)` ausgeführt und der Nutzer landet auf der zuletzt besuchten Seite

#### Scenario: Mehrstufige Navigation
- **WHEN** der Nutzer A → B → C navigiert und auf C den Zurück-Button klickt
- **THEN** landet der Nutzer auf B; ein weiterer Klick auf A

#### Scenario: Mobile-Darstellung
- **WHEN** die Viewport-Breite unter 640px ist und History vorhanden
- **THEN** erscheint der Zurück-Button im Mobile-Header zwischen Hamburger-Button und App-Titel

#### Scenario: Desktop-Darstellung
- **WHEN** die Viewport-Breite 640px oder größer ist und History vorhanden
- **THEN** erscheint der Zurück-Button als kompakte Leiste oberhalb des Page-Contents

### Requirement: Konsolidierung bestehender Zurück-Buttons
Alle bestehenden per-Page-Zurück-Elemente SHALL entfernt werden, sobald der globale Button in AppShell aktiv ist. Doppelte Navigation-Controls auf einer Seite sind nicht erlaubt.

#### Scenario: TermineDetailPage ohne lokalen Zurück-Button
- **WHEN** der Nutzer die TermineDetailPage aufruft
- **THEN** gibt es keinen zweiten Zurück-Button auf der Seite (weder oben noch unten)

#### Scenario: MeinTeamPage ohne lokalen Zurück-Button
- **WHEN** der Nutzer die MeinTeamPage aufruft
- **THEN** gibt es keinen lokalen Zurück-Button auf der Seite

#### Scenario: SpieltagDetailPage ohne lokalen Zurück-Link
- **WHEN** der Nutzer die SpieltagDetailPage aufruft
- **THEN** gibt es keinen lokalen „← Zurück zum Spielplan"-Link auf der Seite

#### Scenario: MembersPage ohne lokalen Zurück-Button
- **WHEN** der Nutzer die MembersPage aufruft
- **THEN** gibt es keinen lokalen Zurück-Button auf der Seite
