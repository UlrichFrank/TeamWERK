## MODIFIED Requirements

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

## MODIFIED Requirements

### Requirement: Mitfahrangebot eintragen
Jeder authentifizierte Nutzer kann sich pro zukünftigem Spiel als Fahrer eintragen (typ='biete') mit Angaben zu Sitzplätzen, Treffpunkt und Notiz. Pro Spiel ist nur EIN Angebot pro Nutzer erlaubt.

#### Scenario: Nutzer trägt Fahrer-Angebot ein
- **WHEN** ein Nutzer für ein Spiel `POST /api/mitfahrgelegenheiten` mit `typ='biete'` sendet
- **THEN** erscheint sein Eintrag in der „Fahrer"-Liste für dieses Spiel

#### Scenario: Doppeltes Angebot wird aktualisiert (Upsert)
- **WHEN** ein Nutzer für dasselbe Spiel ein zweites Angebot mit `typ='biete'` sendet
- **THEN** wird der bestehende Eintrag überschrieben (Upsert — UNIQUE pro biete/game/user)

#### Scenario: Zweites Angebot bei anderem Nutzer möglich
- **WHEN** ein anderer Nutzer für dasselbe Spiel ein Angebot sendet
- **THEN** wird ein neuer Bieter-Eintrag angelegt unabhängig vom ersten
