# kader-games-per-season Specification

## Purpose

Diese Spezifikation beschreibt die Capability `kader-games-per-season`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Feld games_per_season auf kader
Jeder Kader-Eintrag SHALL die Anzahl der Saisonspiele speichern (`games_per_season INTEGER NOT NULL DEFAULT 0`).

#### Scenario: Kader ohne gesetzten Wert
- **WHEN** ein Kader angelegt wird ohne `games_per_season` zu setzen
- **THEN** ist der Wert 0

### Requirement: Admin-UI für games_per_season
In AdminKaderPage SHALL `games_per_season` als nummerisches Input-Feld editierbar sein, rechts neben dem Altersklasse-Feld in der Kader-Zeile.

#### Scenario: Admin ändert Spielanzahl
- **WHEN** ein Admin/Vorstand `games_per_season` auf 20 setzt und speichert
- **THEN** wird der Wert in der DB persistiert und ist beim nächsten Laden sichtbar

#### Scenario: Elternteil sieht Admin-UI nicht
- **WHEN** ein Elternteil die AdminKaderPage aufruft (kein Zugriff per Route-Guard)
- **THEN** sieht er die Seite nicht (401/403 bzw. Sidebar-Eintrag nur für admin/vorstand)
