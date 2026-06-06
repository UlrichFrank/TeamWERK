## Why

Auf Android schlägt der Profilbild-Upload fehl, weil der Browser `image/jpg` statt `image/jpeg` als Content-Type sendet und das Backend diesen Typ ablehnt. Zusätzlich fehlt jede Möglichkeit, einen Bildausschnitt zu wählen oder die Auflösung zu reduzieren — Nutzer laden ungekürzte 8 MP-Fotos hoch.

## What Changes

- **Neues Crop-Modal** (`ImageCropModal`): Öffnet sich nach Dateiauswahl, zeigt kreisförmige Zuschnitt-Vorschau mit Drag (Position) und Pinch-to-Zoom + Slider (Zoom). Kein neues npm-Paket — rein Canvas-basiert.
- **Canvas-Resize**: Ausgabebild wird auf max. 600 × 600 px skaliert und als JPEG (Qualität 85 %) exportiert. Resultierende Dateigröße typisch 50–150 KB statt mehrerer MB.
- **Einheitliche Integration**: Das Modal wird an allen drei Upload-Stellen eingebunden — eigenes Profil, Kindprofil (Elternteil-Sicht), Admin-Mitgliedsstammdaten.
- **Backend-Robustheit**: `image/jpg` als Alias für `image/jpeg` akzeptieren; Magic-Byte-Sniff als Fallback wenn Content-Type fehlt oder leer ist.

## Capabilities

### New Capabilities

- `profilbild-crop-upload`: Interaktiver Bildausschnitt mit Zoom vor dem Upload; Canvas-basierte Resize-Pipeline; robuste MIME-Type-Erkennung im Backend.

### Modified Capabilities

*(keine bestehenden Specs betroffen)*

## Impact

- **Frontend**: Neue Komponente `web/src/components/ImageCropModal.tsx`; Änderungen in `ProfileProfilTab.tsx`, `ChildProfilePage.tsx`, `MemberStammdatenTab.tsx`
- **Backend**: `internal/upload/handler.go` — MIME-Type-Whitelist erweitern, Magic-Byte-Sniff ergänzen
- **Abhängigkeiten**: Keine neuen npm-Pakete, keine neuen Go-Module
- **Datenmenge**: Uploads werden clientseitig auf ~100 KB begrenzt — weniger Speicherverbrauch auf dem VPS
