## 1. Backend — Export-Route

- [x] 1.1 `internal/duties/export_handler.go`: `ExportSlots` mit Pflicht-Zeitraum (`from`/`to`, 400 `invalid_range`), Hauptquery über `duty_slots` + `duty_types` + `games`/`venues`/`game_templates`/`teams`
- [x] 1.2 Tages-Kontext je `(Datum, Saison)` einmal laden und cachen: Ausrichter über `settings.ResolveAusrichterForDayDetailed`, Spiele/Anwurfzeiten/Nachbartage mit denselben Größen wie `loadSameDayContextTx`
- [x] 1.3 Spalten-Rendering: deutsches Datum + Wochentag, Dienst-Ende aus Beginn + `hours_value`, Dauer mit Dezimalkomma, Zielgruppen-Labels, Regel-Texte (`normal`/`entfällt`/`reduziert → Variante`)
- [x] 1.4 UTF-8-BOM + `;`-Trenner (Excel-Import), Dateiname `dienste_<from>_<to>.csv`
- [x] 1.5 Route in `internal/app/router.go` im Tier `vorstand/trainer/sportliche_leitung`

## 2. Backend — Tests

- [x] 2.1 Happy-Path: alle Zeiten, Termin-Felder, Default-Ausrichter, `Ausrichter für Tag gesetzt = nein`
- [x] 2.2 Tageskonstellation: zwei Spiele am Tag, Heimspiel Vor-/Folgetag, explizit gesetzter Ausrichter
- [x] 2.3 Diensttyp-Regeln (`reduced` mit Variantenname, `skip`), `Herkunft = manuell`, Slot ohne Termin
- [x] 2.4 Invariante „keine Belegung/keine Namen" per Volltext-Prüfung der Antwort
- [x] 2.5 Zeitraum: Grenzen inklusive, ISO-Timestamp in `event_date` fällt in den Bereich
- [x] 2.6 Fehlerfälle: 400 bei fehlendem/ungültigem/verdrehtem Zeitraum, 403 für `spieler`
- [x] 2.7 Permission-Matrix: Eintrag in `internal/permissions/matrix_test.go` + `openspec/specs/permissions/spec.md`

## 3. Frontend

- [x] 3.1 `web/src/components/DutyExportModal.tsx`: Zeitraum-Dialog, vorbelegt mit dem angezeigten Monat, Blob-Download, 403-Meldung, gesperrter Button bei verdrehtem Zeitraum
- [x] 3.2 `KalenderPage`: Menüpunkt „Dienste als CSV" hinter `manage_duties`; Aktionsmenü erscheint jetzt auch für Trainer/sportliche Leitung
- [x] 3.3 Vitest für den Dialog (Vorbelegung, Anfrage-URL, Sperre, 403)
- [x] 3.4 `KalenderPage.permissions.test.tsx` auf die neue Menü-Realität umgestellt (Menü = Vereinigung der drei Capabilities, Einträge einzeln gegated)

## 4. Verifikation

- [x] 4.1 `go test ./...`, `go vet ./...`, `golangci-lint run`
- [x] 4.2 `pnpm -C web build`, `pnpm -C web test`, `pnpm -C web lint`
- [x] 4.3 `openspec validate --specs` / `--changes`
- [ ] 4.4 Sichtprüfung im Browser: Menüpunkt, Dialog, Datei in Excel öffnen (Umlaute, Dezimalkomma)
