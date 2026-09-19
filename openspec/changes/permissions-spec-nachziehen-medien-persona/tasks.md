## 1. Persona medien

- [x] 1.1 `internal/permissions/personas_test.go`: Persona `{ID: "medien", Role: "standard", ClubFunctions: ["medien"]}` am Ende ergänzen; Kommentar-Pfad auf `openspec/specs/permissions/spec.md` umstellen. Verifikation: `go test ./internal/permissions/ -run TestPermissionMatrix_Backend` meldet für jede Expected-Map „Persona medien hat keinen Eintrag“ (Nachweis, dass der Drift-Check die neue Persona sieht).
- [x] 1.2 `internal/permissions/matrix_test.go`: `"medien"` in alle 17 Expected-Maps eintragen (Wert wie `spieler`, außer `exMatchReportPublisher`: `httpAllowed`); Drift-Fehlermeldung auf `openspec/specs/permissions/spec.md` umstellen. Verifikation: `go test ./internal/permissions/` grün, `grep -c '"medien":' matrix_test.go` = 17.
- [x] 1.3 `web/src/test/personas.ts`: Persona `medien` (Label „Medien“) ergänzen, Kommentar-Pfad korrigieren. `web/src/test/renderAsPersona.tsx`: `personaNavRoutes` um `/spielberichte/pruefen` für `medien`/`vorstand`/`admin` sowie die neun fehlenden Nav-Routen laut D3 vervollständigen, `/profil`-Regel auf „admin nur mit Mitglied/Kind“ angleichen; `personaCapabilities` um `import_games`, `manage_fees`, `create_root_folder`, `suppress_event_notification`, `bulk_regen_duties` ergänzen. Verifikation: `pnpm -C web test -- renderAsPersona` bzw. betroffene Suites grün.
- [x] 1.4 Frontend-Persona-Tests anpassen: `AppShell.permissions.test.tsx` (`NO_VERWALTUNG_IDS` + `medien`; Erwartung „Berichte prüfen“ für medien), `RoleRoute.permissions.test.tsx` (medien → `/spielberichte/pruefen` rendert, alles andere leitet um), Kommentar-Pfade in den neun `*.permissions.test.tsx`. Verifikation: `pnpm -C web test` grün; ein für `medien` roter Page-Test wird als Befund dokumentiert, nicht wegkonfiguriert.

## 2. Doku

- [x] 2.1 `docs/agent/03-go.md`: Vereinsfunktion `medien` in die Aufzählung und einen Satz zum Freigeber-Tier (`RequireClubFunction("medien","vorstand")`, Spielberichte) aufnehmen. Verifikation: Abschnitt „Rollen und Vereinsfunktionen“ nennt sieben Funktionen.

## 3. Spec-Konsistenz prüfen

- [x] 3.1 Endpunktlisten der Gate-Requirements im Delta gegen die Expected-Maps der Matrix abgleichen (Skript: Routen je Map aus `matrix_test.go` parsen, mit den Listen in `specs/permissions/spec.md` vergleichen; Abweichungen im Delta korrigieren). Verifikation: Skript meldet 0 Abweichungen für exTrainer, exVorstandTrainer, exVorstand, exVorstandKassierer, exMembersList, exSeasonsRead, exMatchReportPublisher.
- [x] 3.2 Frontend-Tabellen (RoleRoute, Nav, Inline-Gates) gegen `App.tsx`, `policy.NavFor` und die `hasCapability`-Aufrufe gegenlesen; `openspec validate --all` grün.

_3.1: Skript im Scratchpad, Ergebnis 0 Abweichungen für alle sieben Gates (die einzige Meldung war `GET /api/seasons/active`, das im Saisons-Requirement bewusst als Authenticated-Gegenstück genannt ist)._

## 4. Abschluss

- [ ] 4.1 `make test` (inkl. Tier- und Objektrechte-Matrix) und `pnpm -C web test` grün; `openspec validate --all` grün.
- [ ] 4.2 Change archivieren, Delta-Specs nach `openspec/specs/permissions`, `me-capabilities`, `nav-visibility` synchronisieren. Verifikation: Archiv-Verzeichnis vorhanden, `openspec validate --specs` grün.
