## Context

TeamWERK bekommt regelmäßig Wartungsfenster — akut den Server-Umzug von `internal.*` (Alt-VPS IONOS) auf `teamwerk.*` (Neu-VPS), perspektivisch weitere Migrationen, DB-Reparaturen und Compliance-Freezes. Während dieser Fenster darf die App weiter erreichbar sein (Lesen bleibt möglich, User bleiben eingeloggt), aber **kein Nicht-Admin-Nutzer darf mutieren**, sonst driften Datenzustände auseinander oder das gerade laufende Reparatur-Skript hat einen Konflikt-Race.

Aktuell existiert kein Wartungsmodus. Die einzigen Wege wären:
- Alle Nicht-GET auf Nginx-Ebene per `if ($request_method ...)` mit 503 blocken — Nutzer sähen kryptische Server-Fehler ohne Kontext, Admin könnte nichts mehr fixen.
- Systemd-Service stoppen — komplette Downtime, nicht zumutbar für ein 3-Stunden-Migrations-Fenster.
- Ad-hoc-Bannerskram nur im Frontend, ohne Server-Enforcement — reine Vertrauens-Sperre, verhindert nichts.

Der bestehende `TransitionalHostnameBanner` (aus Change `internal-hostname-transitional-alias`) ist ein Client-only-Hinweis, ausgelöst durch `window.location.host`, ohne Server-State und ohne Sperre. Er ist die visuelle Vorlage, aber funktional kein Vorbild — der Wartungsmodus braucht Server-Enforcement.

## Goals / Non-Goals

**Goals:**
- **Server-Enforcement**: kein Bypass durch direktes Curl, kein Bypass durch alte Client-Version.
- **Admin-Handlungsfähigkeit**: der Admin bleibt zu jeder Zeit voll schreibfähig. Er darf sich nicht selbst aussperren.
- **UX-Klarheit**: alle Nutzer sehen sofort, dass Wartungsmodus aktiv ist. Fehler bei Schreib-Versuch werden freundlich erklärt, nicht als generisches Netzwerk-Problem.
- **Runtime-Toggle**: Ein/Aus ohne Service-Restart, ohne Deploy.
- **Zwei unabhängige Toggle-Wege**: UI (bequem) und CLI (defensiv, falls UI kaputt ist).
- **Migration-Sync-Kompatibilität**: der `on`-Zustand reist bei `server-sync-data` mit auf den Neu-VPS. Das ist ein bewusstes Feature, kein Bug.
- **Dauer-Capability**: nach der aktuellen Migration bleibt das Feature dauerhaft nutzbar.

**Non-Goals:**
- **Kein feingranulares Read-Only** pro Modul (nicht: "Kader ist read-only, Chat läuft") — global oder gar nicht.
- **Kein Scheduling** (nicht: "Wartungsmodus jeden ersten Sonntag von 03:00–05:00") — manueller Toggle reicht.
- **Kein Queueing** abgewiesener Requests — verlorene Mutations-Versuche müssen vom Nutzer wiederholt werden. 503 mit klarem Signal ist genug.
- **Kein per-Form-Disable im Frontend** — jeder Button zu deaktivieren wäre über alle Formulare hinweg mehr Wartungsoberfläche als das Feature selbst wert ist. Banner + freundlicher 503-Dialog liefern denselben UX-Wert mit deutlich weniger Code.
- **Keine anderen `system_settings`-Keys** in dieser Change — die Tabelle wird generisch angelegt, damit spätere Toggles möglich sind, aber nur `maintenance_mode` wird jetzt eingeführt.
- **Kein Rate-Limiting oder Circuit-Breaker** um wiederholte 503er — kein Bedarf.

## Decisions

### Entscheidung 1: State in DB (`system_settings`), nicht Env-Var oder File-Marker

Alternativen:
- **Env-Var `MAINTENANCE_MODE=1`**: braucht `systemctl restart` zum Toggeln → sichtbare Downtime, widerspricht dem Ziel. Verworfen.
- **File-Marker `/var/lib/teamwerk/.maintenance`**: instant toggelbar, keine DB-Änderung. Aber: braucht SSH+sudo zum Setzen, keine UI ohne Extra-Aufwand, kein `updated_by`/`updated_at`-Audit, kein natürlicher Sync-Pfad zum Neu-VPS. Verworfen.
- **DB-Setting**: Toggelt in-place, natürlich im UI erreichbar, wandert mit `server-sync-data` (Feature), erlaubt Audit-Spalten. **Gewählt.**

Der Store cached den Wert in-memory (`atomic.Bool`), damit die Middleware auf dem Hot-Path keinen DB-Roundtrip pro Request macht. Cache-Invalidierung erfolgt beim eigenen `POST` (Handler ruft `store.Reload()` bzw. schreibt direkt). Für den CLI-Fallback ist das nicht kritisch — der Toggle-Effekt wird beim nächsten Request der Middleware wirksam, weil auch der CLI die DB updatet und das nächste Handler-Startup den Cache neu lädt … aber Vorsicht: **im laufenden Server-Prozess sieht der Cache CLI-Änderungen nicht sofort**. Auflösung: der CLI läuft nicht gegen den laufenden Server, sondern gegen die DB direkt — nach CLI-Aufruf muss der Admin den Server neu starten **oder** wir bauen einen kurzen Poll (alle 10 s) in den Store ein, der bei jeder Nutzung „lazy" nachschaut. Präferenz: **Poll mit Backoff (10 s Intervall, atomar)**, weil das ohne Restart auskommt und Kosten irrelevant sind (eine SELECT alle 10 s).

### Entscheidung 2: Middleware-Position — nach CORS/Recover, **vor** Auth

Warum vor Auth? Damit sie auch für unauthentifizierte Requests greift (die kämen sonst nicht bis zur Sperre durch — Auth würde 401 vor Maintenance werfen). Der Preis: die Middleware muss selbst JWT-Claims decoden, um zu entscheiden „ist der Requester Admin?".

Alternative: Middleware **nach** Auth. Vorteil: `claims.Role` schon parsed vorhanden. Nachteil: unauthentifizierte Mutations-Versuche (z. B. `POST /api/auth/register`) müssten getrennt behandelt werden — dann kann die Middleware sie nicht sperren, obwohl sie es sollte, wenn keine Registrierung während Wartung stattfinden darf. Aktuell gibt's eh keine öffentlichen Mutations-Endpoints außer `/api/auth/*` (Login/Refresh/Logout — die sollen ja durch), aber das kann sich ändern.

**Gewählt: Middleware vor Auth.** Sie parsed den JWT selbst mit `auth.ParseClaims(r, jwtSecret)` (existiert bereits als Helper) und toleriert Parse-Fehler (dann eben kein Admin, sperren). Vermeidet, dass später eine öffentliche Mutations-Route unbemerkt die Sperre umgeht.

Reihenfolge in `router.go`:
```
r.Use(health.InFlightMiddleware)   // bestehend
r.Use(middleware.Recover)          // bestehend
r.Use(middleware.CleanPath)        // bestehend
r.Use(corsMiddleware(...))         // bestehend
r.Use(MaintenanceMiddleware)       // NEU
// ... später: auth.Middleware(...) pro Auth-Tier
```

### Entscheidung 3: Auth-Whitelist per Path-Prefix, nicht per Route-Enum

Die Middleware macht:
```
if r.Method in {GET, HEAD, OPTIONS} → next
if strings.HasPrefix(r.URL.Path, "/api/auth/") → next
if claims.Role == "admin" → next
→ 503
```

Alternative: eine explizite Liste erlaubter Pfade. Präziser, aber pflegeintensiv — jede neue Auth-Route braucht einen Eintrag. Der Prefix-Ansatz ist robust gegen künftige Auth-Routen wie `/api/auth/password-reset` oder `/api/auth/register-invitation`, weil die konsequent unter `/api/auth/` leben.

**Video-Stream-Endpoints** (`/api/videos/{id}/stream/*` mit `?st=` Token) sind alle GETs → automatisch durch. **Preflight** (`OPTIONS`) ist explizit erlaubt.

**Health-Check** `/api/healthz` ist GET → durch.

### Entscheidung 4: `X-Maintenance-Mode: 1` Response-Header + JSON-Body

Der Frontend-Interceptor muss den Maintenance-503 vom generischen 503 (Backend down, Load-Balancer-Fehler, Upstream-Timeout) unterscheiden. Header ist eindeutig, robust gegen Body-Parsing-Fehler und leicht per `curl -I` diagnostizierbar.

Body zusätzlich mit `{"error":"maintenance_mode", "message":"…"}` für Konsumenten, die den Body ohnehin parsen (bestehendes Fehler-Handling).

### Entscheidung 5: Public `/api/maintenance-status` — kein Auth

Der Banner soll auch auf der Login-Seite sichtbar sein, wo noch keine Session existiert. Der Status-Endpoint muss also public sein. Er verrät nur `{enabled: bool}` — keine Metadaten (`updated_by`, `updated_at`), das wäre ein kleiner Info-Leak. Diese Metadaten sind nur im Admin-Endpoint (`GET /api/admin/maintenance-mode`, admin-only) sichtbar.

### Entscheidung 6: Zwei Broadcast-Pfade — UI-Toggle über SSE, CLI über Poll

Der HTTP-Toggle-Handler ruft `h.hub.Broadcast("settings-changed")`. Frontend-Clients aktualisieren binnen SSE-Latency (praktisch sofort).

Der CLI-Toggle hat keinen Zugriff auf `hub` (läuft in eigenem Prozess). Frontend-Clients bekommen den Wechsel erst beim nächsten Poll mit — deshalb der 10-s-Poll im Store und die Empfehlung, den CLI nur als Notfall-Werkzeug zu nutzen. Alternativ könnte der CLI ein internes Signal an den laufenden Prozess senden (SIGUSR1), aber das erhöht Komplexität für einen Nischen-Pfad. Verworfen.

### Entscheidung 7: Frontend-UX — Banner + 503-Dialog, kein Button-Disable

Vergleich:

|                    | Banner + 503-Dialog | Buttons pro Form disablen |
|--------------------|---------------------|---------------------------|
| Server-Enforcement | vorhanden (Middleware) | vorhanden (Middleware) |
| Nutzer-Kommunikation | sofort sichtbar | sofort sichtbar |
| Fehler bei Klick | freundlicher Dialog | Button gar nicht erst klickbar |
| Code-Oberfläche | 2 Dateien + Interceptor | jede Form-Komponente |
| Wartung | zentral | verstreut |
| Risiko Regression | niedrig | mittel (jede neue Form kann's vergessen) |

Der Banner-Weg ist die minimal-invasive Lösung, die dieselbe Enforcement-Garantie liefert. Kein per-Form-Disable.

### Entscheidung 8: `system_settings`-Schema generisch (Key-Value), aber nur ein Key jetzt

Ich könnte nur eine `maintenance_mode`-Boolean-Spalte irgendwo anhängen (z. B. `clubs`-Tabelle). Aber:
- Konzeptuell ist Wartungsmodus keine Vereins-Eigenschaft (der Verein existiert weiter, der Server macht Pause).
- Zukünftige Toggles (Feature-Flags für schrittweise Rollouts, Notfall-Deaktivierungen einzelner Module) würden erneut eine Schema-Migration erzwingen.

Generisches Key-Value ist der übliche Weg für „System-weite Konfiguration, die zur Laufzeit toggelbar sein soll". Kosten: 4 Spalten, 1 Row initial. Nutzen: künftige Erweiterbarkeit ohne Schema-Änderung.

**Grenze**: keine JSON-Werte, keine Typ-System-Ambitionen — String-Values, Parsen im Store. Wenn's mehr wird, refactorn wir. Aber wahrscheinlich reicht das lange.

## Risks / Trade-offs

**Cache-Konsistenz zwischen CLI und laufendem Server** → 10-s-Poll im Store toleriert eine Latenz von bis zu 10 s zwischen CLI-Toggle und Middleware-Wirkung. Für die Notfall-Deaktivierung ist das akzeptabel; das dokumentierte Vorgehen ist „UI benutzen, CLI nur wenn UI hängt".

**Sync-Verhalten auf `server-sync-data`: on-Zustand wandert mit** → im Migrations-Runbook explizit dokumentieren, dass der Admin nach dem finalen Sync **bewusst** auf dem Neu-VPS deaktivieren muss. Für den ersten Nutzer, der es vergisst, ist der Fehlermodus offensichtlich („warum kann keiner schreiben?"), Fix ist ein Klick.

**Middleware parsed JWT selbst** → Duplikation zum bestehenden `auth.Middleware`. Falls JWT-Parsing sich mal ändert (Key-Rotation, andere Signatur), muss die Maintenance-Middleware mitgezogen werden. Mitigation: gemeinsamer Helper `auth.ParseClaimsFromRequest(r, secret)` (der wahrscheinlich schon existiert oder trivial zu extrahieren ist) wird genutzt, kein eigener JWT-Parse-Code.

**Race zwischen POST /api/admin/maintenance-mode und in-flight Mutations** → wenn Admin gerade `on` schaltet, sind Mutations, die schon in Handler-Bodies sind, unterwegs. Sie werden nicht gestoppt. Für ein Wartungsfenster mit vorheriger Ankündigung irrelevant.

**Admin kann selbst nicht sperren** → Feature, nicht Bug. Ein Admin, der aus Versehen den Modus einschaltet, kann ihn ohne Umwege wieder ausschalten.

**Non-Admin-Vorstand kann in aktiven Wartungsfenstern nichts tun** → gewollt. Wenn Vorstand doch schreiben können soll, ist das ein Sonderfall (z. B. „Kassierer soll SEPA-Lauf während des Fensters ausführen"), der über CLI-Deaktivierung oder eine kurze Deaktivierungs-Klammer im UI adressiert wird.

**Frontend-Interceptor unterdrückt 503 → verstärkt „silent failure"?** → Nein: der Interceptor zeigt einen sichtbaren Dialog. Die Verantwortung des Callers ist nur, das Promise nicht als Erfolg zu behandeln (Standard-Axios-Verhalten: Promise rejectet bei 5xx, außer man setzt `validateStatus`). Der Interceptor soll den Fehler **kontextualisieren**, nicht schlucken.

## Migration Plan

**Roll-out (dieselbe Instanz):**
1. Migration `system_settings.up.sql` deployen (idempotent, `INSERT OR IGNORE` für Default-Row).
2. Backend-Deploy mit neuer Middleware + Handlern.
3. Frontend-Deploy mit Banner + Interceptor + Admin-UI.
4. Erst-Verifikation: `GET /api/maintenance-status` → `{"enabled":false}`, App verhält sich unverändert.
5. Verifikation Toggle: als Admin einloggen, `/admin/wartung` öffnen, ein-, wieder ausschalten, dabei mit einem zweiten (nicht-Admin) Browser die Sichtbarkeit des Banners und das 503-Verhalten prüfen.

**Roll-out für den `internal.*` → `teamwerk.*`-Cutover:**
Nachdem dieser Change gemergt+deployt ist, ist er als Werkzeug für den bestehenden Change `internal-hostname-transitional-alias` (Phasen 4.x) verfügbar. Das Runbook `deploy/internal-alias-cutover-runbook.md` wird in einem separaten Follow-up-Commit um „vor Phase B: Wartungsmodus aktivieren" ergänzt — nicht Teil dieser Change.

**Rollback:**
- Kein Datenmigrations-Rollback nötig: `system_settings` ist eine additive Tabelle, kein bestehendes Schema wird verändert.
- Fehler in Middleware → `POST /api/admin/maintenance-mode {enabled:false}` per curl von Admin, oder `teamwerk maintenance off` per SSH, oder DB-Direktupdate.
- Fehler in Frontend-Banner-Rendering → App bleibt nutzbar, Banner-Regression ist kosmetisch.
- Wenn die Middleware selbst kaputt ist (z. B. panisch): `middleware.Recover` fängt Panics, Response wird 500 statt 503 — Fallback ist der Betreiber, der die alte Binary rollt.

## Open Questions

- **Admin-Nav-Position der Wartungs-Seite**: eigenständiger Nav-Punkt oder unter „Einstellungen"? — kann bei Implementierung entschieden werden, kein Blocker.
- **Banner-Text**: „Wartungsarbeiten laufen — Speichern ist gerade deaktiviert." — der genaue Wortlaut kann im Task 2.x abgestimmt werden.
- **Cache-Poll-Intervall**: 10 s ist plausibel, bis 60 s wäre auch ok. Bei der Implementierung feinjustierbar.
- **CLI-Broadcast-Trick**: alternativ könnte der CLI `curl -X POST http://localhost:8080/internal/reload-settings` gegen einen loopback-only Endpoint schießen. Nicht in Scope, aber offen für später.
