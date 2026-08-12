## ADDED Requirements

### Requirement: Vereinsweiter Cap „Max. Kuchen pro Mannschaft"

Das System SHALL eine vereinsweite, konfigurierbare Obergrenze „Max. Kuchen pro Mannschaft"
als positive Ganzzahl in `system_settings` (Key `bewirtung_max_per_team`) vorhalten. Die
Migration MUSS die Row idempotent anlegen und ihren Startwert aus dem größten bereits
konfigurierten Item-Cap übernehmen (Fallback `1`, wenn keiner existiert).

Der Wert SHALL über `GET /api/settings/bewirtung` gemeinsam mit dem Verhältnis lesbar und
über `PUT /api/settings/bewirtung` (Vorstand/Admin) änderbar sein. `PUT` SHALL beide Felder
unabhängig voneinander akzeptieren: ein im Body fehlendes Feld SHALL unverändert bleiben.
Ein Wert `<= 0` SHALL mit `400 invalid_max_per_team` abgelehnt werden, ohne dass **eines**
der beiden Felder persistiert wird.

#### Scenario: Migration übernimmt den größten bestehenden Item-Cap

- **WHEN** vor der Migration Vorlagen-Items mit `rotation_max_per_team` `1` und `3` existieren
- **THEN** hat `system_settings` nach der Migration genau eine Row `bewirtung_max_per_team` mit Wert `'3'`

#### Scenario: Migration ohne konfigurierte Rotation setzt den Default

- **WHEN** vor der Migration kein Item einen `rotation_max_per_team` trägt
- **THEN** hat `system_settings` nach der Migration eine Row `bewirtung_max_per_team` mit Wert `'1'`

#### Scenario: Vorstand ändert den Cap

- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` `PUT /api/settings/bewirtung` mit `{"max_per_team": 2}` aufruft
- **THEN** antwortet das System mit `200`
- **AND** `system_settings.value` für `bewirtung_max_per_team` ist `"2"`
- **AND** der gespeicherte Wert von `bewirtung_verhaeltnis` ist unverändert
- **AND** ein SSE-Event `settings-changed` wird gesendet

#### Scenario: Cap kleiner gleich null wird abgelehnt

- **WHEN** `PUT /api/settings/bewirtung` mit `{"verhaeltnis": 0.5, "max_per_team": 0}` aufgerufen wird
- **THEN** antwortet das System mit `400` und `error=invalid_max_per_team`
- **AND** sowohl `bewirtung_max_per_team` als auch `bewirtung_verhaeltnis` bleiben unverändert

#### Scenario: Nicht-Vorstand kann den Cap nicht ändern

- **WHEN** ein Nutzer ohne Vereinsfunktion `vorstand` und ohne System-Rolle `admin` `PUT /api/settings/bewirtung` mit `{"max_per_team": 2}` aufruft
- **THEN** antwortet das System mit `403`

### Requirement: Rotations-Schalter pro Vorlagen-Item

Das System SHALL ein Feld `rotation_enabled` (INTEGER NOT NULL DEFAULT 0) auf
`game_template_items` vorhalten. `0` (Default) SHALL das bestehende Verhalten unverändert
lassen (ein Slot pro Team des jeweiligen Spiels). `1` SHALL den Rotations-Modus für dieses
Item aktivieren; die Obergrenze pro Mannschaft stammt dann aus der vereinsweiten Einstellung
`bewirtung_max_per_team` und **nicht** mehr aus dem Item.

`PUT /api/admin/duty-templates/{id}` SHALL ein Item mit `rotation_enabled=true` UND einem
referenzierten `duty_types`-Eintrag, dessen `same_day_behavior` oder
`adjacent_day_behavior` ungleich `'normal'` ist, mit `400 rotation_requires_normal_behavior`
ablehnen.

#### Scenario: Vorstand aktiviert Rotation für ein Item

- **WHEN** ein Vorstand `PUT /api/admin/duty-templates/{id}` mit einem Item `{duty_type_id: 11, rotation_enabled: true, ...}` sendet, dessen `duty_types`-Zeile `same_day_behavior='normal'` und `adjacent_day_behavior='normal'` hat
- **THEN** antwortet das System mit `200`
- **AND** `game_template_items.rotation_enabled` ist `1` für dieses Item

#### Scenario: Rotation mit abweichendem same_day_behavior wird abgelehnt

- **WHEN** ein Vorstand ein Item mit `rotation_enabled=true` speichert, dessen Duty-Type `same_day_behavior='skip'` hat
- **THEN** antwortet das System mit `400` und `error=rotation_requires_normal_behavior`
- **AND** keine Änderung wird persistiert

#### Scenario: Bestehende Items ohne Rotation bleiben unverändert

- **WHEN** ein Regen für ein Item mit `rotation_enabled=0` läuft
- **THEN** entsteht wie bisher ein Slot pro Team des jeweiligen Spiels (`game_teams`), ohne Bezug zu einer Warteschlange

#### Scenario: Migration überführt bestehende Caps in den Schalter

- **WHEN** vor der Migration ein Item `rotation_max_per_team=2` trägt und ein zweites `NULL`
- **THEN** hat das erste Item nach der Migration `rotation_enabled=1`, das zweite `rotation_enabled=0`

### Requirement: Vorlagen-Editor zeigt den Rotations-Schalter statt eines Cap-Feldes

Beide Vorlagen-Editoren (Modal auf `/dienstplan-vorlagen` und Detailseite
`/dienstplan-vorlagen/:id`) SHALL pro Item eine Checkbox „Bewirtungsrotation" anbieten, die
`rotation_enabled` schaltet. Ein Zahlenfeld für den Cap SHALL dort NICHT mehr existieren.
Die Checkbox SHALL für den Cap auf die Einstellungen verweisen.

#### Scenario: Checkbox aktiviert die Rotation

- **WHEN** ein Vorstand im Vorlagen-Editor die Checkbox „Bewirtungsrotation" eines Items setzt und speichert
- **THEN** enthält der `PUT`-Body für dieses Item `rotation_enabled: true`

#### Scenario: Editor verweist für den Cap auf die Einstellungen

- **WHEN** ein Vorstand einen Vorlagen-Editor mit gesetzter Checkbox öffnet
- **THEN** wird ein Hinweis angezeigt, dass die maximale Anzahl pro Mannschaft in den Einstellungen unter „Bewirtung" gepflegt wird

## REMOVED Requirements

### Requirement: Max-Kuchen-pro-Team-Cap pro Vorlagen-Item
**Reason:** Der Cap ist eine vereinsweite Bewirtungsregel, keine Eigenschaft einer einzelnen Dienstplan-Vorlage. Als Item-Feld trug er zwei Bedeutungen gleichzeitig (Schalter + Wert) und erlaubte divergierende Caps für denselben Duty-Type in verschiedenen Vorlagen, die die Engine still per „erster gewinnt" auflöste.
**Migration:** Ersetzt durch die Einstellung `bewirtung_max_per_team` (Requirement „Vereinsweiter Cap „Max. Kuchen pro Mannschaft"") plus den booleschen Item-Schalter `rotation_enabled` (Requirement „Rotations-Schalter pro Vorlagen-Item"). Migration `046` überführt bestehende Daten verlustfrei.

## MODIFIED Requirements

### Requirement: Bedarfsermittlung und Greedy-Zuteilung mit Cap

Das System SHALL den Kuchenbedarf eines Spieltags als `min(Anzahl Heimspiele, aufgerundet(Anzahl Heimspiele × Verhältnis))` berechnen. Die chronologisch ersten `Bedarf` Heimspiele des Tages SHALL je einen Rotations-Slot erhalten; weitere Heimspiele erhalten für dieses Item keinen Slot. Die Team-Zuteilung SHALL greedy in Warteschlangen-Reihenfolge erfolgen: die ersten `bewirtung_max_per_team` Slots gehen an Mannschaft 1, die nächsten an Mannschaft 2, usw. Der Cap SHALL einmal pro Regen-Lauf aus `system_settings` gelesen werden und für **alle** rotations-aktiven Items desselben Laufs gelten — unabhängig davon, aus welcher Vorlage ein Item stammt. Ist die Warteschlange erschöpft, bevor der Bedarf gedeckt ist, SHALL der verbleibende Slot ohne Team-Zuordnung (`team_id = NULL`) entstehen, statt den Cap zu überschreiten oder eine andere Mannschaft erneut heranzuziehen.

#### Scenario: Fünf Spiele, drei Teams, Cap zwei

- **WHEN** ein Spieltag fünf Heimspiele hat, das Verhältnis `1` ist, `bewirtung_max_per_team` `2` ist und die Warteschlange `[A, B, C]` lautet
- **THEN** entstehen fünf Rotations-Slots mit Team-Zuordnung `A, A, B, B, C`

#### Scenario: Verhältnis kleiner eins reduziert den Bedarf

- **WHEN** ein Spieltag vier Heimspiele hat und das Verhältnis `0.5` ist
- **THEN** entstehen für die chronologisch ersten zwei Heimspiele je ein Rotations-Slot
- **AND** für die übrigen zwei Heimspiele entsteht kein Rotations-Slot für dieses Item

#### Scenario: Verhältnis größer eins wirkt nur bis zur Spieleanzahl

- **WHEN** ein Spieltag drei Heimspiele hat und das Verhältnis `2` ist
- **THEN** entstehen höchstens drei Rotations-Slots (einer pro Spiel), nicht sechs

#### Scenario: Cap-Überlauf lässt Slots unzugeordnet statt den Cap zu verletzen

- **WHEN** ein Spieltag fünf Heimspiele hat, `bewirtung_max_per_team` `2` ist und nur zwei Teams `[A, B]` in der Warteschlange stehen
- **THEN** entstehen die Slots mit Team-Zuordnung `A, A, B, B` und ein fünfter Slot mit `team_id = NULL`
- **AND** weder `A` noch `B` erhält mehr als zwei zugeordnete Slots

#### Scenario: Ein geänderter Cap wirkt sofort für alle Vorlagen

- **WHEN** zwei verschiedene Vorlagen rotations-aktive Items desselben Duty-Types tragen und `bewirtung_max_per_team` auf `3` gesetzt wird
- **THEN** gilt beim nächsten Regen für beide Vorlagen der Cap `3` (keine vorlagenabhängige Abweichung)

#### Scenario: Unzugeordnete Slots sind im Regen-Summary sichtbar

- **WHEN** ein Regen-Lauf mindestens einen unzugeordneten Rotations-Slot erzeugt
- **THEN** enthält `regen_summary` eine `unassigned`-Liste mit Datum, Duty-Type und Spiel-ID für jeden dieser Slots

### Requirement: Restore-Matching für Rotations-Items ignoriert die Team-Zuordnung

Bei einem erneuten Regen SHALL die Wiederherstellung bestehender `duty_assignments` für Slots rotations-aktivierter Items (`rotation_enabled=1`) ausschließlich über `(duty_type_id, event_time)` matchen, ohne `team_id` einzubeziehen. Für Items ohne aktivierte Rotation SHALL der bestehende Drei-Feld-Match `(duty_type_id, event_time, team_id)` unverändert gelten.

#### Scenario: Zusage überlebt eine team-verschiebende Spielplanänderung

- **WHEN** eine Person für den Rotations-Slot eines Spiels um 10:30 zugesagt hat, dessen Team laut Warteschlange `A` war
- **AND** vor dem Spiel ein neues, früheres Heimspiel eingefügt wird, wodurch der Slot beim nächsten Regen Team `B` zugeteilt bekommt
- **THEN** bleibt die Zusage der Person auf dem Slot um 10:30 erhalten (kein Verlust, keine Benachrichtigung als „entfernt")

#### Scenario: Nicht-Rotations-Item behält das bestehende Matching

- **WHEN** ein Item mit `rotation_enabled=0` regeneriert wird und sich die Spielplanreihenfolge des Tages ändert
- **THEN** matcht das Restore weiterhin über `(duty_type_id, event_time, team_id)`

### Requirement: Einstellungen-UI Tab „Bewirtung"

Das System SHALL unter `/einstellungen` einen Tab „Bewirtung" anzeigen, sichtbar für Nutzer
mit Vereinsfunktion `vorstand` oder System-Rolle `admin` (gleiches Capability-Gate-Muster wie
die übrigen Tabs). Der Tab SHALL **beide** vereinsweiten Bewirtungswerte anzeigen und
editierbar machen: das Spiele-zu-Kuchen-Verhältnis („Kuchen je Spiel") und die Obergrenze
(„Max. Kuchen pro Mannschaft").

Der Tab SHALL sichtbar ausweisen, dass beide Werte die **automatische Dienst-Generierung bei
Heimspielen** steuern — die Zahlen wirken nirgends sonst und sind ohne diesen Bezug nicht
selbsterklärend.

#### Scenario: Vorstand sieht und bearbeitet beide Werte

- **WHEN** ein Vorstand `/einstellungen?tab=bewirtung` öffnet
- **THEN** werden das aktuelle Verhältnis und der aktuelle Cap angezeigt
- **AND** eine Änderung wird nach Speichern über `PUT /api/settings/bewirtung` übernommen

#### Scenario: Tab benennt den Wirkungsbereich

- **WHEN** ein Vorstand `/einstellungen?tab=bewirtung` öffnet
- **THEN** enthält der Tab einen Hinweis, dass die Werte für die Dienst-Generierung bei Heimspielen gelten

#### Scenario: Kassierer ohne Vorstand-Funktion sieht den Tab nicht

- **WHEN** ein Nutzer mit Vereinsfunktion `kassierer` (ohne `vorstand`) `/einstellungen` öffnet
- **THEN** ist der Tab „Bewirtung" nicht in der Tab-Leiste sichtbar
