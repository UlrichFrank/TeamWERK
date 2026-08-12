## MODIFIED Requirements

### Requirement: Bedarfsermittlung und Greedy-Zuteilung mit Cap

Das System SHALL den Kuchenbedarf eines Spieltags als `min(Anzahl Heimspiele, aufgerundet(Anzahl Heimspiele × Verhältnis))` berechnen. In diese Zählung SHALL ein Heimspiel nur eingehen, wenn das rotations-aktive Item das **Ausrichter-Gate** des Tages passiert (`ausrichter_id IS NULL` oder gleich dem aufgelösten Tages-Ausrichter). Das Gate SHALL damit **vor** der Bedarfsrechnung wirken, nicht erst beim Einfügen der Slots — andernfalls verbrauchte die Team-Warteschlange Positionen für Slots, die anschließend verworfen werden, und der ausgewiesene Bedarf wäre falsch.

Die chronologisch ersten `Bedarf` Heimspiele des Tages SHALL je einen Rotations-Slot erhalten; weitere Heimspiele erhalten für dieses Item keinen Slot. Die Team-Zuteilung SHALL greedy in Warteschlangen-Reihenfolge erfolgen: die ersten `bewirtung_max_per_team` Slots gehen an Mannschaft 1, die nächsten an Mannschaft 2, usw. Der Cap SHALL einmal pro Regen-Lauf aus `system_settings` gelesen werden und für **alle** rotations-aktiven Items desselben Laufs gelten — unabhängig davon, aus welcher Vorlage ein Item stammt. Ist die Warteschlange erschöpft, bevor der Bedarf gedeckt ist, SHALL der verbleibende Slot ohne Team-Zuordnung (`team_id = NULL`) entstehen, statt den Cap zu überschreiten oder eine andere Mannschaft erneut heranzuziehen.

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

#### Scenario: Vorstand sieht und bearbeitet beide Bewirtungswerte

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
