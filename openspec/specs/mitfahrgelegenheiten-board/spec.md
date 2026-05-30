## ADDED Requirements

### Requirement: Mitfahrangebot eintragen
Jeder authentifizierte Nutzer kann sich pro zukünftigem Spiel als Fahrer eintragen (typ='biete') mit Angaben zu Sitzplätzen, Treffpunkt und Notiz. Pro Spiel ist nur EIN Angebot pro Nutzer erlaubt.

#### Scenario: Nutzer trägt Fahrer-Angebot ein
- **WHEN** ein Nutzer für ein Spiel `POST /api/mitfahrgelegenheiten` mit `typ='biete'` sendet
- **THEN** erscheint sein Eintrag in der „Fahrer"-Liste für dieses Spiel

#### Scenario: Doppeltes Angebot wird aktualisiert (Upsert)
- **WHEN** ein Nutzer für dasselbe Spiel ein zweites Angebot mit `typ='biete'` sendet
- **THEN** wird der bestehende Eintrag überschrieben (Upsert — UNIQUE pro biete/game/user)

### Requirement: Mitfahrgesuch eintragen
Nutzer können sich als Mitfahrer (typ='suche') für ein zukünftiges Spiel eintragen. Ein Nutzer DARF mehrere Gesuche pro Spiel anlegen, um seine Gruppe manuell auf mehrere Fahrer aufzuteilen. Jedes Gesuch MUSS die Anzahl der benötigten Plätze (`plaetze` ≥ 1) enthalten.

#### Scenario: Nutzer trägt Mitfahrgesuch ein
- **WHEN** ein Nutzer für ein Spiel `POST /api/mitfahrgelegenheiten` mit `typ='suche'` und `plaetze ≥ 1` sendet
- **THEN** erscheint sein Eintrag in der „Gesuche"-Liste für dieses Spiel

#### Scenario: Mehrere Gesuche pro Spiel erlaubt
- **WHEN** ein Nutzer für dasselbe Spiel ein zweites Gesuch mit `typ='suche'` sendet
- **THEN** wird ein neuer Eintrag angelegt (kein Upsert) — der Nutzer hat nun zwei Gesuche für dieses Spiel

#### Scenario: Gesuch ohne Platzangabe wird abgewiesen
- **WHEN** ein Nutzer `POST /api/mitfahrgelegenheiten` mit `typ='suche'` ohne `plaetze` oder `plaetze=0` sendet
- **THEN** antwortet die API mit 400 Bad Request

### Requirement: Eigenen Eintrag zurückziehen
Nutzer können ihren eigenen Eintrag für ein Spiel löschen.

#### Scenario: Eintrag löschen
- **WHEN** ein Nutzer `DELETE /api/mitfahrgelegenheiten/{id}` aufruft für einen eigenen Eintrag
- **THEN** wird der Eintrag entfernt und erscheint nicht mehr in der Liste

#### Scenario: Fremden Eintrag löschen schlägt fehl
- **WHEN** ein Nutzer versucht, den Eintrag eines anderen zu löschen
- **THEN** antwortet die API mit 403 Forbidden

### Requirement: Einträge einsehen
Alle authentifizierten Nutzer sehen für jedes zukünftige Spiel die vollständige Liste der Fahrer-Angebote und Mitfahr-Gesuche.

#### Scenario: Übersicht laden
- **WHEN** `GET /api/mitfahrgelegenheiten` aufgerufen wird
- **THEN** liefert die API alle zukünftigen Spiele (date >= heute) mit je zwei Listen (biete, suche), einem `paarungen`-Array und einem `is_own`-Flag für eigene Einträge

#### Scenario: Keine zukünftigen Spiele
- **WHEN** keine Spiele in der Zukunft existieren
- **THEN** liefert die API eine leere Liste (kein Fehler)

#### Scenario: Vergangene Spiele ausgeblendet
- **WHEN** ein Spiel in der Vergangenheit liegt
- **THEN** erscheint es nicht in der Mitfahrgelegenheiten-Liste

### Requirement: Datenschutz — Nutzernamen
In der Liste werden die Namen der Nutzer angezeigt (aus `users.name`), nicht deren E-Mail-Adressen.
