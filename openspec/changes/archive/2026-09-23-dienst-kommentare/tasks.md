## 1. Datenbank

- [x] 1.1 Migration `internal/db/migrations/062_duty_assignment_comments.up.sql`:
  Tabelle `duty_assignment_comments` (`id`, `assignment_id INTEGER NOT NULL
  UNIQUE REFERENCES duty_assignments(id) ON DELETE CASCADE`, `body TEXT NOT
  NULL`, `created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP`,
  `updated_at DATETIME`)
- [x] 1.2 `062_duty_assignment_comments.down.sql`: Tabelle droppen

## 2. Backend — Endpunkte (`internal/duties`)

- [x] 2.1 Ownership-Helfer extrahieren/wiederverwenden: „darf `claims.UserID`
  für die Zuteilung `assignmentId` handeln" (Zuteilung gehört ihm selbst,
  oder er ist laut `family_links` Elternteil des zugeteilten
  `can_login=0`-Kindes) — dieselbe Regel wie in `Handler.Claim`, nicht neu
  erfinden
- [x] 2.2 `PUT /api/duty-assignments/{id}/comment`: Upsert (SQLite
  `INSERT … ON CONFLICT(assignment_id) DO UPDATE`), Body-Validierung (leer/
  nur Whitespace → 400, > 280 Byte UTF-8 → 400), 404 wenn Zuteilung nicht
  existiert, 403 bei fremder Zuteilung, Broadcast `"duties"` bei Erfolg
- [x] 2.3 `DELETE /api/duty-assignments/{id}/comment`: löscht eigenen
  Kommentar, gleiche Ownership-Prüfung, Broadcast `"duties"` bei Erfolg
- [x] 2.4 `GET /api/duty-slots/{id}/comments`: liefert alle Kommentare eines
  Slots (Name der Person, Text, `created_at`), 404 wenn Slot nicht existiert,
  kein Ownership-Gate (universelles Leserecht)
- [x] 2.5 `comment_count` im Duty-Board-Query (`GET /api/duty-board`)
  ergänzt — Aggregation über `duty_assignments` → `duty_assignment_comments`
  pro Slot, kein Volltext im Response
- [x] 2.6 Neue Routen in `internal/app/router.go` (`BuildRouter`) eingetragen,
  passendes Auth-Tier (Authenticated)

## 3. Backend — Tests

- [x] 3.1 `PUT`: Happy Path (200, Kommentar in DB) + 401 (kein Token) + 403
  (fremde Zuteilung) + 404 (unbekannte `assignmentId`) + 400 (leerer Body) +
  400 (> 280 Byte)
- [x] 3.2 `PUT`: zweiter Aufruf überschreibt bestehenden Kommentar statt
  einen zweiten anzulegen (Upsert-Test)
- [x] 3.3 `PUT`: Elternteil-Account darf für sein Proxy-Kind kommentieren
- [x] 3.4 `DELETE`: Happy Path (204, Kommentar weg) + 401 + 403 + 404
- [x] 3.5 `GET /api/duty-slots/{id}/comments`: Happy Path (liefert Kommentare
  mehrerer Personen) + 404 (unbekannter Slot) — Test explizit mit einem
  Leser, der selbst nicht eingetragen ist (universelles Leserecht)
- [x] 3.6 Cascade-Test: `DELETE /api/duty-board/{slotId}/claim` (Austragen)
  entfernt den Kommentar der ausgetragenen Person automatisch
- [x] 3.7 Cascade-Test: Slot-Löschung über das Kalender-Modal
  (`DELETE /api/duty-slots/{id}`) entfernt die Kommentare aller betroffenen
  Zuteilungen
- [x] 3.8 `comment_count`-Test im Board-Response (0 ohne Kommentare, korrekte
  Zahl bei mehreren kommentierten Zuteilungen desselben Slots)

## 4. Frontend (`web/src/components/DutySlotList.tsx`)

- [x] 4.1 `BoardSlot`-Interface um `comment_count: number` erweitert
  (zusätzlich `my_assignment_id?: number` fürs Schreib-Gate, gespeist aus
  einem neuen `COALESCE(da.id, 0)` im Board-Query, sonst könnte das Frontend
  die eigene `assignmentId` für PUT/DELETE nicht auflösen)
- [x] 4.2 Kommentar-Icon (`MessageCircle` aus lucide-react) neben dem
  bestehenden `BookOpen`-Icon, nur gerendert wenn `comment_count > 0`,
  Klick öffnet das Kommentar-Modal im Lesemodus
- [x] 4.3 Kommentar-Modal: Lesemodus — lädt `GET /api/duty-slots/{id}/comments`
  on-demand beim Öffnen, zeigt Liste (Name + Text), analog zu Struktur/Stil
  der bestehenden Dialoge (`noInstructionOpen`) in derselben Datei
  (`useDialogA11y`, `useEscapeKey`)
- [x] 4.4 Kommentar-Modal: Schreibmodus — Textarea vorbefüllt mit eigenem
  Kommentar (falls vorhanden, per `user_id` im `GET`-Response erkannt),
  „Speichern" ruft `PUT`, „Löschen" (falls Kommentar existiert) ruft
  `DELETE`, danach `onReload()`
- [x] 4.5 Neues ⋮-Aktionsmenü auch auf Desktop (bisher `sm:hidden`):
  Bearbeiten (`canEdit && onEdit`), Anleitung (`has_instruction`,
  weiterhin zusätzlich zum Inline-Icon), Kommentieren (nur wenn
  `claimed_by_me && my_assignment_id`) — Eintragen/Austragen bleiben
  eigene sichtbare Buttons; Mobile-Menü unverändert inkl. Eintragen/
  Austragen (Spaltenbreite `w-11` ist auf reinen Icon-Trigger ausgelegt,
  siehe `colgroup`-Kommentar — Abweichung von der ursprünglichen
  Formulierung, siehe Notiz unten)
- [x] 4.6 Bestehenden Desktop-„Bearbeiten"-Button entfernt (zieht ins
  gemeinsame Menü, keine Dopplung)

## 5. Frontend — Tests

- [x] 5.1 Test: Kommentar-Icon erscheint nur bei `comment_count > 0`
  (inkl. Singular-/Pluralform des aria-label)
- [x] 5.2 Test: Modal lädt Kommentare erst beim Öffnen (on-demand), nicht
  beim initialen Board-Render
- [x] 5.3 Test: „Kommentieren"-Eintrag im ⋮-Menü nur sichtbar, wenn
  `claimed_by_me` (+ Speichern-Flow mit `PUT` und `onReload`)
- [x] 5.4 Test: Desktop-⋮-Menü rendert Bearbeiten/Anleitung, Eintragen bleibt
  eigener sichtbarer Button außerhalb des Menüs

## 6. Verifikation

- [x] 6.1 `go build ./...`, `go vet ./...`, `go test ./...` (48 Pakete),
  `golangci-lint run` grün. Objektrechte-Matrix (`internal/permissions/
  object_matrix_test.go`) und Tier-Matrix (`matrix_test.go`) um die drei
  neuen Routen ergänzt (GET → `openByDesign` analog zu
  `.../assignments`, PUT/DELETE → echtes Fixture mit erwartetem 403).
  Frontend: `tsc --noEmit`, `pnpm lint` (0 Fehler), `pnpm test` (1181/1181),
  `pnpm build` grün. Bestehende `SpieltagDetailModal.deleteSlot.test.tsx`/
  `.duration.test.tsx` angepasst — „Bearbeiten" steckt jetzt auch dort im
  ⋮-Menü statt in einem direkt sichtbaren Button.
- [x] 6.2 `/verify-change` (Route→Tests, Mutation→Broadcast, brand-Tokens,
  lucide-Icons, Migrationsnummer, `openspec validate`)
