## 1. Endpoint

- [x] 1.1 `ListTeamsForUser`: Trainer-Zweig, Trainer-`scope=duties`-Zweig und Nicht-Trainer-Zweig zu einem Zweig zusammenführen (`!HasFunction("sportliche_leitung")`)
- [x] 1.2 Scope-los → `user_accessible_teams`; `?scope=duties` → Stammkader (Selbst + Kinder) ∪ Trainer-Teams
- [x] 1.3 `admin`/`vorstand`, `sportliche_leitung` und die beiden Statistik-Scopes unverändert lassen

## 2. Tests

- [x] 2.1 `TestListTeamsForUser_ScopeDutiesTrainerAlsoParent`: die Assertion „ohne scope genau 1 Team" auf die Vereinigung umstellen, Kommentar zur Herkunft der alten Zusage ergänzen
- [x] 2.2 Neuer Test `TestListTeamsForUser_TrainerElternteilErweiterterKader`: Trainer + Kind im Stammkader + Kind im erweiterten Kader; scope-los enthält alle drei, `?scope=duties` nicht den erweiterten Kader
- [x] 2.3 Prüfen, dass `TestListTeamsForUser_TrainerSeesOwnTeam` (Trainer ohne Kinder) unverändert grün bleibt
- [x] 2.4 Beide geänderten/neuen Tests gegen den alten Stand laufen lassen — müssen rot sein

## 3. Spec & Abschluss

- [x] 3.1 `api-routes`: Anforderung „Teams-Endpoint ist rollenabhängig" auf die Vereinigungs-Regel korrigieren
- [x] 3.2 `make test` + `golangci-lint` grün
- [ ] 3.3 `/verify-change` durchlaufen
- [x] 3.4 Commit-Betreff als `fix(games): …` formulieren — `web/public/CHANGELOG.md` wird von `make build` aus dem Git-Log erzeugt (`scripts/gen-changelog.py`), nicht von Hand gepflegt
