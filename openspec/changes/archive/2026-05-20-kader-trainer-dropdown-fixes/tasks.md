## 1. Backend: DB-Migrationen

- [x] 1.1 `internal/db/migrations/015_member_club_function.up.sql`: `ALTER TABLE members ADD COLUMN club_function TEXT CHECK(club_function IN ('trainer','vorstand','vorstand_beisitzer'))`
- [x] 1.2 `internal/db/migrations/015_member_club_function.down.sql`: SQLite-Rebuild von `members` ohne `club_function` (CREATE TABLE members_new, INSERT SELECT, DROP, RENAME)
- [x] 1.3 `internal/db/migrations/016_kader_trainers.up.sql`: `CREATE TABLE kader_trainers (kader_id INTEGER NOT NULL REFERENCES kader(id) ON DELETE CASCADE, member_id INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE, PRIMARY KEY (kader_id, member_id))`
- [x] 1.4 `internal/db/migrations/016_kader_trainers.down.sql`: `DROP TABLE IF EXISTS kader_trainers`

## 2. Backend: Members-API erweitern

- [x] 2.1 In `internal/members/handler.go` `club_function` in `Member`-Struct ergänzen (`ClubFunction *string`)
- [x] 2.2 `ListMembers` und `GetMember` SQL-Queries: `club_function` in SELECT und Scan ergänzen
- [x] 2.3 `CreateMember` und `UpdateMember`: `club_function` aus Request lesen und in INSERT/UPDATE schreiben
- [x] 2.4 `ListMembers`: Query-Parameter `club_function` auslesen und WHERE-Clause ergänzen wenn gesetzt (`?club_function=trainer`)

## 3. Backend: Kader-API erweitern

- [x] 3.1 In `internal/kader/handler.go` Typ `trainerRow struct { ID int; Name string }` anlegen
- [x] 3.2 `kaderDetail` um `Trainers []trainerRow` (json: `trainers`) erweitern
- [x] 3.3 Hilfsfunktion `loadTrainers(ctx, kaderID) ([]trainerRow, error)`: JOIN `kader_trainers` → `members` (first_name || ' ' || last_name)
- [x] 3.4 `ListKader` und `GetKader`: `loadTrainers` aufrufen und befüllen (leeres Slice statt nil)
- [x] 3.5 `UpdateKader` Request-Struct um `TrainersAdd []int` und `TrainersRemove []int` erweitern
- [x] 3.6 `UpdateKader`: INSERT OR IGNORE / DELETE auf `kader_trainers` in der bestehenden Transaktion ergänzen

## 4. Frontend: Vereinsfunktion in MemberDetailPage

- [x] 4.1 In `MemberDetailPage.tsx` `club_function` zum Member-Interface und Edit-State hinzufügen
- [x] 4.2 Im Bearbeiten-Formular ein Select "Vereinsfunktion" ergänzen mit Optionen: `– keine –`, `Trainer`, `Vorstand`, `Vorstands-Beisitzer`
- [x] 4.3 `PUT /api/members/{id}` sendet `club_function` mit (null wenn keine gewählt)

## 5. Frontend: Dropdown-Bugs beheben

- [x] 5.1 In `AdminKaderPage.tsx` `overflow-hidden` vom äußeren Kader-Karten-`div` entfernen
- [x] 5.2 Dem Jahrgangs-`<select>` ein `key`-Prop geben: `key={k.dedicated_birth_year ?? 'empty'}`

## 6. Frontend: Trainer-Zuweisung in Kader

- [x] 6.1 In `AdminKaderPage.tsx` Typ `TrainerMember { id: number; name: string }` und State `trainerPool: TrainerMember[]` hinzufügen; beim Mount `GET /api/members?club_function=trainer` laden
- [x] 6.2 `Kader`-Interface um `trainers: TrainerMember[]` erweitern
- [x] 6.3 In jeder Kader-Karte Trainer-Bereich hinzufügen: Label "Trainer:", zugewiesene Trainer als Chips (Name + ×)
- [x] 6.4 × auf Chip: `PUT /api/admin/kader/{id}` mit `{ trainers_remove: [member_id] }` → `loadAll()`
- [x] 6.5 `<select>` "Trainer hinzufügen…" darunter, gefiltert auf `trainerPool` minus bereits zugewiesene; Auswahl → `PUT /api/admin/kader/{id}` mit `{ trainers_add: [member_id] }` → Select zurücksetzen → `loadAll()`

## 7. Build & Deploy

- [x] 7.1 `make deploy` — Migrationen 015 + 016 laufen automatisch auf VPS; Funktion im Browser prüfen
