## ADDED Requirements

### Requirement: User kann Erinnerungsmail-Präferenz einstellen

Das System SHALL jedem eingeloggten User ermöglichen, seine Erinnerungsmail-Präferenz im Profil zu konfigurieren. Die Optionen sind: "2 Tage vor dem Event" oder "Nie".

#### Scenario: User aktiviert Erinnerungsmail
- **WHEN** ein User im Profil "Erinnerungsmail 2 Tage vor Event" auswählt und speichert
- **THEN** wird `users.duty_reminder_days = 2` gesetzt und der User erhält zukünftig Erinnerungsmails

#### Scenario: User deaktiviert Erinnerungsmail
- **WHEN** ein User im Profil "Nie" auswählt und speichert
- **THEN** wird `users.duty_reminder_days = NULL` gesetzt und der User erhält keine Erinnerungsmails mehr

#### Scenario: Neuer User hat Erinnerungen standardmäßig deaktiviert
- **WHEN** ein neuer User angelegt wird (Registrierung oder Admin-Anlage)
- **THEN** ist `duty_reminder_days = NULL` (DEFAULT), d.h. Erinnerungen sind standardmäßig deaktiviert und müssen aktiv eingeschaltet werden

### Requirement: API-Endpoint für Präferenz-Update

Das System SHALL einen authentifizierten Endpoint bereitstellen, über den der eingeloggte User seine Reminder-Präferenz aktualisieren kann.

#### Scenario: Erfolgreiche Aktualisierung via PUT
- **WHEN** ein authentifizierter User `PUT /api/profile/reminder-preference` mit `{ "duty_reminder_days": 2 }` oder `{ "duty_reminder_days": null }` aufruft
- **THEN** antwortet der Server mit `200 OK` und die Präferenz ist gespeichert

#### Scenario: Ungültiger Wert wird abgelehnt
- **WHEN** ein User einen Wert übergibt, der weder `2` noch `null` ist
- **THEN** antwortet der Server mit `400 Bad Request`

### Requirement: Profil-UI zeigt aktuellen Reminder-Status

Das System SHALL im Profil-Bereich den aktuellen Status der Erinnerungsmail-Präferenz anzeigen und eine Möglichkeit zur Änderung bieten.

#### Scenario: Toggle zeigt aktuellen Zustand
- **WHEN** ein User die Profil-Seite aufruft
- **THEN** sieht er einen Toggle/Schalter der anzeigt ob Erinnerungsmails aktiviert sind ("2 Tage vor Event") oder deaktiviert ("Nie")

#### Scenario: Änderung wird sofort gespeichert
- **WHEN** ein User den Toggle betätigt
- **THEN** wird die Änderung via `PUT /api/profile/reminder-preference` gespeichert und der neue Zustand visuell bestätigt
