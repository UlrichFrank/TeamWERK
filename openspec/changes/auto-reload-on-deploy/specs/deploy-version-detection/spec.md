## ADDED Requirements

### Requirement: Build-Hash wird zur Compile-Zeit eingebettet

Das Build-System SHALL den Git Short-SHA (`git rev-parse --short HEAD`) zur Compile-Zeit via `-ldflags` in die Go-Binary einbetten. Als Fallback-Wert SHALL `"dev"` verwendet werden, wenn kein Git-Kontext vorhanden ist.

#### Scenario: Produktions-Build enthält Git-SHA

- **WHEN** `make build` oder `make deploy` ausgeführt wird
- **THEN** ist der Build-Hash der aktuelle Git Short-SHA (z.B. `"a3f1b2c"`)

#### Scenario: Dev-Modus verwendet Fallback-Hash

- **WHEN** `go run ./cmd/teamwerk` ohne ldflags gestartet wird
- **THEN** ist der Build-Hash `"dev"`

### Requirement: SSE-Versionscheck erkennt neuen Deployment-Stand

Das Frontend SHALL beim Verbindungsaufbau und bei jedem SSE-Reconnect die empfangene Versionsinformation mit der zuerst bekannten Version vergleichen. Bei Abweichung SHALL der Update-Banner angezeigt werden.

#### Scenario: Erste Verbindung speichert Baseline-Version

- **WHEN** der SSE-Client zum ersten Mal verbindet und ein `__version:`-Event empfängt
- **THEN** wird dieser Hash als bekannte Version gespeichert
- **THEN** wird kein Banner angezeigt

#### Scenario: SSE-Reconnect nach Server-Neustart zeigt Banner

- **WHEN** die SSE-Verbindung nach einem Server-Neustart (deploy) neu aufgebaut wird
- **WHEN** der neue Server einen anderen Hash als die gespeicherte Version sendet
- **THEN** wird der Update-Banner angezeigt

#### Scenario: Reconnect ohne Versionsänderung zeigt keinen Banner

- **WHEN** die SSE-Verbindung kurzzeitig unterbrochen und wieder hergestellt wird
- **WHEN** der Server denselben Hash sendet wie die gespeicherte Version
- **THEN** bleibt der Update-Banner ausgeblendet

#### Scenario: Dev-Modus unterdrückt den Banner

- **WHEN** die App im Dev-Modus läuft (`import.meta.env.DEV === true`)
- **THEN** wird der Update-Banner unabhängig von Versionsänderungen nicht angezeigt

### Requirement: Update-Banner zeigt nicht-aufdringliche Reload-Aufforderung

Das System SHALL einen Update-Banner am unteren Bildschirmrand anzeigen, sobald eine neue Version erkannt wird. Der Banner SHALL über einen Button zum sofortigen Reload und einen Dismiss-Button verfügen.

#### Scenario: Banner erscheint bei erkannter Versionsänderung

- **WHEN** `useVersionCheck` eine Versionsabweichung feststellt
- **THEN** erscheint am unteren Rand ein Banner mit dem Text „Neue Version verfügbar"
- **THEN** enthält der Banner einen „Jetzt neu laden"-Button
- **THEN** enthält der Banner einen Schließen-Button (✕)

#### Scenario: Klick auf Reload-Button lädt die Seite neu

- **WHEN** der Nutzer auf „Jetzt neu laden" klickt
- **THEN** wird `window.location.reload()` ausgeführt

#### Scenario: Dismiss-Button schließt den Banner

- **WHEN** der Nutzer auf ✕ klickt
- **THEN** verschwindet der Banner
- **THEN** wird die Seite NICHT neu geladen

#### Scenario: Banner ist auf Mobile sichtbar und bedienbar

- **WHEN** der Nutzer die App auf einem Mobilgerät nutzt (< 640px)
- **THEN** ist der Banner vollständig sichtbar und die Buttons haben mindestens 44px Touch-Target
