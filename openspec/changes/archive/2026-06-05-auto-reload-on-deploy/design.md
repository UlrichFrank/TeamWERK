## Context

Nach `make deploy` werden Go-Binary und Vite-Bundle gemeinsam deployed (rsync + systemctl restart). Der Server-Neustart unterbricht alle offenen SSE-Verbindungen; der Browser reconnectet automatisch. Dieser Reconnect-Moment ist bisher ungenutzt.

Das System hat zwei bereits vorhandene Erkennung-Mechanismen, die aber nicht genutzt werden:
1. **SSE-Reconnect** — zuverlässiges Signal dass der Server neu gestartet hat
2. **Service Worker `autoUpdate`** — erkennt neue Assets, ruft aber ohne expliziten `onNeedRefresh`-Handler niemanden auf

## Goals / Non-Goals

**Goals:**
- Nutzer sehen nach `make deploy` innerhalb weniger Sekunden einen Banner
- Funktioniert im Browser-Tab und in der installierten PWA
- Kein Polling, kein neuer externer Dienst
- Reload nur auf Nutzerinteraktion — kein erzwungener Reload mitten in einer Aktion

**Non-Goals:**
- Erkennung von Hot-Reloads im Dev-Modus
- Unterscheidung zwischen Breaking und Non-Breaking Changes
- Automatischer Reload ohne Nutzerbestätigung

## Decisions

### Entscheidung 1: SSE-Init-Event statt separatem `/api/version`-Endpoint

**Gewählt:** Der SSE-Handler sendet beim Verbindungsaufbau sofort `data: __version:<hash>\n\n`, bevor reguläre Mutations-Events kommen.

**Alternativen:**
- *Separater `GET /api/version` Endpoint mit Polling (z.B. 60s)*: Polling ist unnötig wenn SSE schon offen ist. Außerdem Latenz bis zu 60s.
- *HTTP Response Header `X-App-Version`*: Würde alle API-Antworten erweitern, funktioniert aber nicht wenn die Seite nur idle ist ohne API-Calls.

**Begründung:** Der SSE-Reconnect nach Server-Neustart ist ohnehin unvermeidlich. Das Init-Event nutzt diesen Moment ohne zusätzliche Roundtrips. Der `__version:`-Prefix ist eindeutig und kollidiert nicht mit bestehenden Event-Typen.

### Entscheidung 2: Git Short-SHA als Build-Hash

**Gewählt:** `git rev-parse --short HEAD` eingebettet via `-ldflags "-X main.buildHash=..."` im Makefile.

**Alternativen:**
- *Timestamp (`date -u +%Y%m%dT%H%M%S`)*: Ändert sich auch ohne Codeänderungen, z.B. bei reinem `make deploy` ohne neue Commits.
- *Datei-Hash des Binaries*: Erst nach dem Build verfügbar, zirkuläre Abhängigkeit.

**Begründung:** Der Git-SHA ist stabil (ändert sich nur bei neuen Commits), menschenlesbar, und bereits in jedem Dev-Workflow verfügbar. Wenn kein Git vorhanden (z.B. CI): fallback `"dev"`.

### Entscheidung 3: Service Worker als zweite Erkennungslinie

**Gewählt:** `useRegisterSW({ onNeedRefresh })` aus `virtual:pwa-register/react` zeigt denselben Banner wenn der SW einen neuen Bundle erkennt.

**Begründung:** Fängt den seltenen Fall ab, dass der Server neu startet aber die SSE-Verbindung noch offen bleibt (z.B. Load Balancer mit Sticky Sessions). Außerdem: transparenter macht was `autoUpdate` eigentlich tut.

### Entscheidung 4: Opt-in Banner, kein erzwungener Reload

**Gewählt:** Fixer Banner am unteren Rand mit „Jetzt neu laden"-Button. Der Nutzer entscheidet.

**Begründung:** Ein Auto-Reload kann Formularinhalte oder laufende Aktionen zerstören. Da TeamWERK eine interne App mit bekannten Nutzern ist, ist ein Banner ohne Countdown ausreichend.

## Risks / Trade-offs

**`__version`-Events in Dev mit `go run`** → Der Build-Hash ist `"dev"` wenn nicht via ldflags gesetzt. In Dev reconnectet SSE oft (z.B. bei `air`-Reload). Der Hook prüft: nur wenn die erste empfangene Version sich von einer späteren unterscheidet, zeigt er den Banner. In Dev bedeutet das: Banner erscheint nach jedem Hot-Reload. Mitigation: `if (import.meta.env.DEV) return` im Hook.

**SSE-Token läuft ab während Tab offen** → Nach 15 min wird das Access Token ungültig. Die bestehende `useLiveUpdates`-Implementierung reconnectet dann nicht mehr (kein Token-Refresh im EventSource). Der `useVersionCheck`-Hook erbt dieses Verhalten. Mitigation: außerhalb des Scope dieser Change — das ist ein bekanntes Problem im bestehenden `useLiveUpdates`.

**PWA standalone: SW-Update-Timing** → In der installierten PWA prüft der SW auf Updates nur beim App-Start. Wer die App den ganzen Tag offen hat, bekommt den SW-basierten Banner erst nach erneutem Start. Der SSE-basierte Banner greift aber sofort. Beide Linien zusammen decken den Fall ab.

## Migration Plan

1. Makefile-Änderung ist rückwärtskompatibel — `var buildHash = "dev"` als Fallback
2. SSE-Init-Event: Bestehende Clients ignorieren `__version:`-Events (sie parsen nur bekannte Event-Typen in `useLiveUpdates`)
3. Frontend-Code ist additiv: neuer Hook + neuer Component, keine bestehenden Komponenten modifiziert außer `App.tsx`
4. Kein Rollback-Aufwand: alle Änderungen sind isoliert und entfernbar
