> **Voraussetzung:** Der Change `bewirtung-cap-global` muss abgeschlossen sein — dieser Change baut auf `game_template_items.rotation_enabled` (statt `rotation_max_per_team`) und auf dem Tab mit bereits zwei Bewirtungsfeldern auf.

## 1. Datenbank

- [x] 1.1 Migrationsnummer bestimmen: `ls internal/db/migrations/ | sort -V | tail -1`. **Ergebnis: `048`** — zwischen Design und Umsetzung ist `047_member_chat_visible` dazugekommen (Commit `b3d1086e`), die im Design genannte `047` ist also vergeben. Nie eine Nummer ≤ aktueller DB-Version vergeben (golang-migrate überspringt sie lautlos).
- [x] 1.2 `048_heimspieltag_ausrichter.up.sql`: Tabelle `ausrichter` (`id`, `name TEXT NOT NULL UNIQUE`, `aktiv INTEGER NOT NULL DEFAULT 1`, `is_default INTEGER NOT NULL DEFAULT 0`, `sort_order INTEGER NOT NULL DEFAULT 0`, `created_at`) + `CREATE UNIQUE INDEX idx_ausrichter_default ON ausrichter(is_default) WHERE is_default = 1;`
- [x] 1.3 `047_*.up.sql`: Tabelle `spieltag_ausrichter` (`date DATE NOT NULL`, `season_id INTEGER NOT NULL REFERENCES seasons(id) ON DELETE CASCADE`, `ausrichter_id INTEGER REFERENCES ausrichter(id) ON DELETE SET NULL`, `updated_at`, `updated_by INTEGER REFERENCES users(id)`, `PRIMARY KEY (date, season_id)`)
- [x] 1.4 `047_*.up.sql`: `ALTER TABLE game_template_items ADD COLUMN ausrichter_id INTEGER REFERENCES ausrichter(id) ON DELETE RESTRICT;` — bewusst `RESTRICT` statt `SET NULL`, damit es keinen stillen Pfad gibt, der eine gebundene Zeile auf „gilt immer" hebt (design.md Decision 6).
- [x] 1.5 `047_*.up.sql`: Seed genau einer Default-Zeile idempotent (`INSERT OR IGNORE … is_default=1`). Ohne diesen Seed ist die Auflösung aus Decision 2 nicht total.
- [x] 1.6 `047_*.down.sql`: Spalte und beide Tabellen entfernen (Reihenfolge: erst `game_template_items.ausrichter_id`, dann `spieltag_ausrichter`, dann `ausrichter`).
- [x] 1.7 Up/Down gegen eine **isolierte Kopie** verifizieren, nicht gegen die echte `teamwerk.db`. Bekannte Repo-Einschränkung: `make migrate-down` ist ein No-Op (`db.Migrate()` ruft nur `m.Up()`), das Down also direkt per `sqlite3 < …down.sql` prüfen.
- [x] 1.8 `internal/db/migrations_test.go` um die neuen Objekte ergänzen (Schema-Assertions wie bei den Vorgänger-Migrationen).

## 2. Backend: Ausrichter-Liste und Auflösung (Foundation)

- [x] 2.1 `internal/settings/ausrichter.go` anlegen: Typ `Ausrichter`, `ListAusrichter`, `GetAusrichter`, `CreateAusrichter`, `UpdateAusrichter`, `DeleteAusrichter`, `AusrichterUsage`. Package-Wahl ist durch den Architektur-Test erzwungen — `settings` ist Foundation und wird von `internal/games` bereits importiert; ein Domain-Package `internal/ausrichter` wäre ein Domain→Domain-Import (design.md Decision 3).
- [x] 2.2 `ResolveAusrichterForDay(ctx, RowQuerier, date string, seasonID int) (int, error)` — nimmt das schmale `RowQuerier`-Interface (wie `GetBewirtungVerhaeltnis`), damit die Regen-Engine innerhalb ihrer `tx` liest. Kein `Store`/Cache (kein Hot-Path). Liefert bei fehlender Zeile **und** bei `ausrichter_id IS NULL` den Default.
- [x] 2.3 Default-Wechsel als eine Transaktion: bisherigen Default auf `0`, neuen auf `1`. Der Partial-Unique-Index ist die eigentliche Garantie — der Handler-Code darf sich nicht allein darauf verlassen, dass er richtig rechnet.
- [x] 2.4 `DeleteAusrichter`: `409 default_ausrichter_undeletable` beim Default; sonst in einer Transaktion `spieltag_ausrichter.ausrichter_id` auf `NULL` und gebundene `game_template_items` **löschen** (design.md Decision 6).
- [x] 2.5 Tests `internal/settings/ausrichter_test.go`: Auflösung total (fehlende Zeile / `NULL` / expliziter Wert), Default-Invariante über alle Schreibpfade, Saison-Trennung bei gleichem Datum, Lösch-Kaskade (Spieltage auf `NULL`, Items weg), Default nicht löschbar.

## 3. Backend: Routen der Ausrichter-Liste

- [x] 3.1 `internal/settings/handler.go`: `ListAusrichter` (`GET /api/ausrichter`), `CreateAusrichter`, `UpdateAusrichter`, `DeleteAusrichter`, `AusrichterUsage` (`GET /api/ausrichter/{id}/usage`). Jede Mutation ruft `h.hub.Broadcast("settings-changed")`.
- [x] 3.2 `internal/app/router.go`: `GET /api/ausrichter` und `GET /api/ausrichter/{id}/usage` in den Authenticated-Block (Kalender und Wizard brauchen die Liste), die drei Mutationen in den bestehenden `vorstand`-Block neben `PUT /api/settings/bewirtung`.
- [x] 3.3 Route-Tests `internal/settings/ausrichter_handler_test.go`: Happy-Path je Route, `401` unauthentifiziert, `403` ohne `vorstand`, `409` bei Namensdublette, `409` beim Default-Löschen — jeweils mit Assertion, dass nichts geschrieben wurde.

## 4. Backend: Vorlagen-Item-Bindung

- [x] 4.1 `ausrichter_id` in `templateItemRow` (Scan als `sql.NullInt64`) und im DTO `templateItem` (JSON `*int`) ergänzen — und in **allen drei** SELECTs auf `game_template_items`: `loadTemplateItems` (non-tx, `handler.go`, für `PreviewSlots`), `scanTemplateItems` (`handler.go`, Admin-CRUD) und `loadTemplateItemsTx` (`regen.go`). Im `INSERT` mitschreiben.
- [x] 4.2 Validierung vor dem ersten Schreibvorgang: gesetztes `ausrichter_id` auf einer Vorlage mit `template_type != 'heim'` → `400 ausrichter_requires_heim_template`; unbekannter `ausrichter_id` → `400`. Beide Prüfungen laufen **vor** allen Inserts, damit ein Fehler keine Teil-Persistenz hinterlässt (gleiches Muster wie bei `PUT /api/settings/bewirtung`).
- [x] 4.3 Tests: Happy-Path (Bindung wird gespeichert und zurückgeliefert), `400` auf Auswärts-/generischer Vorlage, `400` bei unbekanntem Ausrichter, und dass ein Item ohne Bindung unverändert `NULL` bleibt.

## 5. Backend: Gate im Auto-Regen

- [x] 5.1 `internal/games/regen.go`, `loadTemplateItemsTx` und der zweite Item-Query in `regen.go`: `gti.ausrichter_id` mitlesen.
- [x] 5.2 `regenSingleDay`: nach `loadDayGames` einmal `settings.ResolveAusrichterForDay(ctx, tx, date, seasonID)` aufrufen und das Ergebnis an `buildRotationPlan` und `regenGameItems` durchreichen.
- [x] 5.3 `buildRotationPlan`: in der Schleife, die rotations-aktive Items sammelt, zusätzlich das Ausrichter-Gate anwenden — **vor** der Bedarfsrechnung. Ohne das verbraucht die Warteschlange Positionen für Slots, die danach verworfen werden, und der ausgewiesene Bedarf ist falsch (design.md Decision 4).
- [x] 5.4 `regenGameItems`: Item überspringen, wenn `ausrichter_id` gesetzt ist und nicht dem Tages-Ausrichter entspricht. Gate nur bei `event_type='heim'` (Sicherung — Vorlagen anderen Typs dürfen das Feld ohnehin nicht tragen).
- [x] 5.5 Prüfen, ob das Gate Auswirkungen auf `snapshotCustomSlots`/`restoreAssignments` hat: ein ausgegatetes Item erzeugt keinen neuen Slot, die Zusage läuft also regulär in `buildNotificationIntents` als „entfernt". Kein Sonderfall nötig — im Test festhalten, nicht nur annehmen.
- [x] 5.6 Tests `internal/games/regen_test.go`: ausgegatetes Rotations-Item → Bedarf `0` und **kein** `team_id=NULL`-Slot; teilweise gegateter Tag → Bedarf zählt nur passende Spiele; Item ohne Bindung → Slots bitgleich zum Bestandsverhalten (Charakterisierung); Auswärts/generisch unberührt; Ausrichter wird einmal je Tag aufgelöst.

## 6. Backend: Preview/Apply für den Tages-Ausrichter

- [ ] 6.1 `internal/games/ausrichter_handler.go` anlegen: `GET /api/game-days/{date}/host` (aufgelöster Wert + `is_explicit`), `POST /api/game-days/host/preview`, `POST /api/game-days/host/apply`. Muss in `games` liegen, weil es den unexportierten `runAutoRegen` braucht — dieselbe Begründung wie bei `h4aimport_handler.go`.
- [ ] 6.2 Gemeinsame Funktion für beide Endpoints: Tageswert schreiben → `runAutoRegen` für den Tag → Bilanz aus `RegenSummary` + Vor-/Nach-Zählung von `duty_slots`/`duty_assignments` → `Rollback` (preview) oder `Commit` + `Broadcast` (apply). Kein zweiter Codepfad.
- [ ] 6.3 Validierung: unbekannter oder inaktiver `ausrichter_id` → `400`; Capability `manage_games` → `403` sonst.
- [ ] 6.4 Routen in `internal/app/router.go` eintragen. `preview` schreibt nichts und braucht daher einen Eintrag in der `broadcastAllowlist` (`internal/arch/broadcast_test.go`) **mit Begründung** — sonst schlägt das Broadcast-Gate fehl.
- [ ] 6.5 Tests: Vorschau schreibfrei (DB-Zustand vor/nach identisch, inkl. `spieltag_ausrichter`), Apply setzt Wert + regeneriert + broadcastet, Vorschau und Apply liefern dieselbe Bilanz, `400` bei unbekanntem Ausrichter, `403` ohne `manage_games`.

## 7. Backend: Massenlauf

- [ ] 7.1 `internal/games/bulkregen_handler.go`: `HostOverrides []bulkRegenHostOverride` (`{date, ausrichter_id}`) im Request, `Days []bulkRegenDay` (`{date, stored_ausrichter_id, effective_ausrichter_id, is_explicit}`) in der Response.
- [ ] 7.2 Overrides vor dem Lauf validieren (unbekannter Ausrichter → `400`, nichts geschrieben) und im Lauf wie explizit gesetzte Tageswerte behandeln; ohne Angabe gilt der gespeicherte Wert bzw. der Default.
- [ ] 7.3 Tests: Preview mit `host_override` weist die Wirkung aus und schreibt nicht; Apply persistiert den Tageswert und die Dienste entsprechen der Vorschau; ohne `host_overrides` bleibt `spieltag_ausrichter` unverändert; `400` bei unbekanntem Ausrichter.

## 8. Frontend: Einstellungen

- [ ] 8.1 `web/src/pages/AdminSettingsPage.tsx`: Tab-Label `Bewirtung` → `Heimspieltage`. Der alte Query-Parameter `?tab=bewirtung` muss weiterhin auf diesem Tab landen (Alias auf die Tab-ID), damit bestehende Links nicht ins Leere laufen.
- [ ] 8.2 Bestehende Felder in eine Kachel **„Bewirtung"** fassen (Kachel-Klassen aus `docs/agent/05-frontend.md`), Inhalt unverändert.
- [ ] 8.3 Neue Kachel **„Ausrichter"**: CRUD-Liste nach dem Muster von `StammvereineTab` (anlegen, umbenennen, deaktivieren, löschen, sortieren) plus Default-Markierung. `useLiveUpdates` auf `settings-changed`.
- [ ] 8.4 Löschen führt über `GET /api/ausrichter/{id}/usage` und zeigt die betroffenen Spieltage **und Vorlagen-Zeilen** an, bevor bestätigt wird — die Vorlagen-Zeilen verschwinden mit, das darf nicht überraschen.
- [ ] 8.5 Nur `brand-*`-Tokens, `lucide-react`-Icons, Icon-only-Buttons mit `aria-label`. Mobile: Card-Layout statt Tabelle, Touch-Targets `py-2.5`.
- [ ] 8.6 Vitest: Tab-Rendering mit beiden Kacheln, Alias des alten Tab-Parameters, CRUD-Requests, Default-Wechsel, Lösch-Dialog listet Vorlagen-Zeilen.

## 9. Frontend: Vorlagen-Editor

- [ ] 9.1 `web/src/components/DutyTemplateItemFields.tsx`: Ausrichter-Auswahl je Item, nur bei `template_type='heim'`. Leerwert = „gilt immer" und muss im UI als solcher benannt sein, nicht als leeres Feld.
- [ ] 9.2 `AdminDutyTemplatesPage.tsx` und `AdminDutyTemplateDetailPage.tsx`: Feld in Typ und Payload aufnehmen.
- [ ] 9.3 Je Zeile im Klartext anzeigen, wann sie greift (z. B. „nur wenn TV Ötlingen ausrichtet") — ein Konfigurationsfehler fällt sonst erst am erzeugten Dienstplan auf (design.md Risks).
- [ ] 9.4 Vitest: Auswahl erscheint nur bei Heim-Vorlagen, Wert landet im Request, Leerwert sendet `null`.

## 10. Frontend: Kalender

- [ ] 10.1 Ausrichter-Feld im **Termin-Detail-Modal** (`EventInfoModal` in `KalenderPage.tsx`) bei Heim-Terminen — es gibt **keine Tagesansicht**, siehe design.md Decision 10. Sichtbar für alle, änderbar nur mit `manage_games`, gekennzeichnet ob explizit gesetzt oder vom Default geerbt. **Keine** Darstellung in der Monatsübersicht.
- [ ] 10.2 Vorschau-Modal mit der Bilanz (`created`/`deleted`/`assignments_lost`) vor dem Anwenden — gespeist aus `POST /api/game-days/host/preview`.
- [ ] 10.3 Termin-Wizard: Ausrichter-Feld bei Heim-Terminen, vorbelegt mit dem geltenden Wert, **erkennbar tagesbezogen beschriftet** („Ausrichter am 14.09. — gilt für alle Termine dieses Tages"). Weicht der Wert ab, läuft das Speichern über dasselbe Vorschau-Modal (design.md Decision 9).
- [ ] 10.4 Massenlauf-Dialog (`web/src/components/DutyBulkRegenModal.tsx` — eigene Komponente, **nicht** in `KalenderPage.tsx`): Die Zeilen sind heute **flach je Termin**; für die Ausrichter-Spalte je Tag muss `preview.rows` clientseitig nach `row.date` gruppiert und eine Tages-Zwischenebene ins Rendering eingezogen werden. Kennzeichnung geerbt/gesetzt; Änderungen gehen als `host_overrides` mit. Kein zweiter Bestätigungsdialog.
- [ ] 10.5 `useLiveUpdates` für die betroffenen Events, damit fremde Änderungen die Ansicht aktualisieren.
- [ ] 10.6 Vitest: Wizard-Feld ist tagesbezogen beschriftet und vorbelegt; abweichende Wahl öffnet die Vorschau; ohne `manage_games` ist der Picker nur lesend; Massenlauf sendet `host_overrides`.

## 11. Abschluss

- [ ] 11.0 **Permission-Matrix nachziehen** (`internal/permissions/matrix_test.go`): alle sechs neuen Routen ins `matrix`-Array mit erwarteter Berechtigung eintragen (`GET /api/ausrichter`, `GET /api/ausrichter/{id}/usage`, `POST /api/ausrichter`, `PUT|DELETE /api/ausrichter/{id}`, `GET /api/game-days/{date}/host`, `POST /api/game-days/host/preview|apply`). `TestPermissionMatrix_Backend` läuft per `chi.Walk` über den echten Router und failt bei **jeder** nicht eingetragenen Route — ein zweites, vom Broadcast-Gate unabhängiges Gate, das im ursprünglichen Design fehlte.
- [ ] 11.1 `make test` (inkl. Architektur-, Broadcast- und Permission-Matrix-Gate), `make lint`, `pnpm -C web build/test/lint`.
- [ ] 11.2 `/verify-change` durchlaufen (Route→Tests, Mutation→`Broadcast`/`useLiveUpdates`, brand-Tokens, lucide-Icons, Migrationsnummer, `openspec validate`).
- [ ] 11.3 Gotcha in `docs/agent/06-gotchas.md` ergänzen: Ausrichter als Tages-Eigenschaft, totale Auflösung über den Default, Gate an **zwei** Stellen im Regen (inkl. Begründung, warum `buildRotationPlan` mitgatet werden muss), und die asymmetrische Lösch-Semantik (Spieltage `SET NULL`, Vorlagen-Zeilen mitlöschen).
- [ ] 11.4 Ein Commit je Task-Gruppe, Conventional Commits mit Scope `games`/`settings`/`db`.
- [ ] 11.5 Change archivieren.
