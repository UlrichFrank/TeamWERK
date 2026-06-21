# Tasks — Contribution-Infrastruktur

## 1. Leitfäden
- [ ] 1.1 `CONTRIBUTING.md`: OpenSpec-Workflow, Conventional Commits, Hard Rules, Setup, Gate
- [ ] 1.2 `CODE_OF_CONDUCT.md`: Contributor Covenant
- [ ] 1.3 `SECURITY.md`: privater Meldeweg, unterstützte Versionen

## 2. GitHub-Vorlagen
- [ ] 2.1 `.github/ISSUE_TEMPLATE/bug_report.md` + `feature_request.md`
- [ ] 2.2 `.github/pull_request_template.md` (Checkliste: Tests, Broadcast/useLiveUpdates, brand-Tokens, openspec validate)

## 3. Öffentliche CI
- [ ] 3.1 `.github/workflows/ci.yml`: Go 1.25 + pnpm
- [ ] 3.2 Schritte: `go vet`, `go test -race ./...`, `golangci-lint`, `pnpm -C web build/test/lint`, `openspec validate`
- [ ] 3.3 CI auf sauberem Checkout grün; bewusst roter Test wird abgewiesen
- [ ] 3.4 Branch-Protection-Empfehlung dokumentieren (CI required)

## 4. Self-Hosting
- [ ] 4.1 `docs/SELF_HOSTING.md`: VPS-Setup, Nginx/Certbot, systemd, Scheduler-Cron
- [ ] 4.2 Vollständige ENV-Referenz (inkl. Branding-Variablen aus ②, VAPID)
- [ ] 4.3 Backup-Anleitung (DB + Beitragslauf-Protokoll-Dir + storage/)

## 5. AGPL §13
- [ ] 5.1 Source-Link im App-Footer/Info-Seite (Repo-URL aus Config — ②)
- [ ] 5.2 Test, dass der Source-Link gerendert wird

## 6. Verifikation
- [ ] 6.1 `openspec validate --strict` für alle Open-Source-Changes grün
- [ ] 6.2 Dry-Run: frischer Contributor folgt CONTRIBUTING + SELF_HOSTING erfolgreich
