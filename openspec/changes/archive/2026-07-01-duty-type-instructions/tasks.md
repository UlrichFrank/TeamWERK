## 1. Migration

- [x] 1.1 `internal/db/migrations/015_duty_type_instruction.up.sql` anlegen: drei `ALTER TABLE duty_types ADD COLUMN`: `instruction_md TEXT NOT NULL DEFAULT ''`, `instruction_updated_at TEXT`, `instruction_updated_by INTEGER REFERENCES users(id) ON DELETE SET NULL`
- [x] 1.2 `internal/db/migrations/015_duty_type_instruction.down.sql` mit passendem `ALTER TABLE duty_types DROP COLUMN` (SQLite ≥ 3.35, ok in `modernc.org/sqlite`)
- [x] 1.3 Lokal `make migrate-up` + `make migrate-down` + `make migrate-up` grün

## 2. Backend – Read-Pfade erweitern

- [x] 2.1 `internal/duties/handler.go::ListTypes`: `instruction_md`, `instruction_updated_at`, `instruction_updated_by` in `SELECT` und Response-Struct ergänzen (nullable via `sql.NullString`/`sql.NullInt64`)
- [x] 2.2 `internal/duties/handler.go::Board`: LEFT-JOIN nutzt bereits `duty_types`; zusätzlich `dt.instruction_md != '' AS has_instruction` selektieren und im Response-Struct als `has_instruction: bool` sowie `duty_type_id: int` mitgeben
- [x] 2.3 Fixture in `internal/testutil/` prüfen: sofern `CreateDutyType` das neue Feld ignoriert, ergänzen (optionaler Parameter oder Nachbearbeitung im Test)

## 3. Backend – Write-Route

- [x] 3.1 Handler-Methode `func (h *Handler) SetInstruction(w, r)` in `internal/duties/handler.go`: `PUT /api/duty-types/{id}/instruction`, Body `{"markdown": "..."}`; Length-Limit 64 KB, sonst 400
- [x] 3.2 Existenz-Check mit `SELECT 1 FROM duty_types WHERE id = ?` → 404 bei `sql.ErrNoRows`
- [x] 3.3 `UPDATE duty_types SET instruction_md=?, instruction_updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now'), instruction_updated_by=? WHERE id=?`
- [x] 3.4 `h.hub.Broadcast("duties")` nach erfolgreichem Update
- [x] 3.5 Route in `internal/app/router.go` unter dem Vorstand-Tier registrieren (`r.Put("/api/duty-types/{id}/instruction", h.Duties.SetInstruction)`)
- [x] 3.6 Antwort `200 OK` mit `{"instruction_updated_at": "..."}` (Frontend nutzt das für die Header-Anzeige „Zuletzt geändert am …")

## 4. Backend-Tests (`internal/duties/handler_test.go`)

- [x] 4.1 `TestPutInstruction_HappyPath`: Vorstand-User setzt Anleitung → 200, DB-Zeile enthält Markdown + `updated_at`, Broadcast-Kanal empfängt `duties`
- [x] 4.2 `TestPutInstruction_Unauthenticated`: kein JWT → 401, keine DB-Änderung
- [x] 4.3 `TestPutInstruction_ForbiddenForStandard`: User ohne `vorstand`/`admin` → 403
- [x] 4.4 `TestPutInstruction_NotFound`: unbekannte `id` → 404
- [x] 4.5 `TestPutInstruction_MissingBody`: Body ohne `markdown`-Feld → 400
- [x] 4.6 `TestPutInstruction_TooLarge`: Body > 64 KB → 400
- [x] 4.7 `TestListTypes_IncludesInstructionFields`: `GET /api/duty-types` liefert `instruction_md`, `instruction_updated_at` in JSON
- [x] 4.8 `TestBoard_ExposesHasInstruction`: Slot dessen Typ Anleitung hat → `has_instruction=true`; leer → `false`; `duty_type_id` immer gesetzt

## 5. Frontend-Dependencies + Renderer-Helper

- [x] 5.1 `pnpm -C web add react-markdown rehype-sanitize` (SemVer aktuell)
- [x] 5.2 `web/src/lib/dutyInstructionTemplate.ts`: Export `DUTY_INSTRUCTION_TEMPLATE: string` (Text aus `design.md` §5)
- [x] 5.3 `web/src/components/MarkdownRenderer.tsx`: dünner Wrapper um `react-markdown` mit `rehypePlugins={[rehypeSanitize]}`, Klasse `prose prose-sm max-w-none text-brand-text`; keine externen Bild-Preloader
- [x] 5.4 Vitest `MarkdownRenderer.test.tsx`: (a) rendert `<h2>`, (b) verwirft `<script>alert(1)</script>`, (c) rendert `<img src="/dokumente/datei/123">` mit korrektem `src`

## 6. Frontend – Editor auf `AdminDutyTypesPage`

- [x] 6.1 Neue Aktion **„Anleitung"** pro Zeile (Desktop: Button rechts; Mobile: in `ActionMenu`); Icon `<BookOpen>` (lucide)
- [x] 6.2 Modal-Komponente `DutyInstructionEditorModal` (`bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6`): Header „Anleitung: {Dienst-Name}", Textarea (Klassen wie Standard-Input, `min-h-64`), Live-Preview darunter via `MarkdownRenderer`
- [x] 6.3 Vorbelegung: bei `instruction_md === ''` Textarea auf `DUTY_INSTRUCTION_TEMPLATE` setzen; Speichern-Button disabled bis der Text tatsächlich verändert oder ergänzt wurde (`hasChanged`-State)
- [x] 6.4 Hinweis-Zeile über der Textarea: „Bilder aus dem Ordner **Anleitungen** unter /dokumente verlinken: `![Alt](/dokumente/datei/DATEI_ID)`" (Alert Info)
- [x] 6.5 Speichern → `api.put(`/duty-types/${id}/instruction`, {markdown})`, Modal schließen, `onReload()`; Fehler-Toast bei 400/403
- [x] 6.6 Vitest: `prefills example on empty instruction` (Modal öffnet mit leerer Instruction, Textarea = Template) + `save disabled until text changed`

## 7. Frontend – Anleitung anzeigen

- [x] 7.1 Neue Seite `web/src/pages/DutyInstructionPage.tsx`: Lädt `GET /api/duty-types` (bestehender Endpoint), findet Eintrag per `useParams<{typeId}>()`, rendert Header (Dienst-Name, „Zuletzt geändert am …") und `MarkdownRenderer` mit `instruction_md`
- [x] 7.2 Placeholder-Rendering wenn `instruction_md === ''`: „Für diesen Dienst gibt es noch keine Anleitung."
- [x] 7.3 `useLiveUpdates(event => { if (event === 'duties') reload() })`
- [x] 7.4 Route in `web/src/App.tsx` unter `AppShell`-Outlet: `<Route path="/dienste/anleitung/:typeId" element={<DutyInstructionPage />} />` (via `lazy(() => import(...))` für Route-Split)
- [x] 7.5 Vitest: rendert Markdown-Preview korrekt und leere Anleitung zeigt Placeholder

## 8. Frontend – Slot-Link in `DutySlotList`

- [x] 8.1 `BoardSlot`-Interface um `duty_type_id: number` und `has_instruction: boolean` erweitern (Prop-Fluss in `DutyPage.tsx` durchreichen)
- [x] 8.2 Neben `duty_type`-Text im Slot-Item ein Icon-Link `<Link to={`/dienste/anleitung/${slot.duty_type_id}`} aria-label="Anleitung ansehen">` mit `<BookOpen className="w-4 h-4">` genau dann rendern, wenn `slot.has_instruction === true`
- [x] 8.3 Klick auf Icon darf nicht das Slot-Claim/Unclaim-Verhalten triggern (`e.stopPropagation()` oder Link außerhalb des Claim-Bereichs)
- [x] 8.4 Vitest: Slot mit `has_instruction=true` rendert Link mit korrektem `href`; ohne Anleitung kein Link

## 9. Doku, CHANGELOG, Commit, Deploy

- [x] 9.1 `web/public/CHANGELOG.md`: Eintrag `[feat] duties: Anleitung pro Dienst-Typ (Markdown, mit Bild-Referenz aus /dokumente)` unter heutigem Datum
- [x] 9.2 Kurz-Notiz in `docs/agent/06-gotchas.md`, dass Anleitungs-Bilder im Ordner „Anleitungen" (everyone/read) liegen sollen — sonst Broken-Image
- [ ] 9.3 Commits gemäß OpenSpec-Regel (ein Commit pro Task-Sektion): `feat(duties): Anleitung an duty_types + Migration 015`, `feat(duties): Anleitung-Editor auf AdminDutyTypesPage`, `feat(duties): Anleitung-Viewer + Slot-Link`
- [x] 9.4 `/verify-change` grün (Build/Test/Lint + Projekt-Invarianten inkl. Broadcast + brand-Tokens)
- [x] 9.5 `openspec validate duty-type-instructions --strict`
- [ ] 9.6 Push, `make deploy`
- [ ] 9.7 OpenSpec-Change via `openspec archive duty-type-instructions` überführen
