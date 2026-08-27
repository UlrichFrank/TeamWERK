## ADDED Requirements

### Requirement: Ein Dienst-Slot trägt seine eigene Dauer

Jeder `duty_slots`-Eintrag SHALL eine Spalte `hours_value` (`REAL`, Stunden, `NOT NULL`,
`> 0`) tragen. Dieser Wert SHALL zugleich die angezeigte Dauer **und** die Gutschrift auf
das Dienstkonto sein — es SHALL keine zweite Zahl für dieselbe Größe geben.

Der Wert SHALL beim Erzeugen des Slots **materialisiert** werden, wie `event_time`,
`slots_total` und `audiences` es bereits werden. Ein Slot SHALL zur Laufzeit weder die
Vorlage noch den Diensttyp nach seiner Dauer befragen.

`duty_accounts.ist` SHALL aus `SUM(duty_slots.hours_value)` der `fulfilled`-Zuweisungen
aggregiert werden, nicht aus `duty_types.hours_value`.

#### Scenario: Dauer eines Slots weicht vom Diensttyp ab

- **WHEN** ein Vorstand die Dauer eines Slots von 1,0 auf 2,0 Stunden ändert
- **THEN** zeigt die Dienstbörse für diesen Slot die längere Spanne
- **AND** rechnet eine Neuberechnung von `duty_accounts.ist` mit 2,0 Stunden für diesen Slot
- **AND** bleibt die Dauer des zugrundeliegenden Diensttyps unverändert

#### Scenario: Spätere Änderung am Diensttyp erreicht bestehende Slots nicht

- **WHEN** die Dauer eines Diensttyps nach dem Erzeugen von Slots geändert wird
- **THEN** behalten die bestehenden Slots ihre materialisierte Dauer
- **AND** tragen erst neu erzeugte Slots den geänderten Wert

### Requirement: Dienstbörse zeigt Start- und Endzeit

Wo heute der Startzeitpunkt eines Dienst-Slots angezeigt wird, SHALL eine Zeitspanne
`Start–Ende` erscheinen, gebildet aus `event_time` und `hours_value`. Das betrifft die
Dienstbörse (`/dienste`) und die Dienst-Liste im Spieltag-Detail-Modal (`/kalender`).

Die Spanne SHALL mit einem Halbgeviertstrich getrennt und **ohne** nachgestelltes „Uhr"
dargestellt werden. Läuft ein Dienst über Mitternacht, SHALL die Endzeit als Uhrzeit des
Folgetags **ohne** Datumszusatz erscheinen. Trägt ein Slot **kein** `event_time`, SHALL die
bisherige Platzhalter-Darstellung erhalten bleiben.

Die Anstoßzeit des Spiels in der Spieltag-Kopfzeile SHALL unverändert als Zeitpunkt
angezeigt werden — sie ist keine Dienstzeit.

#### Scenario: Dienst mit Startzeit und Dauer

- **WHEN** ein Slot `event_time = 08:00` und eine Dauer von 1,0 Stunden trägt
- **THEN** zeigt die Dienstbörse `8:00–9:00`

#### Scenario: Dienst läuft über Mitternacht

- **WHEN** ein Slot `event_time = 23:30` und eine Dauer von 1,0 Stunden trägt
- **THEN** zeigt die Dienstbörse `23:30–00:30`
- **AND** erscheint kein Datumszusatz

#### Scenario: Dienst ohne Startzeit

- **WHEN** ein Slot kein `event_time` trägt
- **THEN** bleibt die Darstellung der bisherige Platzhalter
- **AND** wird keine Spanne konstruiert

### Requirement: Dauer ist beim Anlegen und Bearbeiten eines Slots editierbar

Die Modale „Dienst hinzufügen" und „Dienst bearbeiten" SHALL neben Uhrzeit und Personenzahl
ein Feld für die Dauer anbieten. `POST /api/duty-slots` und `PUT /api/duty-slots/{id}` SHALL
`hours_value` entgegennehmen.

Beide Routen SHALL `hours_value <= 0` mit HTTP 400 ablehnen. Fehlt `hours_value` im Request
von `POST /api/duty-slots`, SHALL der Server die Dauer des angegebenen `duty_type_id`
einsetzen statt 0 zu speichern.

`PUT /api/duty-slots/{id}` SHALL wie bisher `is_custom=1` setzen — auch dann, wenn
ausschließlich die Dauer geändert wurde.

#### Scenario: Dauer beim Anlegen mitgeben

- **WHEN** ein Vorstand über „+ Dienst hinzufügen" einen Slot mit Dauer anlegt
- **THEN** antwortet die Route mit 201
- **AND** trägt der Slot die angegebene Dauer und `is_custom = 1`

#### Scenario: Alt-Client sendet keine Dauer

- **WHEN** `POST /api/duty-slots` ohne `hours_value` aufgerufen wird
- **THEN** trägt der angelegte Slot die Dauer seines Diensttyps
- **AND** wird nicht 0 gespeichert

#### Scenario: Unzulässige Dauer

- **WHEN** `hours_value` als 0 oder negativ gesendet wird
- **THEN** antwortet die Route mit HTTP 400
- **AND** bleibt der Slot unverändert

### Requirement: Dienst hinzufügen belegt Dauer und Uhrzeit aus dem Diensttyp vor

Wählt ein Vorstand im Modal „Dienst hinzufügen" einen Diensttyp, SHALL das Formular Dauer
und Uhrzeit aus diesem Typ vorbelegen — die Uhrzeit berechnet aus `default_anchor` und
`default_offset_minutes` gegen die Zeit des Termins. Beide Werte SHALL editierbar bleiben.
Die bestehende Vorbelegung der Zielgruppen SHALL erhalten bleiben.

#### Scenario: Diensttyp auswählen

- **WHEN** ein Vorstand im Modal einen Diensttyp mit Anker „Start" und Versatz −30 Minuten
  an einem Termin um 10:00 auswählt
- **THEN** steht im Uhrzeit-Feld 09:30
- **AND** steht im Dauer-Feld die Dauer dieses Diensttyps
- **AND** sind beide Felder weiterhin überschreibbar

### Requirement: Vorlagen-Zeile trägt eine eigene Dauer (Copy-on-pick)

`game_template_items` SHALL eine Spalte `hours_value` (`REAL`, `NOT NULL`) tragen. Der
Vorlagen-Editor SHALL sie beim Auswählen des Diensttyps aus dessen Wert vorbelegen — auf
demselben Weg, auf dem er heute `anchor` und `offset_minutes` vorbelegt. Danach SHALL die
Zeile eigenständig sein: ein leeres Feld als „erben" SHALL es **nicht** geben.

Ein aus der Vorlage erzeugter Slot SHALL die Dauer der **Vorlagen-Zeile** übernehmen.

#### Scenario: Diensttyp in der Vorlage auswählen

- **WHEN** ein Vorstand in einer Vorlagen-Zeile einen Diensttyp auswählt
- **THEN** wird das Dauer-Feld der Zeile mit der Dauer dieses Typs gefüllt
- **AND** kann es anschließend abweichend gesetzt werden

#### Scenario: Vorlage weicht vom Diensttyp ab

- **WHEN** eine Vorlagen-Zeile eine von ihrem Diensttyp abweichende Dauer trägt
- **THEN** tragen die daraus erzeugten Slots die Dauer der Vorlagen-Zeile

### Requirement: Bei reduzierter Variante bestimmt der Varianten-Typ die Dauer

Ersetzt die Varianten-Logik (`same_day_behavior` oder `adjacent_day_behavior` = `reduced`)
den Diensttyp eines Vorlagen-Items durch den hinterlegten Varianten-Typ, SHALL der erzeugte
Slot die Dauer des **Varianten-Typs** tragen. Die in der Vorlagen-Zeile hinterlegte Dauer
SHALL in diesem Fall verworfen werden.

Begründung: Die Vorlage bestimmt Position und Umfang eines Dienstes, der Diensttyp bestimmt
die Art der Arbeit — die Dauer ist eine Eigenschaft der Arbeit, nicht der Platzierung.

Position (`anchor`, `offset_minutes`) und Personenzahl (`slots_count`) SHALL unverändert aus
der Vorlagen-Zeile stammen.

#### Scenario: Mehrfachspieltag löst Variante aus

- **WHEN** an einem Mehrfachspieltag `same_day_behavior = 'reduced'` greift und den
  Diensttyp durch seine Variante ersetzt
- **THEN** trägt der erzeugte Slot die Dauer des Varianten-Typs
- **AND** nicht die Dauer der Vorlagen-Zeile
- **AND** stammen Uhrzeit und Personenzahl weiterhin aus der Vorlagen-Zeile

### Requirement: Eine Dauer-Änderung erhält bestehende Zusagen

Eine geänderte Dauer SHALL den Zeitpunkt eines Slots nicht verschieben und damit den
Wiederherstellungs-Schlüssel `(duty_type_id, event_time, team_id)` nicht berühren. Ein
Regen-Lauf nach einer Dauer-Änderung SHALL alle Zusagen wiederherstellen.

#### Scenario: Regen nach Dauer-Änderung in der Vorlage

- **WHEN** die Dauer einer Vorlagen-Zeile geändert und ein Regen-Lauf ausgelöst wird
- **THEN** tragen die neu erzeugten Slots die geänderte Dauer
- **AND** sind alle zuvor bestehenden Zusagen wiederhergestellt
- **AND** erscheint keine „entfernt"-Benachrichtigung
