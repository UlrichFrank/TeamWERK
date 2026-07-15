## Context

Ausgangslage: TeamWERK-Produktion soll (bzw. hat gerade begonnen) auf `teamwerk.team-stuttgart.org` (VPS `31.70.110.19`) laufen. Vorbereitet ist ein „harter" Cutover-Pfad (`deploy/nginx-redirect.conf` + `make server-cutover` aus dem archivierten `2026-07-03-server-migration`), der den Alt-VPS auf ein permanentes 301 → `teamwerk.*` umschaltet und laufen lässt. Diese Change ersetzt diesen Pfad **nicht**, sondern ergänzt ihn um eine sanftere Variante, die zusätzlich den Alt-Host entbehrlich macht.

Zone `team-stuttgart.org` wird bei Mittwald gehostet. Die App ist eine PWA mit Service-Worker-Scope am jeweiligen Origin, HttpOnly-Refresh-Cookies am jeweiligen Origin und Web-Push-Subscriptions ebenfalls origin-gebunden.

## Goals / Non-Goals

**Goals:**
- `internal.*` bleibt erreichbar, ohne dass ein separater VPS für Redirect am Leben bleibt.
- Bestandsnutzer werden **nicht** synchron ausgeloggt; sie migrieren selbstständig über UI-Hinweis auf ihrem gewohnten Rhythmus.
- Genau **ein** produktiver Host (`31.70.110.19`) servt beide Namen.
- Nach der Übergangsphase kann `internal.*` per Follow-up-Change endgültig auf 301 flippen — Trigger ist der Betreuer, nicht Code.

**Non-Goals:**
- Automatisierte Erkennung, wann „genug" Nutzer auf `teamwerk.*` sind. Bleibt manuelle Ops-Entscheidung.
- Cross-Origin-Login-Migration (Session-Transfer `internal.*` → `teamwerk.*`). Der Nutzer loggt sich einmal auf `teamwerk.*` neu ein.
- Multi-Origin-CORS-Whitelist. Same-origin-Flow deckt beide Hostnames.
- Ausschalten des Alt-VPS. Ist Ops-Handarbeit nach DNS-TTL-Ablauf; nicht Code-Änderung.

## Decisions

### Entscheidung 1: Ein `server`-Block mit zwei `server_name`s, nicht zwei Blöcke

```
server {
    listen 443 ssl http2;
    server_name teamwerk.team-stuttgart.org internal.team-stuttgart.org;
    ssl_certificate     /etc/letsencrypt/live/teamwerk.team-stuttgart.org/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/teamwerk.team-stuttgart.org/privkey.pem;
    ...  # unverändert gegenüber heute
}
```

**Warum:** die App-Semantik ist auf beiden Namen identisch — location-Blöcke, Rate-Limits, CSP, Body-Size-Overrides gelten 1:1. Ein Split würde beide Blöcke stets parallel patchen müssen; Drift-Gefahr. Zertifikat wird mit zwei SANs beantragt (`certbot --nginx -d teamwerk.team-stuttgart.org -d internal.team-stuttgart.org` — Renewal deckt beide).

**Nachteil:** späterer Flip auf 301 (Follow-up) muss den Block aufsplitten. Akzeptabler Preis für die Übergangsphase.

### Entscheidung 2: CORS bleibt strikt auf `BaseURL` (teamwerk.*)

`corsMiddleware` in `internal/app/router.go:524` setzt `Access-Control-Allow-Origin: <BaseURL>` (Single-Value). Für Same-Origin-Requests (Browser auf `internal.*` fetcht `/api/*` am selben Host) wertet der Browser CORS **nicht** aus — der ACAO-Header wird gesendet, aber ignoriert. Cross-Origin (Browser auf `internal.*` fetcht `https://teamwerk.*/api/…`) ist kein Anwendungsfall der SPA.

**Warum keine Multi-Origin-Whitelist:** würde bedeuten, das Origin-Header zu parsen und dynamisch zu echoen. Nicht schwer, aber ist eine echte Änderung am Auth-Perimeter und erfordert eigene Tests. Für den Übergang unnötig; wird bei Flip-auf-301 sowieso obsolet.

### Entscheidung 3: `cfg.BaseURL` zeigt auf `teamwerk.*`, nie auf `internal.*`

Konfig-Default in `config.go` wird auf `https://teamwerk.team-stuttgart.org` gesetzt. Auf dem VPS zusätzlich explizit `BASE_URL=https://teamwerk.team-stuttgart.org` in `/etc/teamwerk/env`. Folge: **alle Mail-Direktlinks** (`internal/notify/notify.go:83`, Password-Reset, Einladungen, Duty-Reminder aus dem Scheduler) zeigen ab sofort auf `teamwerk.*`. Auch wenn der Nutzer die Mail auf einem Gerät liest, das die PWA noch von `internal.*` kennt, landet der Klick auf `teamwerk.*` → dort einmal Login → dann läuft alles.

Der jüngst gefundene Hardcode `scheduler.go:840` (`"Jetzt eintragen: https://internal.team-stuttgart.org/duty-board\n\n"`) ist der einzige Umgehungspfad und wird auf `cfg.BaseURL` refactored. Konfig-Default-Wechsel allein reicht nicht, weil der Scheduler den String hartkodiert baut.

### Entscheidung 4: Banner ist nicht dismissable

Der Banner erscheint auf jeder Seitenladung, solange die Origin `internal.*` ist. Kein Dismiss-State, kein localStorage. Begründung: der einzige Weg, den Banner loszuwerden, ist tatsächlich auf `teamwerk.*` zu wechseln — genau das Zielverhalten. Dismiss-Cookie würde User trainieren, die Meldung wegzuklicken und dann in Support-Tickets zu rennen, wenn Origin-gebundene Funktionen (Neue Push-Sub nach Reinstall) bei `internal.*` hängen.

Der Banner ersetzt **nicht** die App — App bleibt voll funktional auf `internal.*`. Er ist ein persistenter Top-Bar-Hinweis mit CTA-Button. Design: `bg-brand-info/10 border-b border-brand-info/30 text-brand-text` (bestehende Alert-Info-Tokens) über dem `AppShell`-Header, `Sticky`-Positionierung, mobile-tauglich einzeilig scrollend.

### Entscheidung 5: Cutover-Trigger ist offen und wird ad-hoc entschieden

Dass `internal.*` **irgendwann** von "servt App" auf "servt nur 301" flippt, ist gewollt — **wann**, ist bewusst nicht festgelegt. Weder als Datum noch als Metrikschwelle. Diese Change schließt den Dual-Serving-Zustand als tragbaren Dauerbetrieb ein: er kostet nichts (ein SAN im Cert, ein zweiter `server_name`, eine Banner-Komponente) und muss nicht schnell weg.

Als lose Orientierung, wann der Follow-up sinnvoll wird — nicht als Zusage:
- Zugriffe auf `internal.*` im Nginx-Access-Log sind marginal geworden (Faustregel: einstelliger Prozentanteil).
- Keine wiederkehrenden Support-Rückfragen mehr zum Thema „Bin ich noch richtig?".
- Betreuerentscheidung, dass die zusätzliche `server_name`-Zeile jetzt eher Verwirrung als Nutzen stiftet.

Kein Prometheus-Counter, kein automatischer Flip. Der Follow-up-Change ist mechanisch klein (nginx-Block-Split + eine Zeile Doku + Banner löschen), also kostet Aufschub nichts — und die Konsequenz eines verfrühten Flips (koordinierter Ankündigungs-Cutover, User-Kommunikation) wäre unnötig teurer als abwarten.

### Entscheidung 6: Push-Subs an `internal.*` bleiben liegen — kein Cleanup-Sonderweg

Bestehende Rows in `push_subscriptions` sind an den `internal.*`-Service-Worker gebunden. Sie **funktionieren weiter**: Push wird an Apple/Google-Endpoint zugestellt, der Nutzer bekommt die Notification, Klick öffnet `internal.*` → Banner sichtbar → Nutzer wechselt selbst. Wenn er die PWA auf `teamwerk.*` neu installiert, entsteht eine **zweite** Sub-Row am neuen Origin — kurzzeitig Doppel-Push möglich, aber selten (Chat/Duty/Game-Reminder sind selten genug, dass eine Doppel-Benachrichtigung tolerierbar ist).

Alte Subs sterben irgendwann per HTTP-410 (Cleanup-Logik existiert bereits in `internal/notifications/`). Kein extra Housekeeping-Job.

## Alternatives Considered

- **1a: hartes 301 sofort.** Bereits vorbereitet (`nginx-redirect.conf`). Verworfen wegen synchronem Bulk-Logout und weil er den Alt-VPS am Leben halten würde. Falls die 4-Wochen-Übergangsphase zu lange dauert, kann jederzeit auf 1a manuell umgeflippt werden — die Vorarbeit bleibt intakt.
- **IONOS/Mittwald Domain-Forwarding.** Wäre der eleganteste Weg — kein Server für `internal.*` überhaupt — aber die Zone liegt bei Mittwald, und deren Weiterleitungsprodukt bräuchte separate HTTPS-Cert-Klärung. Zusätzlicher Vendor-Touchpoint für minimalen Gewinn, wenn ohnehin ein Nginx auf dem Ziel-Host läuft.
- **Cloudflare vor der Zone.** Sinnvoll für andere Ziele (CDN, DDoS), hier aber Overkill und würde die Vereins-Hauptseite `team-stuttgart.org` (Mittwald-Hosting) mit auf CF-Nameserver ziehen — Eingriff außerhalb TeamWERKs Verantwortungsbereich.
- **Multi-Origin-CORS-Refactor jetzt.** Nicht nötig für den Übergang; Same-Origin-Flow trägt.

## Risks

- **Certbot-Order für neuen SAN scheitert.** Wenn DNS für `internal.*` noch nicht auf die neue IP zeigt, schlägt der HTTP-01-Challenge fehl. Reihenfolge im Runbook: DNS zuerst, TTL abwarten, dann Certbot. Fallback: existierendes `teamwerk.*`-Only-Cert bleibt gültig, `internal.*`-Requests laufen bis dahin auf Cert-Warnung.
- **Banner rendert auf `localhost` in Dev.** Explizit im Component-Test verhindert (`host !== "internal.team-stuttgart.org"` → `null`). Prüfen bei Code-Review.
- **Alte Bookmarks / bookmarks-Deep-Links auf `internal.*` funktionieren nach 4-Wochen-Flip nur noch als 301.** Deep-Links sind Pfad-preserving (`$request_uri` im späteren 301). Akzeptabler Verlust: Refresh-Cookie bei diesen Klicks weg — Nutzer landet auf Login. Follow-up-Change kann das im Betreiber-Kommunikationstext ansagen.
- **Doppel-Push für kurze Zeit.** Siehe Entscheidung 6, akzeptiert.
- **`window.location.host` liefert auf iOS-Homescreen-PWA denselben Origin wie im Browser.** Kein Sonderfall — Banner erscheint in beiden Kontexten wie beabsichtigt.
