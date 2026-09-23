## 1. Client: Video-IDs pro Spiel + Löschen

- [x] 1.1 `ExistingVideo` in `tools/video-encoder/internal/client/client.go` um `IDs []int` erweitern (alle Video-IDs des Spiels, nicht nur die für das Label „gewinnende")
- [x] 1.2 `VideosByGame` befüllt `IDs` für jedes Spiel; Status/DiskBytes-Auswahl für das Label bleibt wie bisher (`ready` gewinnt)
- [x] 1.3 Neue Methode `DeleteVideo(ctx, videoID) error` (`DELETE /api/videos/{id}`, nutzt `c.do`/`statusError` wie die übrigen Methoden)
- [x] 1.4 Unit-Tests: `TestVideosByGame` um `IDs`-Assertion erweitert; `TestDeleteVideo_HappyPath`, `TestDeleteVideo_Forbidden`

## 2. UI: Hinzufügen/Ersetzen-Auswahl

- [x] 2.1 `ui`-Struct: `videosByGame map[int]client.ExistingVideo`, `replaceGroup *widget.RadioGroup`, `replaceHint *widget.Label`, `replaceBox *fyne.Container`
- [x] 2.2 `build()`: RadioGroup mit Optionen „Hinzufügen (weiteres Video)" / „Ersetzen (vorhandenes Video wird gelöscht)", Default „Hinzufügen", in eigenem Container unter dem Formular platziert, initial verborgen
- [x] 2.3 `gameSelect.OnChanged` (`onGameChanged`): zeigt `replaceBox` inkl. `Describe()`-Hinweistext, wenn `videosByGame[gameID]` existiert; verbirgt sie sonst und setzt die Auswahl zurück auf „Hinzufügen"
- [x] 2.4 `onTeamChanged`/Lade-Goroutine speichert `videosByGame` auf dem `ui`-Struct
- [x] 2.5 `setBusy`: RadioGroup während eines laufenden Vorgangs deaktivieren

## 3. Ablauf: Bestätigung, Upload, Löschung

- [x] 3.1 `start()`: bei gewähltem „Ersetzen" zuerst `dialog.ShowConfirm(...)` mit Anzahl der zu löschenden Videos; erst nach Bestätigung den bisherigen Ablauf (`startJob`) anstoßen
- [x] 3.2 `job`-Struct: `replace bool`, `replaceIDs []int` — pro Lauf frisch aus der aktuellen UI-Auswahl gesetzt (nicht Teil der `meta`-Gleichheitsprüfung, damit ein Moduswechsel keinen unnötigen Job-Reset auslöst)
- [x] 3.3 `run()`: nach erfolgreichem Upload, wenn `j.replace`, `DeleteVideo` für jede ID in `j.replaceIDs` außer der gerade neu angelegten `j.videoID` aufrufen; erster Fehler wird als Warnhinweis zurückgegeben, bricht die Schleife aber nicht als Gesamtfehler ab
- [x] 3.4 `finish()`: Erfolgsmeldung um den optionalen Löschungs-Warnhinweis ergänzt; RadioGroup nach jedem erfolgreichen Lauf auf „Hinzufügen" zurückgesetzt

## 4. Doku, CHANGELOG, Commit

- [x] 4.1 `web/public/CHANGELOG.md`: Eintrag `[feat] video-encoder: Hinzufügen-oder-Ersetzen-Auswahl bei bereits vorhandenem Spielvideo` unter heutigem Datum ergänzt
- [x] 4.2 Conventional Commit `feat(video-encoder): Hinzufügen-oder-Ersetzen-Auswahl beim Upload` (Commit `109d58dd`)
- [x] 4.3 `go build`/`go vet`/`go test` für `tools/video-encoder/...` grün
- [x] 4.4 `openspec validate` grün
- [ ] 4.5 OpenSpec-Change `video-upload-anhaengen-oder-ersetzen` nach Verifikation durch den Nutzer archivieren (`openspec archive`)
