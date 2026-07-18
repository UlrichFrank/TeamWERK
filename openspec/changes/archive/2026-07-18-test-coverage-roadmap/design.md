## Context

Ist-Stand (aus `metrics/REPORT.md`, Go `go test -cover ./internal/...`):

| Stack    | Coverage | Notiz                                   |
|----------|---------:|-----------------------------------------|
| Go       |  43,0 %  | ~flach ggü. 42,3 % vor 4 Wochen         |
| Frontend |    n/a   | Vitest-Coverage ungemessen (D4)         |

Per-Package Prod-LOC + Test-Stand + **90-Tage-Churn** (Grundwahrheit Juli 2026, Analyse-Fleet):

| Package        | Prod-LOC | Testfiles | Churn (90d) | Kritikalität                                             |
|----------------|---------:|----------:|------------:|----------------------------------------------------------|
| members        |    3 538 |        16 | **158**     | Handler ok; `Import` ~532 LOC/cog~177 → Refactor first; Authz-Grenze ungetestet (alle Tests nutzen admin) |
| games          |    3 752 |        15 | **126**     | Auto-Duty-Regen (`regenSingleDay` ~125 cog) — Refactor-Vorbehalt |
| auth           |    2 448 |        13 |      96     | gut abgedeckt; Rest-Fehlerpfade (Session/Reauth/Token)   |
| matchreports   |    2 255 |         7 |      50     | `images.go` 0 Tests · Router-Tier via prodserver unsichtbar |
| duties         |    1 207 |         4 |      62     | 18 Roadmap-Tests real **schon erledigt** (tasks-Drift); `match_report_guard` 0 Tests |
| files          |      938 |         1 |       8     | Berechtigungen · **höchstes PII-Leak-Risiko** · 9/12 Routen nur via reine Funktion |
| beitragslauf   |      821 |         4 |      14     | `confirm`/`protocol` (rechtl. Audit) + `export-data`-400 ungetestet |
| absences       |      743 |         2 |       8     | `Calendar?show_team` PII ungetestet · Update/Delete keine Tests |
| attendance     |      730 |         2 |       6     | Cross-Family/Trainer-falsches-Team (Kinder-PII)          |
| venues         |      360 |         1 |       4     | **von Roadmap übersehen** — 5/6 Routen nie getestet (Delete/DeleteAll/Import) |

Bereits existierende Test-Changes: `test-coverage-fachlich` (Duties-Section **de facto fertig**, tasks.md nur Drift) und `frontend-e2e-tests` (Playwright-Setup, 0/38).

## Goals / Non-Goals

**Goals**

- Solo-Dev-optimierte Reihenfolge: höchste Bug-Fang-Wahrscheinlichkeit pro Wartungs-Euro zuerst.
- Vier Prio-Achsen (Risk / Change-hot / Test-erst-Refactor / Sichtbarkeit) explizit machen, damit spätere Ad-hoc-Änderungen begründbar bleiben.
- Verbindliche Nicht-Ziele festhalten — was NICHT auf den Plan kommt, ist wertvoller als noch eine To-do-Liste.
- Präzedenz für Arch-Test-Muster (`broadcast_test.go`) als bevorzugte Testform bei sich wiederholenden Invarianten.

**Non-Goals**

- Diese Roadmap führt selbst keine Tests ein — Änderungen an Testcode passieren in den vier Folge-Proposals.
- Kein Coverage-%-Ziel (weder als Zahl noch als CI-Gate).
- Kein neuer Test-Runner, kein neues Framework (Playwright bleibt, Vitest bleibt, Go stdlib bleibt).

## Decisions

**D1 — Priorisierung nach Risk-first, nicht nach Coverage-Lücke**

Nicht „was hat wenig Tests" bestimmt die Reihenfolge, sondern „was tut am meisten weh, wenn es kaputt geht". Konsequenz: `internal/files` (Berechtigungen) vor `internal/videos` (mehr LOC, aber Fehler = kaputter Player, kein Datenleck).

Begründung: Solo-Dev — Zeit ist der teuerste Rohstoff, Bug-Kosten sind superlinear in Sichtbarkeit. Ein PII-Leak in `files` ist katastrophal, ein kaputter Video-Player ist ärgerlich.

**D2 — Arch-Tests vor Copy-Paste-Unit-Tests, wo die Invariante generisch ist**

Für Autorisierung existieren N Routen mit `RequireClubFunction`/`RequireRole`. Statt N × 2 (401/403) Copy-Paste-Tests: ein Arch-Test analog `internal/arch/broadcast_test.go`, der den Router parst und für jede gated Route prüft, dass mindestens ein Test im Package einen 401/403-Assertion für diese Route enthält. Allowlist mit Begründung für Ausnahmen.

Begründung: Ein Arch-Test veraltet nicht bei neuen Routen, ein Copy-Paste-Test wird bei jeder neuen Route vergessen. Skaliert. Wartungslast pro Route → 0.

Trade-off: Der Arch-Test prüft die *Existenz* eines Tests, nicht die fachliche Korrektheit. Deshalb ersetzt er keine echten Autorisierungs-Assertions in `test-coverage-fachlich` — er ergänzt sie als Backstop.

**D3 — Refactor vor Test bei cog>50**

Funktionen mit gocognit >50 (`members.Import` 177, `games.regenSingleDay` 125, `members.List` 77, `duties.Board` 58) bekommen **erst** ein Extract-Method-Refactoring, **dann** Tests. Nie umgekehrt.

Begründung: Tests auf einer 177-cog-Funktion frieren den Ist-Zustand ein, machen späteres Refactoring teurer, und decken den fachlichen Kern (parse/validate/persist bei `Import`) nicht besser ab als drei kleine Tests auf die extrahierten Funktionen. Reines Coverage-%-Aufblasen ohne Bug-Fang-Nutzen.

Trade-off: Refactor-Aufwand kommt vorher, kein „schnelles Coverage-Add" möglich. Bewusst akzeptiert.

**D4 — Frontend-Coverage vom Boden hoch, aber nicht via Vitest-Zahl**

Vitest-Coverage von 17,9 % ist kein sinnvolles Ziel: die Vitest-Suite kann browser-spezifische Bugs prinzipiell nicht catchen (siehe Motivation in `frontend-e2e-tests`). Golden-Path-E2Es in Playwright fangen mehr echte Regressionen als 40 % mehr Vitest-LOC.

Konsequenz: Nach `frontend-e2e-tests`-Setup werden **E2Es** hinzugefügt (Login, Dienst-Claim, Mitglied bearbeiten), keine Vitest-Renderer-Tests für Pages.

**D5 — Ein neuer Test-Change pro Domäne, nicht ein Mega-Change**

Jeder Prio-Punkt (files, absences+attendance, arch-gate) bekommt einen eigenen OpenSpec-Change mit eigenem Proposal-Zyklus. Kein „Coverage-Sprint"-Change mit 200 Tasks.

Begründung: Kleine Changes werden fertig, große verschimmeln (siehe die 14 In-Flight-Changes). Ein Test-Change pro Domäne ≈ 1–3 Sitzungen, überschaubar.

**D6 — Churn als zweite Achse: Risiko × Churn statt Risiko allein**

Die ursprüngliche Reihenfolge sortierte rein nach Schadenspotenzial (LOC × Kritikalität) und ließ die im Goals-Abschnitt genannte „change-hot"-Achse ungenutzt. Bugs clustern aber empirisch dort, wo Code sich ändert. Risiko und Churn sind orthogonal:

```
                    RISIKO (Schaden wenn kaputt)
                    niedrig                 hoch
         ┌────────────────────────┬──────────────────────────┐
    hoch │ games (126)            │ members (158)             │  ← Regressions-
  CHURN  │ trainings (73)         │ auth (96)                 │    Magnete
 (90 Tg) │ duties (62)            │ matchreports (50)         │
         ├────────────────────────┼──────────────────────────┤
  niedr. │ videos (46)            │ files (8) · beitragslauf  │  ← Write-once-
         │ metrics                │ absences (8) · venues (4) │    Versicherung
         └────────────────────────┴──────────────────────────┘
```

Konsequenz: `files`/`absences`/`beitragslauf` sind **hoch-Risiko × niedrig-Churn** → Write-once-Versicherung (Test rottet kaum, super ROI, aber selten aktiver Bug). `members`/`games`/`auth` sind **hoch-Churn** → dort landet der *nächste* Bug. Die Roadmap adressiert beide, aber die **Vorbedingungs- und Hebel-Achse gewinnt**: Welle 0 (prodserver-Fix + Fixtures + Drift-Detektor) zuerst, weil sie mehrere Wellen erst testbar macht und einen strukturellen Backstop für die Churn-Ecke liefert. Innerhalb der Fach-Wellen zieht der PII-Cluster (Schadenspotenzial) vor.

**D7 — Bug-Verdacht vor Charakterisierung verifizieren**

Die Analyse-Fleet meldet drei mögliche Code-Bugs. Ein Test, der fehlerhaftes Ist-Verhalten festnagelt, zementiert den Bug. Daher: erst am Code verifizieren, dann fixen, dann testen. Verifiziertes Ergebnis:

| Verdacht | Verdikt | Konsequenz |
|---|---|---|
| `members.UpdateStatus` liefert 204 bei unbekannter ID + verschluckt Exec-Fehler | **Bug bestätigt (P1)** | Code-Fix (RowsAffected→404, err-Check) VOR Test |
| `files.checkAntiEscalation` prüft `newRead` nicht gegen eigenen `can_read` | **Gap bestätigt (P2)** | Entscheidung fix-vs-dokumentieren vor Test |
| `files.DownloadFile` Bearer-Pfad ist ein Leck | **kein Bug** — fail-closed 401 | reklassifiziert als Dead-Code-Cleanup; Token-Pfad-Tests fehlen aber wirklich |

**D8 — Der Authz-Gate existiert bereits; das Loch ist die Handler-Verdrahtung**

`internal/permissions/matrix_test.go` läuft schon über `BuildRouter`. Der Blindspot ist `internal/testutil/prodserver`, das `MatchReports`/`Settings`/`Stammvereine` nie verdrahtet (→ gated Routen unsichtbar) und fehlender Nil-Guard bei `/api/stammvereine` (→ 500 wird als „bestanden" gewertet). Welle 0 fixt die Verdrahtung und ergänzt einen Drift-Detektor (analog `TestBroadcastAllowlist_NoOrphans`), der Erwartungs-Maps und `router.go` synchron hält. Damit ist der ursprünglich als „Phase 4" geplante `authz-arch-gate` weitgehend ein **Vervollständigen des Bestehenden**, nicht ein Neubau.

## Risks / Trade-offs

- **Diese Roadmap ist selbst ein Artefakt, das verrotten kann** → Mitigation: kurze `## What Changes`-Liste, nur vier Folge-Changes benannt. Wenn nach Change 1 die Welt anders aussieht, wird Change 2 neu bewertet, nicht sklavisch abgearbeitet.
- **Risk-first ist subjektiv** → die Achsen-Tabelle im Proposal macht die Auswahlkriterien explizit; andere Reihenfolgen sind vertretbar, aber begründungspflichtig.
- **Arch-Test-Gate (`test-authz-arch-gate`) kann falsch positiv/negativ sein** → der `broadcast_test.go`-Präzedenzfall zeigt, wie eine Allowlist mit Begründung die False-Positive-Fälle sauber handhabt. Denselben Mechanismus übernehmen.
- **`members.Import`-Refactor blockiert Tests dort** → Refactor ist kein Teil dieser Roadmap. Ein separater Change (`refactor-members-import`) ist Vorbedingung für Tests auf diesem Codepfad. Falls das nicht passiert, bleibt `Import` eben getestet-frei — bewusster Trade-off gegen Wertlos-Tests.

## Migration Plan

Keine Datenmigration. Keine Route-Änderung. Diese Roadmap ist reines Meta.

Die vier Folge-Changes werden **einzeln** proposed, wenn der jeweils vorherige archiviert ist — kein paralleles Auffahren.

## Open Questions

- **Frontend-Roadmap tiefer**: Braucht der Frontend-Teil eine eigene Roadmap-Change nach Abschluss von `frontend-e2e-tests`, oder reichen ad-hoc-Golden-Path-E2Es? Entscheidung vertagt bis nach Playwright-Setup.
- **Ratchet-Gate**: Soll `make metrics-gate` künftig Coverage-Regression blockieren? Aktuell nur Komplexität/Duplikation/Lint-Dichte im Gate. Entscheidung explizit vertagt, weil das Verhalten ändern würde (ein flaky Test → CI rot).
