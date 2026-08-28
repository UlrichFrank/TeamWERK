## MODIFIED Requirements

### Requirement: Bedarfsermittlung und Kuchen-Zuteilung pro Mannschaft

Das System SHALL den Kuchenbedarf eines Spieltags als `aufgerundet(Anzahl Heimspiele × Verhältnis)` berechnen. Eine Deckelung auf die Anzahl der Heimspiele findet NICHT statt — ein Verhältnis größer eins erhöht den Bedarf entsprechend.

Der Bedarf SHALL greedy in Warteschlangen-Reihenfolge auf Mannschaften verteilt werden: jede Mannschaft erhält `min(bewirtung_max_per_team, verbleibender Bedarf)` Kuchen, bis der Bedarf gedeckt oder die Warteschlange erschöpft ist.

Für jede Mannschaft mit mindestens einem zugeteilten Kuchen SHALL **genau ein** Slot entstehen — an ihrem chronologisch ersten Heimspiel des Tages, also an demselben Spiel, das ihre Position in der Warteschlange bestimmt hat. Der Slot SHALL `slots_total` gleich der Anzahl der zugeteilten Kuchen tragen und `team_id = NULL` — die Mannschaft ergibt sich aus dem Anker-Spiel, an dem der Slot hängt. Trägt ein Heimspiel ausnahmsweise mehrere Teams, SHALL für dieses Spiel **ein** Slot mit der Summe ihrer Kuchen entstehen statt je Mannschaft einer. Heimspiele, deren Mannschaft keinen Kuchen zugeteilt bekommt, erhalten für dieses Item keinen Slot. `game_template_items.slots_count` SHALL für rotations-aktive Items ignoriert werden.

Ist die Warteschlange erschöpft, bevor der Bedarf gedeckt ist, SHALL der Restbedarf verfallen: es entsteht kein zusätzlicher Slot, keine Mannschaft überschreitet den Cap, und es wird kein zusätzlicher Auffang-Slot angelegt. Die Lücke SHALL im `regen_summary` mit Datum, Duty-Type und Anzahl der nicht zugeteilten Kuchen ausgewiesen werden.

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
- **AND** es entsteht kein dritter, den Restbedarf auffangender Slot
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
- **AND** es entsteht überhaupt kein Rotations-Slot

#### Scenario: Teilweise gegatete Vorlagen zählen nur die passenden Spiele

- **WHEN** ein Spieltag vier Heimspiele hat, das Verhältnis `1` ist, und nur zwei davon eine Vorlage tragen, deren rotations-aktives Item das Ausrichter-Gate passiert
- **THEN** beträgt der Bedarf `2` und nicht `4`

#### Scenario: Heimspiel mit zwei Teams bekommt einen zusammengefassten Slot

- **WHEN** ein Heimspiel ausnahmsweise zwei Mannschaften trägt und beiden je ein Kuchen zugeteilt wird
- **THEN** entsteht genau ein Slot mit `slots_total=2` und `team_id = NULL`

### Requirement: Restore-Matching ist für alle Items einheitlich

Bei einem erneuten Regen SHALL die Wiederherstellung bestehender `duty_assignments` für
**alle** Vorlagen-Items — mit und ohne aktivierte Bewirtungsrotation — über
`(duty_type_id, event_time)` matchen. Für rotations-aktive Items gibt es keine
Sonderbehandlung: ein Rotations-Slot hängt am Anker-Spiel seiner Mannschaft, und weil alle
Slots eines Spiels dieselbe Mannschaft betreffen, trägt `team_id` zur Unterscheidung nichts
bei.

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
