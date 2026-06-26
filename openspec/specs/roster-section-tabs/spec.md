# Spec: roster-section-tabs

## Overview
Die Mannschaftskarten auf der Mein-Team-Seite zeigen eine Tab-Navigation mit drei Kategorien: **Team**, **Trainer**, **Eltern**. Jede Karte verwaltet ihren Tab-Zustand unabhängig.

---

## Purpose

Diese Spezifikation beschreibt die Capability `roster-section-tabs`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: RosterSection zeigt Tab-Navigation
Jede Mannschaftskarte in der Mein-Team-Seite SHALL eine Tab-Leiste mit drei Tabs anzeigen: **Team**, **Trainer**, **Eltern**. Der aktive Tab ist beim Laden der Karte immer „Team".

#### Scenario: Standard-Tab beim Öffnen
- **WHEN** die Mein-Team-Seite geladen wird
- **THEN** zeigt jede Mannschaftskarte den Tab „Team" als aktiven Tab

#### Scenario: Tab-Wechsel
- **WHEN** der Nutzer auf einen anderen Tab klickt
- **THEN** wechselt der Inhalt der Karte auf die entsprechende Kategorie (Trainer-Liste oder Eltern-Liste)

#### Scenario: Unabhängiger Tab-Zustand bei mehreren Karten
- **WHEN** der Nutzer bei Karte A auf „Trainer" wechselt
- **THEN** bleibt Karte B auf ihrem eigenen Tab-Zustand (keine Synchronisation)

### Requirement: Leere Tabs zeigen Leertext
Ist eine Tab-Kategorie für ein Team leer (keine Einträge), SHALL der Tab trotzdem angezeigt und auswählbar sein. Der Inhalt zeigt dann den Text `— keine Einträge —`.

#### Scenario: Leerer Trainer-Tab
- **WHEN** ein Team keine Trainer hat und der Nutzer auf „Trainer" klickt
- **THEN** zeigt die Karte den Text `— keine Einträge —`

#### Scenario: Leerer Eltern-Tab
- **WHEN** ein Team keine Eltern hat und der Nutzer auf „Eltern" klickt
- **THEN** zeigt die Karte den Text `— keine Einträge —`

### Requirement: Team-Tab zeigt Abschnitt „Erweiterter Kader"

Der Team-Tab auf der Mein-Team-Seite SHALL unterhalb der regulären Spielertabelle einen Abschnitt „Erweiterter Kader" anzeigen, wenn `extended_players` in der Roster-Antwort mindestens einen Eintrag enthält. Ist `extended_players` leer, wird kein Abschnitt gerendert (kein Leer-Text, kein leerer Block).

#### Scenario: Team hat abgesetzte Spieler

- **WHEN** `GET /api/teams/{id}/roster` gibt `extended_players` mit mindestens einem Eintrag zurück
- **WHEN** der Nutzer den Tab „Team" aktiviert
- **THEN** zeigt die Karte unterhalb der regulären Spielertabelle einen Abschnitt mit Heading „Erweiterter Kader"
- **THEN** listet der Abschnitt die abgesetzten Spieler mit Trikotnummer und Name (gleiche Spalten wie reguläre Spieler)

#### Scenario: Team hat keine abgesetzten Spieler

- **WHEN** `GET /api/teams/{id}/roster` gibt `extended_players: []` zurück
- **WHEN** der Nutzer den Tab „Team" aktiviert
- **THEN** zeigt die Karte keinen „Erweiterter Kader"-Abschnitt
