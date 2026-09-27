# Tasks

## 1. Vorbereitung

- [x] 1.1 Change `dienste-erweiterter-kader` archivieren (alle Tasks erledigt), damit das Delta dieses Changes auf dessen Requirement-Text aufsetzt; verifiziert durch `openspec validate dienst-bilanz-je-kader --strict` grün nach dem Archiv
- [x] 1.2 Prod-Zählquery der Mehrkader-Kinder aus `design.md` („Risks") dem Nutzer übergeben (nicht selbst ausführen); verifiziert durch Nennung in der Abschlussmeldung

## 2. Dienst-Bilanz (`internal/dutyfairness`)

- [x] 2.1 Positionstyp je (Mitglied, Stammteam) einführen (`Snapshot.standings`, `Team.Members []*Standing`), `Member` verliert `Geleistet`/`Vorhersage`, Doc-Kommentare an `Member`/`Team` korrigieren („dieselben Zahlen in beiden Ranglisten" entfällt); `loadMembers` legt je Stammkader-Zeile genau eine Position an; verifiziert durch `go build ./...` und bestehende `fairness_test.go`/`aushilfe_test.go` nach Umstellung ihrer Zugriffe grün
- [x] 2.2 `countAssignments`: zweistufige Teilung (Mitglieder wie bisher, dann je Mitglied auf passende Stammteams: `memberTeams ∩ slot.teams`, generisch/Stufe 5 auf alle Stammteams); verifiziert durch neue Tests `TestCompute_ZweiKader_DienstZaehltNurFuerEigeneMannschaft`, `TestCompute_ZweiKader_GenerischWirdGeteilt`, `TestCompute_ZweiKader_GemeinsamesSpielWirdGeteilt`, `TestCompute_ZweiKader_GeschwisterVorKaderTeilung` (0,5 / 0,25 / 0,25), `TestCompute_ZweiKader_Stufe5WirdGeteilt`
- [x] 2.3 Invariante absichern: `TestCompute_SummeDerPositionenGleichKindZaehlung` — Summe der Positionen je Kind und Familiensumme gleich Zahl der Zuweisungen, Ein-Kader-Kinder unverändert; verifiziert durch den Test
- [x] 2.4 `Team.Ranked()` und `PositionsFor` auf Positionen umstellen; Ranglisten-Handler liest Positionswerte; verifiziert durch `TestRangliste_ZweiKader_EigeneZahlenJeBlock` (Kind mit 3 Diensten in A, 0 in B: Block A = 3, Block B = 0 und hinter einem Kind mit 1) sowie bestehende `handler_test.go` grün

## 3. Dashboard (`internal/dashboard`)

- [x] 3.1 `queryDutyAccount` liest `geleistet`/`vorhersage` aus der Position statt vom Member; verifiziert durch `TestDashboard_DutyAccount_ZweiKaderEigeneZahlen` (zwei Positionen, K1 = 1, K2 = 0, je eigenes `soll`, Werte identisch zur Rangliste) und bestehende Dashboard-Tests grün

## 4. Doku

- [x] 4.1 Gotcha „Dienst-Bilanz je Kind" in `docs/agent/06-gotchas.md` auf Zählung je (Kind, Stammkader) und die zweistufige Teilung umschreiben; verifiziert durch Review des Absatzes
- [x] 4.2 Benutzerhandbuch (`web/public/benutzerhandbuch.html`, Dashboard/Dienste): ein Satz, dass ein Kind in zwei Mannschaften je Mannschaft eigene Zahlen hat und nicht zuordenbare Dienste (Vereinsfest, gemeinsames Spiel) geteilt werden; verifiziert durch Sichtprüfung im Browser

## 5. Abschluss

- [x] 5.1 `/verify-change` ausführen (Build, `make test`, Lint, `pnpm -C web test`, `openspec validate dienst-bilanz-je-kader --strict`); verifiziert durch grünes Gate
