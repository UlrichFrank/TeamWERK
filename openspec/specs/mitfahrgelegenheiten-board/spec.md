## ADDED Requirements

### Requirement: Mitfahrangebot eintragen
Jeder authentifizierte Nutzer kann sich pro zukünftigem Auswärtsspiel als Fahrer eintragen (typ='biete') mit optionalen Angaben zu Sitzplätzen, Treffpunkt und Notiz.

#### Scenario: Nutzer trägt Fahrer-Angebot ein
- **WHEN** ein Nutzer für ein Auswärtsspiel `POST /api/mitfahrgelegenheiten` mit `typ='biete'` sendet
- **THEN** erscheint sein Eintrag in der „Fahrer"-Liste für dieses Spiel

#### Scenario: Doppelter Eintrag wird aktualisiert
- **WHEN** ein Nutzer für dasselbe Spiel erneut einen Eintrag sendet
- **THEN** wird der bestehende Eintrag überschrieben (Upsert)

### Requirement: Mitfahrgesuch eintragen
Nutzer können sich als Mitfahrer (typ='suche') für ein zukünftiges Auswärtsspiel eintragen.

#### Scenario: Nutzer trägt Mitfahrgesuch ein
- **WHEN** ein Nutzer für ein Auswärtsspiel `POST /api/mitfahrgelegenheiten` mit `typ='suche'` sendet
- **THEN** erscheint sein Eintrag in der „Gesuche"-Liste für dieses Spiel

### Requirement: Eigenen Eintrag zurückziehen
Nutzer können ihren eigenen Eintrag für ein Spiel löschen.

#### Scenario: Eintrag löschen
- **WHEN** ein Nutzer `DELETE /api/mitfahrgelegenheiten/{id}` aufruft für einen eigenen Eintrag
- **THEN** wird der Eintrag entfernt und erscheint nicht mehr in der Liste

#### Scenario: Fremden Eintrag löschen schlägt fehl
- **WHEN** ein Nutzer versucht, den Eintrag eines anderen zu löschen
- **THEN** antwortet die API mit 403 Forbidden

### Requirement: Einträge einsehen
Alle authentifizierten Nutzer sehen für jedes zukünftige Auswärtsspiel die vollständige Liste der Fahrer-Angebote und Mitfahr-Gesuche.

#### Scenario: Übersicht laden
- **WHEN** `GET /api/mitfahrgelegenheiten` aufgerufen wird
- **THEN** liefert die API alle zukünftigen Auswärtsspiele (is_home=0, date >= heute) mit je zwei Listen (biete, suche) und einem `is_own`-Flag für eigene Einträge

#### Scenario: Keine zukünftigen Auswärtsspiele
- **WHEN** keine Auswärtsspiele in der Zukunft existieren
- **THEN** liefert die API eine leere Liste (kein Fehler)

#### Scenario: Vergangene Spiele ausgeblendet
- **WHEN** ein Auswärtsspiel in der Vergangenheit liegt
- **THEN** erscheint es nicht in der Mitfahrgelegenheiten-Liste

### Requirement: Datenschutz — Nutzernamen
In der Liste werden die Namen der Nutzer angezeigt (aus `users.name`), nicht deren E-Mail-Adressen.
