# test-members-gaps Specification

## Purpose

Diese Spezifikation beschreibt die Capability `test-members-gaps`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Eigenes Profil lesen
Das System SHALL einem eingeloggten Nutzer das Lesen seiner eigenen Profildaten ermöglichen.

#### Scenario: Eigenes Profil abrufen
- **WHEN** GET /api/profile/me mit gültigem JWT
- **THEN** HTTP 200, Response enthält email, first_name, last_name des eingeloggten Nutzers

### Requirement: Eigenes Profil aktualisieren
Das System SHALL einem eingeloggten Nutzer das Aktualisieren seiner eigenen Profildaten ermöglichen. Andere Nutzer-Profile dürfen nicht überschrieben werden.

#### Scenario: Eigenes Profil aktualisieren
- **WHEN** PUT /api/profile/me mit geänderten Feldern (z.B. first_name)
- **THEN** HTTP 200 oder 204, Änderung in DB persistiert, nur eigene Daten betroffen
