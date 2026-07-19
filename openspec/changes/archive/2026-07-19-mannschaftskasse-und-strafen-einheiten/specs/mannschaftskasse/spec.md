## ADDED Requirements

### Requirement: Kassenbuch pro Kader
Das System SHALL pro Kader ein Kassenbuch (`team_cashbook_entries`) mit signierten Einträgen (Einzahlung positiv, Ausgabe negativ), Notiz, Zeitpunkt und Buchendem führen. Der aktuelle Saldo SHALL als SQL-Summe berechnet werden; es SHALL keine denormalisierte Saldo-Spalte geben. Das Kassenbuch SHALL vom Strafenbuch strukturell getrennt sein; eine Kassenbuchung SHALL keine Strafe modifizieren oder als bezahlt markieren.

#### Scenario: Saldo entspricht der Summe der Einträge
- **WHEN** das Kassenbuch drei Einträge enthält: +50,00 €, +30,00 €, −20,00 €
- **THEN** liefert `GET /api/teams/{id}/cashbook` einen Saldo von 60,00 € (6000 Cent)

#### Scenario: Kassenbuchung ändert keine Strafen
- **WHEN** ein Kassenwart eine Einzahlungs-Buchung anlegt, während offene Strafen für Spieler bestehen
- **THEN** bleiben die Strafen der Spieler unverändert (kein automatischer Reset, keine Statusänderung)

### Requirement: Kassenwart-Appointment pro Kader
Das System SHALL die Rolle „Kassenwart" als per-Kader-Appointment führen (`kader_kassenwarte`, Sibling von `kader_strafenwarte` und `kader_trainers`), nicht als globalen `member_club_functions`-Wert. Der Trainer des Kaders SHALL Kassenwarte ernennen und abberufen (`admin` passt immer). Ein Member kann in verschiedenen Kadern unabhängig Kassenwart sein.

#### Scenario: Trainer ernennt einen Kassenwart
- **WHEN** ein Trainer des Kaders einen Spieler zum Kassenwart ernennt
- **THEN** antwortet das System mit 200/201 und der Spieler ist Kassenwart dieses Kaders

#### Scenario: Kein neuer globaler Vereinsfunktions-Wert
- **WHEN** der CHECK-Constraint von `member_club_functions` geprüft wird
- **THEN** enthält er keinen Wert `kassenwart` (die Rolle lebt ausschließlich in `kader_kassenwarte`)

#### Scenario: Nicht-Trainer darf nicht ernennen
- **WHEN** ein Spieler ohne Trainer-Rolle versucht, einen Kassenwart zu ernennen
- **THEN** antwortet das System mit HTTP 403

### Requirement: Kassenbuchung anlegen und löschen
Das System SHALL es dem Trainer **oder** dem Kassenwart des Kaders erlauben, Kassenbuchungen anzulegen (`POST /api/teams/{id}/cashbook`) und einzelne Einträge hart zu löschen (`DELETE /api/teams/{id}/cashbook/{eid}`). Kein Status-Feld, keine Storno-Zustände. Weder Spieler ohne Rolle noch fremde Kassenwarte anderer Teams SHALL buchen dürfen.

#### Scenario: Trainer legt eine Einzahlungs-Buchung an
- **WHEN** ein Trainer eine Buchung „Startgeld Turnier" über +50,00 € anlegt
- **THEN** antwortet das System mit 200/201 und der Eintrag erscheint im Kassenbuch

#### Scenario: Kassenwart legt eine Ausgaben-Buchung an
- **WHEN** ein Kassenwart eine Buchung „Trainergeschenk" über −25,00 € anlegt
- **THEN** antwortet das System mit 200/201 und der Saldo sinkt um 25,00 €

#### Scenario: Normaler Spieler darf nicht buchen
- **WHEN** ein Spieler ohne Trainer- oder Kassenwart-Rolle eine Buchung anzulegen versucht
- **THEN** antwortet das System mit HTTP 403 und keine Row wird angelegt

#### Scenario: Fremd-Team-Kassenwart darf nicht buchen
- **WHEN** ein Kassenwart von Team A eine Buchung für Team B anzulegen versucht
- **THEN** antwortet das System mit HTTP 403 und keine Row wird angelegt

#### Scenario: Eintrag hart löschen
- **WHEN** der Trainer oder Kassenwart einen Eintrag löscht
- **THEN** ist die Row vollständig entfernt und der Saldo entsprechend angepasst; kein „storniert"-Status bleibt zurück

### Requirement: Teaminternes Read-Gate für Mannschaftskasse
Das System SHALL das Kassenbuch nur an Spieler (`kader_members`), Trainer (`kader_trainers`) und Erweiterten Kader (`kader_extended_members`) des Kaders der aktiven Saison ausliefern. Eltern (`family_links`) und Außenstehende SHALL HTTP 403 erhalten. Das Read-Gate SHALL identisch zum Read-Gate für Strafen sein (`canReadCashbook = canReadPenalties`).

#### Scenario: Spieler liest das Kassenbuch
- **WHEN** ein Spieler des Kaders `GET /api/teams/{id}/cashbook` aufruft
- **THEN** antwortet das System mit 200 und liefert Einträge + Saldo

#### Scenario: Erweiterter Kader liest das Kassenbuch
- **WHEN** ein Member des Erweiterten Kaders `GET /api/teams/{id}/cashbook` aufruft
- **THEN** antwortet das System mit 200

#### Scenario: Eltern dürfen die Kasse nicht sehen
- **WHEN** ein Elternteil (nur `family_links`, kein Kader-Member) `GET /api/teams/{id}/cashbook` aufruft
- **THEN** antwortet das System mit HTTP 403

#### Scenario: Außenstehende dürfen die Kasse nicht sehen
- **WHEN** ein authentifizierter Nutzer ohne jede Team-Zugehörigkeit `GET /api/teams/{id}/cashbook` aufruft
- **THEN** antwortet das System mit HTTP 403

#### Scenario: Roster enthält keine Kassendaten
- **WHEN** `GET /api/teams/{id}/roster` aufgerufen wird (auch als Elternteil)
- **THEN** enthält die Response keinerlei Kassenbuch-Einträge, Saldo oder Kassenwart-Details

### Requirement: Live-Update bei Kassen-Änderung
Das System SHALL bei jeder mutierenden Kassen- oder Kassenwart-Aktion einen SSE-Event broadcasten (`cashbook` für Ledger-Mutationen, `treasurers` für Ernennung/Abberufung), damit Team-interne Clients ihre Sicht ohne Reload aktualisieren.

#### Scenario: Buchung triggert cashbook-Event
- **WHEN** eine Kassenbuchung angelegt oder gelöscht wird
- **THEN** wird ein SSE-Event `cashbook` an alle verbundenen Clients gesendet

#### Scenario: Ernennung triggert treasurers-Event
- **WHEN** ein Kassenwart ernannt oder abberufen wird
- **THEN** wird ein SSE-Event `treasurers` gesendet
