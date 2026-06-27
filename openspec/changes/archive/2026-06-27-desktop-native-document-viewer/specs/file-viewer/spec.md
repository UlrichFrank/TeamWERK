# file-viewer Specification

## MODIFIED Requirements

### Requirement: Dokument-Viewer-Route

Das System SHALL eine Route `/dokumente/anzeigen/:fileId` bereitstellen, die
über `<FileViewerPage>` die Datei-Metadaten lädt und `<FileViewer source="file">`
rendert. `fallbackPath` ist `/dokumente`.

Die Aufrufer (`DocumentsPage.openFile` und `DocumentFileLinkPage`) SHALL eine
**Viewport-basierte Weiche** anwenden:
- Bei kleinem Viewport (`window.matchMedia('(max-width: 639px)').matches`) →
  navigieren zur Viewer-Route (In-App-Render).
- Bei größerem Viewport → die Datei im **nativen Browser-Viewer eines neuen
  Tabs** öffnen (Desktop nutzt die Browser-Chrome zur Navigation; die
  PWA-Standalone-Sackgasse existiert auf Desktop nicht relevant).

`DocumentsPage.openFile` SHALL für den Desktop-Pfad den Popup-Blocker-Workaround
verwenden: `window.open('about:blank', '_blank')` synchron im Click-Handler,
danach Token-Fetch und `tab.location.href = downloadUrl`. Bei Fehler im
Token-Fetch SHALL der Tab geschlossen und ein Fehler-State gesetzt werden.

`DocumentFileLinkPage` SHALL für den Desktop-Pfad `window.location.replace`
auf die Download-URL ausführen (der native Viewer rendert im selben Tab).

#### Scenario: Datei aus Dokumenten-Liste öffnen (Mobile)

- **WHEN** der Nutzer in `/dokumente` auf eine Datei klickt
- **AND** `window.matchMedia('(max-width: 639px)').matches` ist `true`
- **THEN** navigiert die App zu `/dokumente/anzeigen/${fileId}` (In-App-Viewer)
- **THEN** wird **kein** `window.open` aufgerufen

#### Scenario: Datei aus Dokumenten-Liste öffnen (Desktop)

- **WHEN** der Nutzer in `/dokumente` auf eine Datei klickt
- **AND** `window.matchMedia('(max-width: 639px)').matches` ist `false`
- **THEN** ruft die App **synchron** `window.open('about:blank', '_blank')` auf
- **THEN** holt sie anschließend einen Download-Token
- **THEN** setzt sie `tab.location.href = '/api/files/${fileId}/download?token=…'`
- **THEN** wird **nicht** zur In-App-Viewer-Route navigiert

#### Scenario: Datei aus Dokumenten-Liste öffnen (Desktop, Token-Fehler)

- **WHEN** der Token-Fetch beim Desktop-Pfad einen Fehler liefert
- **THEN** ruft die App `tab.close()` auf dem zuvor geöffneten Blank-Tab auf
- **THEN** setzt sie eine Fehler-Meldung („Datei konnte nicht geöffnet werden.")

#### Scenario: Geteilter Datei-Link öffnen (Mobile)

- **WHEN** der Nutzer einen Link `/dokumente/datei/${fileId}` auf einem Gerät
  mit kleinem Viewport öffnet
- **THEN** rendert `DocumentFileLinkPage` ein `<Navigate to="/dokumente/anzeigen/${fileId}" replace />`

#### Scenario: Geteilter Datei-Link öffnen (Desktop)

- **WHEN** der Nutzer einen Link `/dokumente/datei/${fileId}` auf einem Gerät
  mit großem Viewport öffnet
- **THEN** holt `DocumentFileLinkPage` einen Token
- **THEN** führt sie `window.location.replace('/api/files/${fileId}/download?token=…')` aus
- **THEN** rendert der Browser das Dokument im nativen Viewer

#### Scenario: Geteilter Datei-Link öffnen (Desktop, 403)

- **WHEN** der Token-Fetch im Desktop-Pfad HTTP 403 liefert
- **THEN** rendert `DocumentFileLinkPage` eine Fehler-UI „Kein Zugriff auf diese Datei."
- **THEN** bietet sie einen Link zurück zu `/dokumente`
