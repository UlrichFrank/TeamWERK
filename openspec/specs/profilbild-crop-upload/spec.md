# Spec: Profilbild-Crop-Upload

## Purpose

Diese Spezifikation beschreibt die Capability `profilbild-crop-upload`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Nutzer kann Profilbild mit Zuschnitt hochladen
Nach Auswahl einer Bilddatei MUST das System ein modales Crop-Interface öffnen, bevor der Upload stattfindet. Das Modal MUST einen kreisförmigen Ausschnitt (1:1) zeigen, den Nutzer per Drag positionieren und per Pinch-to-Zoom sowie Slider skalieren können. Erst nach Bestätigung wird das Bild verarbeitet und hochgeladen.

#### Scenario: Datei auswählen öffnet Crop-Modal
- **WHEN** ein Nutzer auf „Bild hochladen" / „Bild ersetzen" tippt und eine Bilddatei auswählt
- **THEN** öffnet sich das Crop-Modal mit dem gewählten Bild und einem kreisförmigen Zuschnitt-Overlay

#### Scenario: Bild per Drag positionieren
- **WHEN** der Nutzer im Crop-Modal das Bild mit Maus oder einem Finger zieht
- **THEN** verschiebt sich das Bild unter dem Kreis-Ausschnitt entsprechend der Drag-Bewegung

#### Scenario: Zoom per Pinch-to-Zoom (Touch)
- **WHEN** der Nutzer mit zwei Fingern pincht oder spreizt
- **THEN** wird das Bild im Crop-Modal proportional vergrößert oder verkleinert

#### Scenario: Zoom per Slider
- **WHEN** der Nutzer den Zoom-Slider bewegt
- **THEN** wird das Bild im Crop-Modal entsprechend skaliert

#### Scenario: Upload nach Bestätigung
- **WHEN** der Nutzer im Crop-Modal auf „Hochladen" klickt
- **THEN** wird der Zuschnitt auf max. 600 × 600 px skaliert, als JPEG (Qualität 85 %) exportiert und an die Upload-API gesendet

#### Scenario: Abbrechen ohne Upload
- **WHEN** der Nutzer im Crop-Modal auf „Abbrechen" klickt
- **THEN** schließt das Modal ohne Upload; der Datei-Input wird zurückgesetzt

### Requirement: Nicht unterstützte Bilddateien werden abgefangen
Das System MUST dem Nutzer eine verständliche Fehlermeldung anzeigen, wenn die gewählte Datei nicht als Bild interpretiert werden kann (z. B. beschädigte Datei, nicht unterstütztes Format).

#### Scenario: Fehler beim Laden des Bildes
- **WHEN** das gewählte Bild im Crop-Modal nicht geladen werden kann
- **THEN** schließt das Modal und zeigt eine Fehlermeldung „Bild konnte nicht geladen werden"

### Requirement: Backend akzeptiert robuste MIME-Types für Fotos
Der Server MUST Bildüberträge mit den Content-Types `image/jpeg`, `image/jpg`, `image/png` und `image/webp` akzeptieren. Falls der Content-Type fehlt oder leer ist, MUST der Server die Dateiart anhand der Magic Bytes bestimmen (JPEG: `FF D8`; PNG: `89 50 4E 47`; WebP: `52 49 46 46 ?? ?? ?? ?? 57 45 42 50`).

#### Scenario: Upload mit image/jpg (Android)
- **WHEN** ein Client ein JPEG-Bild mit Content-Type `image/jpg` hochlädt
- **THEN** akzeptiert der Server die Datei wie ein reguläres `image/jpeg`

#### Scenario: Upload ohne Content-Type
- **WHEN** ein Client eine Bilddatei ohne Content-Type-Header hochlädt
- **THEN** bestimmt der Server den Typ via Magic Bytes und akzeptiert JPEG, PNG oder WebP

#### Scenario: Ungültige Datei
- **WHEN** ein Client eine Datei hochlädt, die weder per Content-Type noch per Magic Bytes als erlaubtes Bildformat erkannt wird
- **THEN** antwortet der Server mit HTTP 400

### Requirement: Crop-Modal ist an allen Foto-Upload-Stellen verfügbar
Das Crop-Modal MUST einheitlich eingebunden sein bei: eigenem Profilbild (`ProfileProfilTab`), Kinderfoto durch Elternteil (`ChildProfilePage`), und Mitgliedsfoto durch Admin (`MemberStammdatenTab`).

#### Scenario: Eigenes Profilbild
- **WHEN** ein eingeloggter Nutzer auf der Profil-Seite ein Foto auswählt
- **THEN** öffnet sich das Crop-Modal

#### Scenario: Kinderfoto durch Elternteil
- **WHEN** ein Elternteil auf der Kind-Profil-Seite ein Foto für ein Kind auswählt
- **THEN** öffnet sich das Crop-Modal

#### Scenario: Mitgliedsfoto durch Admin
- **WHEN** ein Admin in den Stammdaten eines Mitglieds ein Foto auswählt
- **THEN** öffnet sich das Crop-Modal
