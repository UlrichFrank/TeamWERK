## Context

Nach der ersten Test-Runde (`test-coverage-fachlich`) haben 9 Pakete Tests, aber die Coverage-Zahlen sind niedrig (9–56 %). Die identifizierten Lücken sind nicht zufällig: sie betreffen entweder sicherheitskritische Pfade (`ChangePassword`), tägliche Nutzerpfade (`GetProfile`) oder seltene aber fehleranfällige Jahresabläufe (`CopyFromSeason`). Das bisherige Testutil-Setup ist komplett; neue Tests können sofort geschrieben werden.

Der zweite Aspekt — den Test-Standard als Konvention zu verankern — ist bewusst leichtgewichtig gehalten: kein automatischer CI-Gate (zu viel Aufwand, zu willkürliche Schwellwerte), sondern ein Werkzeug (`make coverage`) + eine Konvention (Checkliste in Proposals).

## Goals / Non-Goals

**Goals:**
- `make coverage` als Standardbefehl für lokale Coverage-Sichtbarkeit
- Test-Anforderungen als Pflichtfeld in OpenSpec-Proposals (CLAUDE.md-Regel)
- Alle 8 identifizierten fachlich kritischen Lücken geschlossen

**Non-Goals:**
- Kein automatischer CI-Coverage-Gate (kein Schwellwert-Check)
- Kein Coverage-Badge oder externes Reporting
- Kein Mocking-Framework einführen
- Keine Tests für reine Infrastruktur (NewHandler, Hilfsfunktionen ohne Logik)

## Decisions

**D1 — `make coverage` ohne Gate**
Coverage-Zahlen als Information, nicht als Blocker. Ein Gate mit willkürlichem Schwellwert (z.B. 50 %) motiviert Dummy-Tests. Die Konvention in CLAUDE.md — „neue Route = mindestens 2 Tests" — ist das eigentliche Qualitätskriterium.

**D2 — HTML-Report nach `/tmp/`**
Kein eingecheckter Report, kein Build-Artefakt. Der Report wird lokal in `/tmp/teamwerk-coverage.html` erzeugt. Damit kein Lärm im Repository und keine großen Binary-Diffs.

**D3 — Test-Anforderungen in proposal.md, nicht in tasks.md**
Der Vorschlagende denkt beim Schreiben der Proposal über die Fachlichkeit nach und notiert, welche Szenarien getestet werden müssen. Die Implementierung läuft dann in tasks.md. Kein neues Template-Feld im OpenSpec-Schema erforderlich — ein Abschnitt in Freitext reicht.

**D4 — Tests als Ergänzungen in bestehenden `_test.go`-Dateien**
Keine neuen Test-Packages, keine neuen Helfer-Dateien. Alle neuen Tests werden an die bestehenden `handler_test.go`-Dateien angehängt, um den Package-Kontext zu nutzen.

**D5 — Fachliche Priorisierung der Lücken**
Reihenfolge: Sicherheit zuerst (`ChangePassword`), dann täglich genutzte Pfade (`GetProfile`/`UpdateProfile`), dann Workflow-Pfade (`Fulfill`, `ApproveMembership`), dann seltene aber kritische Operationen (`CopyFromSeason`).

## Risks / Trade-offs

- [GetProfile ist komplex, ~100 Zeilen Query] → Test prüft nur die Kernfelder (user_id, email, name), nicht jedes Nullable-Feld
- [CopyFromSeason hat viele Varianten von member_source] → Nur `same-age-previous` getestet (häufigster Weg); `auto-assign` hat bereits eigene Tests
- [members/ hat 9.5 % Coverage, 2178 Zeilen] → Nach diesem Change: ~20–25 % erwartet; vollständige Abdeckung aller 46 Handler-Methoden wäre unverhältnismäßig
