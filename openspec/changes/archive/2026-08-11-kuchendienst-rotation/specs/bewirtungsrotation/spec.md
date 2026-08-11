## ADDED Requirements

### Requirement: Vereinsweites Spiele-zu-Kuchen-Verhältnis

Das System SHALL ein vereinsweites, konfigurierbares Verhältnis „Spiele zu Kuchen" als Dezimalzahl in `system_settings` (Key `bewirtung_verhaeltnis`) vorhalten. Bei der Migration MUSS eine Default-Row mit Wert `1` idempotent angelegt werden. Der Wert ist über `GET /api/settings/bewirtung` lesbar und über `PUT /api/settings/bewirtung` (Vorstand/Admin) änderbar; eine erfolgreiche Änderung SHALL `h.hub.Broadcast("settings-changed")` auslösen.

#### Scenario: Migration legt Default-Row idempotent an

- **WHEN** die Migration zweimal hintereinander auf derselben DB ausgeführt wird
- **THEN** existiert genau eine Row mit `key='bewirtung_verhaeltnis'` und `value='1'`, und die zweite Ausführung endet ohne Fehler

#### Scenario: Vorstand ändert das Verhältnis

- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` `PUT /api/settings/bewirtung` mit `{"verhaeltnis": 0.5}` aufruft
- **THEN** antwortet das System mit `200` und dem neuen Wert
- **AND** `system_settings.value` für `bewirtung_verhaeltnis` ist `"0.5"`
- **AND** ein SSE-Event `settings-changed` wird gesendet

#### Scenario: Nicht-Vorstand kann das Verhältnis nicht ändern

- **WHEN** ein Nutzer ohne Vereinsfunktion `vorstand` und ohne System-Rolle `admin` `PUT /api/settings/bewirtung` aufruft
- **THEN** antwortet das System mit `403`

#### Scenario: Negativer oder nicht-numerischer Wert wird abgelehnt

- **WHEN** `PUT /api/settings/bewirtung` mit `{"verhaeltnis": -1}` oder einem nicht-numerischen Wert aufgerufen wird
- **THEN** antwortet das System mit `400`
- **AND** der gespeicherte Wert bleibt unverändert

### Requirement: Max-Kuchen-pro-Team-Cap pro Vorlagen-Item

Das System SHALL ein optionales Feld `rotation_max_per_team` (positive Ganzzahl, nullable) auf `game_template_items` vorhalten. `NULL` (Default) SHALL das bestehende Verhalten unverändert lassen (ein Slot pro Team des jeweiligen Spiels). Ein gesetzter Wert SHALL den Rotations-Modus für dieses Item aktivieren. `PUT /api/admin/duty-templates/{id}` SHALL ein Item mit gesetztem `rotation_max_per_team` UND einem referenzierten `duty_types`-Eintrag, dessen `same_day_behavior` oder `adjacent_day_behavior` ungleich `'normal'` ist, mit `400 rotation_requires_normal_behavior` ablehnen.

#### Scenario: Vorstand aktiviert Rotation für ein Item

- **WHEN** ein Vorstand `PUT /api/admin/duty-templates/{id}` mit einem Item `{duty_type_id: 11, rotation_max_per_team: 2, ...}` sendet, dessen `duty_types`-Zeile `same_day_behavior='normal'` und `adjacent_day_behavior='normal'` hat
- **THEN** antwortet das System mit `200`
- **AND** `game_template_items.rotation_max_per_team` ist `2` für dieses Item

#### Scenario: Rotation mit abweichendem same_day_behavior wird abgelehnt

- **WHEN** ein Vorstand ein Item mit `rotation_max_per_team` gesetzt speichert, dessen Duty-Type `same_day_behavior='skip'` hat
- **THEN** antwortet das System mit `400` und `error=rotation_requires_normal_behavior`
- **AND** keine Änderung wird persistiert

#### Scenario: Bestehende Items ohne Cap bleiben unverändert

- **WHEN** ein Regen für ein Item ohne gesetztes `rotation_max_per_team` läuft
- **THEN** entsteht wie bisher ein Slot pro Team des jeweiligen Spiels (`game_teams`), ohne Bezug zu einer Warteschlange

### Requirement: Tagesweite Team-Warteschlange für Rotations-Items

Für jeden Regen-Lauf eines Tages SHALL das System für jede Gruppe rotations-aktivierter Items (gruppiert nach `duty_type_id`) eine Team-Warteschlange aus den Heimspielen des Tages (`event_type='heim'`) aufbauen: Reihenfolge nach chronologischem Anpfiff (`time`, dann `id`), jedes Team genau einmal an der Position seines ersten Heimspiels. Die Warteschlange SHALL bei jedem Regen-Lauf neu aufgebaut werden (kein persistenter, saisonweiter Rotationszustand).

#### Scenario: Team mit mehreren Spielen erscheint einmal

- **WHEN** an einem Spieltag Team A um 9:00 und erneut um 11:00 spielt, Team B um 10:00
- **THEN** ist die Warteschlange `[A, B]`

#### Scenario: Warteschlange startet bei jedem Spieltag neu

- **WHEN** zwei aufeinanderfolgende Spieltage jeweils dieselbe Team-Konstellation haben
- **THEN** beginnt die Zuteilung an beiden Tagen unabhängig voneinander wieder bei Position 1 der jeweiligen Tages-Warteschlange

### Requirement: Bedarfsermittlung und Greedy-Zuteilung mit Cap

Das System SHALL den Kuchenbedarf eines Spieltags als `min(Anzahl Heimspiele, aufgerundet(Anzahl Heimspiele × Verhältnis))` berechnen. Die chronologisch ersten `Bedarf` Heimspiele des Tages SHALL je einen Rotations-Slot erhalten; weitere Heimspiele erhalten für dieses Item keinen Slot. Die Team-Zuteilung SHALL greedy in Warteschlangen-Reihenfolge erfolgen: die ersten `rotation_max_per_team` Slots gehen an Mannschaft 1, die nächsten an Mannschaft 2, usw. Ist die Warteschlange erschöpft, bevor der Bedarf gedeckt ist, SHALL der verbleibende Slot ohne Team-Zuordnung (`team_id = NULL`) entstehen, statt den Cap zu überschreiten oder eine andere Mannschaft erneut heranzuziehen.

#### Scenario: Fünf Spiele, drei Teams, Cap zwei

- **WHEN** ein Spieltag fünf Heimspiele hat, das Verhältnis `1` ist, der Cap `2` ist und die Warteschlange `[A, B, C]` lautet
- **THEN** entstehen fünf Rotations-Slots mit Team-Zuordnung `A, A, B, B, C`

#### Scenario: Verhältnis kleiner eins reduziert den Bedarf

- **WHEN** ein Spieltag vier Heimspiele hat und das Verhältnis `0.5` ist
- **THEN** entstehen für die chronologisch ersten zwei Heimspiele je ein Rotations-Slot
- **AND** für die übrigen zwei Heimspiele entsteht kein Rotations-Slot für dieses Item

#### Scenario: Verhältnis größer eins wirkt nur bis zur Spieleanzahl

- **WHEN** ein Spieltag drei Heimspiele hat und das Verhältnis `2` ist
- **THEN** entstehen höchstens drei Rotations-Slots (einer pro Spiel), nicht sechs

#### Scenario: Cap-Überlauf lässt Slots unzugeordnet statt den Cap zu verletzen

- **WHEN** ein Spieltag fünf Heimspiele hat, Cap `2` ist und nur zwei Teams `[A, B]` in der Warteschlange stehen
- **THEN** entstehen die Slots mit Team-Zuordnung `A, A, B, B` und ein fünfter Slot mit `team_id = NULL`
- **AND** weder `A` noch `B` erhält mehr als zwei zugeordnete Slots

#### Scenario: Unzugeordnete Slots sind im Regen-Summary sichtbar

- **WHEN** ein Regen-Lauf mindestens einen unzugeordneten Rotations-Slot erzeugt
- **THEN** enthält `regen_summary` eine `unassigned`-Liste mit Datum, Duty-Type und Spiel-ID für jeden dieser Slots

### Requirement: Restore-Matching für Rotations-Items ignoriert die Team-Zuordnung

Bei einem erneuten Regen SHALL die Wiederherstellung bestehender `duty_assignments` für Slots rotations-aktivierter Items ausschließlich über `(duty_type_id, event_time)` matchen, ohne `team_id` einzubeziehen. Für Items ohne aktivierte Rotation SHALL der bestehende Drei-Feld-Match `(duty_type_id, event_time, team_id)` unverändert gelten.

#### Scenario: Zusage überlebt eine team-verschiebende Spielplanänderung

- **WHEN** eine Person für den Rotations-Slot eines Spiels um 10:30 zugesagt hat, dessen Team laut Warteschlange `A` war
- **AND** vor dem Spiel ein neues, früheres Heimspiel eingefügt wird, wodurch der Slot beim nächsten Regen Team `B` zugeteilt bekommt
- **THEN** bleibt die Zusage der Person auf dem Slot um 10:30 erhalten (kein Verlust, keine Benachrichtigung als „entfernt")

#### Scenario: Nicht-Rotations-Item behält das bestehende Matching

- **WHEN** ein Item ohne `rotation_max_per_team` regeneriert wird und sich die Spielplanreihenfolge des Tages ändert
- **THEN** matcht das Restore weiterhin über `(duty_type_id, event_time, team_id)` wie vor diesem Change

### Requirement: Einstellungen-UI Tab „Bewirtung"

Das System SHALL unter `/einstellungen` einen neuen Tab „Bewirtung" anzeigen, sichtbar für Nutzer mit Vereinsfunktion `vorstand` oder System-Rolle `admin` (gleiches Capability-Gate-Muster wie die übrigen Tabs). Der Tab SHALL das aktuelle Spiele-zu-Kuchen-Verhältnis anzeigen und editierbar machen.

#### Scenario: Vorstand sieht und bearbeitet den Bewirtung-Tab

- **WHEN** ein Vorstand `/einstellungen?tab=bewirtung` öffnet
- **THEN** wird das aktuelle Verhältnis angezeigt
- **AND** eine Änderung wird nach Speichern über `PUT /api/settings/bewirtung` übernommen

#### Scenario: Kassierer ohne Vorstand-Funktion sieht den Tab nicht

- **WHEN** ein Nutzer mit Vereinsfunktion `kassierer` (ohne `vorstand`) `/einstellungen` öffnet
- **THEN** ist der Tab „Bewirtung" nicht in der Tab-Leiste sichtbar
