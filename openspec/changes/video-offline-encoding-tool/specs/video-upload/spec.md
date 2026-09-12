## REMOVED Requirements

### Requirement: Client-seitige Progress-Throttle
**Reason**: Die Anforderung bezog sich ausschließlich auf `web/src/pages/VideoUploadPage.tsx`, die mit diesem Change entfällt — Video-Uploads laufen ab sofort ausschließlich über das Offline-Encoding-Tool, nicht mehr über den Browser.
**Migration**: Die äquivalente Verantwortung (Fortschritts-/Restzeit-Anzeige ohne UI-Blockade) liegt jetzt im Tool, siehe `video-offline-encoding-tool`, Requirement „Einfache Bedienung ohne Zusatzfunktionen". Da das Tool keine React-Rendering-Schleife hat, gilt die ursprüngliche 1-Hz-State-Update-Motivation dort nicht 1:1 — das Tool SHALL dennoch keine UI-Blockade durch zu häufige Fortschritts-Events verursachen.
