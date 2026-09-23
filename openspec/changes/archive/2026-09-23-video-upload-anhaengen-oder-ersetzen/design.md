## Context

`video-speicherplatz` hat dem Video-Encoder-Tool bereits beigebracht, in der Spiel-Auswahl anzuzeigen, wenn zu einem Termin schon ein Video existiert (`client.VideosByGame`, Label-Zusatz „Video vorhanden: …" in `ui.go`). Diese Änderung baut direkt darauf auf: aus der reinen Anzeige wird eine Entscheidung.

Das Tool spricht ausschließlich bestehende Server-Routen an (`POST /api/videos`, tus-Upload, `DELETE /api/videos/{id}`) — genau wie ein Browser. Diese Änderung fügt keine neue Route hinzu, sondern nutzt `DELETE /api/videos/{id}` (existiert seit `spielvideo-ablage`, Berechtigung über `CanManageTeamVideos`: Trainer des Teams, Vorstand, Admin) aus dem Tool heraus.

## Goals / Non-Goals

**Goals:**
- Im Tool zwischen „Hinzufügen" (weiteres Video, Default) und „Ersetzen" (vorhandene(s) Video(s) zu diesem Spiel löschen) wählen können.
- Datenverlust nur nach einem tatsächlich erfolgreichen neuen Upload — nie vorher.
- Eine fehlgeschlagene Löschung (Berechtigung, Netzwerk) darf einen ansonsten erfolgreichen Upload nicht als Fehlschlag melden.

**Non-Goals:**
- Kein serverseitiges Zusammenfügen mehrerer Dateien zu einer durchgehenden Aufnahme/Timeline (Concat). „Hinzufügen" bleibt ein zweites, eigenständiges Video — das ist die von `video-multi-upload` bereits getragene, einfachere Lösung für „Film in mehreren Teilen", und die Video-Detailseite/-Liste kennt Mehrfach-Videos pro Spiel bereits (auch wenn ihre gebündelte Darstellung erst mit `videos-grouping` kommt).
- Keine Ersetzen-Option für „Freier Titel" (Videos ohne `game_id`). Titel-basiertes Matching wäre unscharf (Tippfehler, Groß-/Kleinschreibung) — das Risiko, versehentlich das falsche Video zu löschen, überwiegt den Nutzen für den selteneren Fall.
- Kein neuer Server-Endpunkt, keine Migration.

## Decisions

### Entscheidung: Löschen erst NACH erfolgreichem Upload, nie vorher

**Gewählt:** Reihenfolge im Tool ist immer Encode → `POST /api/videos` → tus-Upload bis zum Abschluss → **erst danach** `DELETE /api/videos/{id}` für jedes zuvor vorhandene Video.

**Warum:** Ein vorzeitiges Löschen (vor oder während des neuen Uploads) würde bei einem Abbruch — Netzwerk, Absturz, falsches Format, das der Server erst beim Transcode entdeckt — sowohl das alte als auch das (noch unfertige) neue Video verlieren. Die gewählte Reihenfolge stellt sicher: im schlimmsten Fall bleibt das alte Video erhalten und der Nutzer verliert nur Zeit, nie Daten.

**Alternative verworfen:** Alte(s) Video(s) vor dem Upload löschen, damit „Ersetzen" sich wie ein echtes Atomic-Replace anfühlt. Verworfen, weil das genau die Fehlerklasse öffnet, die diese Funktion eigentlich vermeiden soll.

### Entscheidung: Fehlschlagendes Löschen ist eine Warnung, kein Gesamt-Fehlschlag

**Gewählt:** `run()` gibt neben `(videoID, error)` zusätzlich einen optionalen Hinweistext zurück. Schlägt das Löschen einer alten Video-ID fehl, wird der Lauf trotzdem als Erfolg gemeldet (`error == nil`), aber die Erfolgsmeldung enthält einen zusätzlichen Absatz „das alte Video konnte nicht automatisch gelöscht werden — bitte manuell in TeamWERK löschen".

**Warum:** Der neue Upload ist zu diesem Zeitpunkt bereits sicher auf dem Server (tus-Session abgeschlossen). Diesen Erfolg als „Fehler" zu melden, nur weil ein nachgelagerter Aufräumschritt scheitert, wäre irreführend und würde den Nutzer zu einem unnötigen Retry verleiten, der den ganzen Upload wiederholt. Der typischste Fehlschlag ist ohnehin erwartbar: `sportliche_leitung` darf laut `CanUploadToTeam` hochladen, steht aber nicht in `CanManageTeamVideos` (nur Trainer/Vorstand/Admin) — für diese Rolle liefert das Löschen planmäßig 403.

**Alternative verworfen:** Löschfehler als Gesamt-Fehlschlag melden. Verworfen, weil es dem Nutzer suggeriert, der Upload sei nicht angekommen, obwohl er es ist.

### Entscheidung: Bestätigungsdialog vor dem Start, wenn „Ersetzen" gewählt ist

**Gewählt:** Klickt der Nutzer „Encodieren und hochladen" mit ausgewähltem „Ersetzen", zeigt das Tool zuerst einen `dialog.ShowConfirm` mit der Anzahl der zu löschenden Videos, bevor der eigentliche Lauf (Encode/Upload) startet.

**Warum:** Löschen ist unwiderruflich (kein serverseitiges Recovery für Videodateien). Ein Klick auf einen Button, der weit vorher (bei der Spiel-Auswahl) auf „Ersetzen" gestellt wurde, ist als alleinige Bestätigung zu schwach — der Encode-Vorgang dauert oft Minuten, in denen der Nutzer die Auswahl vergessen haben kann.

**Alternative verworfen:** Keine zusätzliche Bestätigung, da die Radio-Auswahl selbst schon eine bewusste Entscheidung ist. Verworfen zugunsten von Sicherheit gegen einen unbeabsichtigten Klick, zumal das Encoding lange genug dauert, dass der Kontext beim späteren Löschen nicht mehr präsent sein muss.

### Entscheidung: „Ersetzen" löscht ALLE vorhandenen Videos des Spiels, nicht nur eines

**Gewählt:** `ExistingVideo` trägt jetzt `IDs []int` — alle Video-Zeilen mit diesem `game_id`, nicht nur die für das Label „gewinnende". „Ersetzen" löscht sie alle (mit Sicherheitsnetz: die gerade neu hochgeladene `video_id` wird dabei nie gelöscht, auch wenn sie aus irgendeinem Grund in der Liste auftauchen sollte).

**Warum:** „Ersetzen" soll den Zustand herstellen, den der Nutzer erwartet — „zu diesem Spiel gibt es jetzt genau mein neues Video", nicht „eines von vorher ist weg, der Rest bleibt liegen". Bei zwei vorhandenen Halbzeiten-Videos, die beide durch einen fehlerhaften Dreh ungültig sind, wäre ein Teil-Ersatz verwirrend.

**Alternative verworfen:** Nur das zuletzt hochgeladene/„relevanteste" Video ersetzen. Verworfen, weil unklar bliebe, welches der mehreren Videos gemeint ist, und der Rest als Karteileiche zurückbliebe.

### Entscheidung: Kein Merge/Concat mehrerer Dateien zu einem Video

**Gewählt:** „Mehrteiliger Film" bleibt technisch mehrere Video-Zeilen (wie schon in `video-multi-upload` vorgesehen); diese Änderung fügt nur die Ersetzen-Option hinzu.

**Warum:** Ein echtes Server-seitiges Concat (eine durchgehende HLS-Timeline aus zwei Quelldateien) bräuchte einen neuen Encoding-Schritt, eine neue Datenmodell-Frage („was passiert mit der Wiedergabeposition beim Wechsel?") und stünde in keinem Verhältnis zum eigentlichen Bedarf — zwei Halbzeiten als zwei anklickbare Videos sind für die Praxis ausreichend, zumal `videos-grouping` (separate, noch offene Änderung) genau das in der Anzeige bündeln soll.

**Alternative verworfen:** Serverseitiges Concat der Rohdateien vor dem Transcode. Verworfen als deutlich größerer Umfang ohne klaren Mehrwert gegenüber zwei separaten Videos.
