## MODIFIED Requirements

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
