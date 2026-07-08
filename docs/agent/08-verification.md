# Harness / Verifikation

Konventionen werden mechanisch durchgesetzt, nicht nur dokumentiert.

- **Git-Hooks** (`make hooks`, in `make init`): `pre-commit` = gofmt auf gestagete Go-Dateien; `pre-push` = volles Gate (`go vet`, `go test ./...` inkl. Architektur-Test, `golangci-lint`, `pnpm -C web build/test/lint`, `openspec validate`). Race-Detector läuft separat via `make test-race` (~10× langsamer). Notausgang: `git push --no-verify`.
- **Architektur-Test** `internal/arch/arch_test.go` (stdlib, Teil von `make test`): Domain-Packages importieren sich nicht gegenseitig; Foundation importiert keine Domain/Composition; jedes neue `internal/`-Package muss klassifiziert werden.
- **Broadcast-Gate** `internal/arch/broadcast_test.go` (stdlib, Teil von `make test`): erzwingt die SSE-Hard-Rule „jede Mutations-Route broadcastet". Parst `BuildRouter`, prüft für jede `POST`/`PUT`/`PATCH`/`DELETE`-Route, dass der Handler-Rumpf einen Broadcast-artigen Aufruf enthält (`h.hub.Broadcast`/`BroadcastToUsers` **oder** einen Helfer wie `broadcastMembers`). Bewusste Ausnahmen (kein Hub-Live-Update, eigener Kanal wie Chat, pro-Nutzer/Session, Mailversand) stehen mit Begründung in der `broadcastAllowlist`; ein verwaister Allowlist-Eintrag (Route existiert nicht mehr) lässt den Test ebenfalls fehlschlagen.
- **gofmt-Selbstkorrektur:** `PostToolUse`-Hook (`scripts/claude-gofmt-hook.sh`) formatiert via Edit/Write geänderte `*.go`-Dateien.
- **Pre-Completion:** Slash-Command **`/verify-change`** prüft Build/Test/Lint + Projekt-Invarianten (Route→Tests, Mutation→`Broadcast`/`useLiveUpdates`, brand-Tokens, lucide-Icons, Migrationsnummer, `openspec validate`).
- **Permissions:** geteilte Routine-Befehle in `.claude/settings.json`; `.claude/settings.local.json` (gitignored) nur maschinenspezifisch.

## Eskalation (wenn etwas klemmt)

Nicht raten, nicht in einer Schleife festfahren — sauber an den Menschen übergeben, sobald:

- **Gate bleibt rot:** Nach **3** erfolglosen Versuchen, denselben Test/Lint/Build-Fehler zu beheben → stoppen, Stand + Fehlausgabe + Hypothese zusammenfassen, fragen. Kein `--no-verify`, keine Symptom-Workarounds, keine abgeschwächten Assertions zur Grün-Färbung.
- **Irreversibel/riskant:** Prod-Migration, `make deploy`, Daten löschen/überschreiben, Secrets — vor der Ausführung bestätigen lassen.
- **Mehrdeutig:** Anforderung lässt mehrere sinnvolle Interpretationen zu → kurz nachfragen statt eine Variante zu unterstellen.
