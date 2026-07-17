## ADDED Requirements

### Requirement: Priorisierung nach Risiko, nicht nach Coverage-Lücke
Neue Test-Investitionen SHALL nach Bug-Kosten-Risiko priorisiert werden, nicht nach fehlenden Coverage-Prozentpunkten. Ein Package mit hoher LOC und niedriger Coverage darf hinter einem kleineren Package zurücktreten, wenn dort die fachliche Konsequenz eines Fehlers geringer ist.

#### Scenario: PII-relevantes Package vor UX-Package
- **WHEN** ein Team-Backlog-Eintrag „Tests für `internal/videos`" gegen einen Eintrag „Tests für `internal/files`" priorisiert wird
- **THEN** `internal/files` (Berechtigungen, PII-Leak-Risiko) SHALL zuerst umgesetzt werden, unabhängig davon dass `internal/videos` mehr LOC hat

#### Scenario: Refactoring statt Testen bei extrem hoher Komplexität
- **WHEN** ein Vorschlag „Tests für `members.Import`" (gocognit = 177) eingeht
- **THEN** die Antwort SHALL sein „erst Extract-Method-Refactoring, dann Tests" — nicht „Tests auf den Ist-Stand hinzufügen"

### Requirement: Arch-Tests vor Copy-Paste-Unit-Tests bei generischen Invarianten
Wenn eine Test-Invariante über N Routen/Handler wiederholbar ist (z.B. „jede gated Route muss einen 401/403-Test haben"), SHALL ein Arch-Test analog `internal/arch/broadcast_test.go` implementiert werden statt N identischer Copy-Paste-Tests.

#### Scenario: Neue Route wird hinzugefügt
- **WHEN** in `BuildRouter` eine neue Route mit `auth.RequireClubFunction(...)` mountet und im Ziel-Package kein Autorisierungs-Test existiert
- **THEN** der Arch-Test (`internal/arch/authz_test.go`) SHALL fehlschlagen und die Route beim Namen nennen

#### Scenario: Route ist bewusste Ausnahme
- **WHEN** eine Route explizit ohne Autorisierungs-Test bleiben soll (dokumentierte Ausnahme)
- **THEN** ein Eintrag in der `authzAllowlist` mit textueller Begründung SHALL vorhanden sein; ein verwaister Allowlist-Eintrag (Route existiert nicht mehr) SHALL den Test ebenfalls fehlschlagen lassen

### Requirement: Coverage-Prozent ist Indikator, kein Gate
Weder `make test` noch `make metrics-gate` noch die CI SHALL bei einer Coverage-Regression fehlschlagen. Coverage-Zahlen werden ausschließlich in `metrics/REPORT.md` berichtet, aber nicht durchgesetzt.

#### Scenario: Neuer Test-Change wird proposed
- **WHEN** ein Proposal argumentiert „hebt Coverage von 42 % auf 55 %"
- **THEN** dieses Argument SHALL alleine keine Priorisierung rechtfertigen; die Begründung MUST auf fachliches Risiko / Bug-Fang / strukturelle Invariante zeigen

#### Scenario: Coverage sinkt lokal
- **WHEN** ein Refactoring-PR die Coverage-Zahl senkt, aber fachlich äquivalent bleibt
- **THEN** der PR SHALL nicht blockiert werden; die Zahl im Report ist informativ

### Requirement: Ein Test-Change pro Domäne
Test-Ergänzungen SHALL in OpenSpec-Changes zerlegt werden, die eine einzelne Domäne oder eine einzelne strukturelle Invariante abdecken. Ein Mega-Change „Coverage-Sprint" mit >50 Tasks über mehrere Domänen SHALL nicht erstellt werden.

#### Scenario: Domänen-Grenzen
- **WHEN** Tests für `files`, `absences` und `attendance` alle gewünscht sind
- **THEN** mindestens zwei separate Changes (`test-files-permissions`, `test-absences-attendance`) SHALL erstellt werden

#### Scenario: Fach- vs. Struktur-Change
- **WHEN** ein Change fachliche Tests (Handler-Level) und einen Arch-Test-Gate im selben Proposal bündelt
- **THEN** die zwei Teile SHALL in separate Changes zerlegt werden (der Arch-Test-Gate ist konzeptionell orthogonal zur Fach-Testsuite)

### Requirement: Frontend-Test-Investition priorisiert E2E vor Vitest-Coverage
Neue Frontend-Test-Investitionen SHALL vorrangig in Playwright-E2Es fließen, nicht in Vitest-Rendering-Tests, solange die Vitest-Suite browser-spezifische Bugs prinzipiell nicht catchen kann (Scroll, Focus, Layout-Physik).

#### Scenario: Vorschlag „Vitest-Coverage von 17 % auf 40 %"
- **WHEN** ein Test-Vorschlag ausschließlich Vitest-Coverage erhöht (Rendering-Snapshots, Komponenten-Tests ohne Interaktions-Assertions)
- **THEN** die Priorisierung SHALL stattdessen zugunsten von Playwright-Golden-Path-E2Es umgelenkt werden

#### Scenario: Vitest bleibt für Unit-Logik
- **WHEN** eine reine Utility-Funktion (`lib/sepa.ts`, `lib/crypto.ts`, `lib/sepaXml.ts`) getestet werden soll
- **THEN** Vitest SHALL die richtige Wahl bleiben (kein Browser-Verhalten involviert)
