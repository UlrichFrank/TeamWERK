## ADDED Requirements

### Requirement: Übungsgruppen-Sichtbarkeit im Authenticated-Tier mit Handler-Scope

`GET /api/practice-groups/my` SHALL im Authenticated-Tier liegen: jede eingeloggte
Persona erreicht die Route, ohne Token antwortet sie mit HTTP 401. Die Einschränkung
auf die eigenen Übungsgruppen (Trainer via `kader_trainers`, Spieler/Eltern via
`kader_members`/`kader_extended_members`/`family_links`) SHALL der Handler selbst
durchsetzen — für die System-Rolle `admin` und die Vereinsfunktionen `vorstand` und
`sportliche_leitung` entfällt die Einschränkung, sie sehen alle Übungsgruppen der
aktiven Saison.

#### Scenario: Eingeloggte Persona erreicht die Route
- **WHEN** eine beliebige Persona `GET /api/practice-groups/my` mit gültigem Token
  aufruft
- **THEN** antwortet das System nicht mit 401

#### Scenario: Aufruf ohne Token
- **WHEN** `GET /api/practice-groups/my` ohne Token aufgerufen wird
- **THEN** antwortet das System mit HTTP 401
