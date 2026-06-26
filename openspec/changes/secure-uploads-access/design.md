## Context

`/api/uploads/*` liefert vor allem Mitglieder-/Nutzerfotos, die im Frontend über `<img src="/api/uploads/...">` eingebettet werden. `<img>`-Requests senden **keinen** `Authorization`-Header — ein simples Verschieben hinter die `auth.Middleware` (Bearer) würde die Bilddarstellung brechen. Das Projekt hat dieses Problem für SEPA-Mandate bereits gelöst: ein kurzlebiges, signiertes Download-Token (`SepaDownloadToken` → `SepaDownload`, Capability `file-download-token`), das als Query-Parameter an die URL gehängt wird.

## Goals / Non-Goals

**Goals:**
- Kein tokenloser, unauthentifizierter Abruf von Fotos mehr.
- Bildanzeige im Frontend bleibt funktionsfähig (kein Bearer im `<img>`).
- Berechtigungsprüfung (wer darf das Foto sehen) beim Token-Ausstellen.

**Non-Goals:**
- Keine Verschlüsselung der Fotos at rest (anderes Schutzniveau als Bankdaten; nicht Teil dieses Befunds).
- Kein Umbau des SEPA-Pfads (bleibt wie ist; dient als Vorlage).

## Decisions

**D1 — Cookie-Auth statt Download-Token (Revision).** Ursprünglich war ein HMAC-Download-Token pro Foto vorgesehen (wie SEPA). Beim Umsetzen zeigte sich, dass die Codebase dasselbe Problem für SSE bereits idiomatisch löst: `<img>`/`EventSource` senden kein Bearer-Token, deshalb laufen diese Routen unter `auth.CookieMiddleware` (HttpOnly-Refresh-Cookie). `/api/uploads/*` wird in dieselbe Cookie-Auth-Group verschoben. Vorteil: **null Frontend-Änderungen** (same-origin `<img src>` sendet das Cookie automatisch), schließt den eigentlichen Befund (unauthentifizierter Zugriff). Die Token-Variante (großer Frontend-Umbau über alle `photoURL`-Consumer) wurde als unverhältnismäßig für einen Low-Befund verworfen.

**D2 — Bewusst kein Pro-Foto-Sichtbarkeitscheck.** Cookie-Auth prüft nur „eingeloggt", nicht „darf genau dieses Foto sehen". Akzeptiert, weil Fotos in Mitgliederlisten ohnehin breit an berechtigte Nutzer ausgespielt werden und der behobene Befund der *unauthentifizierte* Leak war (Referrer/Logs/Cache). UUIDv4-Namen bleiben als Defense-in-Depth.

**D3 — Defense-in-Depth bleibt.** UUIDv4-Dateinamen, `..`-Abwehr und `filepath.Join`-Re-Rooting bleiben; zusätzlich `Referrer-Policy: no-referrer` und `Cache-Control: private, no-store` auf der Auslieferung.

**D4 — Doc-Kommentar korrigieren.** Der irreführende `// Auth required`-Kommentar wird an die jetzt tatsächlich geschützte Mountierung angeglichen.

## Risks / Trade-offs

- **[Kein Pro-Foto-Ownership]** → akzeptierte Grenze (D2); jeder Eingeloggte kann ein Foto per nicht-erratbarer UUID-URL laden.
- **[Cache-Verlust bei Fotos]** → `private, no-store` verhindert Shared-Cache; akzeptabel, Fotos sind klein.
- **[SameSite=Strict + Cookie-Pfad]** → das Refresh-Cookie liegt unter Path `/` mit SameSite=Strict; same-origin `<img>`-GETs senden es. Bestätigt durch das identische SSE-Muster.

## Migration Plan

Ein-Schritt-Deploy (nur Backend; kein Frontend-Change). Breaking nur für bisher unauthentifizierte Direktzugriffe — beabsichtigt. Rollback durch Zurücknehmen des Commits.
