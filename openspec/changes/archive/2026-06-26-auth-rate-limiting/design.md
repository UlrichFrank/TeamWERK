## Context

Single-Node-Deployment (ein 1-GB-VPS, ein `teamwerk`-Prozess, nginx davor). Es gibt keinen geteilten State-Store (Redis o.ä.) und soll auch keiner eingeführt werden (RAM-Druck, Betriebsaufwand). Rate-Limiting muss daher mit In-Process-State auskommen; das ist für ein Single-Node-Setup ausreichend.

## Goals / Non-Goals

**Goals:**
- Mail-Bombing über `forgot-password` und bcrypt-CPU-DoS abschneiden.
- Online-Bruteforce einzelner Konten begrenzen.
- Keine neuen Laufzeitabhängigkeiten (Redis o.ä.), keine Flakiness in bestehenden Auth-Tests.

**Non-Goals:**
- Verteiltes Rate-Limiting über mehrere Nodes.
- CAPTCHA / Proof-of-Work.
- Schutz gegen ein botnet-skaliges, IP-rotierendes DDoS (gehört auf die Netz-/Reverse-Proxy-Ebene).

## Decisions

**D1 — In-Process-Limiter via `go-chi/httprate`.** Als Chi-Middleware ausschließlich auf der Public-Auth-Routengruppe, Schlüssel = Client-IP (über `RealIP`/`X-Forwarded-For` hinter nginx, korrekt konfiguriert). Alternative `golang.org/x/time/rate` mit eigener IP-Map verworfen: mehr Eigencode, `httprate` liefert IP-Keying, Fenster und 429 fertig. Alternative „nur nginx `limit_req`" verworfen als alleinige Lösung: greift nicht für account-basierten Lockout und koppelt Security an die Deploy-Config; nginx-Limit bleibt aber als optionale zweite Schicht empfohlen.

**D2 — Account-Lockout in der DB, nicht im Speicher.** `failed_login_count` + `locked_until` auf `users`, weil der Zustand einen Prozess-Neustart überleben muss und an das Konto (nicht die IP) gebunden ist. Exponentielles Backoff (z.B. Schwelle 5 → Sperre wächst je weiterer Fehlversuchsserie).

**D3 — Reihenfolge: erst Limiter/Sperre, dann bcrypt/Mail.** Die teure Operation darf nie vor der Drosselungsentscheidung laufen, sonst bleibt der CPU-/Mail-DoS-Vektor offen.

**D4 — Generische Antworten erhalten.** Lockout-/Drosselungsantworten dürfen die bestehende Anti-Enumeration (generische `invalid credentials`, konstant-zeitiger Dummy-Hash) nicht aushebeln; gesperrtes existierendes Konto und gedrosselte nicht-existente E-Mail sind ununterscheidbar.

**D5 — Konfigurierbar + Test-Override.** Limits aus `internal/config` (`.env`); in `testutil`-Servern hoch/aus, damit Persona- und Happy-Path-Tests deterministisch bleiben.

## Risks / Trade-offs

- **[Falsch erkannte Client-IP hinter Proxy → ganze Nutzergruppe hinter einem NAT gedrosselt]** → `RealIP`-Middleware korrekt an nginx `X-Forwarded-For` koppeln; Limit nicht zu aggressiv (≥5/min).
- **[Lockout als DoS gegen ein bekanntes Konto]** → Sperre zeitlich begrenzt (`locked_until`), nicht permanent; IP-Limit fängt den Massenfall ab; erfolgreicher Login hebt sofort auf.
- **[Prozess-Neustart leert IP-Limiter]** → akzeptiert (kurzes Fenster); der persistente Teil (Account-Lockout) liegt bewusst in der DB.

## Migration Plan

Migration `010_user_login_throttle` ergänzt zwei Spalten mit Defaults (kein Backfill nötig). Deploy in einem Schritt; `make migrate-remote-up` vor Binary-Restart. Rollback: `.down.sql` entfernt die Spalten, Middleware-Commit zurücknehmen. Limits zunächst großzügig setzen und nach Beobachtung nachziehen.

## Open Questions

- Konkrete Default-Werte (Versuche/Fenster, Lockout-Schwelle/-Dauer) — Startwerte im Design vorgeschlagen, finale Kalibrierung beim Apply.
- nginx `limit_req` als zweite Schicht jetzt mit ausrollen oder separat? (Empfehlung: jetzt mitnehmen, da `deploy/nginx-intern.conf` ohnehin angefasst wird.)
