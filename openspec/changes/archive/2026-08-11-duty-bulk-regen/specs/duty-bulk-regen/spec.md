## ADDED Requirements

### Requirement: Massen-Regeneration der Dienst-Slots über einen wählbaren Zeitraum

Das System SHALL die Endpoints `POST /api/duty-slots/bulk-regen/preview` und
`POST /api/duty-slots/bulk-regen/apply` bereitstellen, die einen Zeitraum (`from`, `to`),
eine Vorbelegung je Terminart (`defaults`), Überschreibungen je Termin (`overrides`), eine
Ausnahmeliste (`excluded_game_ids`) und ein Benachrichtigungs-Flag (`notify`) entgegennehmen
und daraus die Dienst-Slots aller Termine der aktiven Saison im Zeitraum neu erzeugen.

Nur Nutzer mit der Capability `bulk_regen_duties` (Vereinsfunktion `vorstand` oder
Systemrolle `admin`) SHALL die Endpoints aufrufen dürfen.

Das System SHALL `from` ≤ heute mit HTTP 400 `range_in_past` ablehnen und den Zeitraum NICHT
stillschweigend verschieben. Fehlen `from`/`to`, SHALL das System den Default-Zeitraum
`[morgen, MAX(games.date) der aktiven Saison]` verwenden und ihn in der Antwort unter `range`
zurückliefern.

Das System SHALL bei fehlender aktiver Saison mit HTTP 400 antworten und bei einer
`template_id`, die nicht in `game_templates` existiert, mit HTTP 400 `invalid_template`.

#### Scenario: Vorstand startet einen Lauf über die Restsaison
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` `POST /api/duty-slots/bulk-regen/apply` mit gültigem Zeitraum und Vorbelegung aufruft
- **THEN** antwortet der Server mit HTTP 200 und einem Body `{ range, rows, totals, warnings, applied: true }`
- **THEN** tragen alle einbezogenen Termine die zugewiesene `template_id` und die daraus erzeugten `duty_slots`

#### Scenario: Zeitraum wird vom Server vorbelegt
- **WHEN** ein Preview-Request ohne `from` und `to` gestellt wird
- **THEN** antwortet der Server mit HTTP 200 und `range.from` = morgen sowie `range.to` = spätestes Termindatum der aktiven Saison

#### Scenario: Ohne Vorbelegung behält jeder Termin sein Template
- **WHEN** ein Preview-Request ohne `defaults` und ohne `overrides` gestellt wird
- **THEN** wird für jeden Termin die gespeicherte `games.template_id` verwendet (bzw. `none`, wenn sie `NULL` ist)
- **THEN** weisen die Zeilen die Differenz zwischen den heute gespeicherten Slots und dem aus, was die Templates aktuell erzeugen würden

#### Scenario: Standard-Nutzer ohne Vorstand-Funktion wird abgewiesen
- **WHEN** ein Nutzer ohne Capability `bulk_regen_duties` einen der beiden Endpoints aufruft
- **THEN** antwortet der Server mit HTTP 403 und verändert keinen Datensatz

#### Scenario: Zeitraum in der Vergangenheit wird abgelehnt
- **WHEN** ein Request mit `from` ≤ heute gestellt wird
- **THEN** antwortet der Server mit HTTP 400 und `{"error":"range_in_past"}`
- **THEN** wird kein `duty_slot` mit `event_date <= heute` verändert

#### Scenario: Keine aktive Saison
- **WHEN** keine Saison `is_active = 1` gesetzt ist
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Unbekanntes Template
- **WHEN** eine `template_id` übergeben wird, die in `game_templates` nicht existiert
- **THEN** antwortet der Server mit HTTP 400 und `{"error":"invalid_template"}`

### Requirement: Vier Zustände pro Termin

Das System SHALL pro Termin genau einen der Zustände `template`, `none` oder `purge`
anwenden, oder den Termin als ausgenommen unangetastet lassen. Der Zustand SHALL aus der
Vorbelegung der Terminart (`heim`, `auswärts`, `generisch`) stammen und durch einen Eintrag
in `overrides` für einzelne Termine überschreibbar sein.

| Zustand | `games.template_id` | `is_custom=0`-Slots | `is_custom=1`-Slots |
|---|---|---|---|
| `template` | wird auf die gewählte ID gesetzt | gelöscht und aus dem Template neu erzeugt | bleiben erhalten |
| `none` | wird auf `NULL` gesetzt | gelöscht, keine neuen | bleiben erhalten |
| `purge` | wird auf `NULL` gesetzt | gelöscht, keine neuen | **gelöscht** |
| ausgenommen | unverändert | unangetastet | unangetastet |

Das System SHALL die Ausnahmeliste (`excluded_game_ids`) unabhängig von der Zustandswahl
auswerten: ein ausgenommener Termin wird auch dann nicht verändert, wenn seine Terminart
eine Vorbelegung trägt.

#### Scenario: Pauschalwahl wirkt auf alle Termine einer Terminart
- **WHEN** `defaults.heim = { action: "template", template_id: 3 }` gesetzt ist und kein Override greift
- **THEN** tragen alle nicht ausgenommenen Heimspiele im Zeitraum `template_id = 3` und die daraus erzeugten Slots

#### Scenario: Zeilen-Override sticht die Pauschalwahl
- **WHEN** `defaults.heim` Template 3 vorbelegt und `overrides` für Spiel 43 Template 9 setzt
- **THEN** trägt Spiel 43 `template_id = 9`, alle übrigen Heimspiele `template_id = 3`

#### Scenario: „keine Dienste anlegen" behält handgemachte Slots
- **WHEN** ein Termin mit einem `is_custom=0`- und einem `is_custom=1`-Slot im Zustand `none` verarbeitet wird
- **THEN** ist der `is_custom=0`-Slot gelöscht, der `is_custom=1`-Slot unverändert vorhanden und `games.template_id` ist `NULL`

#### Scenario: „alle Dienste löschen" entfernt auch handgemachte Slots
- **WHEN** derselbe Termin im Zustand `purge` verarbeitet wird
- **THEN** existiert kein `duty_slot` mehr zu diesem Termin, unabhängig von `is_custom`

#### Scenario: Ausgenommener Termin bleibt unverändert
- **WHEN** ein Termin in `excluded_game_ids` steht und seine Terminart eine Vorbelegung trägt
- **THEN** sind seine `duty_slots` inklusive ihrer IDs unverändert und `games.template_id` ist unverändert

### Requirement: Der Nachbar-Kontext für die Dienstoptimierung hängt nur an der Existenz des Nachbarspiels, nicht an seinem Massenlauf-Zustand

Das System SHALL für die Berechnung von `same_day_behavior` (aufeinanderfolgende Heimspiele
am selben Tag) und `adjacent_day_behavior` (aufeinanderfolgende Heimspiele an Folgetagen)
ausschließlich prüfen, ob am betrachteten Tag bzw. an den Nachbartagen ein Heimspiel
existiert (`games.is_home=1`). Der im selben Massenlauf gewählte Zustand des Nachbarspiels
(`template`/`none`/`purge`/ausgenommen) SHALL diese Berechnung NICHT beeinflussen, und ein
Nachbarspiel außerhalb des gewählten Zeitraums (`from`/`to`) SHALL genauso in die Berechnung
einfließen wie eines innerhalb.

Ein ausgenommener Termin SHALL insbesondere weiterhin in die Tageskonstellation einbezogen
werden, aus der `same_day_behavior` und `adjacent_day_behavior` der übrigen Termine berechnet
werden — er beeinflusst die Reduktion benachbarter Termine genauso wie ein einbezogener.

#### Scenario: Ausgenommenes Spiel beeinflusst weiterhin die Reduktion des Nachbarspiels
- **WHEN** an einem Tag zwei Spiele stattfinden, eines davon ausgenommen ist, und das einbezogene Spiel eine Dienstart mit `same_day_behavior` trägt
- **THEN** erhält das einbezogene Spiel dieselbe reduzierte Dienstart wie bei einem Lauf ohne Ausnahme
- **THEN** sind die Slot-IDs des ausgenommenen Spiels unverändert

#### Scenario: Nachbarspiel im Zustand „keine Dienste"/„alle Dienste löschen" reduziert trotzdem
- **WHEN** zwei Heimspiele an aufeinanderfolgenden Tagen im selben Lauf verarbeitet werden und das zweite den Zustand `none` oder `purge` erhält
- **THEN** erhält das erste Spiel dieselbe `adjacent_day_behavior`-Reduktion, als hätte das zweite Spiel den Zustand `template` erhalten
- **THEN** ist dabei unerheblich, dass das zweite Spiel selbst keine Dienst-Slots mehr trägt

#### Scenario: Heimspiel außerhalb des gewählten Zeitraums löst die Reduktion trotzdem aus
- **WHEN** am Tag vor `from` ein Heimspiel stattfindet, das damit außerhalb des Massenlauf-Zeitraums liegt und unangetastet bleibt
- **THEN** erhält der erste Termin innerhalb des Zeitraums dieselbe `adjacent_day_behavior`-Reduktion wie bei einem Einzelspiel-Regen mit demselben Vortag

### Requirement: Vorschau ist schreibfrei und deckungsgleich mit dem Lauf

Das System SHALL `POST /api/duty-slots/bulk-regen/preview` als vollständigen Dry-Run
ausführen — dieselbe Transaktion wie `apply`, abgeschlossen mit `ROLLBACK` statt `COMMIT` —
und dabei KEINEN Datensatz verändern, KEINEN Broadcast senden und KEINE Benachrichtigung
versenden.

Das System SHALL pro Termin eine Zeile mit `created`, `deleted_auto`, `deleted_custom`,
`assignments_kept`, `assignments_lost`, `conflicts` sowie dem Bestand vor und nach dem Lauf
(`slots_before`/`slots_after`, je nach `auto`/`custom` getrennt) zurückliefern, dazu eine
Summenzeile `totals`. Die Zeilenliste SHALL nicht gekappt werden.

Ein `preview`- und ein unmittelbar folgender `apply`-Request mit identischem Body SHALL
identische `totals` liefern.

#### Scenario: Preview verändert die Datenbank nicht
- **WHEN** ein Preview-Request mit `purge` über den gesamten Zeitraum gestellt wird
- **THEN** ist der Datenbankinhalt nach dem Request identisch zu dem davor
- **THEN** wurde kein Broadcast und keine Benachrichtigung ausgelöst

#### Scenario: Preview-Zahlen stimmen mit dem tatsächlichen Lauf überein
- **WHEN** derselbe Request zuerst an `preview` und danach an `apply` gestellt wird
- **THEN** sind die `totals` beider Antworten identisch
- **THEN** entsprechen die tatsächlichen Zählungen in `duty_slots` nach dem Apply den Preview-Werten

#### Scenario: Konflikte mit handgemachten Slots werden ausgewiesen
- **WHEN** ein Template-Slot auf dieselbe Kombination aus Dienstart, Uhrzeit und Mannschaft fällt wie ein vorhandener `is_custom=1`-Slot
- **THEN** wird der Template-Slot nicht angelegt und die Zeile weist `conflicts` ≥ 1 aus

### Requirement: Ein Broadcast und ein Regen-Lauf pro Massenlauf

Das System SHALL bei `apply` genau einen `runAutoRegen`-Lauf über die Vereinigungsmenge aller
betroffenen Datumsfenster ausführen und danach genau einen `Broadcast("duties")` sowie genau
einen `Broadcast("games")` senden — unabhängig von der Anzahl betroffener Termine und Tage.

#### Scenario: Lauf über viele Tage erzeugt keinen Broadcast-Sturm
- **WHEN** ein Apply-Request 40 Termine an 25 verschiedenen Tagen betrifft
- **THEN** wird genau ein `Broadcast("duties")` und ein `Broadcast("games")` gesendet

### Requirement: Benachrichtigung der Betroffenen ist abschaltbar

Das System SHALL bei `notify: false` KEINE Push-/Notification-Zustellung an betroffene Nutzer
auslösen, die Wirkung des Laufs auf `duty_slots` und `duty_assignments` aber unverändert
ausführen. Der Default SHALL `notify: true` sein.

Das Flag SHALL ausschließlich die Zustellung an Betroffene steuern; die Ergebnisdarstellung
(`rows`, `totals`, `regen_summary`) an den auslösenden Nutzer SHALL davon unberührt bleiben.

#### Scenario: Lauf ohne Benachrichtigung
- **WHEN** ein Apply-Request mit `notify: false` Zuweisungen entfernt
- **THEN** wird keine Benachrichtigung versendet
- **THEN** weist die Antwort die betroffenen Nutzer weiterhin in `totals.notified_users` aus

#### Scenario: Default benachrichtigt
- **WHEN** ein Apply-Request ohne das Feld `notify` Zuweisungen entfernt
- **THEN** erhält jede betroffene Person genau eine Benachrichtigung

### Requirement: Einstieg über das Aktionsmenü im Kalender

Das System SHALL den Massenlauf auf `/kalender` über das bestehende Dropdown am
`+ Event`-Split-Button anbieten, als zweiten Eintrag neben dem H4A-Spielimport, sichtbar für
Nutzer mit der Capability `bulk_regen_duties`. Das Dropdown SHALL sichtbar sein, sobald der
Nutzer `import_games` ODER `bulk_regen_duties` besitzt.

Das Modal SHALL die Vorschau bei jeder Änderung an Zeitraum, Vorbelegung, Override oder
Ausnahmeliste neu anfordern, die Anfragen entprellen und eine noch laufende Anfrage
abbrechen. Zeilen im Zustand `purge` SHALL als destruktiv gekennzeichnet sein.

#### Scenario: Nutzer mit nur einer der beiden Capabilities sieht das Menü
- **WHEN** ein Nutzer `bulk_regen_duties`, aber nicht `import_games` besitzt
- **THEN** ist das Dropdown am `+ Event`-Button sichtbar und enthält „Dienste aktualisieren"

#### Scenario: Vorschau folgt der Eingabe
- **WHEN** der Nutzer die Vorbelegung einer Terminart ändert
- **THEN** wird nach kurzer Verzögerung eine neue Vorschau angefordert, eine noch laufende abgebrochen, und die Summenzeile spiegelt die neue Wahl
