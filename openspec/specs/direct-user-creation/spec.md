# direct-user-creation Specification

## Purpose

Diese Spezifikation beschreibt die Capability `direct-user-creation`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Direktes Anlegen eines Nutzeraccounts

Das System SHALL es Vorstand und Admin ermöglichen, einen vollständig login-fähigen Nutzeraccount direkt anzulegen — ohne Einladungs-E-Mail und ohne Wartezeit.

Der neue Account MUSS folgende Felder haben: E-Mail (unique), Vorname, Nachname, Passwort (bcrypt-gehasht), Rolle fest `standard`.

#### Scenario: Erfolgreicher Account-Anlage

- **WHEN** Vorstand oder Admin das Modal „Account anlegen" ausfüllt (E-Mail, Vorname, Nachname) und abspeichert
- **THEN** wird der Account sofort in der Datenbank angelegt, die Nutzerliste refresht, und das Modal schließt sich

#### Scenario: E-Mail bereits vergeben

- **WHEN** die eingegebene E-Mail-Adresse bereits einem bestehenden Nutzer gehört
- **THEN** gibt das Backend HTTP 409 zurück und das Modal zeigt eine Fehlermeldung

#### Scenario: Pflichtfelder nicht ausgefüllt

- **WHEN** E-Mail, Vorname oder Nachname leer sind
- **THEN** verhindert das Formular das Absenden (HTML5-Validierung `required`)

### Requirement: Passwort-Generierung und Copy-Button

Das System SHALL beim Öffnen des Modals automatisch ein starkes Passwort (~16 Zeichen, Groß-/Kleinbuchstaben + Ziffern + Sonderzeichen) generieren und im Passwortfeld (readonly) anzeigen.

Ein Copy-Button SHALL das Passwort in die Zwischenablage kopieren und visuelles Feedback (Icon-Wechsel zu Checkmark für 2 s) geben.

#### Scenario: Passwort kopieren

- **WHEN** Admin auf den Copy-Button neben dem Passwortfeld klickt
- **THEN** wird das Passwort via `navigator.clipboard.writeText()` in die Zwischenablage kopiert und der Button zeigt kurz ein Checkmark-Icon

#### Scenario: Passwort neu generieren

- **WHEN** Admin auf „Neu generieren" klickt (oder das Modal erneut öffnet)
- **THEN** wird ein neues zufälliges Passwort erzeugt und im Feld angezeigt

### Requirement: Zugriffskontrolle

Der Endpunkt `POST /api/users` MUSS auf Vorstand und Admin beschränkt sein. Alle anderen Rollen erhalten HTTP 403.

#### Scenario: Unbefugter Zugriff

- **WHEN** ein Nutzer ohne Vorstand- oder Admin-Rolle `POST /api/users` aufruft
- **THEN** antwortet der Server mit HTTP 403 Forbidden
