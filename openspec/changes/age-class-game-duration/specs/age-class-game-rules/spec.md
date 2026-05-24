## ADDED Requirements

### Requirement: Altersklassen-Regeln abrufen

`GET /api/admin/age-class-rules` SHALL alle vier Einträge (A–D) als geordnete Liste zurückgeben. Die Route ist für `admin`, `vorstand` und `trainer` zugänglich.

#### Scenario: Admin ruft Regeln ab

- **WHEN** ein User mit Rolle admin `GET /api/admin/age-class-rules` aufruft
- **THEN** antwortet der Server mit HTTP 200 und einem Array mit genau vier Objekten `{age_class, half_duration_minutes, break_minutes}`

#### Scenario: Trainer darf lesen

- **WHEN** ein User mit Rolle trainer `GET /api/admin/age-class-rules` aufruft
- **THEN** antwortet der Server mit HTTP 200

#### Scenario: Spieler hat keinen Zugriff

- **WHEN** ein User mit Rolle spieler oder elternteil `GET /api/admin/age-class-rules` aufruft
- **THEN** antwortet der Server mit HTTP 403

---

### Requirement: Altersklassen-Regel bearbeiten

`PUT /api/admin/age-class-rules/{ageClass}` SHALL `half_duration_minutes` und `break_minutes` einer Altersklasse aktualisieren. Nur `admin` darf schreiben.

#### Scenario: Admin aktualisiert A-Jugend-Halbzeit

- **WHEN** admin `PUT /api/admin/age-class-rules/A` mit `{half_duration_minutes: 30, break_minutes: 15}` aufruft
- **THEN** antwortet der Server mit HTTP 200 und dem aktualisierten Objekt

#### Scenario: Ungültige Altersklasse

- **WHEN** `PUT /api/admin/age-class-rules/E` aufgerufen wird
- **THEN** antwortet der Server mit HTTP 404

#### Scenario: Ungültige Werte

- **WHEN** `PUT /api/admin/age-class-rules/B` mit `half_duration_minutes: 0` aufgerufen wird
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Trainer darf nicht schreiben

- **WHEN** ein User mit Rolle trainer `PUT /api/admin/age-class-rules/A` aufruft
- **THEN** antwortet der Server mit HTTP 403

---

### Requirement: Admin-UI Altersklassen-Regeln

Eine Admin-Seite unter `/admin/altersklassen` SHALL eine Tabelle mit allen vier Altersklassen und ihren Zeitwerten anzeigen. Jede Zeile ist inline editierbar (Zahlenfelder für Halbzeit und Pause). Nur Nutzer mit Rolle `admin` sehen die Seite und können speichern.

#### Scenario: Seite lädt Daten

- **WHEN** ein Admin die Seite `/admin/altersklassen` öffnet
- **THEN** wird eine Tabelle mit vier Zeilen (A, B, C, D) angezeigt, jede mit den aktuellen Minuten-Werten

#### Scenario: Inline-Speichern

- **WHEN** der Admin den Wert für „A-Jugend Halbzeit" ändert und „Speichern" klickt
- **THEN** wird `PUT /api/admin/age-class-rules/A` gesendet und die Änderung bestätigt

#### Scenario: Validierung im Frontend

- **WHEN** der Admin einen Wert ≤ 0 eingibt und speichert
- **THEN** wird eine Fehlermeldung angezeigt und kein API-Aufruf gesendet

#### Scenario: Nav-Eintrag nur für Admin sichtbar

- **WHEN** ein User mit Rolle trainer eingeloggt ist
- **THEN** ist der Menüeintrag „Altersklassen" nicht in der Navigation sichtbar
