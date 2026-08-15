# Änderungs-Benachrichtigungen mit Termin und Aktor

## Why

`absage-benachrichtigung` hat die **Lösch**-Meldungen repariert: Name, Datum, Aktor, Grund.
Die **Änderungs**-Meldungen desselben Termins sind dabei unangetastet geblieben und stehen
jetzt als der auffälligste Rest da. Was heute auf dem Handy ankommt:

> **Spielinfo geändert** — Ferientraining mB — Details aktualisiert

Drei Probleme in einer Zeile:

1. **Welcher Termin?** Weder Datum noch Uhrzeit. Ein Ferientraining läuft über zwei Wochen;
   „Ferientraining mB" identifiziert nichts. Beim Löschen desselben Termins steht das Datum
   längst drin.
2. **Durch wen?** Steht nirgends — also auch nicht, wen man fragen kann. `notify.ActorName`
   existiert seit `absage-benachrichtigung` und wird hier nicht aufgerufen.
3. **„Spielinfo" bei einem generischen Termin.** Ein Ferientraining ist kein Spiel. Beim
   Löschen wählt `cancellationTitle(eventType)` bereits „Termin abgesagt"; der
   Änderungspfad kennt diese Unterscheidung nicht.

Bei den Trainings ist es dieselbe Lücke, eine Stufe leerer: „Training geändert — Eine
Trainingseinheit wurde aktualisiert" (`internal/trainings/handler.go:966`) nennt nicht
einmal den Titel der Einheit.

Und ein vierter Fall hat gar keine Meldung: `PUT /api/training-series/{id}` löscht und
regeneriert **alle** Sessions der Serie ab `from_date` (`handler.go:499-525`) — verschiebt
also potenziell Dutzende Termine im Kalender des Teams — und verschickt dabei nichts. Nur
das SSE-Live-Update läuft. Wer die App nicht offen hat, erfährt von der Verschiebung nichts.

Der Sonderfall, der den ganzen Change trägt: Wird ein Termin **verschoben**, nennt eine
Meldung mit nur dem neuen Datum den Termin, den der Empfänger noch gar nicht kennt. Er
sucht im Kalender nach dem alten und findet ihn nicht. Deshalb trägt die Meldung bei
tatsächlicher Verschiebung beide Zeitpunkte.

## What Changes

- **Neuer Textbaustein `notify.ChangeBody`** analog zu `CancellationBody` — eine Stelle,
  an der die Zusage „keine Änderungsmeldung ohne Terminname, Zeitpunkt und Aktor" prüfbar
  ist:

  ```
  Ferientraining mB am 20.08.2026 um 18:00 Uhr wurde geändert. Geändert von Tim Meier.
  ```

- **Vorher-Klammer bei Verschiebung.** Ändert der Request Datum oder Uhrzeit, nennt die
  Meldung zusätzlich den alten Zeitpunkt:

  ```
  Ferientraining mB am 20.08.2026 um 18:00 Uhr wurde geändert (vorher 19.08.2026,
  17:00 Uhr). Geändert von Tim Meier.
  ```

  Bleibt der Zeitpunkt gleich (Ort, Gegner, RSVP-Voreinstellung geändert), entfällt die
  Klammer ersatzlos.

- **Titel nach `event_type`** bei `PUT /api/games/{id}`: „Termin geändert" für
  `event_type='generisch'`, sonst weiterhin „Spielinfo geändert". Dieselbe Unterscheidung,
  die `cancellationTitle` beim Löschen schon trifft.

- **Trainingseinheiten** nennen ihren Titel (`sessionSubject`, bereits vorhanden), Datum,
  Startzeit und den Aktor. Der Absage-Zweig (`status → cancelled`) bleibt unverändert.

- **Trainingsserien bekommen erstmals eine Änderungs-Meldung** („Trainingsserie geändert")
  mit dem betroffenen Zeitraum, dem Aktor und — bei Verschiebung des Wochentags oder der
  Startzeit — dem alten Rhythmus („vorher montags 18:00 Uhr").

- **Die Neuanlage** (`POST /api/games`) reiht sich ein. Sie meldete bisher das rohe
  ISO-Datum aus dem Request und keinen Aktor („Heimspiel vs. HSG Ostfildern am
  2026-09-14"); künftig „… am 14.09.2026 um 18:00 Uhr. Angelegt von Tim Meier." Der Titel
  folgt auch hier dem `event_type` („Neuer Termin" statt „Neues Spiel" bei `generisch`).
  Die Trainings-Routen legen ohne Benachrichtigung an und bleiben unangetastet — dort
  fehlt die Meldung ganz, das ist ein eigener Wunsch.

## What Does Not Change

- **Kein `silent`-Flag für Änderungen.** Die Capability `suppress_event_notification` bleibt
  auf Löschungen beschränkt. Eine Änderung ohne Benachrichtigung ist ein anderer Wunsch mit
  eigenem Risiko (stille Verschiebung) und gehört in einen eigenen Change.
- **Kein Feld-Diff.** Die Meldung sagt „wurde geändert", nicht „Ort: Halle A → Halle B".
  Ein vollständiger Diff über alle Felder (inkl. RSVP-Voreinstellungen, Vorlage, Teams)
  wäre in einem Push-Body unlesbar; der Direktlink führt zum aktuellen Stand.
- **Kein Grundfeld.** Anders als bei der Löschung gibt es kein `reason` — die Änderung ist
  im Termin selbst sichtbar, die Löschung war es nicht.
- **Publikum, Kategorie und Link bleiben** wie gehabt (`teamMembersAndParents`, Kategorie
  `games`/`trainings`, `/termine?focus=…`). Bei der Serie ist der Link `/termine` ohne
  `focus` — die Serie ist kein einzelner Termin.

## Impact

- `internal/notify/change.go` (neu): `ChangeBody`, `CreationBody`, `EventWhen`,
  `EventMoment`, `PreviousMoment`, `FormatTimeHM`
- `internal/games/handler.go`: `UpdateGame` — Pre-Update-SELECT um `time` erweitert,
  Meldungsaufbau, `changeTitleFor`; `CreateGame` — Meldungsaufbau, `creationTitle`
- `internal/trainings/handler.go`: `UpdateSession` (Pre-Read um `date`/`start_time`
  erweitert), `UpdateSeries` (Pre-Read um `day_of_week`/`start_time`, neue Meldung)
- Specs: `push-games`, `push-trainings`
- Kein Frontend, keine Migration, keine neue Route.
