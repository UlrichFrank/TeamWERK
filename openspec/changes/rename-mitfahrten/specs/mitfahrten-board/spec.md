## ADDED Requirements

### Requirement: Mitfahrangebot eintragen
Jeder authentifizierte Nutzer kann sich pro zukünftigem Spiel als Fahrer eintragen (typ='biete') mit Angaben zu Sitzplätzen, Treffpunkt und Notiz. Pro Spiel ist nur EIN Angebot pro Nutzer erlaubt.

#### Scenario: Nutzer trägt Fahrer-Angebot ein
- **WHEN** ein Nutzer für ein Spiel `POST /api/mitfahrten` mit `typ='biete'` sendet
- **THEN** erscheint sein Eintrag in der „Fahrer"-Liste für dieses Spiel

#### Scenario: Doppeltes Angebot wird aktualisiert (Upsert)
- **WHEN** ein Nutzer für dasselbe Spiel ein zweites Angebot mit `typ='biete'` sendet
- **THEN** wird der bestehende Eintrag überschrieben (Upsert — UNIQUE pro biete/game/user)

### Requirement: Mitfahrgesuch eintragen
Nutzer können sich als Mitfahrer (typ='suche') für ein zukünftiges Spiel eintragen. Pro Nutzer ist nur EIN Gesuch pro Spiel erlaubt. Jedes Gesuch MUSS die Anzahl der benötigten Plätze (`plaetze` ≥ 1) enthalten.

#### Scenario: Nutzer trägt Mitfahrgesuch ein
- **WHEN** ein Nutzer für ein Spiel `POST /api/mitfahrten` mit `typ='suche'` und `plaetze ≥ 1` sendet
- **THEN** erscheint sein Eintrag in der „Gesuche"-Liste für dieses Spiel

#### Scenario: Gesuch ohne Platzangabe wird abgewiesen
- **WHEN** ein Nutzer `POST /api/mitfahrten` mit `typ='suche'` ohne `plaetze` oder `plaetze=0` sendet
- **THEN** antwortet die API mit 400 Bad Request

### Requirement: Eigenen Eintrag zurückziehen
Nutzer können ihren eigenen Eintrag für ein Spiel löschen.

#### Scenario: Eintrag löschen
- **WHEN** ein Nutzer `DELETE /api/mitfahrten/{id}` aufruft für einen eigenen Eintrag
- **THEN** wird der Eintrag entfernt und erscheint nicht mehr in der Liste

#### Scenario: Fremden Eintrag löschen schlägt fehl
- **WHEN** ein Nutzer versucht, den Eintrag eines anderen zu löschen
- **THEN** antwortet die API mit 403 Forbidden

### Requirement: Einträge einsehen
Alle authentifizierten Nutzer sehen für jedes zukünftige Spiel die vollständige Liste der Fahrer-Angebote und Mitfahr-Gesuche.

#### Scenario: Übersicht laden
- **WHEN** `GET /api/mitfahrten` aufgerufen wird
- **THEN** liefert die API alle zukünftigen Spiele (date >= heute) mit je zwei Listen (biete, suche), einem `paarungen`-Array und einem `is_own`-Flag für eigene Einträge

#### Scenario: Keine zukünftigen Spiele
- **WHEN** keine Spiele in der Zukunft existieren
- **THEN** liefert die API eine leere Liste (kein Fehler)

#### Scenario: Vergangene Spiele ausgeblendet
- **WHEN** ein Spiel in der Vergangenheit liegt
- **THEN** erscheint es nicht in der Mitfahrten-Liste

### Requirement: Datenschutz — Nutzernamen
In der Liste werden die Namen der Nutzer angezeigt (aus `users.name`), nicht deren E-Mail-Adressen.

#### Scenario: Liste enthält Namen statt E-Mails
- **WHEN** ein Nutzer `GET /api/mitfahrten` aufruft
- **THEN** enthält die Antwort für jeden Eintrag `user_name`, aber keine `user_email`

### Requirement: Mitfahrt anlegen ist idempotent (suche)
Ein Nutzer SHALL pro Spiel und Typ höchstens eine Mitfahrt besitzen. `POST /api/mitfahrten` mit `typ='suche'` MUSS einen bestehenden Eintrag aktualisieren statt einen neuen anzulegen, wenn bereits ein `suche`-Eintrag des Nutzers für dasselbe Spiel existiert.

#### Scenario: Erster suche-Eintrag wird angelegt
- **WHEN** ein Nutzer `POST /api/mitfahrten` mit `typ='suche'` für ein Spiel aufruft, für das er noch keinen suche-Eintrag hat
- **THEN** wird ein neuer Eintrag angelegt und HTTP 200 zurückgegeben

#### Scenario: Bestehender suche-Eintrag wird aktualisiert
- **WHEN** ein Nutzer `POST /api/mitfahrten` mit `typ='suche'` für ein Spiel aufruft, für das er bereits einen suche-Eintrag hat
- **THEN** wird der bestehende Eintrag aktualisiert (kein neuer Row) und HTTP 200 zurückgegeben

#### Scenario: Datenbank verhindert suche-Duplikate
- **WHEN** durch eine Race Condition zwei gleichzeitige suche-Inserts für denselben Nutzer und dasselbe Spiel eintreffen
- **THEN** schlägt einer der Inserts mit einem Constraint-Fehler fehl

### Requirement: Modal zeigt bestehende Einträge des Nutzers
Das Eintrag-Modal SHALL beim Öffnen die Daten eines bereits vorhandenen eigenen Eintrags des gewählten Typs vorausfüllen. Beim Typ-Wechsel innerhalb des Modals werden die Felder mit dem bestehenden Eintrag des neuen Typs befüllt — oder auf Defaults gesetzt, falls kein Eintrag existiert.

#### Scenario: Modal öffnet mit bestehendem biete-Eintrag
- **WHEN** ein Nutzer das Modal für ein Spiel öffnet, für das er bereits einen biete-Eintrag hat
- **THEN** sind Treffpunkt, Notiz und Plätze mit den gespeicherten Werten vorausgefüllt

#### Scenario: Typ-Wechsel zeigt bestehenden Eintrag des neuen Typs
- **WHEN** ein Nutzer im Modal von biete zu suche wechselt und bereits einen suche-Eintrag für dieses Spiel hat
- **THEN** werden die Felder mit den Werten des suche-Eintrags befüllt

#### Scenario: Typ-Wechsel ohne bestehenden Eintrag zeigt leere Felder
- **WHEN** ein Nutzer im Modal von biete zu suche wechselt und noch keinen suche-Eintrag hat
- **THEN** sind Treffpunkt und Notiz leer; Plätze ist auf 1 gesetzt

### Requirement: Generische Events mit mehreren Teams erscheinen genau einmal
Ein generisches Event, das mehreren Teams zugeordnet ist, SHALL in der Mitfahrten-Liste genau einmal erscheinen. Der angezeigte Team-Name zeigt alle beteiligten Teams komma-separiert.

#### Scenario: Generisches Event mit zwei Teams — eine Zeile
- **WHEN** ein generisches Event den Teams A und B zugeordnet ist
- **THEN** erscheint es genau einmal in der Liste mit Team-Name „A, B"

#### Scenario: Heimspiel mit einem Team — unverändert
- **WHEN** ein Heim- oder Auswärtsspiel genau einem Team zugeordnet ist
- **THEN** erscheint es unverändert einmal mit dem Team-Namen
