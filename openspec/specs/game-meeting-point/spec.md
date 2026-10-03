# game-meeting-point Specification

## Purpose
Trainer und Verantwortliche hinterlegen je Spiel eine optionale Treffzeit samt Treffpunkt-Ort. Die Treffzeit hängt am Anwurf und erscheint überall dort, wo Spieler und Eltern den Termin sehen.

## Requirements

### Requirement: Treffzeit und Treffpunkt-Ort je Spiel

Das System SHALL an jedem Eintrag der Tabelle `games` eine optionale **Treffzeit** und einen optionalen **Treffpunkt-Ort** führen. Das gilt für alle `event_type`-Werte (`heim`, `auswärts`, `generisch`). Trainings sind nicht Teil dieser Capability.

Die Treffzeit SHALL als **Abstand in Minuten vor dem Anwurf** gespeichert werden, nicht als Uhrzeit. Die angezeigte Treffzeit SHALL bei jedem Lesen aus dem aktuellen Anwurf (`games.date` + `games.time`) minus diesem Abstand berechnet werden. Ändert sich der Anwurf, auf welchem Weg auch immer (Bearbeiten, H4A-Import), SHALL sich die Treffzeit um denselben Betrag verschieben.

Der Treffpunkt-Ort SHALL ein Freitext mit höchstens 100 Zeichen sein. Er SHALL nur zusammen mit einer Treffzeit existieren. Ein Spiel ohne Treffzeit hat auch keinen Ort.

Jede API-Antwort, die den Anwurf eines Spiels (`time`) ausliefert, SHALL zusätzlich `meet_time` (`"HH:MM"` oder `null`), `meet_date` (Datum der Treffzeit `"YYYY-MM-DD"` oder `null`) und `meet_place` (String, leer ohne Ort) liefern. `meet_date` weicht genau dann vom Spieldatum ab, wenn die Treffzeit vor 00:00 des Spieltags liegt.

#### Scenario: Treffzeit folgt einem verlegten Anwurf

- **WHEN** ein Spiel mit Anwurf 15:00 die Treffzeit 13:30 trägt
- **AND** der Anwurf wird (manuell oder per H4A-Import) auf 17:00 verlegt
- **THEN** liefert die API für dieses Spiel `meet_time = "15:30"`
- **AND** `meet_place` bleibt unverändert

#### Scenario: Spiel ohne Treffzeit

- **WHEN** für ein Spiel keine Treffzeit gesetzt ist
- **THEN** liefert die API `meet_time = null`, `meet_date = null` und `meet_place = ""`

#### Scenario: Treffzeit am Vortag nach Verlegung

- **WHEN** ein Spiel am 11.10. mit Anwurf 15:00 die Treffzeit 08:00 trägt (Abstand 7 h)
- **AND** der Anwurf wird auf 06:00 verlegt
- **THEN** liefert die API `meet_time = "23:00"` und `meet_date` = 10.10.

### Requirement: Treffzeit setzen und entfernen

Das System SHALL die Route `PUT /api/games/{id}/meeting` mit dem Body `{"meet_time": "HH:MM" | "", "meet_place": string}` bereitstellen.

- Ein nicht leerer `meet_time` SHALL als Uhrzeit am Spieltag gelesen werden. Gespeichert wird der Abstand zum aktuellen Anwurf.
- Liegt `meet_time` nach dem Anwurf, SHALL das Backend HTTP 400 mit `{"error": "meet_after_start"}` liefern.
- Ist der Abstand größer als 720 Minuten, SHALL das Backend HTTP 400 mit `{"error": "meet_offset_out_of_range"}` liefern.
- Ein ungültiges Uhrzeitformat SHALL zu HTTP 400 führen.
- `meet_place` SHALL getrimmt werden. Mehr als 100 Zeichen SHALL zu HTTP 400 mit `{"error": "meet_place_too_long"}` führen.
- Ein leerer `meet_time` SHALL Treffzeit **und** Ort entfernen. Ein nicht leerer `meet_place` zusammen mit leerem `meet_time` SHALL zu HTTP 400 mit `{"error": "meet_place_without_time"}` führen.
- Bei einem abgelehnten Request SHALL keine Spalte verändert werden.

Eine erfolgreiche Änderung SHALL HTTP 200 liefern, die berechneten Felder `meet_time`, `meet_date` und `meet_place` zurückgeben und an alle offenen Sitzungen, die das Spiel sehen, das Live-Update-Ereignis `games` senden.

#### Scenario: Treffzeit mit Ort setzen

- **WHEN** ein berechtigter Trainer für ein Spiel mit Anwurf 15:00 `PUT /api/games/{id}/meeting` mit `{"meet_time": "13:30", "meet_place": "Parkplatz Vereinsheim"}` aufruft
- **THEN** liefert das Backend HTTP 200 mit `meet_time = "13:30"` und `meet_place = "Parkplatz Vereinsheim"`
- **AND** ein anschließendes `GET /api/games/{id}` liefert dieselben Werte

#### Scenario: Treffzeit nach dem Anwurf wird abgelehnt

- **WHEN** für ein Spiel mit Anwurf 15:00 `meet_time = "15:30"` gesendet wird
- **THEN** liefert das Backend HTTP 400 mit `meet_after_start`
- **AND** eine zuvor gesetzte Treffzeit bleibt unverändert

#### Scenario: Ort ohne Uhrzeit wird abgelehnt

- **WHEN** `{"meet_time": "", "meet_place": "Halle"}` gesendet wird
- **THEN** liefert das Backend HTTP 400 mit `meet_place_without_time`

#### Scenario: Treffzeit entfernen

- **WHEN** für ein Spiel mit gesetzter Treffzeit und Ort `{"meet_time": "", "meet_place": ""}` gesendet wird
- **THEN** liefert das Backend HTTP 200
- **AND** das Spiel hat danach weder Treffzeit noch Ort

#### Scenario: Unbekanntes Spiel

- **WHEN** `PUT /api/games/{id}/meeting` für eine nicht existierende ID aufgerufen wird
- **THEN** liefert das Backend HTTP 404

### Requirement: Berechtigung zum Pflegen der Treffzeit

Das System SHALL `PUT /api/games/{id}/meeting` nur für **Trainer eines am Spiel beteiligten Teams** (Saison des Trainer-Eintrags aktiv), **sportliche_leitung**, **Vorstand** und **Admin** erlauben. Das ist derselbe Kreis, der den Hinweistext eines Spiels setzen darf. Alle anderen authentifizierten Nutzer SHALL HTTP 403 erhalten, ohne dass etwas verändert wird. Lesen dürfen alle, die das Spiel sehen dürfen.

#### Scenario: Trainer eines fremden Teams

- **WHEN** ein Trainer, der keinem Team des Spiels zugeordnet ist, `PUT /api/games/{id}/meeting` aufruft
- **THEN** liefert das Backend HTTP 403
- **AND** Treffzeit und Ort bleiben unverändert

#### Scenario: Spieler oder Elternteil

- **WHEN** ein Nutzer ohne die Vereinsfunktionen `trainer`, `sportliche_leitung` oder `vorstand` und ohne System-Rolle `admin` die Route aufruft
- **THEN** liefert das Backend HTTP 403

#### Scenario: Sportliche Leitung ohne Team-Bezug

- **WHEN** ein Nutzer mit der Vereinsfunktion `sportliche_leitung` die Treffzeit eines beliebigen Spiels setzt
- **THEN** liefert das Backend HTTP 200

### Requirement: Pflege der Treffzeit in der Oberfläche

Das System SHALL im Kalender-Modal eines Spiels und auf der Termin-Detailseite eines Spiels einen Editor für Treffzeit (Uhrzeitfeld) und Treffpunkt-Ort (Textfeld, optional) anbieten, und zwar nur Nutzern, die nach der Berechtigungsregel schreiben dürfen. Der Editor SHALL den Hinweis zeigen, dass sich die Treffzeit mit dem Anwurf verschiebt. Er SHALL ein Entfernen der Treffzeit erlauben. Fehlercodes des Backends SHALL als verständlicher Text erscheinen. Der Bearbeiten-Dialog eines Spiels und der Anlegen-Dialog SHALL in diesem Change kein Treffzeit-Feld erhalten.

#### Scenario: Berechtigter sieht den Editor

- **WHEN** ein Trainer eines beteiligten Teams das Kalender-Modal oder die Detailseite eines Spiels öffnet
- **THEN** sieht er den Treffzeit-Editor mit den aktuellen Werten

#### Scenario: Unberechtigter sieht nur die Anzeige

- **WHEN** ein Spieler die Detailseite eines Spiels mit gesetzter Treffzeit öffnet
- **THEN** sieht er Treffzeit und Ort, aber keinen Editor

### Requirement: Anzeige der Treffzeit

Das System SHALL eine gesetzte Treffzeit an folgenden Stellen anzeigen. Ohne Treffzeit SHALL dort nichts zusätzlich erscheinen.

- **Terminliste `/termine`**: eigene Zeile an der Spielkarte, „Treffen HH:MM" und, falls gesetzt, „ · Ort".
- **Termin-Detailseite** und **Kalender-Modal**: unter dem Anwurf „Treffen HH:MM Uhr" und, falls gesetzt, „ · Ort".
- **Dashboard**: an der Terminzeile die Ergänzung „(Treffen HH:MM)", ohne Ort.
- **iCal-Feed**: als erster Absatz der `DESCRIPTION` „Treffen: HH:MM Uhr" und, falls gesetzt, „, Ort". `DTSTART` SHALL weiterhin der Anwurf sein.

Liegt `meet_date` vor dem Spieldatum, SHALL jede dieser Anzeigen den Hinweis „Vortag" bzw. das Datum der Treffzeit tragen.

#### Scenario: Terminliste zeigt Treffzeit und Ort

- **WHEN** ein Spieler `/termine` öffnet und ein Spiel seines Teams die Treffzeit 13:30 mit Ort „Parkplatz Vereinsheim" trägt
- **THEN** zeigt die Spielkarte „Treffen 13:30 · Parkplatz Vereinsheim"

#### Scenario: iCal-Eintrag behält den Anwurf als Beginn

- **WHEN** ein Kalender-Feed ein Spiel mit Anwurf 15:00 und Treffzeit 13:30 ausliefert
- **THEN** ist `DTSTART` 15:00 Berlin-Zeit
- **AND** die `DESCRIPTION` beginnt mit „Treffen: 13:30 Uhr"

#### Scenario: Live-Aktualisierung

- **WHEN** ein Trainer die Treffzeit ändert, während ein Spieler `/termine` geöffnet hat
- **THEN** zeigt die Seite des Spielers die neue Treffzeit ohne manuelles Neuladen

### Requirement: Gebündelte Push bei Änderung der Treffzeit

Das System SHALL **5 Minuten nach der letzten Änderung** von Treffzeit oder Ort eines Spiels eine Push-Benachrichtigung der Kategorie `games` versenden. Die Empfängermenge SHALL der Regel der Capability `terminmeldung-empfaenger` folgen. Jeder weitere erfolgreiche `PUT` innerhalb des Fensters SHALL den Timer zurücksetzen, sodass mehrere Korrekturen nur **eine** Meldung auslösen.

Maßgeblich ist der **Netto-Unterschied** zwischen dem Stand vor der ersten Änderung des Fensters und dem Stand zum Sendezeitpunkt:
- Unterscheiden sich beide nicht (Abstand und Ort gleich), SHALL keine Meldung versendet werden.
- Ist zum Sendezeitpunkt eine Treffzeit gesetzt, SHALL die Meldung Team, Gegner bzw. Terminname, Datum, Treffzeit und, falls gesetzt, Ort enthalten.
- Wurde eine zuvor bestehende Treffzeit entfernt, SHALL die Meldung angeben, dass die Treffzeit entfällt.

Die `url` SHALL auf `/termine?focus=game-<id>` zeigen. Für Spiele, deren Datum vor dem heutigen Tag (Europe/Berlin) liegt, und für inzwischen gelöschte Spiele SHALL keine Meldung versendet werden. Der wartende Eintrag SHALL nach der Verarbeitung in jedem Fall entfernt werden.

#### Scenario: Mehrere Korrekturen ergeben eine Meldung

- **WHEN** ein Trainer die Treffzeit auf 13:30 setzt und zwei Minuten später auf 13:15 korrigiert
- **THEN** versendet das System etwa 5 Minuten nach der zweiten Änderung genau eine Meldung mit „Treffen 13:15 Uhr"

#### Scenario: Zurückgenommene Änderung löst nichts aus

- **WHEN** eine Treffzeit von 13:30 auf 13:00 und innerhalb des Fensters zurück auf 13:30 geändert wird
- **THEN** versendet das System keine Meldung

#### Scenario: Entfernte Treffzeit

- **WHEN** eine bestehende Treffzeit entfernt wird und bis zum Sendezeitpunkt nicht neu gesetzt ist
- **THEN** erhalten die Empfänger eine Meldung, dass die Treffzeit entfällt

#### Scenario: Gelöschtes Spiel

- **WHEN** ein Spiel mit wartender Treffzeit-Meldung vor dem Sendezeitpunkt gelöscht wird
- **THEN** wird keine Treffzeit-Meldung versendet

### Requirement: Vorbefüllung der Mitfahrgelegenheiten

Das System SHALL beim Anlegen eines **neuen** Angebots oder Gesuchs für ein Spiel mit gesetztem Treffpunkt-Ort das Feld „Treffpunkt" mit diesem Ort vorbefüllen. Der Nutzer SHALL den Wert ändern oder leeren können. Beim Bearbeiten eines bestehenden Eintrags SHALL der gespeicherte Treffpunkt des Eintrags erhalten bleiben, auch wenn er vom Spiel-Ort abweicht. Eine spätere Änderung des Spiel-Orts SHALL bestehende Einträge nicht verändern.

#### Scenario: Neues Angebot übernimmt den Ort

- **WHEN** ein Elternteil für ein Spiel mit Treffpunkt-Ort „Parkplatz Vereinsheim" ein neues Angebot öffnet
- **THEN** steht im Feld „Treffpunkt" bereits „Parkplatz Vereinsheim"

#### Scenario: Bestehender Eintrag bleibt unberührt

- **WHEN** ein Nutzer sein bestehendes Angebot mit Treffpunkt „Bahnhof" bearbeitet und das Spiel den Ort „Parkplatz Vereinsheim" trägt
- **THEN** steht im Feld „Treffpunkt" weiterhin „Bahnhof"
