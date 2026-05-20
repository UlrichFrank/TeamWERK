## ADDED Requirements

### Requirement: Spieltag-Detail zeigt Zeitleiste der verknüpften Dienste
Das System SHALL für jeden Spieltag eine Detailansicht anzeigen mit: Spiel-Metadaten (Datum, Uhrzeit, Gegner, Mannschaft) und einer chronologischen Zeitleiste aller verknüpften Duty Slots mit Besetzungsstand.

#### Scenario: Spieltag-Detail aufrufen
- **WHEN** ein authentifizierter Admin oder Trainer auf einen Spieltag im Kalender klickt
- **THEN** wird die Route `/spielplan/:gameId` geladen mit Spiel-Metadaten und einer Liste der Duty Slots sortiert nach Event-Zeit

#### Scenario: Besetzungsstand pro Slot anzeigen
- **WHEN** die Detailansicht eines Spieltags geladen wird
- **THEN** zeigt jeder Slot: Diensttyp-Name, Uhrzeit, Rollenbezeichnung, Fortschrittsanzeige `slots_filled / slots_total`

#### Scenario: Spiel ohne verknüpfte Slots
- **WHEN** ein Spieltag keine Duty Slots hat (kein Template vorhanden war)
- **THEN** wird ein Hinweis „Keine Dienste für dieses Spiel angelegt" angezeigt

### Requirement: Admin kann Dienste aus der Detailansicht heraus anlegen
Das System SHALL Admins erlauben, direkt aus der Spieltag-Detailansicht heraus zusätzliche Duty Slots anzulegen, die automatisch mit dem Spiel verknüpft werden.

#### Scenario: Manuellen Dienst zum Spiel hinzufügen
- **WHEN** ein Admin in der Detailansicht auf „+ Dienst anlegen" klickt und das Formular ausfüllt
- **THEN** wird ein neuer Duty Slot mit `game_id = spielId` angelegt und erscheint sofort in der Zeitleiste

### Requirement: API liefert Spieldetails mit Slot-Aggregation
Das System SHALL `GET /api/games/{id}` bereitstellen, der Spiel-Metadaten und alle verknüpften Duty Slots mit Besetzungsstand zurückgibt.

#### Scenario: Spieldetail-API
- **WHEN** ein authentifizierter Admin oder Trainer `GET /api/games/{id}` aufruft
- **THEN** antwortet das System mit HTTP 200 und: game-Objekt + slots-Array mit id, duty_type_name, event_time, role_description, slots_total, slots_filled

#### Scenario: Spiel nicht gefunden
- **WHEN** `GET /api/games/{id}` mit einer nicht existierenden ID aufgerufen wird
- **THEN** antwortet das System mit HTTP 404
