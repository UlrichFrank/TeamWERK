## MODIFIED Requirements

### Requirement: Explizite Vorlage bei Erstellung

`POST /api/admin/games` SHALL ein optionales Feld `template_id` akzeptieren. Wenn angegeben, wird diese Vorlage direkt für die Slot-Generierung verwendet. Die Dauern-Quelle für die Slot-Zeitberechnung ist strikt durch `event_type` bestimmt:

- `heim` / `auswärts`: Dauer = `half_duration_minutes * 2 + break_minutes` aus `age_class_game_rules` des Teams. Das Feld `duration_minutes` der Vorlage wird ignoriert.
- `generisch`: Dauer = `duration_minutes` der Vorlage. Altersklassen-Regeln werden nicht konsultiert.

#### Scenario: Vorlage explizit übergeben

- **WHEN** `POST /api/admin/games` mit `template_id: 5` aufgerufen wird
- **THEN** werden Slots aus Vorlage 5 generiert, ohne nach `template_type` zu suchen

#### Scenario: Kein template_id — Fallback

- **WHEN** `POST /api/admin/games` ohne `template_id` aufgerufen wird
- **THEN** wählt das Backend automatisch anhand `event_type` eine passende Vorlage (bisheriges Verhalten)

#### Scenario: Heim-Spiel nutzt Altersklassen-Regel

- **WHEN** ein Heim-Spiel für ein B-Jugend-Team angelegt wird
- **THEN** berechnet das Backend Slot-Zeiten mit 25 Min Halbzeit und 10 Min Pause (Gesamtdauer 60 Min), unabhängig vom `duration_minutes`-Wert der Vorlage

#### Scenario: Generisches Event nutzt Vorlagen-Dauer

- **WHEN** ein Event mit `event_type: 'generisch'` und einer Vorlage mit `duration_minutes: 90` angelegt wird
- **THEN** werden Slot-Zeiten auf Basis von 90 Minuten berechnet; Altersklassen-Regeln werden ignoriert

#### Scenario: Heim-Spiel ohne Team-Altersklasse

- **WHEN** ein Heim-Spiel für ein Team ohne `age_class` (NULL) angelegt wird
- **THEN** antwortet der Server mit HTTP 422 und einer Fehlermeldung

#### Scenario: Generisches Event ohne Vorlagen-Dauer

- **WHEN** ein Event mit `event_type: 'generisch'` und einer Vorlage ohne `duration_minutes` (NULL) angelegt wird
- **THEN** antwortet der Server mit HTTP 422 und einer Fehlermeldung
