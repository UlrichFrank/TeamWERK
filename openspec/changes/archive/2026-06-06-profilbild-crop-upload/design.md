## Context

Profilbilder werden aktuell direkt nach Dateiauswahl per `FormData` an den Server gesendet. Der Backend-Handler prüft `Content-Type` aus dem Multipart-Header — auf Android Chrome liefert der Browser dabei `image/jpg` statt `image/jpeg`, was zur Ablehnung führt. Es gibt keine clientseitige Aufbereitung: Originaldateien (ggf. 8–20 MB) landen unverändert auf dem VPS.

Drei Upload-Stellen existieren:
- `ProfileProfilTab` → eigenes User-Foto (`/api/upload/user-photo`)
- `ChildProfilePage` → Kinderfoto via Elternteil (`/api/profile/kind/:memberId/photo`)
- `MemberStammdatenTab` → Mitgliedsfoto durch Admin (`/api/upload/member-photo/:id`)

## Goals / Non-Goals

**Goals:**
- Android-Upload reparieren (Content-Type-Toleranz im Backend)
- Kreisförmigen Zuschnitt mit Drag + Pinch-to-Zoom + Slider anbieten
- Ausgabegröße auf max. 600 × 600 px / JPEG 85 % begrenzen
- Wiederverwendbare Komponente an allen drei Upload-Stellen

**Non-Goals:**
- Originalbild aufbewahren (kein Multi-Resolution-Storage)
- Serverside Image Processing (kein Go-Imaging-Package)
- Freies Seitenverhältnis (immer 1:1 für Profilbilder)
- HEIC/HEIF-Support über Browser-Canvas hinaus

## Decisions

### 1. Resize + Crop im Frontend (Canvas API), nicht serverseitig

**Entscheidung:** Canvas-basierte Pipeline im Browser, kein Go-Imaging-Package.

**Begründung:**
- Canvas ist für den Crop sowieso nötig → Resize ist gratis (Ziel-Canvas-Größe = 600 px)
- `canvas.toBlob('image/jpeg', 0.85)` liefert immer sauberen Content-Type → Android-Bug entschärft
- Kein neues Go-Modul, kein RAM-Druck auf dem 1 GB VPS
- Upload-Payload ~100 KB statt mehrerer MB

**Alternative verworfen:** Go `imaging`-Lib serverseitig — braucht CGo-freies Fork (`disintegration/imaging` ist pure Go, aber ~2 MB binärer Overhead und kein User-Feedback vor dem Upload).

### 2. Keine neue npm-Abhängigkeit (kein react-image-crop)

**Entscheidung:** Eigene `ImageCropModal`-Komponente mit Canvas/Touch-Events.

**Begründung:**
- Anforderung ist definiert (Kreis, 1:1, Drag, Pinch, Slider) — kein unbekannter Scope
- `react-image-crop` würde ~20 KB hinzufügen, bietet aber Freeform-Crop den wir nicht wollen
- Canvas-Implementierung für 1:1 Crop ist überschaubar (~150 Zeilen)

**Alternative verworfen:** `react-image-crop` — gute Library, aber Overhead für einen klar abgegrenzten Use Case nicht gerechtfertigt.

### 3. Backend: MIME-Whitelist erweitern + Magic-Byte-Fallback

**Entscheidung:** `image/jpg` in die Allowlist aufnehmen; bei leerem/fehlendem Content-Type die ersten 12 Bytes des Streams prüfen (JPEG: `FF D8`, PNG: `89 50 4E 47`, WebP: `52 49 46 46...57 45 42 50`).

**Begründung:**
- Defense-in-depth: Canvas sendet immer `image/jpeg`, aber direkte API-Calls (Admin-Tools, zukünftige Clients) profitieren von robuster Erkennung
- Magic-Byte-Check ist ein Standard-Pattern, ~15 Zeilen Go, keine Abhängigkeit

### 4. Crop-Interaktion: Drag + Pinch-to-Zoom + Slider

**Entscheidung:** 
- Maus/Touch-Drag verschiebt das Bild unter dem Kreis-Ausschnitt
- Pinch-to-Zoom (zwei Finger) auf Touch-Geräten skaliert das Bild
- Slider (`<input type="range">`) als universelle Zoom-Alternative

**Technische Umsetzung:**
- `touchstart`/`touchmove` mit `e.touches.length === 2` für Pinch-Erkennung (Abstandsdelta zwischen zwei Touchpoints)
- `mousedown`/`mousemove` für Desktop-Drag
- State: `{ offsetX, offsetY, scale }` — alle anderen Größen sind davon abgeleitet
- Kreis-Maske via `canvas.arc()` + `clip()` im Preview-Canvas

## Risks / Trade-offs

- **Canvas-Qualität auf Low-End-Android** → Canvas-Resize ist ausreichend für Profilbilder (600 px); kein merklicher Qualitätsverlust bei Köpfen/Portraits.
- **HEIC auf Android nicht decodierbar** → Betrifft nur Fotos die direkt von iPhone auf Android-Browser übertragen werden (theoretisch). Canvas.drawImage schlägt dann lautlos fehl → Fehler-Handling im `onload` des Image-Elements mit User-Feedback.
- **Pinch-Präzision auf kleinen Screens** → Slider als Fallback ist immer sichtbar; kein Showstopper.
- **Sehr kleine Ausgangsbilder** (< 600 px) → Scale-up via Canvas ist möglich aber pixelig. Mitigation: Mindest-Zoom auf 1.0 setzen, kein Hochskalieren erzwingen.

## Migration Plan

1. Backend-Fix deployen (MIME-Whitelist) — sofort wirksam, rückwärtskompatibel
2. Frontend: `ImageCropModal` einbauen, in allen drei Upload-Stellen verdrahten
3. Keine DB-Migration nötig (Dateinamen/Pfade bleiben unverändert)
4. Rollback: Frontend-Änderung revertieren; bestehende Bilder sind nicht betroffen
