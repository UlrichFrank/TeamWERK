# whatsapp-sichtbarkeit Specification

## Purpose

Diese Spezifikation beschreibt die Capability `whatsapp-sichtbarkeit`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: WhatsApp-Sichtbarkeitsfeld in DB und API
Das System SHALL ein Feld `whatsapp_visible` in der `user_visibility`-Tabelle speichern und über
`PUT /api/profile/visibility` setzen können.

#### Scenario: Nutzer aktiviert WhatsApp-Sichtbarkeit
- **WHEN** ein authentifizierter Nutzer `PUT /api/profile/visibility` mit `{ whatsapp_visible: true }` aufruft
- **THEN** wird `whatsapp_visible=1` in `user_visibility` gespeichert

#### Scenario: Standard-Wert für neue Nutzer
- **WHEN** ein neuer Nutzer noch keinen Eintrag in `user_visibility` hat
- **THEN** gilt `whatsapp_visible=false` (Default 0)

### Requirement: Toggle-Schalter für alle Sichtbarkeitsoptionen
Das System SHALL im Profil-Tab alle fünf Sichtbarkeitsoptionen als Toggle-Schalter anzeigen
(kein Checkbox-Element), in dieser Reihenfolge: Telefonnummern, WhatsApp, Adresse, Profilbild, E-Mail-Adresse.

#### Scenario: Alle Toggles sichtbar
- **WHEN** ein Nutzer mit User-Account den Profil-Tab öffnet
- **THEN** sieht er fünf Toggle-Schalter: „Telefonnummern sichtbar", „WhatsApp sichtbar",
  „Adresse sichtbar", „Profilbild sichtbar", „E-Mail-Adresse sichtbar"

#### Scenario: Toggle-Zustand wird geladen
- **WHEN** der Profil-Tab geladen wird
- **THEN** spiegeln alle Toggles den aktuell gespeicherten Zustand wider

#### Scenario: Toggle-Änderung löst Speichern aus
- **WHEN** ein Nutzer einen Sichtbarkeits-Toggle umlegt
- **THEN** wird der geänderte Zustand markiert und beim Klick auf „Speichern" via `PUT /api/profile/visibility` gespeichert

#### Scenario: Toggle-Komponente einheitlich
- **WHEN** Toggle-Schalter in Sichtbarkeits-Controls und Push-Benachrichtigungen dargestellt werden
- **THEN** verwenden beide dieselbe `Toggle`-Komponente aus `web/src/components/Toggle.tsx`
