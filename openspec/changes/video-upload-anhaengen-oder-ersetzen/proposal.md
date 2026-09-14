## Why

Ein Spiel wird in der Praxis oft nicht in einer einzigen Datei gefilmt — getrennte Aufnahmen je Halbzeit sind der Normalfall, dazu kommt ein missglückter erster Versuch (falsches Format, abgebrochene Aufnahme), der komplett neu hochgeladen werden soll. Seit `video-speicherplatz` zeigt der Video-Encoder in der Spiel-Auswahl bereits an, wenn zu einem Termin schon (ein) Video(s) existieren — aber das Tool bietet keine Möglichkeit, direkt dort zu entscheiden, ob der neue Upload **zusätzlich** (2. Halbzeit) oder **anstelle** des bestehenden (fehlgeschlagener erster Versuch) hochgeladen werden soll. Heute bleibt nur der Umweg über die Video-Liste in TeamWERK, das alte Video von Hand zu finden und zu löschen — und `video-multi-upload` verbietet aktuell sogar ausdrücklich, dass ein Upload ein bestehendes Video automatisch ersetzt.

## What Changes

- **Video-Encoder-Tool**: Sobald das gewählte Spiel bereits (ein) Video(s) hat, erscheint eine Auswahl **„Hinzufügen"** (Default, heutiges Verhalten unverändert — ein weiteres Video entsteht, z. B. 2. Halbzeit) vs. **„Ersetzen"** (das/die bestehende(n) Video(s) zu diesem Spiel werden ersetzt).
- Bei **„Ersetzen"** fragt das Tool vor dem Start noch einmal ausdrücklich nach Bestätigung (unwiderrufliche Löschung), lädt danach das neue Video wie gewohnt hoch und löscht **erst nach dessen erfolgreichem Abschluss** die zuvor vorhandene(n) Video-Zeile(n) zu diesem Spiel über die bestehende `DELETE /api/videos/{id}`-Route — nie umgekehrt, damit ein Datenverlust nur bei einem tatsächlich erfolgreichen Ersatz eintritt.
- Schlägt das Löschen fehl (z. B. `sportliche_leitung` darf hochladen, aber nicht löschen; Netzwerkfehler), bleibt der Lauf trotzdem ein Erfolg: das Tool meldet den Upload als abgeschlossen und weist zusätzlich darauf hin, das alte Video manuell in TeamWERK zu löschen.
- **Kein Schema-Change, keine neue Server-Route.** Das Tool kombiniert ausschließlich bestehende Endpunkte (`POST /api/videos`, tus-Upload, `DELETE /api/videos/{id}`) in neuer Reihenfolge.
- Passt die bestehende `video-multi-upload`-Anforderung „ein Upload ersetzt nie automatisch" an: das bleibt der Standardfall, aber ein **ausdrücklich vom Nutzer gewählter** Ersatz-Upload darf jetzt gezielt die zuvor vorhandenen Videos desselben Spiels löschen.

**Nicht Teil dieser Änderung:** kein Zusammenfügen mehrerer Dateien zu einem einzigen durchgehenden Video (Server-seitiges Concat/eine gemeinsame Timeline) — „Hinzufügen" legt weiterhin ein eigenständiges zweites Video an, wie es `video-multi-upload` bereits vorsieht. Die Auswahl gilt außerdem nur für Spiel-gebundene Videos (`game_id`); bei „Freier Titel" bleibt das Verhalten unverändert (immer ein neues Video).

## Capabilities

### Modified Capabilities

- `video-multi-upload`: die Anforderung „ein Upload ersetzt nie automatisch ein bestehendes Video" bekommt eine ausdrückliche, vom Nutzer aktiv gewählte Ausnahme.
- `video-offline-encoding-tool`: neue Anforderung für die Hinzufügen/Ersetzen-Auswahl bei bereits vorhandenem Video zum gewählten Spiel.

## Impact

- `tools/video-encoder/internal/client/client.go` — `ExistingVideo` trägt zusätzlich alle vorhandenen Video-IDs des Spiels; neue Methode `DeleteVideo(ctx, videoID)`.
- `tools/video-encoder/ui.go` — Hinzufügen/Ersetzen-Steuerelement (sichtbar nur bei vorhandenem Video zum gewählten Spiel), Bestätigungsdialog vor dem Start, Löschaufruf nach erfolgreichem Upload, Warnhinweis bei fehlgeschlagenem Löschen.
- `internal/videos/` (Server): keine Code-Änderung — `DELETE /api/videos/{id}` und dessen Berechtigungsprüfung (`CanManageTeamVideos`) existieren bereits unverändert.
- Tests: Go-Unit-Tests in `tools/video-encoder/internal/client/client_test.go` für `DeleteVideo`/erweitertes `VideosByGame`.
- Keine Migration, kein Deploy-Sonderschritt — reine Tool-Änderung, die beim nächsten Tool-Release automatisch verfügbar ist (siehe `docs/agent/10-deployment.md` „Cutover Video-Encoder-Tool" für den Release-Mechanismus).

## Test-Anforderungen

| Bereich | Test | Erwartung |
|---|---|---|
| `client.VideosByGame` | `TestVideosByGame` (erweitert) | liefert je Spiel alle vorhandenen Video-IDs, nicht nur die für die Anzeige „gewinnende" |
| `client.DeleteVideo` | `TestDeleteVideo_HappyPath` | `DELETE /api/videos/{id}` wird aufgerufen, Erfolg bei HTTP 200 |
| `client.DeleteVideo` | `TestDeleteVideo_Forbidden` | HTTP 403 wird als Fehler zurückgegeben (Tool wertet das als Warnhinweis, nicht als Gesamt-Fehlschlag) |

**Garantierte Invariante:** Ein „Ersetzen"-Lauf löscht das alte Video **nur nach** einem bereits erfolgreich abgeschlossenen neuen Upload; ein fehlgeschlagener neuer Upload lässt das alte Video unangetastet. Das gerade hochgeladene Video wird nie versehentlich mitgelöscht.
