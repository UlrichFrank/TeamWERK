## ADDED Requirements

### Requirement: e2e-seed CLI-Subcommand

Das `teamwerk`-Binary SHALL einen Subcommand `e2e-seed --db=<path>`
bereitstellen, der eine frische SQLite-DB unter `<path>` anlegt, alle
Migrations anwendet und einen deterministischen Test-Datensatz einträgt.
Nach erfolgreichem Lauf ist die DB von Playwright direkt nutzbar (kein
weiterer Setup-Schritt nötig).

Der Datensatz umfasst mindestens:
- 1 Admin-User: `e2e@test.local` mit Passwort `E2ETestPassword!` (bcrypt
  gehasht wie im Prod-Login-Flow)
- 3 Standard-Test-User
- 1 Gruppenkonversation „E2E Chat mit Bildern" mit ≥ 10 Text- und
  ≥ 3 Bild-Nachrichten, alle für den Admin als gelesen markiert
- 1 Gruppenkonversation „E2E Chat unread" mit ≥ 20 Text-Nachrichten,
  letzte 3 nicht für den Admin gelesen (unreadCount = 3)

Der Subcommand SHALL idempotent bei existierender Ziel-Datei sein: wenn
`<path>` existiert, wird die Datei ohne Rückfrage überschrieben.

#### Scenario: e2e-seed legt frische DB an

- **WHEN** `teamwerk e2e-seed --db=./tmp-e2e.db` in einem leeren
  Verzeichnis ausgeführt wird
- **THEN** existiert `./tmp-e2e.db` mit vollständig migriertem Schema
  und dem Test-Datensatz

#### Scenario: e2e-seed überschreibt bestehende Datei

- **GIVEN** eine Datei `./tmp-e2e.db` existiert bereits
- **WHEN** `teamwerk e2e-seed --db=./tmp-e2e.db` erneut ausgeführt wird
- **THEN** wird die Datei überschrieben (deterministisch derselbe Inhalt)
  und der Exit-Code ist 0

#### Scenario: Login gegen geseedete DB funktioniert

- **GIVEN** die DB wurde per `e2e-seed` erzeugt und das Backend läuft
  gegen sie
- **WHEN** ein Request `POST /api/auth/login` mit
  `e2e@test.local` / `E2ETestPassword!` gesendet wird
- **THEN** antwortet der Server mit HTTP 200 und einem gültigen JWT

### Requirement: Trennung Vitest vs. Playwright — Konvention

Die Projekt-Konvention in `docs/agent/07-testing.md` SHALL beschreiben,
welche Test-Ebene für welche Klasse von Bugs zuständig ist:

- **Vitest + jsdom**: JS-Logik, Handler, Component-Rendering-Ausgabe,
  API-Mocks, State-Übergänge.
- **Playwright**: Browser-Verhalten (Scroll-Physik, Bild-Decode-Timing,
  echtes Layout, Focus, Animation, IntersectionObserver mit echtem
  Layout).

Bei UI-Änderungen an Scroll/Layout/Animation/Focus-Verhalten SHALL die
Konvention einen E2E-Test als „Reminder" vorschlagen (nicht harte
Pflicht, um Overhead zu vermeiden — der Autor entscheidet).

#### Scenario: 07-testing.md enthält den Abschnitt

- **WHEN** ein Entwickler `docs/agent/07-testing.md` liest
- **THEN** findet er einen Abschnitt „Wann Vitest, wann Playwright" mit
  klarem Kriterienkatalog und dem Verweis auf `make test-e2e`
