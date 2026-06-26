## MODIFIED Requirements

### Requirement: Public Endpoints sind ohne Auth zugänglich

Die Routen `POST /api/auth/login`, `POST /api/auth/refresh`, `POST /api/auth/logout`, `POST /api/auth/request-membership`, `POST /api/auth/register`, `GET /api/auth/token-info`, `POST /api/auth/forgot-password`, `POST /api/auth/reset-password`, `GET /api/profile/email/confirm`, `GET /api/files/{id}/download`, `GET /api/members/{id}/sepa-mandat/download` SHALL ohne Bearer-Token erreichbar sein und dürfen NICHT mit 401 antworten, nur weil kein Token vorliegt. `GET /api/uploads/*` ist NICHT mehr Teil dieser Liste und SHALL nicht ohne Authentifizierung ausgeliefert werden (siehe Anforderung „Upload-Auslieferung erfordert Authentifizierung").

#### Scenario: Login ohne Token
- **WHEN** ein Aufruf an `POST /api/auth/login` mit gültigem Body und ohne `Authorization`-Header gemacht wird
- **THEN** antwortet der Server NICHT mit 401 (200/400 je nach Body-Validität ist erlaubt)

#### Scenario: Tokenloser Upload-Zugriff wird abgelehnt
- **WHEN** ein Aufruf an `GET /api/uploads/<datei>` ohne gültiges Download-Token gemacht wird
- **THEN** antwortet der Server mit 401 oder 403 und liefert die Datei NICHT aus

## ADDED Requirements

### Requirement: Upload-Auslieferung erfordert Authentifizierung

Das System SHALL Dateien unter `/api/uploads/*` nur an authentifizierte Aufrufer ausliefern. Da `<img>`-Requests keinen Bearer-Header senden, erfolgt die Authentifizierung über das HttpOnly-Refresh-Cookie (`auth.CookieMiddleware`, analog zu den SSE-Routen). Ohne gültiges Cookie SHALL der Server mit HTTP 401 antworten und die Datei NICHT ausliefern. Die Auslieferung SHALL `Referrer-Policy: no-referrer` und `Cache-Control: private, no-store` setzen, damit die (UUID-)URL nicht über Referrer oder Caches weiterleakt. Die UUID-Dateinamen bleiben als Defense-in-Depth erhalten.

> Bewusste Grenze: Es gibt keinen Pro-Foto-Sichtbarkeitscheck — jeder authentifizierte Nutzer kann ein Foto über seine (nicht erratbare) UUID-URL laden. Der behobene Befund war der *unauthentifizierte* Zugriff; per-Foto-Granularität wäre angesichts der bereits breiten Foto-Sichtbarkeit in Mitgliederlisten unverhältnismäßig.

#### Scenario: Authentifizierter Aufrufer erhält die Datei
- **WHEN** ein Aufrufer mit gültigem Refresh-Cookie `GET /api/uploads/<datei>` aufruft
- **THEN** wird die Datei mit 200 sowie `Referrer-Policy: no-referrer` und `Cache-Control: private, no-store` ausgeliefert

#### Scenario: Unauthentifizierter Zugriff wird abgelehnt
- **WHEN** `GET /api/uploads/<datei>` ohne gültiges Refresh-Cookie aufgerufen wird
- **THEN** antwortet der Server mit HTTP 401 und liefert die Datei NICHT aus
