## ADDED Requirements

### Requirement: Vereinsweite Ausrichter-Liste mit genau einem Default

Das System SHALL eine vereinsweite Liste von Ausrichtern (`ausrichter`: `name`, `aktiv`, `is_default`, `sort_order`) vorhalten. Zu jedem Zeitpunkt SHALL **genau ein** Eintrag `is_default = 1` tragen; diese Invariante SHALL mechanisch über einen Partial-Unique-Index (`WHERE is_default = 1`) abgesichert sein und nicht allein durch Handler-Logik. Die Migration SHALL genau eine Default-Zeile idempotent anlegen.

Die Liste SHALL über `GET /api/ausrichter` für alle Eingeloggten lesbar sein (Kalender und Termin-Wizard brauchen sie) und über `POST`/`PUT`/`DELETE /api/ausrichter[/{id}]` nur für Nutzer mit Vereinsfunktion `vorstand` oder System-Rolle `admin` änderbar. Jede erfolgreiche Mutation SHALL `h.hub.Broadcast("settings-changed")` auslösen.

Namen SHALL eindeutig sein; ein Duplikat SHALL mit HTTP 409 abgelehnt werden, ohne zu schreiben.

Der Default-Eintrag SHALL zusätzlich gegen zwei Zustände geschützt sein, die die totale Auflösung aushebeln würden: er SHALL weder **abwählbar** sein (`is_default` auf `false` setzen, es bliebe kein Default übrig) noch **deaktivierbar** (`aktiv` auf `false`, jeder ungepflegte Spieltag löste dann auf einen inaktiven Ausrichter auf — einen Wert, den die Schreibrouten selbst mit HTTP 400 zurückweisen). Beide SHALL mit HTTP 409 abgelehnt werden, ohne zu schreiben. Der Weg bleibt in beiden Fällen: erst einen anderen Eintrag zum Default machen, dann den alten ändern.

#### Scenario: Migration legt genau eine Default-Zeile an

- **WHEN** die Migration zweimal hintereinander auf derselben DB ausgeführt wird
- **THEN** existiert genau eine Zeile mit `is_default = 1`
- **AND** die zweite Ausführung endet ohne Fehler

#### Scenario: Vorstand legt einen Ausrichter an

- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` `POST /api/ausrichter` mit `{"name": "TV Ötlingen"}` aufruft
- **THEN** antwortet das System mit HTTP 201 und dem angelegten Eintrag
- **AND** ein SSE-Event `settings-changed` wird gesendet

#### Scenario: Doppelter Name wird abgelehnt

- **WHEN** ein Ausrichter mit einem bereits vergebenen Namen angelegt werden soll
- **THEN** antwortet das System mit HTTP 409
- **AND** es wird keine Zeile geschrieben

#### Scenario: Default-Wechsel entzieht dem bisherigen Default die Markierung

- **WHEN** ein Vorstand einen anderen Eintrag per `PUT /api/ausrichter/{id}` als Default markiert
- **THEN** trägt danach genau eine Zeile `is_default = 1`, nämlich die neu markierte

#### Scenario: Der Default lässt sich nicht abwählen

- **WHEN** ein Vorstand den Default-Eintrag per `PUT` auf `is_default: false` setzen will
- **THEN** antwortet das System mit HTTP 409
- **AND** es bleibt genau eine Zeile mit `is_default = 1`

#### Scenario: Der Default lässt sich nicht deaktivieren

- **WHEN** ein Vorstand den Default-Eintrag per `PUT` auf `aktiv: false` setzen will
- **THEN** antwortet das System mit HTTP 409
- **AND** der Eintrag bleibt aktiv

#### Scenario: Standard-Nutzer darf die Liste lesen, aber nicht ändern

- **WHEN** ein eingeloggter Nutzer ohne Vereinsfunktion `vorstand` `GET /api/ausrichter` aufruft
- **THEN** antwortet das System mit HTTP 200 und der Liste
- **WHEN** derselbe Nutzer `POST /api/ausrichter` aufruft
- **THEN** antwortet das System mit HTTP 403 und verändert keinen Datensatz

#### Scenario: Unauthentifizierter Zugriff wird abgewiesen

- **WHEN** ein Request ohne gültigen Access-Token `GET /api/ausrichter` aufruft
- **THEN** antwortet das System mit HTTP 401

### Requirement: Der Ausrichter ist eine Eigenschaft des Spieltags und immer aufgelöst

Das System SHALL den Ausrichter je Spieltag in `spieltag_ausrichter` mit dem Schlüssel `(date, season_id)` speichern — nicht je Spiel und nicht je Veranstaltungsort. Der gespeicherte Wert SHALL für **alle** Termine dieses Tages gelten, unabhängig davon, ob sie vor oder nach dem Setzen angelegt wurden.

Die Auflösung SHALL total sein:

```
ausrichter(tag) = spieltag_ausrichter[(date, season_id)] ?? ausrichter.is_default
```

Ein fehlender Eintrag und ein Eintrag mit `ausrichter_id IS NULL` SHALL identisch behandelt werden und beide auf den Default fallen. Es SHALL keinen Zustand „Spieltag ohne Ausrichter" geben.

`GET /api/game-days/{date}/host` SHALL den aufgelösten Ausrichter zusammen mit der Information liefern, ob er explizit gesetzt oder vom Default geerbt ist (`is_explicit`).

#### Scenario: Tag ohne Eintrag fällt auf den Default

- **WHEN** für einen Spieltag kein Eintrag in `spieltag_ausrichter` existiert
- **AND** `GET /api/game-days/{date}/host` aufgerufen wird
- **THEN** antwortet das System mit dem Default-Ausrichter und `is_explicit: false`

#### Scenario: NULL-Eintrag verhält sich wie kein Eintrag

- **WHEN** ein Eintrag mit `ausrichter_id IS NULL` existiert (etwa nach dem Löschen eines Ausrichters)
- **THEN** liefert die Auflösung den Default-Ausrichter und `is_explicit: false`

#### Scenario: Der Tageswert gilt auch für später angelegte Termine

- **WHEN** für den 14.09. ein abweichender Ausrichter gesetzt ist
- **AND** danach ein weiteres Heimspiel am 14.09. angelegt wird
- **THEN** gilt für dieses neue Spiel derselbe Ausrichter, ohne dass er erneut gesetzt werden muss

#### Scenario: Derselbe Kalendertag in zwei Saisons kollidiert nicht

- **WHEN** in zwei verschiedenen Saisons Einträge für dasselbe Datum existieren
- **THEN** liefert die Auflösung je Saison den jeweils eigenen Wert

### Requirement: Vorlagen-Items können an einen Ausrichter gebunden werden

Das System SHALL auf `game_template_items` ein optionales Feld `ausrichter_id` führen. `NULL` (Default) SHALL bedeuten: die Zeile erzeugt an jedem Spieltag Slots (bisheriges Verhalten, unverändert). Ein gesetzter Wert SHALL bedeuten: die Zeile erzeugt nur an Spieltagen Slots, deren aufgelöster Ausrichter übereinstimmt.

Das Feld SHALL nur auf Vorlagen mit `template_type = 'heim'` zulässig sein. Ein gesetztes `ausrichter_id` auf einer Vorlage anderen Typs SHALL mit HTTP 400 `ausrichter_requires_heim_template` abgelehnt werden, ohne zu schreiben.

#### Scenario: Bestehende Vorlagen verhalten sich unverändert

- **WHEN** ein Vorlagen-Item ohne `ausrichter_id` regeneriert wird
- **THEN** entstehen dieselben Slots wie vor Einführung dieses Feldes, unabhängig vom Ausrichter des Tages

#### Scenario: Gebundene Zeile erzeugt nur bei passendem Ausrichter

- **WHEN** ein Item an Ausrichter `A` gebunden ist und der Spieltag `A` als Ausrichter auflöst
- **THEN** entstehen die Slots dieses Items
- **WHEN** derselbe Spieltag stattdessen `B` auflöst
- **THEN** entsteht für dieses Item kein Slot

#### Scenario: Ausrichter auf einer Auswärts-Vorlage wird abgelehnt

- **WHEN** ein Item mit gesetztem `ausrichter_id` auf einer Vorlage mit `template_type = 'auswärts'` gespeichert werden soll
- **THEN** antwortet das System mit HTTP 400 und `{"error":"ausrichter_requires_heim_template"}`
- **AND** es wird kein Item geschrieben

### Requirement: Das Ausrichter-Gate wirkt vor der Bedarfsrechnung und nur bei Heimspielen

Das System SHALL den Ausrichter eines Tages **einmal** je `regenSingleDay`-Lauf innerhalb der laufenden Transaktion auflösen und an beide nachgelagerten Stellen durchreichen:

1. an die tagesweite Bewirtungs-Vorausberechnung (`buildRotationPlan`), **bevor** dort der Kuchenbedarf ermittelt wird, und
2. an die Pro-Spiel-Erzeugung (`regenGameItems`).

Ein Item SHALL genau dann Slots erzeugen, wenn `ausrichter_id IS NULL` oder `ausrichter_id` dem aufgelösten Tages-Ausrichter entspricht. Das Gate SHALL ausschließlich auf Termine mit `event_type = 'heim'` wirken.

#### Scenario: Ausgegatetes Rotations-Item erzeugt keinen Bedarf

- **WHEN** das rotations-aktive Kuchen-Item an Ausrichter `A` gebunden ist und der Spieltag `B` auflöst
- **THEN** ist der ermittelte Kuchenbedarf des Tages `0`
- **AND** die Team-Warteschlange verbraucht keine Positionen für dieses Item

#### Scenario: Auswärts- und generische Termine bleiben unberührt

- **WHEN** ein Spieltag Auswärts- und generische Termine enthält
- **THEN** verändert der Ausrichter des Tages die für diese Termine erzeugten Slots nicht

#### Scenario: Der Ausrichter wird einmal je Tag aufgelöst

- **WHEN** ein Regen-Lauf einen Tag mit mehreren Heimspielen verarbeitet
- **THEN** wird der Tages-Ausrichter einmal gelesen und für alle Spiele dieses Tages verwendet

### Requirement: Ausrichter-Änderungen laufen über eine schreibfreie Vorschau

Das System SHALL `POST /api/game-days/host/preview` und `POST /api/game-days/host/apply` bereitstellen. Beide SHALL denselben Codepfad nutzen und sich ausschließlich im Abschluss unterscheiden (`Rollback` statt `Commit`) — die Vorschau SHALL nicht clientseitig nachgebaut werden.

`preview` SHALL die Bilanz der Änderung ausweisen (erzeugte und gelöschte Slots, erhaltene und verlorene Zuweisungen) und **keinen** Datensatz verändern. `apply` SHALL den Wert persistieren, den Regen für den betroffenen Tag ausführen und broadcasten.

Beide Endpoints SHALL die Capability `manage_games` verlangen. Ein unbekannter oder inaktiver `ausrichter_id` SHALL mit HTTP 400 abgelehnt werden, ohne zu schreiben.

#### Scenario: Vorschau zeigt die Bilanz ohne zu schreiben

- **WHEN** ein Berechtigter `POST /api/game-days/host/preview` für einen Tag mit bestehenden Zusagen aufruft
- **THEN** antwortet das System mit HTTP 200 und der Bilanz gelöschter Dienste und betroffener Zusagen
- **AND** `duty_slots`, `duty_assignments` und `spieltag_ausrichter` sind danach unverändert

#### Scenario: Anwenden setzt den Wert und regeneriert den Tag

- **WHEN** derselbe Request an `POST /api/game-days/host/apply` geht
- **THEN** ist `spieltag_ausrichter` für diesen Tag gesetzt
- **AND** die Dienst-Slots des Tages entsprechen dem in der Vorschau ausgewiesenen Ergebnis
- **AND** ein Broadcast wird gesendet

#### Scenario: Unbekannter Ausrichter wird abgelehnt

- **WHEN** ein `ausrichter_id` übergeben wird, das nicht existiert
- **THEN** antwortet das System mit HTTP 400 und verändert keinen Datensatz

#### Scenario: Nutzer ohne manage_games wird abgewiesen

- **WHEN** ein Nutzer ohne Capability `manage_games` einen der beiden Endpoints aufruft
- **THEN** antwortet das System mit HTTP 403 und verändert keinen Datensatz

### Requirement: Löschen eines Ausrichters entkoppelt Spieltage, löscht aber gebundene Vorlagen-Zeilen

Das System SHALL beim Löschen eines Ausrichters zwei Referenzen unterschiedlich behandeln:

- `spieltag_ausrichter.ausrichter_id` SHALL auf `NULL` gesetzt werden — die betroffenen Spieltage fallen damit auf den Default zurück.
- `game_template_items` mit diesem `ausrichter_id` SHALL **mitgelöscht** werden. Ein `SET NULL` ist hier ausdrücklich unzulässig, weil die Zeile dadurch auf „gilt immer" gehoben würde und nach dem Löschen **mehr** Dienste erzeugte als vorher.

`GET /api/ausrichter/{id}/usage` SHALL vor dem Löschen die betroffenen Spieltage und Vorlagen-Zeilen benennen. Der Default-Ausrichter SHALL nicht löschbar sein (HTTP 409 `default_ausrichter_undeletable`), weil sonst die Auflösung nicht mehr total wäre.

#### Scenario: Betroffene Spieltage fallen auf den Default

- **WHEN** ein Ausrichter gelöscht wird, der an drei Spieltagen explizit gesetzt war
- **THEN** tragen diese Spieltage danach `ausrichter_id IS NULL`
- **AND** ihre Auflösung liefert den Default-Ausrichter

#### Scenario: Gebundene Vorlagen-Zeilen verschwinden mit

- **WHEN** ein Ausrichter gelöscht wird, an den zwei Vorlagen-Items gebunden waren
- **THEN** existieren diese beiden Items danach nicht mehr
- **AND** kein Item trägt ein verwaistes oder auf `NULL` gesetztes `ausrichter_id` aus dieser Bindung

#### Scenario: Verwendungsübersicht vor dem Löschen

- **WHEN** ein Vorstand `GET /api/ausrichter/{id}/usage` aufruft
- **THEN** antwortet das System mit den betroffenen Spieltagen und Vorlagen-Zeilen

#### Scenario: Der Default-Ausrichter ist nicht löschbar

- **WHEN** ein Vorstand den als Default markierten Ausrichter löschen will
- **THEN** antwortet das System mit HTTP 409 und `{"error":"default_ausrichter_undeletable"}`
- **AND** es wird kein Datensatz verändert

### Requirement: Der Ausrichter ist im Kalender und im Termin-Wizard als Tages-Eigenschaft erkennbar

Das System SHALL den Ausrichter im **Termin-Detail-Modal jedes Heim-Termins** anzeigen und für Nutzer mit der Capability `manage_games` änderbar machen. Dabei SHALL sichtbar sein, ob der Wert explizit gesetzt oder vom Default geerbt ist.

Weil der Kalender keine Tagesansicht besitzt, erscheint derselbe Tageswert am Modal **jedes** Heim-Termins dieses Tages. Das Feld SHALL deshalb dort ebenso erkennbar tagesbezogen beschriftet sein wie im Wizard, und eine Änderung SHALL über dieselbe Vorschau laufen. In der Monatsübersicht SHALL der Ausrichter bewusst **nicht** dargestellt werden.

Im Termin-Wizard SHALL das Feld bei Heim-Terminen erscheinen, mit dem aktuell geltenden Wert vorbelegt sein und **erkennbar tagesbezogen** beschriftet sein (etwa „Ausrichter am 14.09. — gilt für alle Termine dieses Tages"), weil eine Änderung dort den ganzen Tag samt bereits bestehender Termine betrifft. Weicht der gewählte Wert vom gespeicherten ab, SHALL das Speichern über dieselbe Vorschau laufen wie im Kalender.

#### Scenario: Termin-Modal unterscheidet gesetzt von geerbt

- **WHEN** ein Berechtigter das Detail-Modal eines Heim-Termins an einem Tag ohne expliziten Eintrag öffnet
- **THEN** wird der Default-Ausrichter angezeigt und als geerbt gekennzeichnet

#### Scenario: Alle Termine desselben Tages zeigen denselben Wert

- **WHEN** an einem Tag zwei Heim-Termine existieren und für einen davon der Ausrichter geändert wird
- **THEN** zeigt auch das Modal des anderen Termins den neuen Wert

#### Scenario: Monatsübersicht bleibt frei vom Ausrichter

- **WHEN** ein Berechtigter die Monatsübersicht öffnet
- **THEN** wird dort kein Ausrichter dargestellt

#### Scenario: Wizard-Feld ist als Tages-Feld beschriftet

- **WHEN** ein Berechtigter im Wizard ein Heimspiel anlegt
- **THEN** trägt das Ausrichter-Feld eine Beschriftung, die den Tagesbezug benennt
- **AND** es ist mit dem für diesen Tag geltenden Ausrichter vorbelegt

#### Scenario: Abweichende Wahl im Wizard führt über die Vorschau

- **WHEN** ein Berechtigter im Wizard einen anderen als den geltenden Ausrichter wählt
- **AND** an diesem Tag bereits Termine mit Diensten existieren
- **THEN** erscheint vor dem Speichern die Bilanz der betroffenen Dienste und Zusagen

#### Scenario: Nutzer ohne manage_games sieht den Wert nur

- **WHEN** ein eingeloggter Nutzer ohne Capability `manage_games` die Tagesansicht öffnet
- **THEN** ist der Ausrichter sichtbar, aber nicht änderbar
