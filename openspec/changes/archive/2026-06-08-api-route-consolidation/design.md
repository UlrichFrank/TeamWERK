## Context

Die API hat historisch einen `/admin`-Präfix für privilegierte Routen bekommen. Dieser Präfix ist inkonsistent angewandt: Einige privilegierte Routen liegen unter `/api/admin/...`, andere ohne diesen Prefix (z.B. `POST /api/members`). Schlimmer: Der Präfix vermittelt Sicherheit, die nicht existiert — z.B. ist `GET /api/admin/membership-requests` für Trainer zugänglich, nicht nur für Admins.

Betroffen im Frontend: 11 Seiten in `web/src/pages/` sowie `web/src/pages/KalenderPage.tsx` und `MemberDetailPage.tsx`.

## Goals / Non-Goals

**Goals:**
- Alle Ressource-Pfade folgen einer einheitlichen Konvention: `/api/{ressource}`
- Berechtigungsprüfung ausschließlich in Middleware-Gruppen in `main.go`, nie implizit über Pfadstruktur
- Hard-Cut: Frontend und Backend gleichzeitig deployen, keine Übergangsphase

**Non-Goals:**
- Keine Änderungen an Middleware-Gruppen oder Berechtigungslogik
- Kein API-Versioning einführen
- Keine Redirects von alten zu neuen Pfaden

## Decisions

### 1. Teams-Endpoint zusammenführen

**Problem:** `GET /api/admin/teams` (alle Teams, für Vorstand) und `GET /api/teams` (Kader-gefilterter View, für alle) kollidieren nach Umbenennung.

**Entscheidung:** Den bestehenden `ListTeamsForUser`-Handler in `internal/games/handler.go` erweitern: Wenn der aufrufende User `HasFunction("vorstand")` oder `IsAdmin()`, wird die ungefilterte Query aus dem bisherigen `ListTeams` ausgeführt. Andernfalls bleibt das bisherige Kader-gefilterte Verhalten.

**Alternative verworfen:** Zwei getrennte Pfade (`/api/teams` und `/api/teams/all`) — widerspricht dem Ziel einer sauberen Ressource-Adressierung.

**Alternative verworfen:** Query-Parameter `?all=true` als Berechtigungsschalter — ungewöhnliches Muster, Berechtigung sollte nicht von Query-Params abhängen.

### 2. Hard-Cut statt Redirects

**Entscheidung:** Kein `http.Redirect` von alten auf neue Pfade. Da Frontend und Backend als eine Einheit deployt werden (`make deploy` baut beides und setzt neu), gibt es keine Phase in der ein Client die alten Pfade noch erwartet.

**Begründung:** Redirects für interne API-Pfade wären totes Gewicht — kein externer Konsument nutzt diese API. Würden nur für Verwirrung sorgen.

### 3. Route-Registrierung bleibt in main.go

**Entscheidung:** Alle Route-Strings werden direkt in `main.go` geändert. Kein Routing-Layer abstrahieren.

**Begründung:** Das bestehende Pattern ist bewusst flach gehalten. Keine Abstraktion für diesen einmaligen Refaktor einführen.

## Risks / Trade-offs

- **Vergessene Frontend-Calls** → Mitigation: Grep nach `/admin/` in `web/src/` nach dem Umbenennen als Verification-Schritt
- **CLAUDE.md veraltet** → Mitigation: CLAUDE.md-Update ist expliziter Task
- **Deploy-Timing**: Wenn Backend ohne neues Frontend deployt wird (oder umgekehrt), gibt es 404s → Mitigation: `make deploy` baut immer beides atomisch

## Migration Plan

1. Backend: Alle Route-Strings in `main.go` umbenennen
2. Backend: `ListTeamsForUser` in `games/handler.go` um Vorstand-Check erweitern
3. Frontend: Alle `/admin/`-Calls in 11 betroffenen Pages ersetzen
4. Verification: `grep -r "'/admin/" web/src/` muss leer sein
5. Deploy: `make deploy` (baut Frontend + Backend + migrate up + systemctl restart)
6. CLAUDE.md aktualisieren
