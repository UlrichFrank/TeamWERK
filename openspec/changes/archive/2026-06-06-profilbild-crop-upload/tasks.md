## 1. Backend: Robuste MIME-Type-Erkennung

- [x] 1.1 `image/jpg` zur Allowlist in `imageTypes` hinzufügen (`internal/upload/handler.go`)
- [x] 1.2 Hilfsfunktion `sniffImageType(r io.Reader) string` schreiben: liest erste 12 Bytes, erkennt JPEG (`FF D8`), PNG (`89 50 4E 47`), WebP (`52 49 46 46 … 57 45 42 50`)
- [x] 1.3 `saveFile` anpassen: wenn `contentType` leer oder unbekannt → `sniffImageType` aufrufen; bereits gelesene Bytes via `io.MultiReader` an den Stream zurückführen

## 2. Frontend: ImageCropModal-Komponente

- [x] 2.1 Datei `web/src/components/ImageCropModal.tsx` anlegen; Props: `file: File | null`, `onConfirm: (blob: Blob) => void`, `onCancel: () => void`
- [x] 2.2 State-Modell implementieren: `{ offsetX, offsetY, scale }` + Initialwerte beim Bildladen (Bild zentriert, scale = fit-to-circle)
- [x] 2.3 Preview-Canvas mit kreisförmiger Maske rendern (`ctx.arc` + `clip`); Bild mit aktuellem offset/scale zeichnen
- [x] 2.4 Drag-Interaktion implementieren: `mousedown`/`mousemove`/`mouseup` für Desktop
- [x] 2.5 Touch-Drag implementieren: `touchstart`/`touchmove`/`touchend` (1 Finger = Drag)
- [x] 2.6 Pinch-to-Zoom implementieren: `touchstart`/`touchmove` mit `e.touches.length === 2`; Abstandsdelta berechnen und `scale` anpassen
- [x] 2.7 Zoom-Slider (`<input type="range" min=1 max=3 step=0.01>`) mit `scale`-State verdrahten
- [x] 2.8 „Hochladen"-Button: Export-Canvas 600 × 600 px erstellen, Zuschnitt zeichnen, `canvas.toBlob('image/jpeg', 0.85)` → `onConfirm(blob)` aufrufen
- [x] 2.9 Fehlerbehandlung: `img.onerror` → Modal schließen + Fehlermeldung „Bild konnte nicht geladen werden"
- [x] 2.10 Styling: Modal-Overlay, Kreis-Rahmen sichtbar machen, Slider + Buttons nach Brand-Vorgaben

## 3. Integration: Crop-Modal einbinden

- [x] 3.1 `ProfileProfilTab.tsx`: `handlePhotoUpload` umbauen → erst Modal öffnen, `onConfirm(blob)` führt den bestehenden FormData-Upload durch
- [x] 3.2 `ChildProfilePage.tsx` verwendet `ProfileProfilTab` → durch 3.1 abgedeckt
- [x] 3.3 `MemberStammdatenTab.tsx` (Admin-Upload): ebenso Modal vorschalten

## 4. Testen & Verifizieren

- [ ] 4.1 Android Chrome: Foto aus Galerie wählen → Modal öffnet sich → Upload erfolgreich (kein 400-Fehler mehr)
- [ ] 4.2 iOS Safari: Foto aufnehmen oder aus Galerie → Modal + Upload funktioniert
- [x] 4.3 Desktop (Chrome/Firefox): Drag, Slider, Upload prüfen — verifiziert
- [x] 4.4 Abbrechen-Flow: Datei-Input nach Abbrechen zurückgesetzt, kein Upload — verifiziert
- [ ] 4.5 Ungültige Datei (z. B. PDF umbenennen zu .jpg): Fehlermeldung erscheint
- [ ] 4.6 Sehr kleines Bild (< 600 px): kein ungewolltes Hochskalieren, Upload funktioniert
