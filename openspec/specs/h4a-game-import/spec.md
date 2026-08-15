# h4a-game-import Specification

## Purpose
TBD - created by archiving change h4a-import. Update Purpose after archive.
## Requirements
### Requirement: Server-seitiger Abruf des H4A-Spielplans mit eingegebenen Zugangsdaten

Das System SHALL einen Endpoint `POST /api/games/import/h4a/preview` bereitstellen, der
Handball4All-Zugangsdaten (`user`, `pw`) sowie eine H4A-`period_id` entgegennimmt, sich
server-seitig bei Handball4All anmeldet, den Spielplan über `edit.php` (xajax) mit dem Filter
„nur eigene Beteiligung" abruft und einen Diff-Plan zurückliefert, **ohne** die Zugangsdaten
zu persistieren. Nur Nutzer mit Vereinsfunktion `vorstand` oder Systemrolle `admin` dürfen
den Endpoint aufrufen.

Das System SHALL die Zugangsdaten ausschließlich für die Dauer dieses Requests im Speicher
halten und sie NICHT loggen, NICHT in Datenbank/Datei/Env schreiben und ausgehende
H4A-Requests nur über HTTPS ausführen.

#### Scenario: Vorstand ruft Preview mit gültigen Zugangsdaten ab
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` `POST /api/games/import/h4a/preview` mit gültigen H4A-Zugangsdaten und einer `period_id` aufruft
- **THEN** antwortet der Server mit HTTP 200 und einem Body `{ plan: { new, changed, unchanged }, mappings, warnings }`
- **THEN** enthält der Plan nur Spiele mit eigener Vereinsbeteiligung (Team Stuttgart / Team Stuttgart 2)

#### Scenario: Falsche H4A-Zugangsdaten liefern generischen Fehler
- **WHEN** die H4A-Anmeldung mit den übergebenen Zugangsdaten fehlschlägt
- **THEN** antwortet der Server mit HTTP 502 und `{"error":"h4a_login_failed"}`
- **THEN** enthält die Antwort weder das übermittelte Passwort noch H4A-interne Fehlermeldungen

#### Scenario: Standard-Nutzer ohne Vorstand-Funktion wird abgewiesen
- **WHEN** ein Nutzer ohne Vereinsfunktion `vorstand` und ohne Systemrolle `admin` `POST /api/games/import/h4a/preview` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Zugangsdaten werden nicht persistiert
- **WHEN** ein Preview-Request abgeschlossen ist (Erfolg oder Fehler)
- **THEN** existiert kein Datensatz, keine Datei und kein Log-Eintrag, der das übermittelte Passwort enthält

### Requirement: Idempotenter Diff über die BWHV-Spielnummer

Das System SHALL jedes importierte Spiel über `games.external_id` = BWHV-Spielnummer (`Nr.`)
identifizieren und beim Preview jede Zeile als `new`, `changed` oder `unchanged` einordnen.
Ein Spiel gilt als `changed`, wenn zu einer bestehenden `external_id` mindestens eines der
Felder `date`, `time`, `opponent`, `is_home` oder `venue_id` abweicht.

Das System SHALL aus dem Abruf KEINE Löschungen ableiten — im Abruf fehlende Bestandsspiele
bleiben unangetastet.

#### Scenario: Bekanntes Spiel ohne Änderung
- **WHEN** ein Spiel mit einer `external_id` abgerufen wird, das in `games` bereits mit identischen Feldern existiert
- **THEN** erscheint es im Plan unter `unchanged` und wird bei `apply` nicht verändert

#### Scenario: Verschobenes Spiel wird als Änderung erkannt
- **WHEN** ein Spiel mit bekannter `external_id` abgerufen wird, dessen `time` von der gespeicherten abweicht
- **THEN** erscheint es im Plan unter `changed` mit Alt- und Neu-Wert des Feldes `time`

#### Scenario: Neues Spiel
- **WHEN** ein Spiel mit einer `external_id` abgerufen wird, die in `games` nicht existiert
- **THEN** erscheint es im Plan unter `new`

#### Scenario: Im Abruf fehlendes Bestandsspiel bleibt erhalten
- **WHEN** ein Spiel in `games` eine `external_id` trägt, die im aktuellen Abruf nicht vorkommt
- **THEN** bleibt dieses Spiel unverändert bestehen und erscheint nicht als Löschvorschlag

### Requirement: Zuordnung von Staffel, Spielort und Dienst-Template im Import

Das System SHALL im Preview je Spiel die H4A-Staffel einer TeamWERK-Mannschaft
(`teams.id`), die H4A-Hallennummer einem `venues`-Eintrag (über `venues.hall_number`) und
den Spieltyp (`heim`/`auswärts`) zuordnen. Der `apply`-Endpoint SHALL die Wahl eines
Dienst-Templates je Spieltyp als Batch (ein Template für alle Spiele eines Typs) und selektiv
je Spiel akzeptieren.

Das System SHALL Zeilen, deren Staffel keiner Mannschaft zugeordnet werden kann, im Plan
markieren und beim `apply` überspringen, bis eine Zuordnung vorliegt.

#### Scenario: Bekannte Staffel ist vorbelegt
- **WHEN** eine Staffel abgerufen wird, für die bereits eine Mannschaftszuordnung gelernt wurde
- **THEN** ist die Zielmannschaft im Plan vorbelegt

#### Scenario: Unbekannte Staffel erfordert Zuordnung
- **WHEN** eine Staffel abgerufen wird, für die keine Mannschaftszuordnung existiert
- **THEN** wird die Zeile als „Mannschaft fehlt" markiert und ist ohne Zuordnung nicht importierbar

#### Scenario: Hallennummer wird zu Venue aufgelöst
- **WHEN** ein Spiel eine Hallennummer trägt, für die genau ein Venue mit passender `hall_number` existiert
- **THEN** ist dieses Venue dem Spiel im Plan zugeordnet

#### Scenario: Batch-Template für alle Heimspiele
- **WHEN** der Admin im Apply-Request ein Dienst-Template für den Typ `heim` als Batch wählt und für einzelne Heimspiele keine abweichende Wahl trifft
- **THEN** wird allen als `heim` importierten Spielen dieses Template zugewiesen

#### Scenario: Selektive Template-Wahl überschreibt Batch
- **WHEN** der Admin für ein einzelnes Spiel ein anderes Template als das Batch-Template wählt
- **THEN** wird diesem Spiel das selektiv gewählte Template zugewiesen

### Requirement: Bestätigter Import schreibt Spiele ohne erneuten H4A-Zugriff

Das System SHALL einen Endpoint `POST /api/games/import/h4a/apply` bereitstellen, der die im
Preview bestätigten Entscheidungen entgegennimmt, jede Zeile gegen die Datenbank
re-validiert (aktive Saison vorhanden, Mannschaften und Venues existieren, Template gültig)
und die ausgewählten Spiele in `games` (+ `game_teams`) schreibt. Der Endpoint SHALL dabei
NICHT erneut auf Handball4All zugreifen und KEINE Zugangsdaten benötigen.

Das System SHALL für den gesamten Apply-Lauf genau einen Hub-Broadcast `games` senden und
die Dienst-Slot-Regeneration in einem einzigen `runAutoRegen`-Aufruf über die
Vereinigungsmenge aller betroffenen Datumsfenster ausführen.

#### Scenario: Neue und geänderte Spiele werden geschrieben
- **WHEN** der Admin `POST /api/games/import/h4a/apply` mit bestätigten `new`- und `changed`-Zeilen aufruft
- **THEN** werden neue Spiele eingefügt, geänderte aktualisiert und der Server antwortet mit `{ imported, updated, skipped, regen_summary }`

#### Scenario: Apply ohne aktive Saison schlägt fehl
- **WHEN** keine `seasons`-Zeile mit `is_active=1` existiert und `apply` aufgerufen wird
- **THEN** antwortet der Server mit HTTP 400 und es wird kein Spiel geschrieben

#### Scenario: Ein einziger Broadcast und Regen-Lauf
- **WHEN** ein Apply mehrere Spiele über mehrere Tage schreibt
- **THEN** wird genau ein `games`-Broadcast gesendet und `runAutoRegen` genau einmal über alle betroffenen Datumsfenster ausgeführt

#### Scenario: Nicht zugeordnete Zeile wird übersprungen
- **WHEN** eine Apply-Zeile keine Mannschaftszuordnung trägt
- **THEN** wird sie übersprungen und in `skipped` gezählt, ohne den restlichen Import abzubrechen

