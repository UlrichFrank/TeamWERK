## Context

Welle 1 der `test-coverage-roadmap`. Ziel: die PII-tragenden Routen (Dokumente/Ordner,
Abwesenheiten, Anwesenheits-Recording, Spielbericht-Bilder, Spielbericht-Slot-Guard) auf
Autorisierungs-Ebene mechanisch festnageln. Enabler ist Welle 0 (`test-harness`): zentrale
Fixtures und ein voll verdrahteter `prodserver`-Router stehen bereit.

Der Scope wurde gegen den realen Code geprüft (nicht gegen Roadmap-Annahmen). Wesentliche
Korrektur: attendance-**Stats** (`GetMemberStats`/`GetTeamStats`/`GetTeamOpen`) sind bereits
umfassend getestet — Welle 1 fasst dort nur das ungetestete **Recording** an.

## Goals / Non-Goals

**Goals:**

- Jede PII-tragende Route hat mindestens einen Happy-Path- und einen Fremdzugriff-Test
  (401/403/404), der die Autorisierungs-Invariante festnagelt.
- Tests laufen über den echten Router/Handler, nicht gegen isolierte ACL-Hilfsfunktionen
  (Route-Ebene statt nur Unit) — außer dort, wo eine Unit bereits existiert und die Route
  denselben Pfad nutzt (dann nur die HTTP-Ergänzung, keine Duplikation).
- Keine bestehenden Tests duplizieren: `TestResolveAccess_*`, `TestFolderContents_*`,
  `TestListPermissions_*`, `TestCheckAntiEscalation_*` (files) sowie die attendance-Stats-Tests
  bleiben unangetastet.

**Non-Goals:**

- Keine Geschäftslogik-, API-, Schema- oder SSE-Änderung. Reiner Test-Change.
- Keine Coverage-Prozent-Ziele (lt. `test-strategy` kein Gate).
- Keine Frontend-Tests (Playwright läuft separat, Welle 7).

## Decisions

**D1 — Eine Capability `pii-route-authz` statt Aufsplittung pro Domäne.**
`test-strategy` fordert „ein Test-Change pro Domäne" und nennt explizit den Fall
files/absences/attendance → „mindestens zwei separate Changes". Wave 1 bündelt dennoch fünf
Bereiche in einem Change. Begründung: die abgedeckte Invariante ist **strukturell eine** —
„PII-Route ist fail-closed autorisiert" — nicht fünf fachlich unabhängige Feature-Suiten. Die
Roadmap gruppiert sie bewusst nach Risiko-Cluster (DSGVO-Leak) und geteilten Welle-0-Fixtures.
Das entspricht `test-strategy`'s eigenem Vorrang für strukturelle Invarianten. **Trade-off**
offen dokumentiert: sollte der Change zu groß werden (> ~30 Tasks), wird er entlang der
Package-Grenzen in `test-pii-files` / `test-pii-absences-attendance` / `test-pii-matchreports`
gesplittet — die `## Test-Anforderungen` sind bereits so gegliedert, dass ein Split
verlustfrei möglich ist.

**D2 — attendance-Recording lebt in `trainings`/`games`, nicht in `attendance`.**
`POST /api/training-sessions/{id}/attendances` → `Trainings.SaveAttendances`,
`POST /api/games/{id}/attendances` → `Games.SaveAttendances`. Beim Umsetzen zeigte sich (Roadmap
9.2): das Package heißt `internal/trainings` (mit „s") und **hat** bereits Tests
(`PlayerForbidden`, `TrainerOK`); `internal/games` deckt das Recording **vollständig** ab
(inkl. `TrainerOfOtherTeam_403`, `Unauthenticated_401`, `NotFound_404`). Einziger echter Gap:
der Fremd-Team-Trainer bei `trainings.SaveAttendances` (nur `CreateSeries` hatte das Muster).
Welle 1 ergänzt daher gezielt zwei `trainings`-Tests und dupliziert games **nicht**. Beide
Recording-Routen sind per Router mit `RequireClubFunction("trainer","sportliche_leitung")`
gated — der Team-Match ist Handler-Logik (`hasTeamAccess`/`canRecordGameAttendance`).

**D3 — Bug-Verdacht vor Charakterisierung.** Wenn ein Test einen echten Fail-Open aufdeckt
(z.B. `download-token` gibt Token ohne can_read aus, oder `ServeImage` verpasst einen
`published`-State), wird der Code-Fix in einem eigenen `fix(...)`-Commit **vorangestellt** und
erst der korrigierte Pfad getestet — nie das fehlerhafte Ist-Verhalten zementieren.

**D4 — matchreports `ServeImage` Erwartungsmatrix (aus `images.go:215`):**
401 (kein Claim) · 400 (bad id) · 404 (unbekannter Report/Image) · 403 (`authorID != UserID`
und kein Reviewer) · 200 (Autor oder Reviewer = medien/vorstand/admin).

**D5 — duties Guard-Erwartung (aus `assertSlotTakePermitted`):** Namens-Match auf duty_type
`"Spielbericht"`. Nicht-Spielbericht-Slot → immer erlaubt. Spielbericht-Slot: nur
`presseteam`/`admin`, sonst `role_required` (→ 403). Proxy-Kind-Fall: der Guard wertet
`claims.Role` des **handelnden** Users (Elternteil) — ein Elternteil ohne `presseteam` darf
einen Spielbericht-Slot auch dann nicht ziehen, wenn das Kind Presseteam wäre. Genau diese
Rollenverschiebung wird festgenagelt.

## Risks / Trade-offs

- **Change-Größe:** fünf Bereiche in einem Change (siehe D1). Mitigiert durch split-fähige
  Gliederung und Task-Obergrenze.
- **Package-Streuung:** Recording-Tests in `training`/`games` (D2) erweitern den Scope über die
  „PII-Package"-Intuition hinaus — bewusst akzeptiert, weil die Route den PII-Cluster trägt.
- **Echte Bugs möglich:** die ungetesteten Pfade (`download-token`, `SaveAttendances`) könnten
  Fail-Open enthalten. Das ist ein Feature dieser Welle, kein Risiko — jeder Fund wird per D3
  sauber als Fix-vor-Test behandelt und kann den Change zeitlich strecken.
