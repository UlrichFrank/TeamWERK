# trainings Specification

## Purpose

Diese Spezifikation beschreibt die Capability `trainings`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Training-Series mit Venue
Training-Series SHALL einen optionalen Venue (venue_id FK) statt eines Freitext-Ortsfeldes haben.

#### Scenario: Series mit Venue anlegen
- **WHEN** Nutzer legt Training-Series mit Venue an
- **THEN** venue_id wird gespeichert; Response enthält venue-Objekt

#### Scenario: Series ohne Venue
- **WHEN** Nutzer legt Training-Series ohne Venue an
- **THEN** venue_id ist null; kein Maps-Link wird angezeigt

---

### Requirement: Training-Session mit Venue
Training-Sessions SHALL einen optionalen Venue (venue_id FK) statt eines Freitext-Ortsfeldes haben. Der Venue der Series wird als Default vorausgefüllt und kann je Session überschrieben werden.

#### Scenario: Session erbt Venue der Series
- **WHEN** Nutzer öffnet Formular für neue einzelne Training-Session aus einer Series
- **THEN** venue_id wird mit dem Venue der Series vorausgefüllt (sofern gesetzt)

#### Scenario: Session mit abweichendem Venue
- **WHEN** Nutzer wählt anderen Venue für eine einzelne Session
- **THEN** Session-venue_id überschreibt den Series-Venue für diese Session

#### Scenario: Venue in Response eingebettet
- **WHEN** GET /api/trainings oder Training-Detail aufgerufen wird
- **THEN** Response enthält venue-Objekt (oder null) für Series und Session

---

> **Replaced**: Das Freitext-Feld `location TEXT` wurde durch strukturierte `venue_id` FK ersetzt. Bestehende location-Texte werden nicht migriert; Orte müssen neu als Venues angelegt werden.
