## ADDED Requirements

### Requirement: Eine Dienst-Zuweisung überlebt eine Regeneration, die ihren Slot unverändert wiederherstellt

Das System SHALL bei jeder Regeneration von Dienst-Slots die vorhandenen
`duty_assignments` der gelöschten `is_custom=0`-Slots erfassen und nach dem Neuanlegen
wiederherstellen, sofern ein neuer Slot mit identischer Kombination aus `duty_type_id`,
`event_time` und `team_id` entstanden ist.

Wiederhergestellt SHALL die Zuweisung samt `user_id`, `status`, `cash_amount` und
`fulfilled_at` werden. Das System SHALL dabei `duty_slots.slots_filled` auf die Anzahl der
tatsächlich wiederhergestellten Zuweisungen setzen.

Diese Zusicherung SHALL für **alle** Regenerations-Pfade gelten — die Regeneration nach einer
einzelnen Spieländerung, den H4A-Import und den Massenlauf.

Für eine wiederhergestellte Zuweisung SHALL **keine** Benachrichtigung versendet werden.

#### Scenario: Unveränderte Regeneration erhält die Zuweisung
- **WHEN** ein Termin mit einem belegten Dienst-Slot ohne Änderung an Template, Uhrzeit oder Mannschaft regeneriert wird
- **THEN** existiert die Zuweisung derselben Person danach weiterhin
- **THEN** stimmt `slots_filled` des neuen Slots mit der Anzahl der Zuweisungen überein
- **THEN** wurde keine Benachrichtigung an diese Person versendet

#### Scenario: Status und Betrag einer Ersatzzahlung bleiben erhalten
- **WHEN** eine Zuweisung mit `status = 'cash_substitute'` und gesetztem `cash_amount` regeneriert wird und ihr Slot identisch wiederkommt
- **THEN** trägt die wiederhergestellte Zuweisung denselben `status` und denselben `cash_amount`

#### Scenario: Unbeteiligte Spieländerung entfernt keine Zuweisungen
- **WHEN** an einem Spiel ein Feld geändert wird, das die Slot-Erzeugung nicht beeinflusst (z. B. der Spielort), und die Regeneration dieselben Slots erzeugt
- **THEN** bleiben alle Dienstzuweisungen dieses Spiels bestehen und es wird keine Benachrichtigung versendet

### Requirement: Deterministische Auswahl, wenn die Kapazität schrumpft

Das System SHALL höchstens `slots_total` Zuweisungen pro Slot wiederherstellen. Reichen die
Plätze nicht für alle erfassten Zuweisungen, SHALL das System die Zuweisungen in aufsteigender
Reihenfolge ihrer ursprünglichen `duty_assignments.id` wiederherstellen — die ältesten
zuerst.

Für jede nicht wiederhergestellte Zuweisung SHALL das System genau eine
„Dienst entfernt"-Benachrichtigung erzeugen.

#### Scenario: Von drei auf zwei Plätze
- **WHEN** ein Slot mit drei Zuweisungen regeneriert wird und das geänderte Template nur noch zwei Plätze vorsieht
- **THEN** überleben die zwei Zuweisungen mit den kleinsten ursprünglichen IDs
- **THEN** erhält genau die dritte Person eine „Dienst entfernt"-Benachrichtigung
- **THEN** ist `slots_filled` des neuen Slots 2

### Requirement: Ohne passenden Slot wird nicht wiederhergestellt

Das System SHALL eine Zuweisung NICHT wiederherstellen, wenn kein neuer Slot mit identischem
`duty_type_id`, `event_time` und `team_id` entstanden ist. Insbesondere SHALL eine durch
`same_day_behavior`/`adjacent_day_behavior` in eine andere Dienstart überführte Position
NICHT als Treffer gelten — hier bleibt die bestehende
„Variante geändert"-Benachrichtigung unverändert bestehen.

#### Scenario: Verschobene Uhrzeit löst die Zuweisung
- **WHEN** die Anstoßzeit eines Termins geändert wird und die Slots dadurch auf eine andere `event_time` fallen
- **THEN** wird die Zuweisung nicht wiederhergestellt und die betroffene Person erhält eine „Dienst entfernt"-Benachrichtigung

#### Scenario: Reduzierte Dienstart gilt nicht als Treffer
- **WHEN** eine Dienstart durch `same_day_behavior` in ihre reduzierte Variante überführt wird
- **THEN** wird die Zuweisung nicht wiederhergestellt
- **THEN** erhält die betroffene Person die bestehende „Variante geändert"-Benachrichtigung mit dem Namen der neuen Dienstart

#### Scenario: Gelöschter Slot ohne Ersatz
- **WHEN** ein Termin so regeneriert wird, dass eine Dienstart entfällt
- **THEN** ist die zugehörige Zuweisung entfernt und die betroffene Person wurde genau einmal benachrichtigt
