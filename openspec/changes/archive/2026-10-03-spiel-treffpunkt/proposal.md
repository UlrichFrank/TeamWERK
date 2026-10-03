# Proposal

## Why

Vor fast jedem Spiel treffen sich Mannschaft und Trainer vorher, vor Auswärtsspielen und Turnieren meist an einem anderen Ort als der Halle („Treffen 13:30 am Parkplatz Vereinsheim"). TeamWERK kennt heute nur den Anwurf (`games.time`) und den Spielort (`venue_id`). Die Treffzeit wandert deshalb über den freien Hinweistext oder über Chat-Nachrichten, ohne feste Stelle in Terminliste, Kalender-Feed oder Erinnerung.

## What Changes

- **Neue optionale Treffzeit plus Treffpunkt-Ort je Spiel** für alle drei `event_type` (`heim`, `auswärts`, `generisch`). Die Treffzeit wird als **Abstand zum Anwurf** gespeichert (`games.meet_offset_minutes`), eingegeben und angezeigt aber als Uhrzeit. Ein verlegter Anwurf, auch durch den H4A-Import, nimmt die Treffzeit dadurch mit, statt eine veraltete Uhrzeit stehen zu lassen. Der Ort ist ein optionaler Freitext (`games.meet_place`) und existiert nur zusammen mit einer Treffzeit.
- **Neue Route `PUT /api/games/{id}/meeting`** mit `{meet_time: "HH:MM" | "", meet_place: string}`. Berechtigt ist derselbe Kreis wie beim Hinweistext: Trainer eines beteiligten Teams, sportliche Leitung, Vorstand, Admin. Ein leerer `meet_time` entfernt Treffzeit und Ort.
- **Pflege inline**: der neue `MeetingPointEditor` sitzt neben dem `EventNoteEditor` im Kalender-Modal (`EventInfoModal`) und auf der Termin-Detailseite. Das `GameEditModal` und der Anlegen-Dialog bekommen das Feld in diesem Change bewusst nicht.
- **Anzeige** überall, wo der Anwurf eines Spiels steht: Liste `/termine`, Termin-Detail, Kalender-Modal, Dashboard, iCal-`DESCRIPTION` (erster Absatz; `DTSTART` bleibt der Anwurf).
- **Spielerinnerungen (24 h/3 h)** nennen die Treffzeit und den Ort, wenn gesetzt.
- **Gebündelte Push bei Änderung**: 5 Minuten nach der letzten Änderung erhält die Empfängermenge der Terminmeldungen (`notify.TeamAudience`) eine Meldung mit der aktuellen Treffzeit, nach dem Muster der Hinweis-Push.
- **Mitfahrgelegenheiten**: der Treffpunkt-Ort befüllt das Feld `treffpunkt` eines **neuen** Angebots bzw. Gesuchs vor; bestehende Einträge bleiben unberührt.

## Capabilities

### New Capabilities
- `game-meeting-point`: Treffzeit und Treffpunkt-Ort je Spiel, Speicherung relativ zum Anwurf, Schreibroute und Berechtigung, Anzeige in Termine/Detail/Kalender-Modal/Dashboard/iCal, gebündelte Änderungs-Push, Vorbefüllung der Mitfahrgelegenheiten.

### Modified Capabilities
- `push-reminders`: Die Spielerinnerung (24 h und 3 h) nennt eine gesetzte Treffzeit samt Ort im Meldungstext.

## Impact

- **Backend:** `internal/games` (neuer Handler `UpdateGameMeeting`, Treffzeit-Felder in allen Spiel-Antworten, die `time` ausliefern), `internal/scheduler` (Debounce-Job für die Änderungs-Push, Text der Spielerinnerung), `internal/calendar` (DESCRIPTION), `internal/dashboard`, `internal/carpooling` (Spiel-Kopf liefert den Ort mit), `internal/app/router.go` (eine Route im selben Block wie `PUT /api/games/{id}/note`).
- **Datenbank:** additive Migration `073`: zwei Spalten an `games` und eine Debounce-Tabelle `pending_game_meeting_push`. Kein Tabellen-Rebuild nötig, die Konvention „Migrationen sind additiv" bleibt gewahrt.
- **Gates:** Die neue Mutationsroute broadcastet (`games`), daher kein Eintrag in die `broadcastAllowlist`. Die Objektrechte-Matrix braucht eine Fixture für `PUT /api/games/{id}/meeting` (Spiel eines fremden Teams, Trainer A → 403). Die Push läuft über `notify.Send`, also ist das Push-Fan-out-Gate erfüllt.
- **Frontend:** neue Komponente `MeetingPointEditor`, Anzeige in `TerminePage`, `TermineDetailPage`, `EventInfoModal`, `DashboardPage`, Vorbefüllung in `MitfahrgelegenheitenPage`.
- **Berechtigungsmodell:** kein neues Tier, Wiederverwendung von `canEditGameNote` (Vereinsfunktionen `trainer` mit Team-Bezug, `sportliche_leitung`, `vorstand`, System-Rolle `admin`). Lesen dürfen alle, die das Spiel sehen.
- **RAM/Last:** vernachlässigbar; ein minütlicher Scheduler-Schritt über eine meist leere Tabelle.

## Test-Anforderungen

| Route / Pfad | Test | Erwartet |
|---|---|---|
| `PUT /api/games/{id}/meeting` | `TestUpdateGameMeeting_TrainerSetztTreffzeit` | 200, `GET` liefert `meet_time`/`meet_place` |
| | `TestUpdateGameMeeting_FremderTrainer` | 403, Spalten unverändert |
| | `TestUpdateGameMeeting_SpielerVerboten` | 403 |
| | `TestUpdateGameMeeting_NachAnwurf` | 400 `meet_after_start`, Spalten unverändert |
| | `TestUpdateGameMeeting_ZuWeitVorher` | 400 `meet_offset_out_of_range` |
| | `TestUpdateGameMeeting_OrtOhneZeit` | 400 `meet_place_without_time` |
| | `TestUpdateGameMeeting_OrtZuLang` | 400 `meet_place_too_long` |
| | `TestUpdateGameMeeting_Entfernen` | 200, Abstand `NULL`, Ort leer |
| | `TestUpdateGameMeeting_Unbekannt` | 404 |
| Anwurf-Verlegung | `TestMeetingFolgtVerlegtemAnwurf` | `meet_time` verschiebt sich mit `PUT /api/games/{id}` |
| `timez.MeetTime` | `TestMeetTime_Vortag` | Anwurf 06:00, Abstand 420 → Vortag 23:00 |
| Scheduler | `TestMeetingPush_NettoUnveraendert` | keine Meldung |
| | `TestMeetingPush_MehrfachKorrekturEineMeldung` | genau eine Meldung, aktueller Stand |
| | `TestMeetingPush_Entfernt` | Meldung „entfällt" |
| | `TestMeetingPush_VergangenesSpiel` | keine Meldung, Zeile gelöscht |
| | `TestGameReminder_NenntTreffzeit` | Body enthält „Treffen 13:30 Uhr" |
| iCal | `TestCalendar_DescriptionNenntTreffzeit` | `DTSTART` = Anwurf, DESCRIPTION beginnt mit „Treffen:" |

**Invariante:** Die angezeigte Treffzeit ist immer `Anwurf − gespeicherter Abstand`. Kein Schreibpfad auf `games.time` muss die Treffzeit kennen.
