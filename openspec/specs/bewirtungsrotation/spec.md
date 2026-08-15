# bewirtungsrotation Specification

## Purpose
Bewirtungs-/Kuchendienste (Kuchen) werden nicht pro Spiel an das jeweils spielende Team
vergeben, sondern über einen ganzen Spieltag hinweg reihum unter den an dem Tag
spielenden Mannschaften verteilt. Die Capability umfasst die beiden vereinsweiten
Stellschrauben (Bedarf je Spiel, Obergrenze je Mannschaft) samt Einstellungen-UI, den
Rotations-Schalter am Vorlagen-Eintrag und die tagesweite Zuteilung im Auto-Regen.
## Requirements
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

### Requirement: Tagesweite Team-Warteschlange für Rotations-Items

Für jeden Regen-Lauf eines Tages SHALL das System für jede Gruppe rotations-aktivierter Items (gruppiert nach `duty_type_id`) eine Team-Warteschlange aus den Heimspielen des Tages (`event_type='heim'`) aufbauen: Reihenfolge nach chronologischem Anpfiff (`time`, dann `id`), jedes Team genau einmal an der Position seines ersten Heimspiels. Die Warteschlange SHALL bei jedem Regen-Lauf neu aufgebaut werden (kein persistenter, saisonweiter Rotationszustand).

#### Scenario: Team mit mehreren Spielen erscheint einmal

- **WHEN** an einem Spieltag Team A um 9:00 und erneut um 11:00 spielt, Team B um 10:00
- **THEN** ist die Warteschlange `[A, B]`

#### Scenario: Warteschlange startet bei jedem Spieltag neu

- **WHEN** zwei aufeinanderfolgende Spieltage jeweils dieselbe Team-Konstellation haben
- **THEN** beginnt die Zuteilung an beiden Tagen unabhängig voneinander wieder bei Position 1 der jeweiligen Tages-Warteschlange

### Requirement: Einstellungen-UI Tab „Bewirtung"

Das System SHALL unter `/einstellungen` einen Tab **„Heimspieltage"** anzeigen, sichtbar für
Nutzer mit Vereinsfunktion `vorstand` oder System-Rolle `admin` (gleiches Capability-Gate-Muster
wie die übrigen Tabs). Der Tab SHALL in zwei Kacheln gegliedert sein:

- **Kachel „Bewirtung"** mit **beiden** vereinsweiten Bewirtungswerten: dem Spiele-zu-Kuchen-Verhältnis
  („Kuchen je Spiel") und der Obergrenze („Max. Kuchen pro Mannschaft").
- **Kachel „Ausrichter"** mit der editierbaren Ausrichter-Liste inklusive Default-Markierung.

Der Tab SHALL sichtbar ausweisen, dass die Werte die **automatische Dienst-Generierung bei
Heimspielen** steuern — sie wirken nirgends sonst und sind ohne diesen Bezug nicht
selbsterklärend.

Ein bestehender Aufruf mit dem alten Tab-Parameter (`?tab=bewirtung`) SHALL weiterhin auf diesem
Tab landen, damit vorhandene Links und Lesezeichen nicht ins Leere laufen.

#### Scenario: Vorstand sieht und bearbeitet beide Werte

- **WHEN** ein Vorstand den Tab „Heimspieltage" öffnet
- **THEN** werden in der Kachel „Bewirtung" das aktuelle Verhältnis und der aktuelle Cap angezeigt
- **AND** eine Änderung wird nach Speichern über `PUT /api/settings/bewirtung` übernommen

#### Scenario: Ausrichter-Kachel ist im selben Tab erreichbar

- **WHEN** ein Vorstand den Tab „Heimspieltage" öffnet
- **THEN** wird die Ausrichter-Liste mit Markierung des Default-Eintrags angezeigt
- **AND** Einträge lassen sich anlegen, umbenennen, deaktivieren und löschen

#### Scenario: Tab benennt den Wirkungsbereich

- **WHEN** ein Vorstand den Tab „Heimspieltage" öffnet
- **THEN** enthält der Tab einen Hinweis, dass die Einstellungen für die Dienst-Generierung bei Heimspielen gelten

#### Scenario: Alter Tab-Link bleibt gültig

- **WHEN** ein Vorstand `/einstellungen?tab=bewirtung` aufruft
- **THEN** wird der Tab „Heimspieltage" angezeigt

#### Scenario: Kassierer ohne Vorstand-Funktion sieht den Tab nicht

- **WHEN** ein Nutzer mit Vereinsfunktion `kassierer` (ohne `vorstand`) `/einstellungen` öffnet
- **THEN** ist der Tab „Heimspieltage" nicht in der Tab-Leiste sichtbar

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
lassen (ein Slot pro Team des jeweiligen Spiels, mit `slots_total = slots_count`). `1` SHALL
den Rotations-Modus für dieses Item aktivieren; die Obergrenze pro Mannschaft stammt dann aus
der vereinsweiten Einstellung `bewirtung_max_per_team` und **nicht** mehr aus dem Item, und
`slots_count` SHALL ohne Wirkung bleiben (die Personenzahl ergibt sich aus der Zuteilung).

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
- **THEN** entsteht wie bisher ein Slot pro Team des jeweiligen Spiels (`game_teams`) mit `slots_total = slots_count`, ohne Bezug zu einer Warteschlange

#### Scenario: Migration überführt bestehende Caps in den Schalter

- **WHEN** vor der Migration ein Item `rotation_max_per_team=2` trägt und ein zweites `NULL`
- **THEN** hat das erste Item nach der Migration `rotation_enabled=1`, das zweite `rotation_enabled=0`

### Requirement: Vorlagen-Editor zeigt den Rotations-Schalter statt eines Cap-Feldes

Beide Vorlagen-Editoren (Modal auf `/dienstplan-vorlagen` und Detailseite
`/dienstplan-vorlagen/:id`) SHALL pro Item eine Checkbox „Bewirtungsrotation" anbieten, die
`rotation_enabled` schaltet. Ein Zahlenfeld für den Cap SHALL dort NICHT mehr existieren.
Die Checkbox SHALL für den Cap auf die Einstellungen verweisen.

Ist die Checkbox für ein Item gesetzt, SHALL das Feld „Anzahl" (`slots_count`) dieses Items
deaktiviert und als wirkungslos gekennzeichnet werden, weil die Personenzahl eines
Rotations-Slots aus der Zuteilung stammt.

#### Scenario: Checkbox aktiviert die Rotation

- **WHEN** ein Vorstand im Vorlagen-Editor die Checkbox „Bewirtungsrotation" eines Items setzt und speichert
- **THEN** enthält der `PUT`-Body für dieses Item `rotation_enabled: true`

#### Scenario: Editor verweist für den Cap auf die Einstellungen

- **WHEN** ein Vorstand einen Vorlagen-Editor mit gesetzter Checkbox öffnet
- **THEN** wird ein Hinweis angezeigt, dass die maximale Anzahl pro Mannschaft in den Einstellungen unter „Bewirtung" gepflegt wird

#### Scenario: Anzahl-Feld ist bei aktiver Rotation deaktiviert

- **WHEN** ein Vorstand die Checkbox „Bewirtungsrotation" eines Items setzt
- **THEN** ist das Feld „Anzahl" dieses Items deaktiviert
- **AND** ein Hinweis nennt die Zuteilung als Quelle der Personenzahl

### Requirement: Bedarfsermittlung und Kuchen-Zuteilung pro Mannschaft

Das System SHALL den Kuchenbedarf eines Spieltags als `aufgerundet(Anzahl Heimspiele × Verhältnis)` berechnen. Eine Deckelung auf die Anzahl der Heimspiele findet NICHT statt — ein Verhältnis größer eins erhöht den Bedarf entsprechend.

Der Bedarf SHALL greedy in Warteschlangen-Reihenfolge auf Mannschaften verteilt werden: jede Mannschaft erhält `min(bewirtung_max_per_team, verbleibender Bedarf)` Kuchen, bis der Bedarf gedeckt oder die Warteschlange erschöpft ist.

Für jede Mannschaft mit mindestens einem zugeteilten Kuchen SHALL **genau ein** Slot entstehen — an ihrem chronologisch ersten Heimspiel des Tages, also an demselben Spiel, das ihre Position in der Warteschlange bestimmt hat. Der Slot SHALL `slots_total` gleich der Anzahl der zugeteilten Kuchen und `team_id` gleich dieser Mannschaft tragen. Heimspiele, deren Mannschaft keinen Kuchen zugeteilt bekommt, erhalten für dieses Item keinen Slot. `game_template_items.slots_count` SHALL für rotations-aktive Items ignoriert werden.

Ist die Warteschlange erschöpft, bevor der Bedarf gedeckt ist, SHALL der Restbedarf verfallen: es entsteht kein zusätzlicher Slot, keine Mannschaft überschreitet den Cap, und es wird KEIN Slot ohne Team-Zuordnung angelegt. Die Lücke SHALL im `regen_summary` mit Datum, Duty-Type und Anzahl der nicht zugeteilten Kuchen ausgewiesen werden.

Der Cap SHALL einmal pro Regen-Lauf aus `system_settings` gelesen werden und für **alle** rotations-aktiven Items desselben Laufs gelten — unabhängig davon, aus welcher Vorlage ein Item stammt.

In die Bedarfsrechnung SHALL ein Heimspiel nur eingehen, wenn das rotations-aktive Item das
**Ausrichter-Gate** des Tages passiert (`ausrichter_id IS NULL` oder gleich dem aufgelösten
Tages-Ausrichter). Das Gate SHALL damit **vor** Warteschlange und Bedarfsrechnung wirken, nicht
erst beim Einfügen der Slots — andernfalls verbrauchte die Team-Warteschlange Positionen für
Slots, die anschließend verworfen werden, und der ausgewiesene Bedarf wäre falsch.

#### Scenario: Fünf Spiele, vier Teams, Cap zwei

- **WHEN** ein Spieltag fünf Heimspiele hat, das Verhältnis `1` ist, `bewirtung_max_per_team` `2` ist und die Warteschlange `[A, B, C, D]` lautet
- **THEN** entstehen genau drei Rotations-Slots: `A` mit `slots_total=2`, `B` mit `slots_total=2`, `C` mit `slots_total=1`
- **AND** für `D` entsteht kein Slot
- **AND** jeder Slot hängt am chronologisch ersten Heimspiel seiner Mannschaft

#### Scenario: Verhältnis kleiner eins reduziert den Bedarf

- **WHEN** ein Spieltag vier Heimspiele hat, das Verhältnis `0.5` ist, `bewirtung_max_per_team` `2` ist und die Warteschlange `[A, B, C, D]` lautet
- **THEN** ist der Bedarf zwei Kuchen
- **AND** es entsteht genau ein Slot für `A` mit `slots_total=2`
- **AND** für `B`, `C` und `D` entsteht kein Slot

#### Scenario: Verhältnis größer eins erhöht den Bedarf

- **WHEN** ein Spieltag drei Heimspiele hat, das Verhältnis `2` ist, `bewirtung_max_per_team` `2` ist und die Warteschlange `[A, B, C]` lautet
- **THEN** ist der Bedarf sechs Kuchen
- **AND** es entstehen drei Slots mit je `slots_total=2` für `A`, `B` und `C`

#### Scenario: Slot hängt am eigenen Termin der Mannschaft

- **WHEN** an einem Spieltag `A` um 10:00 und `B` um 11:30 ein Heimspiel hat und beide einen Kuchen zugeteilt bekommen
- **THEN** ist der Slot von `A` dem 10:00-Spiel zugeordnet (`game_id`) und seine `event_time` aus `anchor`/`offset_minutes` relativ zu diesem Spiel berechnet
- **AND** der Slot von `B` ist dem 11:30-Spiel zugeordnet

#### Scenario: Mannschaft mit zwei Heimspielen bekommt einen Slot am früheren

- **WHEN** `A` an einem Spieltag um 9:00 und um 13:00 ein Heimspiel hat und Kuchen zugeteilt bekommt
- **THEN** entsteht genau ein Slot, zugeordnet zum 9:00-Spiel

#### Scenario: Restbedarf verfällt, wenn die Warteschlange erschöpft ist

- **WHEN** ein Spieltag fünf Heimspiele hat, das Verhältnis `1` ist, `bewirtung_max_per_team` `2` ist und nur zwei Mannschaften `[A, B]` in der Warteschlange stehen
- **THEN** entstehen genau zwei Slots mit je `slots_total=2` für `A` und `B`
- **AND** es entsteht kein Slot mit `team_id = NULL`
- **AND** weder `A` noch `B` erhält mehr als zwei Kuchen

#### Scenario: Nicht zugeteilte Kuchen sind im Regen-Summary sichtbar

- **WHEN** ein Regen-Lauf den Bedarf eines Tages nicht vollständig zuteilen kann
- **THEN** enthält `regen_summary` eine `unassigned`-Liste mit Datum, Duty-Type und der Anzahl der nicht zugeteilten Kuchen

#### Scenario: slots_count der Vorlage bleibt ohne Wirkung

- **WHEN** ein rotations-aktives Item `slots_count=3` trägt, der Cap `2` ist und einer Mannschaft zwei Kuchen zugeteilt werden
- **THEN** entsteht ein Slot mit `slots_total=2`

#### Scenario: Ein geänderter Cap wirkt sofort für alle Vorlagen

- **WHEN** zwei verschiedene Vorlagen rotations-aktive Items desselben Duty-Types tragen und `bewirtung_max_per_team` auf `3` gesetzt wird
- **THEN** gilt beim nächsten Regen für beide Vorlagen der Cap `3` (keine vorlagenabhängige Abweichung)

#### Scenario: Ausgegatetes Rotations-Item erzeugt keinen Bedarf

- **WHEN** das rotations-aktive Item an Ausrichter `A` gebunden ist und der Spieltag `B` auflöst
- **THEN** ist der Kuchenbedarf des Tages `0`
- **AND** es entsteht kein Rotations-Slot, auch keiner mit `team_id = NULL`

#### Scenario: Teilweise gegatete Vorlagen zählen nur die passenden Spiele

- **WHEN** ein Spieltag vier Heimspiele hat, das Verhältnis `1` ist, und nur zwei davon eine Vorlage tragen, deren rotations-aktives Item das Ausrichter-Gate passiert
- **THEN** beträgt der Bedarf `2` und nicht `4`

### Requirement: Restore-Matching ist für alle Items einheitlich

Bei einem erneuten Regen SHALL die Wiederherstellung bestehender `duty_assignments` für
**alle** Vorlagen-Items — mit und ohne aktivierte Bewirtungsrotation — über
`(duty_type_id, event_time, team_id)` matchen. Für rotations-aktive Items gibt es keine
Sonderbehandlung mehr, weil die Team-Zuordnung eines Rotations-Slots die Mannschaft des
Spiels ist, an dem der Slot hängt, und damit ein stabiles Merkmal des Slots.

Sinkt die zugeteilte Kuchenzahl einer Mannschaft, SHALL das Restore die Zusagen in
aufsteigender Reihenfolge ihrer Entstehung bis `slots_total` zurückschreiben; darüber
hinausgehende Zusagen SHALL regulär als „entfernt" benachrichtigt werden.

#### Scenario: Zusage überlebt einen Regen bei gleichbleibender Zuteilung

- **WHEN** eine Person für den Rotations-Slot ihrer Mannschaft zugesagt hat und ein erneuter Regen dieselbe Zuteilung ergibt
- **THEN** bleibt die Zusage erhalten und es wird keine Benachrichtigung ausgelöst

#### Scenario: Gesunkene Kuchenzahl verliert die jüngste Zusage

- **WHEN** ein Slot mit `slots_total=2` zwei Zusagen trägt und ein erneuter Regen der Mannschaft nur noch einen Kuchen zuteilt
- **THEN** bleibt die ältere Zusage erhalten
- **AND** die jüngere Zusage wird als „entfernt" benachrichtigt

#### Scenario: Nicht mehr herangezogene Mannschaft verliert ihre Zusage

- **WHEN** eine Mannschaft nach einer Spielplanänderung keinen Kuchen mehr zugeteilt bekommt
- **THEN** entsteht für sie kein Slot mehr
- **AND** eine bestehende Zusage wird als „entfernt" benachrichtigt

