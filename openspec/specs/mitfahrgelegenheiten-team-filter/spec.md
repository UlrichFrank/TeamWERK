# mitfahrgelegenheiten-team-filter Specification

## Purpose

Diese Spezifikation beschreibt die Capability `mitfahrgelegenheiten-team-filter`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Team-Dropdown filtert Mitfahrgelegenheiten
Die Mitfahrgelegenheiten-Seite SHALL einen Team-Dropdown anzeigen, wenn der Nutzer Zugang zu mehr als einem Team hat. Der gewählte Filter wird als `?team_id=X` an `GET /api/mitfahrgelegenheiten` übergeben.

#### Scenario: Nutzer mit einem Team sieht keinen Dropdown
- **WHEN** ein Nutzer Zugang zu genau einem Team hat
- **THEN** ist kein Team-Dropdown sichtbar und alle Events dieses Teams werden ohne Filterinteraktion angezeigt

#### Scenario: Nutzer mit mehreren Teams kann filtern
- **WHEN** ein Nutzer Zugang zu mehreren Teams hat und einen Team aus dem Dropdown wählt
- **THEN** zeigt die Seite nur Events des gewählten Teams an

#### Scenario: Kein Filter zeigt alle zugänglichen Teams
- **WHEN** ein Nutzer mit mehreren Teams die Option „Alle" im Dropdown wählt
- **THEN** werden Events aller zugänglichen Teams angezeigt

#### Scenario: Team-Filter und Ansicht (Teams/Meine) sind kombinierbar
- **WHEN** ein Nutzer gleichzeitig einen Team-Filter und „Meine" aktiviert hat
- **THEN** werden nur eigene Einträge des gewählten Teams angezeigt
