## Why

Trainer/sportliche Leitung/Vorstand sind aktuell die einzigen, die Spielvideos hochladen dürfen — auch wenn ein Verein die eigentliche Filmarbeit über die Dienstbörse an beliebige Mitglieder vergibt (Dienst-Typ „Video"). Wer diesen Dienst übernimmt, kann das aufgenommene Material bislang nicht selbst einstellen und muss es an eine berechtigte Person weiterreichen. Zusätzlich lässt sich ein Video bisher nur streamen, nicht herunterladen — für ein Offline-Archiv oder externe Nachbearbeitung fehlt der Weg.

## What Changes

- Vorstand kann einen Diensttyp unter `/admin/duty-types` als „berechtigt zum Video-Upload" markieren (neues Flag `duty_types.grants_video_upload`).
- Wer für ein konkretes Spiel eine Dienst-Zuweisung auf einen so markierten Diensttyp hat, darf für **genau dieses Spiel** ein Video hochladen — zusätzlich zur bestehenden Trainer-/sportliche-Leitung-/Vorstand-/Admin-Berechtigung. Kein teamweiter Freifahrtschein: die Berechtigung hängt am Termin, nicht am Team.
- Neuer Endpoint liefert dem Desktop-Tool (und potenziell dem Web-Frontend) die Menge der Spiele, für die der angemeldete Nutzer aktuell hochladen darf (Rollen-basiert + Dienst-basiert), statt serverseitig erst beim Anlegen mit 403 zu scheitern.
- Desktop-Tool (`tools/video-encoder`) stellt seine Spielauswahl auf diesen Endpoint um und fragt die aktive Saison über `GET /api/seasons/active` (Authenticated-Tier) statt `GET /api/seasons` ab, damit ein 403 dort nicht mehr fälschlich als „kein Upload-Recht" gilt.
- Neuer Download-Endpoint für fertige Videos: der Server remuxt die vorhandenen 720p-HLS-Segmente on-demand per `ffmpeg` zu einer MP4-Datei und liefert sie mit `Content-Disposition: attachment`, ohne die Datei dauerhaft zu cachen.
- Web-Frontend (`VideoDetailPage.tsx`) bekommt einen Download-Button neben dem bestehenden Cast-Button.
- Sichtbarkeit fertiger Videos (wer ein Video ansehen/streamen darf) bleibt **unverändert** — dieselbe Berechtigung gilt für Streaming und Download.

## Capabilities

### New Capabilities
- `video-download`: On-demand-Remux fertiger Videos zu MP4 und Auslieferung als Download, mit derselben Berechtigung wie das Streaming.

### Modified Capabilities
- `video-upload`: Upload-Berechtigung erweitert sich um einen dienst-basierten Pfad (Dienst-Zuweisung auf einen als upload-berechtigt markierten Diensttyp, gebunden an das konkrete Spiel der Zuweisung); neuer Endpoint zur Ermittlung der für den Nutzer aktuell upload-berechtigten Spiele.

## Impact

- **DB-Migration**: neue Spalte `duty_types.grants_video_upload` (nächste freie Migrationsnummer).
- **Backend**: `internal/videos/access.go` (`CanUploadToTeam` → zusätzlicher dienst-basierter Zweig, gebunden an `game_id`), `internal/videos/upload.go` (`CreateUpload` re-validiert den neuen Pfad hart serverseitig), neuer Handler/Route für „meine upload-berechtigten Spiele", neuer Handler/Route für Download (`internal/videos`, ffmpeg-Subprozess), `internal/duties` (Admin-Handler für das neue Flag).
- **Frontend**: `web/src/pages/admin/AdminDutyTypesPage.tsx` (neue Checkbox), `web/src/pages/VideoDetailPage.tsx` (Download-Button).
- **Desktop-Tool**: `tools/video-encoder/internal/client/client.go` (`ActiveSeasonID` auf `/api/seasons/active` umgestellt, neue Methode für upload-berechtigte Spiele), UI-Schicht des Tools (Spielauswahl-Filterung, Fehlermeldung bei fehlender Berechtigung für ein gewähltes Spiel).
- **Tests**: neue Go-Tests für Upload-Berechtigung (Happy-Path + Fehlerfall), neuen Endpoint für berechtigte Spiele, Download-Endpoint (inkl. Objektrechte-Matrix-Fixture, da `{id}`-Route); Vitest für die neue Admin-Checkbox und den Download-Button.
