## MODIFIED Requirements

### Requirement: Eigenen Eintrag zurückziehen
Nutzer können ihren eigenen Eintrag für ein Spiel löschen. Elternteile können zusätzlich Einträge löschen, die einem ihrer Kinder gehören (geprüft via `family_links`).

#### Scenario: Eintrag löschen
- **WHEN** ein Nutzer `DELETE /api/mitfahrgelegenheiten/{id}` aufruft für einen eigenen Eintrag
- **THEN** wird der Eintrag entfernt und erscheint nicht mehr in der Liste

#### Scenario: Elternteil löscht Kind-Eintrag
- **WHEN** ein Elternteil `DELETE /api/mitfahrgelegenheiten/{id}` für einen Eintrag aufruft, der einem seiner Kinder gehört
- **THEN** wird der Eintrag entfernt

#### Scenario: Fremden Eintrag löschen schlägt fehl
- **WHEN** ein Nutzer versucht, den Eintrag eines anderen zu löschen, der weder ihm noch einem seiner Kinder gehört
- **THEN** antwortet die API mit 403 Forbidden

### Requirement: Einträge einsehen
Alle authentifizierten Nutzer sehen für jedes zukünftige Spiel die vollständige Liste der Fahrer-Angebote und Mitfahr-Gesuche. Das `isOwn`-Flag ist `true` für eigene Einträge UND für Einträge von Kindern des eingeloggten Nutzers.

#### Scenario: Übersicht laden
- **WHEN** `GET /api/mitfahrgelegenheiten` aufgerufen wird
- **THEN** liefert die API alle zukünftigen Spiele mit je zwei Listen (biete, suche), einem `paarungen`-Array, einem `isOwn`-Flag sowie einem `children`-Array mit `userId` und `name` der eigenen Kinder

#### Scenario: isOwn für Kind-Einträge
- **WHEN** ein Elternteil die Liste abruft und ein Kind einen Eintrag hat
- **THEN** hat dieser Eintrag `isOwn: true`

#### Scenario: Keine zukünftigen Spiele
- **WHEN** keine Spiele in der Zukunft existieren
- **THEN** liefert die API eine leere Liste (kein Fehler)

#### Scenario: Vergangene Spiele ausgeblendet
- **WHEN** ein Spiel in der Vergangenheit liegt
- **THEN** erscheint es nicht in der Mitfahrgelegenheiten-Liste
