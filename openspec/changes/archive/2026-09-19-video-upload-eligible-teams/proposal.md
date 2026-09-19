## Why

Das Desktop-Tool (`tools/video-encoder`) navigiert Mannschaft → Spiel und füllt beide Listen aus `GET /api/teams` und `GET /api/games`. Beide Endpunkte kennen nur die Kader-Zugehörigkeit (Trainer, Spieler, Eltern), nicht den Dienst-Pfad über `duty_types.grants_video_upload`. Wer einen Video-/Kamera-Dienst bei einer Mannschaft hat, mit der er sonst nicht verbunden ist, sieht diese Mannschaft im Tool nie und erreicht das Spiel nicht, das `GET /api/videos/upload-eligible-games` ihm ausdrücklich zuspricht. Die Server-Zusage „Vereinigung aus Rolle und Dienst" (Change `video-download-duty-upload`) wird vom Tool unterlaufen. Prod-Fall 2026-09-19: Florian Steinle (Trainer mB2, Elternteil) hat den Kamera-Dienst bei der mA2, konnte aber nur mB2 wählen. Sein Eltern-Weg ist seit `teamfilter-trainer-elternteil` gedeckt; der strukturelle Rest (Dienst bei einer Mannschaft ohne jede Kader-Verbindung) nicht.

## What Changes

- `GET /api/videos/upload-eligible-games` liefert neben `game_ids` (bleibt, für ältere Tool-Versionen) zwei neue Felder: `teams` (Mannschaften, für die der Nutzer hochladen darf, mit Kennzeichen `upload_without_game`, ob ein Upload ohne Spielbezug erlaubt ist) und `games` (die berechtigten Spiele mit Datum, Gegner, Saison und Team-IDs). Ein Endpoint beantwortet „was darf ich wo hochladen" vollständig.
- Das Desktop-Tool bezieht Mannschafts- und Spielliste ausschließlich aus diesem Endpoint; die Aufrufe von `GET /api/teams` und `GET /api/games` im Tool entfallen. „Freier Titel" (Upload ohne Spiel) wird nur für Mannschaften mit `upload_without_game` angeboten, sonst liefe der Upload in einen 403.
- Die Auswahl-Logik des Tools (Mannschaften, Spiele je Mannschaft, Regel für „Freier Titel") wird in ein reines, unit-getestetes Paket ausgelagert; die Fyne-Oberfläche bleibt dünn.
- Härtung in `POST /api/videos`: Wer nur über den Dienst-Pfad berechtigt ist, MUSS ein `team_id` angeben, das zu den Mannschaften des Spiels gehört, sonst HTTP 400. Bisher ließ sich ein Dienst-Upload an eine beliebige, fremde Mannschaft hängen und wurde dort sichtbar.
- Veralteter Hinweistext im Tool („für Trainer nur die eigenen") wird ersetzt.
- Kein Eingriff in `GameVisibilityClause` oder `GET /api/teams`: die Sichtbarkeit von Terminen ist eine andere Frage als die Upload-Berechtigung.

## Capabilities

### New Capabilities

_keine_

### Modified Capabilities

- `video-upload`: Requirement „Eigene upload-berechtigte Spiele abrufen" liefert zusätzlich Mannschaften (mit Kennzeichen für Upload ohne Spielbezug) und Spieldaten; Requirement „Upload-Initialisierung mit Pre-Disk-Check" bindet den Dienst-Pfad an ein `team_id` des Spiels.
- `video-offline-encoding-tool`: neues Requirement „Auswahl folgt der Upload-Berechtigung" — Mannschafts- und Spielauswahl stammen aus dem Berechtigungs-Endpoint, „Freier Titel" nur bei Rollen-Berechtigung.

## Impact

- **Backend:** `internal/videos/eligible_games.go` (Response erweitern), `internal/videos/upload.go` (team_id-Prüfung im Dienst-Pfad), Tests in `internal/videos/`. Keine Migration, keine neue Route, kein Broadcast (reine Lese-Route bleibt in der `broadcastAllowlist`-Begründung unverändert).
- **Desktop-Tool:** `tools/video-encoder/internal/client` (neues Response-Modell, `Teams()`/`Games()` entfallen), neues Paket `tools/video-encoder/internal/pick`, `tools/video-encoder/ui.go`. CI-Job `video-encoder` testet `./internal/...`.
- **Rollout:** Server-Deploy allein genügt nicht — der Fix kommt erst mit einem neuen Tool-Release bei den Filmenden an (`release.yml` hängt die Binaries an jedes Release). Alte Tool-Versionen laufen gegen den erweiterten Endpoint unverändert weiter (`game_ids` bleibt).
- **Sichtbarkeit fertiger Videos** bleibt unberührt (`CanViewVideo`/`visibilityFilter`/`pushRecipients`).
