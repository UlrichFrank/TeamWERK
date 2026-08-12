## 1. Migration

- [x] 1.1 `internal/db/migrations/046_bewirtung_cap_global.up.sql`:
  - `INSERT OR IGNORE INTO system_settings (key, value)` für `bewirtung_max_per_team`, Startwert `COALESCE((SELECT MAX(rotation_max_per_team) FROM game_template_items), 1)`.
  - `ALTER TABLE game_template_items ADD COLUMN rotation_enabled INTEGER NOT NULL DEFAULT 0;`
  - `UPDATE game_template_items SET rotation_enabled = 1 WHERE rotation_max_per_team IS NOT NULL;`
  - `ALTER TABLE game_template_items DROP COLUMN rotation_max_per_team;`
- [x] 1.2 `.down.sql` in umgekehrter Reihenfolge (Spalte zurück, Wert aus dem Setting je aktiviertem Item, Setting-Row löschen).

## 2. Settings-Backend

- [x] 2.1 `internal/settings/bewirtung.go`: `keyBewirtungMaxPerTeam`, `ErrInvalidMaxPerTeam`, `GetBewirtungMaxPerTeam(ctx, RowQuerier) (int, error)` (fehlende Row → Default `1`, gleiches Fail-Safe-Muster wie das Verhältnis), `SetBewirtungMaxPerTeam(ctx, *sql.DB, int, updatedBy)` (validiert `> 0`, schreibt sonst nichts).
- [x] 2.2 `internal/settings/handler.go`: `GetBewirtung` liefert `{verhaeltnis, max_per_team}`. `SetBewirtung` nimmt beide Felder als Pointer entgegen (fehlend = unverändert), validiert **beide vor** dem ersten Schreibvorgang (Teil-Persistenz vermeiden), antwortet `400 invalid_verhaeltnis` / `400 invalid_max_per_team` und broadcastet `settings-changed` nur bei Erfolg.
- [x] 2.3 Tests `internal/settings/bewirtung_test.go`: Default ohne Row, Roundtrip, Ablehnung `0`/`-1` ohne Persistenz.
- [x] 2.4 Route-Tests: `GET` liefert beide Felder; `PUT` nur `max_per_team` lässt `verhaeltnis` unverändert; `PUT` mit gültigem `verhaeltnis` + ungültigem `max_per_team` persistiert **keines** von beiden (400); Nicht-Vorstand → 403.

## 3. Games-Backend

- [x] 3.1 `internal/games/handler.go`: `templateItemRow.RotationMaxPerTeam sql.NullInt64` → `RotationEnabled bool`; `templateItem.RotationMaxPerTeam *int` → `RotationEnabled bool` mit JSON-Tag `rotation_enabled`. SELECTs (`loadTemplateItems`, `loadTemplateItemsTx`, `scanTemplateItems`) und der INSERT in `UpdateTemplate` auf die neue Spalte umstellen.
- [x] 3.2 `UpdateTemplate`-Validierung: `rotation_requires_normal_behavior` hängt jetzt an `it.RotationEnabled`; die Prüfung `invalid_rotation_max_per_team` entfällt (der Wert lebt in den Settings und wird dort validiert).
- [x] 3.3 `internal/games/regen.go`: `rotationGroup.cap` entfernen; `buildRotationPlan` liest `settings.GetBewirtungMaxPerTeam(ctx, tx)` einmal — zusammen mit dem Verhältnis, in derselben `tx` — und nutzt ihn für alle Gruppen. Item-Filter auf `it.RotationEnabled` umstellen (`buildRotationPlan`, `regenGameItems`-Schleife, `rotationTypes`-Aufbau in `regenSingleDay`).
- [x] 3.4 Tests `internal/games/rotation_template_test.go`: `rotation_enabled` statt Cap im Request; `rotation_requires_normal_behavior`-Fall bleibt; entfallenen `invalid_rotation_max_per_team`-Test streichen.
- [x] 3.5 Tests `internal/games/rotation_regen_test.go`: Cap-Setup je Testfall über `system_settings` statt über das Item; neuer Test „ein Cap gilt für Items aus zwei verschiedenen Vorlagen".

## 4. Frontend

- [x] 4.1 `web/src/components/DutyTemplateItemFields.tsx`: `RotationCapField` → `RotationEnabledField` (Checkbox „Bewirtungsrotation", `accent-brand-yellow` wie `TeamScopeField`). Hinweistext: Cap wird unter Einstellungen → Bewirtung gepflegt; Voraussetzung „Normal (immer)" bleibt erhalten.
- [x] 4.2 `AdminDutyTemplatesPage.tsx` + `AdminDutyTemplateDetailPage.tsx` (3 Stellen): Item-Typ auf `rotation_enabled?: boolean`, Feldaufruf und `updateItem`-Callback anpassen.
- [x] 4.3 `AdminSettingsPage.tsx` → `BewirtungTab`: zweites Feld „Max. Kuchen pro Mannschaft" (Ganzzahl ≥ 1), aus `GET` befüllt, im gemeinsamen `PUT` mitgesendet; clientseitige Validierung analog zum Verhältnis. Einleitender Hinweis, dass beide Werte die automatische Dienst-Generierung bei **Heimspielen** steuern.
- [x] 4.4 Tests `web/src/pages/__tests__/AdminSettingsPage.bewirtung.test.tsx`: beide Felder werden geladen und gemeinsam gesendet; ungültiger Cap zeigt Fehler und sendet nicht.
- [x] 4.5 Tests `web/src/pages/__tests__/AdminDutyTemplateDetailPage.rotationCap.test.tsx` (→ `.rotationToggle.test.tsx`): Checkbox schaltet `rotation_enabled` im `PUT`-Body; Modal-Editor auf `/dienstplan-vorlagen` zeigt sie ebenfalls.

## 5. Doku

- [x] 5.1 `docs/agent/06-gotchas.md`, Gotcha „Bewirtungs-/Kuchendienst-Rotation": Cap-Herkunft (Settings statt Item), Schalter `rotation_enabled`, Wegfall des „erster gewinnt"-Sonderfalls.
- [x] 5.2 `make test` + `make lint` + `pnpm -C web test`/`build` grün; `openspec validate --all`.
- [ ] 5.3 Change archivieren.
