## Why

Das Architektur-Review vom 11.09.2026 (`docs/reviews/2026-09-11-architektur-review.md`) hat
sieben Sicherheitsbefunde der Stufen kritisch und hoch ergeben. Alle liegen in einer Klasse:
Objekt-Level-Autorisierung und der Upload-Pfad, also dort, wo die vorhandenen Gates
(Permission-Matrix, Broadcast-, Push-, Audience-Gate) per Konstruktion nicht hinsehen. Der
kritische Befund hebelt die Zero-Knowledge-Zusage für Bankdaten vollständig aus: ein
beliebiges Konto kann über den Foto-Upload HTML und JavaScript unter dem App-Origin
ausliefern lassen und bei entsperrtem Tresor den exportierbaren Vereins-Privatschlüssel aus
`sessionStorage` lesen. Die Fixes sind klein bis mittel; der Schaden im Eintrittsfall ist
irreversibel (kein Recovery, keine Rotation der verschlüsselten Bestandsdaten).

## What Changes

1. **Upload-Härtung** (`internal/upload`): Byte-Sniffing immer, Dateiendung ausschließlich aus
   dem erkannten Typ (Muster `media.extByMime`), Auslieferung mit `Content-Disposition:
   attachment` für alles außer Bildern und expliziter, aus der DB gelesener `Content-Type`.
2. **Tresor-Schlüssel nicht exportierbar** (`web/src/contexts/VaultContext.tsx`): der
   entschlüsselte Privatschlüssel liegt als `CryptoKey` mit `extractable=false` in IndexedDB
   statt als PKCS8-Base64 in `sessionStorage`; Timeout-Semantik (30 min) bleibt.
3. **`GET /api/media/{id}` gaten**: nur Mitglieder der Konversation bzw. Empfänger der
   Mitteilung (`broadcast_reads`) und der Hochladende. Permission-Matrix korrigiert.
4. **Sechs Objekt-Gates nachziehen**: `GET /api/training-sessions/{id}` (Kader-Zugriff),
   `POST /api/games/{id}/lineup` (Trainer eines beteiligten Teams statt vereinsweit),
   `POST /api/chat/conversations/{id}/read` und `DELETE .../members/me` (aktive
   Mitgliedschaft), `POST /api/duty-assignments/{id}/fulfill` und `/cash-substitute`
   (bestehendes Recht admin/trainer/sportliche_leitung jetzt auch im Handler, Existenzprüfung,
   Fehler geprüft).
5. **HSTS aktivieren, `JWT_SECRET` mit Mindestlänge**: `HSTS_ENABLED=true` im Deploy-Env
   (idempotent nachgezogen wie die Storage-Pfade), Server verweigert Start bei Secret unter
   32 Byte oder gleich dem Beispielwert; `.env.example` ohne funktionsfähiges Secret.
6. **Impersonation auditieren**: JWT-Claim `impersonated_by`, `slog.Warn` mit Akteur und Ziel,
   Eintrag im Event-Log des Admins (neue Kategorie `admin`, Migration `060`).
7. **Abhängigkeiten**: offene Dependabot-Alerts (7 high, 10 moderate, 1 low, alle npm)
   schließen; `govulncheck` und `pnpm audit --audit-level=high` als CI-Job.

**BREAKING (Betrieb):** Punkt 5 lässt den Server nicht mehr starten, wenn `JWT_SECRET` zu kurz
ist. Vor dem Deploy prüfen: `grep JWT_SECRET /etc/teamwerk/env`. Punkt 2 sperrt beim ersten
Laden einen bereits entsperrten Tresor (Format-Wechsel); einmaliges Neu-Entsperren.

## Capabilities

### New Capabilities

_(keine)_

### Modified Capabilities

- `profilbild-crop-upload`: Backend akzeptiert Fotos nur nach Byte-Sniffing; Endung folgt
  dem erkannten Typ; Auslieferung nie als HTML.
- `media-storage`: Bild abrufen setzt Sichtbarkeit des referenzierenden Objekts voraus.
- `permissions`: Objekt-Gates für die sechs Routen; Upload-Auslieferung ohne
  Content-Type-Vertrauen; Impersonation ist auditiert.
- `client-side-bank-encryption`: sitzungsgebundenes Schlüssel-Caching ohne exportierbares
  Schlüsselmaterial.
- `security-headers`: HSTS ist im Produktivbetrieb aktiv.
- `admin-impersonation`: Impersonation hinterlässt Audit-Spur und ist im Token erkennbar.
- `auth`: `JWT_SECRET` hat eine Mindestlänge.

## Impact

- Backend: `internal/upload`, `internal/media`, `internal/trainings`, `internal/games`,
  `internal/chat`, `internal/duties`, `internal/auth` (Claims + Impersonate),
  `internal/config`, `internal/eventlog`, Migration `060`, `internal/permissions/matrix_test.go`.
- Frontend: `VaultContext.tsx` (+ Tests), `lib/crypto.ts` (Import-Flag).
- Betrieb: `deploy/nginx-teamwerk.conf`, `Makefile` (`deploy`-Env-Schleife), `.env.example`,
  `.github/workflows/ci.yml`, `web/package.json` + Lockfile.
- Keine API-Form-Änderung; betroffene Routen antworten bei fehlender Berechtigung 403/404
  statt 200.
