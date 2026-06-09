## Context

TeamWERK verwendet Server-Sent Events (SSE) für Live-Updates. Das Hub in `internal/hub/` broadcasted Events an alle verbundenen Clients. Frontend-Seiten subscriben via `useLiveUpdates()`. Die Infrastruktur existiert und funktioniert — sie ist nur unvollständig ausgerollt.

Aktuelle Lücken:
- `kader/handler.go` hat kein `hub`-Feld → keine Broadcasts bei Kader-Mutationen
- Mehrere Handler in `members/handler.go` und `games/handler.go` rufen `hub.Broadcast()` nicht auf
- 11 Frontend-Seiten haben `useLiveUpdates()` nicht eingebunden

## Goals / Non-Goals

**Goals:**
- Alle mutativen Backend-Handler broadcasten das passende Event
- Alle Frontend-Seiten, die Daten anzeigen, lauschen auf die relevanten Events
- Kader-Domäne wird vollständig in die SSE-Infrastruktur eingebunden (neues `"kader"`-Event)

**Non-Goals:**
- Keine Änderung an der SSE-Infrastruktur selbst (Hub, EventSource-Verbindungsmanagement)
- Keine Einführung von User-spezifischem Filtering (BroadcastToUser) für neue Events
- Keine Änderung an der Daten-Granularität (kein Payload in SSE-Events, nur Event-Name)

## Decisions

### 1. Neues `"kader"`-Event statt Wiederverwendung von `"members"`

Kader (Saisonzugehörigkeit, Trainer-Zuweisung, Alterskategorien) ist konzeptuell getrennt von Mitgliederstammdaten. Ein eigenes Event erlaubt gezieltes Reload ohne unnötige API-Calls auf Seiten, die nur Kader oder nur Members anzeigen.

**Alternative:** `"members"` wiederverwenden — abgelehnt, da MembersPage dann bei jedem Kader-Update die komplette Mitgliederliste neu lädt (expensive pagination query).

### 2. `kader/handler.go` bekommt `hub *hub.EventHub`-Feld

Pattern entspricht allen anderen Handlern. `NewHandler(db, hub)` in `main.go` anpassen.

**Alternative:** Hub als globale Variable — abgelehnt (widerspricht dem bestehenden Pattern).

### 3. Reload-Strategie im Frontend: silent reload (kein Spinner)

Bei SSE-Events wird `load(true)` bzw. `load()` mit `silent = true` aufgerufen, wo diese Signatur existiert. Kein sichtbarer Ladezustand beim Hintergrund-Refresh. Entspricht dem Muster, das bereits in anderen Seiten (z.B. DashboardPage, MitfahrgelegenheitenPage) verwendet wird.

### 4. Template-CRUD in `games/handler.go` broadcasted `"games"`

Duty-Templates sind eng mit Games verknüpft (Slot-Generierung). Das `"games"`-Event triggert in der `AdminDutyTemplatesPage` ein Reload. Kein neues `"templates"`-Event nötig — vereinfacht die Subscriber-Logik.

## Risks / Trade-offs

**[Flood-Risiko bei Massen-Operationen]** → CopyFromSeason, AutoAssign und BulkImport broadcasten ein einziges Event am Ende der Transaktion, nicht pro Zeile. Dieses Pattern wird bereits in `games/handler.go` (RegenerateSlots) so gehandhabt.

**[DashboardPage reagiert auf viele Events]** → Dashboard-Reload ist ein einzelner `/dashboard`-API-Call. Bei gleichzeitigen Änderungen (z.B. gleichzeitig `"games"` und `"trainings"`) könnten zwei Reloads kurz hintereinander feuern. Akzeptiertes Trade-off — kein Debouncing nötig bei realistischer Nutzerzahl (< 50 gleichzeitig).

**[AdminKaderPage ohne Paginierung]** → Kein Silent-Reload-Parameter, daher `loadKader()` direkt aufrufen. Kein sichtbarer Spinner-Flicker bei SSE-Update da die Seite keine globale Loading-State-Logik hat.
