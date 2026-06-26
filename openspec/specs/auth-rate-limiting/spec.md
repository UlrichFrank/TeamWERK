# auth-rate-limiting Specification

## Purpose
TBD - created by archiving change auth-rate-limiting. Update Purpose after archive.
## Requirements
### Requirement: IP-basiertes Rate-Limiting der Auth-Endpunkte

Das System SHALL die unauthentifizierten Auth-Routen `POST /api/auth/login`, `POST /api/auth/refresh`, `POST /api/auth/forgot-password` und `POST /api/auth/reset-password` pro Client-IP drosseln. Überschreitet eine IP das konfigurierte Limit innerhalb des Zeitfensters, SHALL der Server mit HTTP 429 und einem `Retry-After`-Header antworten, OHNE die teure Verarbeitung (bcrypt-Hashing, Mailversand) auszuführen. Das Limit SHALL über die Konfiguration einstellbar und in der Testumgebung deaktivierbar sein.

#### Scenario: Zu viele Login-Versuche von einer IP
- **WHEN** dieselbe IP `POST /api/auth/login` häufiger als das konfigurierte Limit innerhalb des Fensters aufruft
- **THEN** antwortet der Server für weitere Versuche mit HTTP 429 und setzt `Retry-After`

#### Scenario: Innerhalb des Limits keine Drosselung
- **WHEN** eine IP `POST /api/auth/login` seltener als das Limit aufruft
- **THEN** antwortet der Server normal (200/400/401 je nach Body und Credentials), nicht mit 429

#### Scenario: 429 vermeidet teure Verarbeitung
- **WHEN** eine gedrosselte Anfrage an `POST /api/auth/forgot-password` eingeht
- **THEN** wird weder eine E-Mail versendet noch ein bcrypt-Hash berechnet

---

### Requirement: Account-basierter Login-Lockout

Das System SHALL pro Benutzerkonto aufeinanderfolgende fehlgeschlagene Login-Versuche zählen (`users.failed_login_count`) und das Konto nach einer konfigurierten Schwelle für ein Zeitfenster sperren (`users.locked_until`). Während der Sperre SHALL `POST /api/auth/login` für dieses Konto mit HTTP 429 (oder 403) antworten, OHNE bcrypt auszuführen. Ein erfolgreicher Login SHALL den Zähler zurücksetzen und eine etwaige Sperre aufheben. Das Antwortverhalten SHALL keine Information preisgeben, ob die E-Mail existiert (generische Meldung, kein Enumerationsvorteil).

#### Scenario: Konto wird nach zu vielen Fehlversuchen gesperrt
- **WHEN** für ein existierendes Konto mehr als die konfigurierte Anzahl falscher Passwörter in Folge gesendet werden
- **THEN** ist das Konto bis `locked_until` gesperrt und weitere Login-Versuche antworten ohne bcrypt-Prüfung mit einer Drosselungs-/Sperrmeldung

#### Scenario: Erfolgreicher Login setzt den Zähler zurück
- **WHEN** ein Konto mit korrekten Credentials einloggt, nachdem zuvor (unterhalb der Schwelle) Fehlversuche registriert wurden
- **THEN** wird `failed_login_count` auf 0 gesetzt und keine Sperre gesetzt

#### Scenario: Sperre verrät keine Konto-Existenz
- **WHEN** Login-Versuche gegen eine nicht existierende E-Mail das Limit überschreiten
- **THEN** ist die Antwort nicht von der für ein existierendes, gesperrtes Konto unterscheidbar

