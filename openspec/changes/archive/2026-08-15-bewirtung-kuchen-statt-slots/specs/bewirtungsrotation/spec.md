## MODIFIED Requirements

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

## ADDED Requirements

### Requirement: Bedarfsermittlung und Kuchen-Zuteilung pro Mannschaft

Das System SHALL den Kuchenbedarf eines Spieltags als `aufgerundet(Anzahl Heimspiele × Verhältnis)` berechnen. Eine Deckelung auf die Anzahl der Heimspiele findet NICHT statt — ein Verhältnis größer eins erhöht den Bedarf entsprechend.

Der Bedarf SHALL greedy in Warteschlangen-Reihenfolge auf Mannschaften verteilt werden: jede Mannschaft erhält `min(bewirtung_max_per_team, verbleibender Bedarf)` Kuchen, bis der Bedarf gedeckt oder die Warteschlange erschöpft ist.

Für jede Mannschaft mit mindestens einem zugeteilten Kuchen SHALL **genau ein** Slot entstehen — an ihrem chronologisch ersten Heimspiel des Tages, also an demselben Spiel, das ihre Position in der Warteschlange bestimmt hat. Der Slot SHALL `slots_total` gleich der Anzahl der zugeteilten Kuchen und `team_id` gleich dieser Mannschaft tragen. Heimspiele, deren Mannschaft keinen Kuchen zugeteilt bekommt, erhalten für dieses Item keinen Slot. `game_template_items.slots_count` SHALL für rotations-aktive Items ignoriert werden.

Ist die Warteschlange erschöpft, bevor der Bedarf gedeckt ist, SHALL der Restbedarf verfallen: es entsteht kein zusätzlicher Slot, keine Mannschaft überschreitet den Cap, und es wird KEIN Slot ohne Team-Zuordnung angelegt. Die Lücke SHALL im `regen_summary` mit Datum, Duty-Type und Anzahl der nicht zugeteilten Kuchen ausgewiesen werden.

Der Cap SHALL einmal pro Regen-Lauf aus `system_settings` gelesen werden und für **alle** rotations-aktiven Items desselben Laufs gelten — unabhängig davon, aus welcher Vorlage ein Item stammt.

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

## REMOVED Requirements

### Requirement: Bedarfsermittlung und Greedy-Zuteilung mit Cap

**Reason**: Das Zuteilungsmodell hat gewechselt. Zugeteilt werden jetzt **Kuchen statt Slots**:
der Tagesbedarf wird auf möglichst wenige Mannschaften gebündelt, jede herangezogene Mannschaft
bekommt genau einen Slot mit `slots_total` gleich ihrer Kuchenzahl — an ihrem eigenen Termin
statt am i-ten Spiel des Tages. Damit entfallen zwei Festlegungen dieser Requirement
ersatzlos: die Deckelung des Bedarfs auf die Anzahl der Heimspiele (ein Verhältnis > 1 ist
jetzt ausdrückbar) und der Auffang-Slot ohne Team-Zuordnung (`team_id = NULL`) bei erschöpfter
Warteschlange — der Restbedarf verfällt stattdessen und wird im `regen_summary` ausgewiesen.

**Migration**: Ersetzt durch die Requirement „Bedarfsermittlung und Kuchen-Zuteilung pro Mannschaft". Kein Datenumbau nötig — die
Rotations-Slots werden bei jedem Regen-Lauf neu aufgebaut, das neue Modell greift also ab dem
nächsten Lauf. Bestehende Zusagen an Rotations-Slots werden vom Restore-Matching übernommen,
soweit `slots_total` sie noch trägt.

### Requirement: Restore-Matching für Rotations-Items ignoriert die Team-Zuordnung

**Reason**: Die Ausnahme war nötig, solange die Team-Zuordnung eines Rotations-Slots aus der
Tagesreihenfolge abgeleitet und damit durch Spielplanänderungen verschiebbar war. Seit ein
Rotations-Slot am eigenen Termin der zugeteilten Mannschaft hängt, ist `team_id` wieder ein
stabiles Merkmal — die Ausnahme würde nun Slots verschiedener Mannschaften zur selben Uhrzeit
verwechseln.

**Migration**: Ersetzt durch die Requirement „Restore-Matching ist für alle Items
einheitlich". Kein Datenumbau nötig; das geänderte Matching wirkt ab dem nächsten Regen-Lauf.
