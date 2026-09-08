## 1. Sichtbarkeit

- [x] 1.1 `internal/videos/access.go` — `userBelongsToTeam` um zwei `UNION`-Zweige erweitern (Mitglied im erweiterten Kader, Elternteil eines solchen), Statusfilter `<> 'ausgetreten'` mit Begründungs-Kommentar (Förderkinder)
- [x] 1.2 `internal/videos/crud.go` — `visibilityFilter` um dieselben zwei Zweige erweitern; der Kommentar „spiegelt exakt userBelongsToTeam" muss weiter stimmen
- [x] 1.3 Prüfen, dass damit auch Detailabruf und Stream-Token abgedeckt sind (`CanViewVideo` → `Play` → `?st=`), also kein vierter Pfad offen bleibt

## 2. Benachrichtigung

- [x] 2.1 `internal/videos/worker.go` — `pushRecipients` um dieselben zwei Zweige erweitern
- [x] 2.2 Eintrag in `internal/arch/audience_test.go` (`audienceAllowlist`) auf den neuen Stand bringen: die Video-Menge bleibt eigenständig, die Begründung nennt jetzt nicht mehr das Fehlen des erweiterten Kaders

## 3. Tests

- [x] 3.1 `TestListVideos_ErwKaderSiehtTeamvideos`, `TestListVideos_ElternDesErwKadersSehenTeamvideos`
- [x] 3.2 `TestListVideos_FoerderkindWirdNichtWegenStatusGefiltert` — der Fall, an dem ein `= 'aktiv'`-Filter stillschweigend scheitern würde
- [x] 3.3 `TestListVideos_AusgetretenesErwMitgliedSiehtNichts`
- [x] 3.4 `TestGetVideo_ErwKaderDarfDetailLesen` (200) und `TestGetVideo_FremdesTeamBleibtVerboten` (403)
- [x] 3.5 `TestPushRecipients_ErwKaderUndEltern` — Ready-Meldung erreicht Mitglied und Elternteil

## 4. Abschluss

- [x] 4.1 `make test` + `golangci-lint` grün; `openspec validate videos-erweiterter-kader --strict`
- [x] 4.2 Gotcha in `docs/agent/06-gotchas.md` ergänzen: Video-Sichtbarkeit steht an drei Stellen und muss deckungsgleich bleiben; Statusfilter-Falle `foerderkind`
