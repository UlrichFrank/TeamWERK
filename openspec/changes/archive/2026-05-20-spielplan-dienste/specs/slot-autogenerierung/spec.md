## ADDED Requirements

### Requirement: Vorschau der zu generierenden Slots vor Bestätigung
Das System SHALL beim Anlegen eines Spiels eine Vorschau der zu generierenden Duty Slots anzeigen, bevor der Admin bestätigt. Der Admin kann die Generierung bestätigen, einzelne Vorschau-Items entfernen oder die Generierung überspringen.

#### Scenario: Vorschau mit aktivem Template
- **WHEN** ein Admin das „Spiel anlegen"-Formular ausgefüllt hat und ein aktives Template existiert
- **THEN** zeigt das Frontend eine Vorschauliste der zu generierenden Slots (Diensttyp, berechnete Uhrzeit, Personenanzahl) bevor das Formular abgesendet wird

#### Scenario: Admin entfernt einzelne Slots aus der Vorschau
- **WHEN** ein Admin in der Vorschau einen Slot-Eintrag entfernt und dann bestätigt
- **THEN** wird das Spiel mit nur den verbleibenden Vorschau-Slots angelegt; der entfernte Slot wird nicht generiert

#### Scenario: Admin überspringt Generierung
- **WHEN** ein Admin die Option „Ohne Dienste anlegen" wählt
- **THEN** wird das Spiel ohne Duty Slots gespeichert; Slots können danach manuell ergänzt werden

#### Scenario: Kein Template — Vorschau entfällt
- **WHEN** kein aktives Template existiert
- **THEN** entfällt der Vorschau-Schritt; das Spiel wird direkt ohne Slots angelegt

### Requirement: Duty Slots werden beim Anlegen eines Spiels automatisch generiert
Das System SHALL beim Bestätigen des Spiel-Anlegen-Formulars alle vom Admin ausgewählten Vorschau-Slots in einer Datenbank-Transaktion erzeugen und mit dem Spiel verknüpfen (`game_id`). Die Liste der zu generierenden Items wird vom Frontend übergeben (nicht serverseitig aus dem Template abgeleitet), sodass Anpassungen aus der Vorschau wirksam werden.

#### Scenario: Slots aus bestätigter Vorschau generieren
- **WHEN** ein Admin `POST /api/admin/games` mit `slots`-Array (aus Vorschau) aufruft
- **THEN** werden genau diese Slots angelegt mit: event_date = Spieldatum, event_time berechnet, team_id = Spiel-Team, game_id = neue Spiel-ID, season_id = Spiel-Saison; HTTP 201

#### Scenario: Spiel ohne Slots anlegen
- **WHEN** `POST /api/admin/games` mit leerem `slots`-Array aufgerufen wird
- **THEN** wird das Spiel ohne Duty Slots gespeichert, HTTP 201

#### Scenario: Atomarität — Fehler rollt Spiel zurück
- **WHEN** die Transaktion beim Anlegen eines Duty Slots fehlschlägt
- **THEN** wird auch das Spiel nicht gespeichert und HTTP 500 zurückgegeben

### Requirement: Slots können manuell neu generiert werden (Overwrite)
Das System SHALL Admins erlauben, die Slot-Generierung aus dem Template für ein bestehendes Spiel nachträglich manuell auszulösen. Dabei können bestehende unbesetzte Slots des Spiels überschrieben werden.

#### Scenario: Neugen erierung aus Spieltag-Detail
- **WHEN** ein Admin in der Spieltag-Detailansicht auf „Dienste neu generieren" klickt
- **THEN** zeigt das System eine Vorschau der zu generierenden Slots (analog zum Anlegen-Dialog)

#### Scenario: Overwrite — bestehende unbesetzte Slots werden gelöscht
- **WHEN** ein Admin die Neugenerierung bestätigt und das Spiel bereits unbesetzte Slots hat
- **THEN** werden die bestehenden Slots mit `slots_filled = 0` gelöscht und durch die neuen Slots ersetzt; die Transaktion ist atomar

#### Scenario: Overwrite — besetzte Slots bleiben erhalten
- **WHEN** ein Admin die Neugenerierung bestätigt und das Spiel bereits Slots mit `slots_filled > 0` hat
- **THEN** werden diese belegten Slots nicht gelöscht; nur Slots mit `slots_filled = 0` werden ersetzt; das System zeigt eine Warnung über beibehaltene belegte Slots

#### Scenario: Neugenerierung ohne Template
- **WHEN** kein aktives Template existiert und Admin „Dienste neu generieren" klickt
- **THEN** zeigt das System die Meldung „Kein Template konfiguriert" und bricht ab

### Requirement: Generierte Slots sind nachträglich einzeln bearbeitbar
Duty Slots die durch Auto-Generierung entstanden sind, unterscheiden sich nicht von manuell angelegten Slots. Sie können über die bestehende Duty-Slot-Verwaltung bearbeitet, ergänzt und gelöscht werden. Zusätzlich kann aus der Spieltag-Detailansicht direkt ein neuer Slot zum Spiel hinzugefügt werden.

#### Scenario: Generierter Slot wird manuell angepasst
- **WHEN** ein Admin einen generierten Duty Slot via `PUT /api/duty-slots/{id}` anpasst (z.B. Uhrzeit, Personenanzahl)
- **THEN** wird der Slot aktualisiert; die `game_id`-Verknüpfung bleibt erhalten

#### Scenario: Zusätzlichen Slot zum Spiel hinzufügen
- **WHEN** ein Admin in der Spieltag-Detailansicht „+ Dienst hinzufügen" klickt und ein Formular ausfüllt
- **THEN** wird ein neuer Duty Slot mit `game_id = spielId` angelegt und erscheint sofort in der Zeitleiste

#### Scenario: Einzelner Slot wird gelöscht
- **WHEN** ein Admin einen Slot in der Spieltag-Detailansicht löscht
- **THEN** wird nur dieser Slot entfernt; das Spiel und alle anderen Slots bleiben unverändert
