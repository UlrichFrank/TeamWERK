# deploy-version-detection Specification

## Purpose

Diese Spezifikation beschreibt die Capability `deploy-version-detection`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: SSE-Versionscheck erkennt neuen Deployment-Stand

Das Frontend SHALL beim Verbindungsaufbau und bei jedem SSE-Reconnect die empfangene Versionsinformation mit der zuerst bekannten Version vergleichen. Bei Abweichung SHALL der Update-Banner angezeigt werden. Der Hook SHALL neben `updateAvailable` auch die aktuell bekannte `version` (erster empfangener Hash, oder `null` vor dem ersten Event, oder `'dev'` im Dev-Modus) zurückgeben. Die SSE-Verbindung SHALL nach Verbindungsabbruch automatisch reconnecten; der Hook SHALL `EventSource.close()` NICHT im `onerror`-Handler aufrufen. Der Hook SHALL beim Wechsel des eingeloggten Nutzers (Login, Logout, Impersonation) die SSE-Verbindung schließen und neu aufbauen. Die SSE-URL SHALL **keinen** `?token=`-Query-Parameter mehr enthalten — die Authentifizierung läuft ausschließlich über das HttpOnly-`refresh_token`-Cookie via `CookieMiddleware`. Der Hook liefert KEIN `updateDescription` mehr — der `changes.json`-Fetch entfällt.

#### Scenario: Erste Verbindung speichert Baseline-Version

- **WHEN** der SSE-Client zum ersten Mal verbindet und ein `__version:`-Event empfängt
- **THEN** wird dieser Hash als bekannte Version gespeichert
- **THEN** gibt der Hook `{ updateAvailable: false, version: "<hash>" }` zurück

#### Scenario: SSE-Reconnect nach Server-Neustart zeigt Banner

- **WHEN** die SSE-Verbindung nach einem Server-Neustart (deploy) neu aufgebaut wird
- **WHEN** der neue Server einen anderen Hash als die gespeicherte Version sendet
- **THEN** gibt der Hook `{ updateAvailable: true, version: "<alter-hash>" }` zurück
- **THEN** wird der Update-Banner angezeigt

#### Scenario: Reconnect ohne Versionsänderung zeigt keinen Banner

- **WHEN** die SSE-Verbindung kurzzeitig unterbrochen und wieder hergestellt wird
- **WHEN** der Server denselben Hash sendet wie die gespeicherte Version
- **THEN** gibt der Hook `{ updateAvailable: false, version: "<hash>" }` zurück
- **THEN** bleibt der Update-Banner ausgeblendet

#### Scenario: Login öffnet, Logout schließt die SSE-Verbindung

- **WHEN** ein Nutzer sich einloggt und der `user`-State in `AuthContext` von `null` auf einen Wert wechselt
- **THEN** öffnet der Hook eine neue SSE-Verbindung zu `/api/events`
- **WHEN** der Nutzer sich ausloggt und `user` zurück auf `null` wechselt
- **THEN** schließt der Hook die SSE-Verbindung
- **THEN** setzt der Hook `version` auf `null` zurück

#### Scenario: Dev-Modus zeigt sichtbaren Platzhalter

- **WHEN** die App im Dev-Modus läuft (`import.meta.env.DEV === true`)
- **THEN** öffnet der Hook keine SSE-Verbindung
- **THEN** gibt der Hook `{ updateAvailable: false, version: "dev" }` zurück
- **THEN** wird der Update-Banner unabhängig von Versionsänderungen nicht angezeigt
- **THEN** ist der Versions-Link in der Sidebar sichtbar und zeigt `v dev`


### Requirement: CHANGELOG.md wird bei Build aus git log generiert

`make build` SHALL `web/public/CHANGELOG.md` aus der vollständigen `git log`-Historie erzeugen. Nur `feat`- und `fix`-Conventional-Commits werden einbezogen. Einträge werden nach Commit-Datum gruppiert (neuestes Datum zuerst). `changes.json` entfällt.

#### Scenario: CHANGELOG.md enthält alle feat/fix-Commits
- **WHEN** `make build` ausgeführt wird
- **THEN** enthält `web/public/CHANGELOG.md` alle `feat`- und `fix`-Commits aus der git-Historie, gruppiert nach Datum im Format `## DD.MM.YYYY`
- **THEN** jeder Eintrag hat die Form `- [feat] scope: Beschreibung` oder `- [fix] scope: Beschreibung`

#### Scenario: changes.json wird nicht mehr generiert
- **WHEN** `make build` ausgeführt wird
- **THEN** wird `web/public/changes.json` NICHT erzeugt

### Requirement: VersionContext zentralisiert die Versionsabfrage

Es SHALL einen `VersionProvider` geben, der die App-weit einzige SSE-Versionsabfrage hält. UI-Komponenten SHALL ausschließlich über den Hook `useVersion()` auf `version` und `updateAvailable` zugreifen. Der direkte Aufruf von `useVersionCheck()` aus mehreren Komponenten parallel SHALL NICHT mehr stattfinden — die parallelen SSE-Verbindungen, die daraus heute resultieren, entfallen.

#### Scenario: Eine SSE-Verbindung pro Tab für Versionsabfrage

- **WHEN** die App in einem Tab mit eingeloggtem Nutzer läuft
- **THEN** existiert genau eine SSE-Verbindung zu `/api/events`, die ausschließlich vom `VersionProvider` gehalten wird
- **THEN** liefert `useVersion()` in allen Konsumenten denselben State

#### Scenario: Konsumenten erhalten Updates synchron

- **WHEN** die SSE eine neue Version meldet
- **THEN** sehen `AppUpdateBanner` und der Versions-Link in der `AppShell`-Sidebar denselben Wert von `version` und `updateAvailable` im selben React-Render-Zyklus

### Requirement: Banner-Dismiss ist versionsbezogen

Wenn der Update-Banner per „Schließen"/Dismiss-Button geschlossen wird, SHALL die aktuell angezeigte Version als „dismissed" gespeichert werden. Der Banner SHALL erneut sichtbar werden, sobald eine andere Version (Hash) als die zuletzt dismissed-Version erkannt wird.

#### Scenario: Dismiss verbirgt den Banner nur für die aktuelle Version

- **WHEN** der Banner für Version A sichtbar ist und der Nutzer „Schließen" klickt
- **THEN** verschwindet der Banner
- **WHEN** anschließend Version B erkannt wird (anderer Hash)
- **THEN** wird der Banner wieder angezeigt

#### Scenario: Wiederholtes Dismiss derselben Version bleibt dismissed

- **WHEN** der Banner für Version A dismissed wurde
- **WHEN** anschließend erneut Version A gemeldet wird (Reconnect ohne Versionswechsel)
- **THEN** bleibt der Banner verborgen
