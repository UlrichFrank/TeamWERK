## Why

Coverage steht bei **Go 42,3 %** und **Frontend 17,9 %**. Zwei In-Flight-Changes adressieren Teile (`test-coverage-fachlich`: auth ✅, duties/members/kader ⏳; `frontend-e2e-tests`: Playwright-Setup ⏳), aber es fehlt eine **strategische Reihenfolge und eine klare Grenze, was NICHT getestet wird**. Ohne Roadmap drohen zwei Anti-Patterns: (a) Coverage-% als Ziel statt als Indikator, (b) Unit-Tests auf Code, der vor dem Testen refactored gehört (`members.Import` cog=177 ist der Prototyp).

Zusätzlich klaffen konkrete Blindspots, die keine der bestehenden Changes abdeckt:

- **`internal/files/` — 938 LOC / 1 Test**: Berechtigungsmatrix (`folder_permissions.everyone.can_read/write`, Rollen-Gates), Grundlage für Anleitungen-Bilder (Gotcha „Dienst-Anleitungen") und Vereinsdokumente. Ein Autorisierungs-Bug hier ist PII-Leak-Risiko.
- **`internal/absences/` — 743 LOC / 2 Tests** und **`internal/attendance/` — 730 LOC / 2 Tests**: Kalender-, Anwesenheits- und Autorisierungslogik ohne Netz.
- **Strukturelle Autorisierungs-Gates fehlen**: das `broadcast_test.go`-Muster (Router parsen, Invariante mechanisch prüfen) skaliert besser als N Copy-Paste-Tests, wird aber bisher nur für die SSE-Regel genutzt.

## What Changes

Diese Roadmap ist **selbst kein Test-Code**, sondern legt Reihenfolge, Prinzipien und explizite Nicht-Ziele fest. Sie mündet in vier untergeordnete Changes (jeweils eigener Proposal-Zyklus), die in dieser Reihenfolge angegangen werden:

1. **`test-coverage-fachlich` fertigstellen** (existiert, 15 % done) — duties, members, kader, kleinere games/trainings/absences/chat-Ergänzungen zu Ende bringen.
2. **`test-files-permissions`** (neu) — Berechtigungsmatrix von `internal/files/` fachlich absichern (Ordner-CRUD, Datei-Upload/Download, Rollen- und `everyone`-Flags, Rekursion).
3. **`test-absences-attendance`** (neu) — Autorisierungsgrenzfälle, Kalender-Aggregation, Eltern-Sichtbarkeit.
4. **`test-authz-arch-gate`** (neu) — Arch-Test analog `broadcast_test.go`: jede Route hinter `RequireClubFunction`/`RequireRole` besitzt einen Autorisierungs-Test (401/403). Allowlist mit Begründung für dokumentierte Ausnahmen.

**Parallel weiter**: `frontend-e2e-tests` (Playwright-Setup abschließen, dann Golden-Path-E2Es für Login → Team → Dienstbörse → Claim).

Explizit **nicht** Teil der Roadmap:
- Vitest-Coverage-Zahl heben (misleading Metric — Playwright > Vitest-% für den Solo-Dev).
- Coverage-% als CI-Gate (widerspricht Projektregel „Coverage ist Indikator, kein Gate").
- Tests auf Hotspots mit cog>50, **bevor** dort refactored wurde (`members.Import`, `members.List`, `games.regenSingleDay`).

## Capabilities

### New Capabilities

- `test-strategy`: Verbindliche Grundsätze für neue Tests im Projekt (Prinzipien, Nicht-Ziele, Refactor-vor-Test-Regel, Arch-Test-Präferenz vor Copy-Paste). Referenzdokument, das nachfolgende Test-Changes verpflichtet.

### Modified Capabilities

*(keine — bestehende Fach-Specs bleiben unverändert)*

## Impact

- **Betroffen**: `openspec/` (neue Change-Ordner in Folge-Iterationen), `docs/agent/07-testing.md` (Verweis auf Strategie-Spec ergänzen).
- **Kein Produktionscode** in diesem Change. Die nachfolgenden Test-Changes fügen ausschließlich `*_test.go`-Dateien und ggf. `testutil`-Helfer hinzu, außer der Refactor-Meilenstein für `members.Import` (dort ist Produktionscode-Änderung Vorbedingung, kein Bestandteil dieser Roadmap).
- **CI**: keine Änderung an Gates in diesem Change. Ein optionaler späterer Ratchet-Schritt (Coverage darf nur steigen, per `metrics/thresholds.yml`) ist bewusst nicht Teil dieser Roadmap.
- **Team-Kontext**: Solo-Dev. Priorisierung optimiert daher **Wartungslast der Tests** gleich stark wie Bug-Fang — begünstigt Contract-Tests und Arch-Gates gegenüber tiefen Unit-Bäumen.
