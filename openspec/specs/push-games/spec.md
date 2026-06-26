# push-games Specification

## Purpose

Diese Spezifikation beschreibt die Capability `push-games`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Push bei Spiel-Ereignissen
Das System SHALL allen berechtigten Team-Mitgliedern und deren Elternteilen eine Push Notification senden, wenn ein Spiel erstellt, geändert oder gelöscht wird — sofern der Nutzer Push für die Kategorie `games` nicht deaktiviert hat. Die Notification-`url` MUSS auf den konkreten Spieltermin in der Termine-Seite zeigen (`/termine?focus=game-<id>`), damit der Empfänger direkt zu- oder absagen kann. Für gelöschte Spiele (kein navigierbarer Termin mehr) zeigt die `url` auf `/termine`.

#### Scenario: Neues Spiel erstellt
- **WHEN** ein Admin oder Trainer ein neues Spiel über `POST /api/games` anlegt
- **THEN** erhalten alle aktiven Mitglieder des betroffenen Teams + deren Elternteile eine Push Notification mit Titel „Neues Spiel" und der Gegnerinfo
- **THEN** zeigt der Klick-Link auf `/termine?focus=game-<id>` des neu erstellten Spiels

#### Scenario: Spiel verschoben oder geändert
- **WHEN** ein Admin oder Trainer ein Spiel über `PUT /api/games/{id}` aktualisiert (Datum, Zeit oder Ort geändert)
- **THEN** erhalten alle aktiven Mitglieder des betroffenen Teams + deren Elternteile eine Push Notification „Spielinfo geändert"
- **THEN** zeigt der Klick-Link auf `/termine?focus=game-<id>`

#### Scenario: Spiel abgesagt (gelöscht)
- **WHEN** ein Admin oder Trainer ein Spiel über `DELETE /api/games/{id}` löscht
- **THEN** erhalten alle aktiven Mitglieder des betroffenen Teams + deren Elternteile eine Push Notification „Spiel abgesagt"
- **THEN** zeigt der Klick-Link auf `/termine` (kein `focus`, da das Spiel nicht mehr existiert)

#### Scenario: Nutzer mit deaktiviertem Push erhält keine Notification
- **WHEN** ein Spiel-Ereignis eintritt und ein Nutzer hat `push_enabled=0` für Kategorie `games` in `notification_preferences`
- **THEN** erhält dieser Nutzer keine Push Notification für dieses Ereignis
