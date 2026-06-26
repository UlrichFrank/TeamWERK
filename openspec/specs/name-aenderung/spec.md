# name-aenderung Specification

## Purpose

Diese Spezifikation beschreibt die Capability `name-aenderung`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Eingeloggter Nutzer kann seinen Anzeigenamen ändern
Das System SHALL jedem authentifizierten Nutzer erlauben, seinen eigenen Anzeigenamen (`users.name`) zu ändern. Eine Passwort-Verifikation ist nicht erforderlich.

#### Scenario: Name erfolgreich ändern
- **WHEN** ein eingeloggter Nutzer `PUT /api/profile/account` mit `{ "name": "Neuer Name" }` aufruft
- **THEN** wird `users.name` für den aufrufenden Nutzer aktualisiert und HTTP 204 zurückgegeben

#### Scenario: Leerer Name wird abgelehnt
- **WHEN** `PUT /api/profile/account` mit leerem oder fehlendem `name`-Feld aufgerufen wird
- **THEN** antwortet das System mit HTTP 400

#### Scenario: Nicht eingeloggte Anfrage wird abgelehnt
- **WHEN** `PUT /api/profile/account` ohne gültigen Bearer-Token aufgerufen wird
- **THEN** antwortet das System mit HTTP 401

### Requirement: Profilseite zeigt bearbeitbares Name-Feld
Das Frontend SHALL auf der Profilseite ein Eingabefeld für den Anzeigenamen anzeigen, das mit dem aktuellen Wert vorbelegt ist und per Speichern-Button gespeichert werden kann.

#### Scenario: Namensfeld vorbelegt
- **WHEN** ein Nutzer die Profilseite aufruft
- **THEN** ist das Name-Feld mit dem aktuellen `users.name`-Wert vorbelegt

#### Scenario: Speichern zeigt Bestätigung
- **WHEN** ein Nutzer den Namen ändert und speichert
- **THEN** erscheint eine kurze Erfolgsmeldung; der neue Name bleibt im Feld stehen
