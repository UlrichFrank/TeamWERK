# video-offline-encoding-tool Specification

## Purpose

Ein einmalig herunterladbares Desktop-Werkzeug, das Spielvideos lokal auf dem Rechner der Filmenden in das von TeamWERK erwartete Zielformat encodiert und direkt hochlädt, damit die CPU-intensive Encoding-Arbeit nicht mehr auf dem 1-GB-RAM-Produktions-VPS anfällt.

## Requirements

### Requirement: Lokales Encoding mit deterministischem Zielformat

Das Tool SHALL eine ausgewählte Videodatei lokal so encodieren, dass das Ergebnis mit dem bisherigen Server-Zielformat identisch ist: H.264-Video, `yuv420p`-Pixelformat, Skalierung auf 720p Höhe, CRF 26, Preset `medium`, erzwungene Keyframes alle 4 Sekunden, AAC-Audio (128 kbit/s, oder Stream-Copy wenn die Quelle bereits AAC ist). Zusätzlich MUST das Tool ein festes Video-Profil und -Level (z. B. High@3.1) an ffmpeg übergeben, statt sich auf den libx264-Standardwert zu verlassen, damit das erzeugte Profil unabhängig vom Quellinhalt über alle Tool-Versionen hinweg stabil bleibt.

#### Scenario: Encoding-Ergebnis entspricht dem Zielformat
- **WHEN** eine Videodatei durch das Tool encodiert wird
- **THEN** ist die Ausgabedatei H.264/`yuv420p` mit dem konfigurierten festen Profil/Level, Höhe 720p, und Keyframes alle 4 Sekunden

#### Scenario: Profil bleibt über verschiedene Quellinhalte stabil
- **WHEN** zwei unterschiedlich komplexe Quellvideos mit derselben Tool-Version encodiert werden
- **THEN** tragen beide Ausgabedateien exakt dasselbe Video-Profil/-Level

### Requirement: Gebündeltes ffmpeg ohne externe Installation

Das Tool SHALL eine ffmpeg-Binary für die jeweilige Zielplattform (Windows, macOS) in sich eingebettet mitliefern und benötigt keine separat installierte ffmpeg-Version auf dem Zielsystem. Beim ersten Start MUST das Tool die eingebettete Binary in ein lokales Cache-Verzeichnis entpacken, falls dort noch keine (oder eine veraltete) Kopie vorhanden ist, und diese für alle folgenden Encodes wiederverwenden.

#### Scenario: Erster Start ohne vorhandenes Cache-Verzeichnis
- **WHEN** das Tool zum ersten Mal auf einem Rechner gestartet wird
- **THEN** entpackt es die eingebettete ffmpeg-Binary ins Cache-Verzeichnis, bevor der erste Encode-Vorgang beginnt

#### Scenario: Folgestart nutzt bereits entpackte Binary
- **WHEN** das Tool ein zweites Mal gestartet wird und das Cache-Verzeichnis bereits eine passende ffmpeg-Version enthält
- **THEN** wird nicht erneut entpackt, der Encode-Vorgang startet direkt

### Requirement: Anmeldung wie ein Browser gegen bestehende Auth-Endpunkte

Das Tool SHALL sich mit E-Mail und Passwort gegen den bestehenden `POST /api/auth/login`-Endpunkt anmelden und dabei einen Cookie-Jar verwenden, der Set-Cookie-Header (Refresh-Token) genauso verarbeitet wie ein Webbrowser. Es SHALL KEINE neuen, tool-spezifischen Auth-Endpunkte oder -Flows auf dem Server voraussetzen.

#### Scenario: Erfolgreicher Login
- **WHEN** ein Nutzer im Tool gültige E-Mail/Passwort-Kombination eingibt
- **THEN** erhält das Tool einen Access-Token aus der Response und der Cookie-Jar speichert den Refresh-Token-Cookie für Folge-Requests

#### Scenario: Ungültige Zugangsdaten
- **WHEN** ein Nutzer falsche Zugangsdaten eingibt
- **THEN** zeigt das Tool eine verständliche Fehlermeldung und startet keinen Encode-/Upload-Vorgang

### Requirement: Token-Refresh während eines langen Uploads

Wenn ein Chunk-Request während des Uploads mit HTTP 401 beantwortet wird (Access-Token nach 15 Minuten abgelaufen), SHALL das Tool automatisch `POST /api/auth/refresh` aufrufen und den fehlgeschlagenen Chunk mit dem neuen Access-Token wiederholen, ohne den Upload sichtbar zu unterbrechen oder neu zu starten. Schlägt der Refresh selbst mit HTTP 401 fehl (Refresh-Token abgelaufen), SHALL das Tool den Upload sauber abbrechen und eine Meldung anzeigen, die zu einer erneuten Anmeldung auffordert.

#### Scenario: Access-Token läuft mitten im Upload ab
- **WHEN** ein Chunk-Request nach mehr als 15 Minuten Laufzeit 401 zurückliefert
- **THEN** ruft das Tool den Refresh-Endpunkt auf und wiederholt denselben Chunk mit dem neuen Token, ohne den Gesamt-Upload abzubrechen

#### Scenario: Refresh-Token abgelaufen
- **WHEN** der Refresh-Aufruf selbst mit HTTP 401 fehlschlägt
- **THEN** bricht das Tool den Upload ab und fordert den Nutzer zur erneuten Anmeldung auf

### Requirement: Wiederverwendung des bestehenden Upload-Wegs

Das Tool SHALL zum Hochladen ausschließlich die bestehenden, unveränderten Server-Endpunkte verwenden: `POST /api/videos` zur Metadaten-Initialisierung (Titel, Team, Saison, optional Spiel) und den tus-Protokoll-Endpunkt `/api/videos/upload/` für die eigentliche Dateiübertragung. Es SHALL KEIN neues Upload-Dateiformat (z. B. Archiv aus HLS-Segmenten) und KEINEN neuen Server-Endpoint für den Datei-Transfer voraussetzen.

#### Scenario: Erfolgreicher Upload über bestehende Endpunkte
- **WHEN** das Tool einen Encode-Vorgang abgeschlossen hat und der Nutzer den Upload bestätigt
- **THEN** ruft das Tool zuerst `POST /api/videos` und anschließend den tus-Endpunkt mit der encodierten Datei auf, identisch zum bisherigen Browser-Upload-Vertrag

#### Scenario: Wiederaufnahme nach Verbindungsabbruch
- **WHEN** die Netzwerkverbindung während eines tus-Chunk-Uploads abbricht und später wiederhergestellt wird
- **THEN** setzt das Tool denselben tus-Session-Upload an der zuletzt bestätigten Byte-Position fort, statt neu zu beginnen

### Requirement: Einfache Bedienung ohne Zusatzfunktionen

Das Tool SHALL eine minimale Oberfläche bieten: Dateiauswahl per Systemdialog oder per Drag & Drop, sowie eine Fortschrittsanzeige für Encoding und Upload. Es SHALL KEINE Kapitel-/Marken-Funktion, keine Vorschau-/Trimm-Funktion und keine sonstige Videobearbeitung anbieten.

#### Scenario: Datei per Dialog auswählen
- **WHEN** ein Nutzer den Datei-auswählen-Button klickt
- **THEN** öffnet sich der native Systemdateidialog und die gewählte Datei wird als Encoding-Quelle übernommen

#### Scenario: Datei per Drag & Drop
- **WHEN** ein Nutzer eine Videodatei auf das Tool-Fenster zieht und ablegt
- **THEN** wird diese Datei als Encoding-Quelle übernommen, ohne dass der Systemdialog geöffnet werden muss

### Requirement: Verteilung über GitHub Releases

Fertige Tool-Binaries für Windows und macOS SHALL als Assets an GitHub-Releases des öffentlichen TeamWERK-Repositories angehängt werden. Der Download-Dropdown auf `/videos` SHALL direkt auf die Asset-URLs des jeweils neuesten Releases verlinken; es SHALL KEIN Hosting der Binaries auf dem TeamWERK-VPS und KEIN Authentifizierungs-Proxy für den Download erforderlich sein.

#### Scenario: Download-Link zeigt auf aktuelles Release
- **WHEN** ein Nutzer auf `/videos` „Tool für Windows herunterladen" klickt
- **THEN** wird die Windows-Binary des jeweils neuesten GitHub-Releases heruntergeladen

#### Scenario: Neues Release aktualisiert automatisch die Download-Links
- **WHEN** ein neues GitHub-Release mit aktualisierten Tool-Binaries veröffentlicht wird
- **THEN** verweisen die Download-Links auf `/videos` ohne manuelle Änderung auf die neuen Binaries
