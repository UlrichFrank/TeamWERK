## MODIFIED Requirements

### Requirement: Massen-Regeneration der Dienst-Slots über einen wählbaren Zeitraum

Das System SHALL die Endpoints `POST /api/duty-slots/bulk-regen/preview` und
`POST /api/duty-slots/bulk-regen/apply` bereitstellen, die einen Zeitraum (`from`, `to`),
eine Vorbelegung je Terminart (`defaults`), Überschreibungen je Termin (`overrides`), eine
Ausnahmeliste (`excluded_game_ids`), **Ausrichter-Überschreibungen je Spieltag
(`host_overrides`: `{date, ausrichter_id}`)** und ein Benachrichtigungs-Flag (`notify`)
entgegennehmen und daraus die Dienst-Slots aller Termine der aktiven Saison im Zeitraum neu
erzeugen.

Die Antwort SHALL neben den termingenauen `rows` eine tagesweite Liste `days` führen, die je
Spieltag im Zeitraum den gespeicherten und den wirksamen Ausrichter sowie die Information
ausweist, ob der Wert explizit gesetzt oder vom Default geerbt ist. Ein `host_override` SHALL
im Lauf wie ein explizit gesetzter Tageswert wirken; ohne Angabe SHALL der gespeicherte
Tageswert bzw. der Default gelten („wie bisher").

Nur Nutzer mit der Capability `bulk_regen_duties` (Vereinsfunktion `vorstand` oder
Systemrolle `admin`) SHALL die Endpoints aufrufen dürfen.

Das System SHALL `from` ≤ heute mit HTTP 400 `range_in_past` ablehnen und den Zeitraum NICHT
stillschweigend verschieben. Fehlen `from`/`to`, SHALL das System den Default-Zeitraum
`[morgen, MAX(games.date) der aktiven Saison]` verwenden und ihn in der Antwort unter `range`
zurückliefern.

Das System SHALL bei fehlender aktiver Saison mit HTTP 400 antworten, bei einer
`template_id`, die nicht in `game_templates` existiert, mit HTTP 400 `invalid_template`, und
bei einem `ausrichter_id`, das nicht in `ausrichter` existiert, mit HTTP 400 — jeweils ohne
zu schreiben.

#### Scenario: Vorstand startet einen Lauf über die Restsaison
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` `POST /api/duty-slots/bulk-regen/apply` mit gültigem Zeitraum und Vorbelegung aufruft
- **THEN** antwortet der Server mit HTTP 200 und einem Body `{ range, rows, days, totals, warnings, applied: true }`
- **THEN** tragen alle einbezogenen Termine die zugewiesene `template_id` und die daraus erzeugten `duty_slots`

#### Scenario: Zeitraum wird vom Server vorbelegt
- **WHEN** ein Preview-Request ohne `from` und `to` gestellt wird
- **THEN** antwortet der Server mit HTTP 200 und `range.from` = morgen sowie `range.to` = spätestes Termindatum der aktiven Saison

#### Scenario: Ohne Vorbelegung behält jeder Termin sein Template
- **WHEN** ein Preview-Request ohne `defaults` und ohne `overrides` gestellt wird
- **THEN** wird für jeden Termin die gespeicherte `games.template_id` verwendet (bzw. `none`, wenn sie `NULL` ist)
- **THEN** weisen die Zeilen die Differenz zwischen den heute gespeicherten Slots und dem aus, was die Templates aktuell erzeugen würden

#### Scenario: Ohne host_override bleibt der gespeicherte Ausrichter wirksam
- **WHEN** ein Preview-Request ohne `host_overrides` gestellt wird
- **THEN** weist jede Zeile in `days` den gespeicherten Tageswert bzw. den Default als wirksamen Ausrichter aus
- **THEN** wird kein Eintrag in `spieltag_ausrichter` verändert

#### Scenario: Ausrichter-Überschreibung wirkt auf die erzeugten Dienste
- **WHEN** ein Apply-Request für einen Spieltag ein `host_override` setzt, das eine gebundene Vorlagen-Zeile ausgatet
- **THEN** entstehen für diese Zeile an diesem Tag keine Slots
- **THEN** ist der Tageswert in `spieltag_ausrichter` persistiert

#### Scenario: Unbekannter Ausrichter wird abgelehnt
- **WHEN** ein `ausrichter_id` übergeben wird, das in `ausrichter` nicht existiert
- **THEN** antwortet der Server mit HTTP 400 und verändert keinen Datensatz

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

## ADDED Requirements

### Requirement: Der Massenlauf-Dialog zeigt und setzt den Ausrichter je Spieltag

Das System SHALL im Massenlauf-Dialog des Kalenders je Spieltag im gewählten Zeitraum den
wirksamen Ausrichter anzeigen und zur Änderung anbieten. Dabei SHALL erkennbar sein, ob der
Wert explizit gesetzt oder vom Default geerbt ist.

Eine im Dialog vorgenommene Ausrichter-Änderung SHALL erst mit `apply` persistiert werden;
die Vorschau SHALL ihre Wirkung auf die erzeugten und gelöschten Dienste in derselben Bilanz
ausweisen wie Vorlagen-Änderungen — ein zweiter, separater Bestätigungsdialog SHALL nicht
nötig sein.

#### Scenario: Tagesliste weist geerbte Werte aus

- **WHEN** ein Berechtigter den Massenlauf-Dialog für einen Zeitraum ohne explizite Tageswerte öffnet
- **THEN** zeigt jede Tageszeile den Default-Ausrichter, gekennzeichnet als geerbt

#### Scenario: Änderung im Dialog erscheint in der Bilanz

- **WHEN** ein Berechtigter im Dialog für einen Tag einen anderen Ausrichter wählt und die Vorschau anstößt
- **THEN** weist die Bilanz die dadurch entfallenden Dienste und betroffenen Zusagen aus
- **AND** es wurde noch nichts persistiert

#### Scenario: Nur ein Bestätigungsschritt

- **WHEN** ein Berechtigter einen Lauf mit Vorlagen- und Ausrichter-Änderungen anwendet
- **THEN** werden beide Arten von Änderungen in einem Durchlauf übernommen, ohne zusätzlichen Dialog
