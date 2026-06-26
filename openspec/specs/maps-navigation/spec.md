# maps-navigation Specification

## Purpose

Diese Spezifikation beschreibt die Capability `maps-navigation`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Maps-Deep-Link anzeigen
Überall wo ein Venue mit Adresse angezeigt wird, SHALL ein anklickbarer Maps-Link erscheinen.

#### Scenario: Link öffnet Maps-App (Präferenz-abhängig)
- **WHEN** Nutzer klickt auf den Maps-Link eines Venues
- **THEN** öffnet der Browser die URL zum konfigurierten Kartendienst des Nutzers (google, apple oder auto-erkannt) mit der kodierten Adresse in einem neuen Tab

#### Scenario: Kein Venue vorhanden
- **WHEN** Event hat venue_id = null
- **THEN** Kein Maps-Link wird angezeigt; Ort-Bereich bleibt leer oder zeigt „Kein Ort angegeben"

---

### Requirement: Maps-Link in Spielplan-Ansicht
Spielplan-Einträge mit Venue SHALL den Maps-Link inline neben dem Venue-Namen zeigen.

#### Scenario: Auswärts-/Heimspiel mit Venue
- **WHEN** Nutzer sieht einen Spielplan-Eintrag mit gesetztem Venue
- **THEN** Venue-Name und Maps-Icon-Link sind sichtbar

---

### Requirement: Maps-Link in Training-Ansicht
Training-Sessions mit Venue SHALL den Maps-Link inline zeigen.

#### Scenario: Training mit Venue
- **WHEN** Nutzer sieht eine Training-Session mit gesetztem Venue
- **THEN** Venue-Name und Maps-Icon-Link sind sichtbar
