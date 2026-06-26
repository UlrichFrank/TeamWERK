// Öffnet ein Blob im nativen Datei-Viewer des Geräts (iOS Quick-Look /
// Android-Download → System-App). Reproduziert exakt das „Download"-Verhalten
// des In-App-Viewers, damit Mobilgeräte den qualitativ hochwertigen nativen
// Viewer (zoomen/scrollen/schließen) direkt nutzen, statt den schwachen
// In-App-Canvas-Render zu durchlaufen.
export function openBlobNatively(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  // Object-URL erst verzögert freigeben — der native Viewer hält die Datei
  // sonst evtl. noch offen, wenn wir sofort revoken.
  setTimeout(() => URL.revokeObjectURL(url), 60_000)
}
