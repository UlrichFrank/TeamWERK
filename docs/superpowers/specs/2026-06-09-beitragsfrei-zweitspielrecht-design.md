# Design: Felder „Beitragsfrei" und „Zweitspielrecht"

**Datum:** 2026-06-09  
**Status:** Approved

## Überblick

Zwei neue boolesche Felder auf der `members`-Tabelle:

| Feld | Anzeige | Tab |
|------|---------|-----|
| `beitragsfrei` | Checkbox im Bankdaten-Block | Kontakt-Tab |
| `zweitspielrecht` | Checkbox nach Stammverein | Stammdaten-Tab |

Beide Felder sind standardmäßig `false`. `beitragsfrei` wird zusätzlich beim CSV-Import aus dem Status-Wert `"beitragsfrei"` abgeleitet.

---

## Datenbank

**Migration 035** (`035_beitragsfrei_zweitspielrecht.up.sql`):

```sql
ALTER TABLE members ADD COLUMN beitragsfrei    INTEGER NOT NULL DEFAULT 0;
ALTER TABLE members ADD COLUMN zweitspielrecht INTEGER NOT NULL DEFAULT 0;
```

Down-Migration entfernt die Spalten (via Table-Rebuild, da SQLite kein DROP COLUMN vor 3.35 unterstützt — modernc.org/sqlite unterstützt es, daher einfaches DROP COLUMN möglich).

---

## Backend (`internal/members/handler.go`)

### Member-Struct

```go
Beitragsfrei    bool `json:"beitragsfrei,omitempty"`
Zweitspielrecht bool `json:"zweitspielrecht,omitempty"`
```

### GET `/api/members/{id}`

- SELECT-Query um `m.beitragsfrei, m.zweitspielrecht` erweitern
- Scan: `var beitragsfrei, zweitspielrecht int64` → `base.Beitragsfrei = beitragsfrei == 1`

### PUT `/api/members/{id}`

- SET-Klausel um `beitragsfrei=?, zweitspielrecht=?` ergänzen
- Werte aus Request-Body dekodieren (wie bei `sepa_mandat`)

### CSV-Import

**normalizeStatus**: neuer Case vor dem Default:
```go
case "beitragsfrei":
    return "passiv"
```

**Beitragsfrei-Ableitung**: Vor dem `normalizeStatus`-Aufruf den Raw-CSV-Wert prüfen:
```go
csvBeitragsfrei := strings.ToLower(strings.TrimSpace(col(row, "Status"))) == "beitragsfrei"
```

- **INSERT** (neues Mitglied): `beitragsfrei`-Spalte im INSERT mit `csvBeitragsfrei`-Wert (als 0/1) übergeben
- **UPDATE** (Bestandsmitglied): Change-Erkennung wie bei `sepa_mandat` — nur wenn CSV-Wert gesetzt und verschieden vom DB-Wert

`zweitspielrecht` wird **nicht** aus dem CSV abgeleitet (kein entsprechendes CSV-Feld).

---

## Frontend

### Member-Interface-Erweiterungen

In `MemberDetailPage.tsx`, `MemberKontaktTab.tsx`, `MemberStammdatenTab.tsx`:
```ts
beitragsfrei?: boolean
zweitspielrecht?: boolean
```

### MemberDetailPage.tsx

`form`-State-Initialwerte:
```ts
beitragsfrei: false, zweitspielrecht: false
```

`applyMemberToForm`:
```ts
beitragsfrei: m.beitragsfrei ?? false,
zweitspielrecht: m.zweitspielrecht ?? false,
```

### MemberKontaktTab.tsx

Im Bankdaten-Block (`<div className="space-y-3">`), nach dem IBAN-Feld:

```tsx
<label className="flex items-center gap-2 cursor-pointer mt-2">
  <input
    type="checkbox"
    checked={form.beitragsfrei || false}
    onChange={e => onFormChange({ beitragsfrei: e.target.checked })}
    className="w-4 h-4 accent-brand-yellow"
  />
  <span className="text-sm text-brand-text">Beitragsfrei</span>
</label>
```

### MemberStammdatenTab.tsx

Nach dem Stammverein-Textfeld (innerhalb des `!isHonorar`-Blocks):

```tsx
<label className="flex items-center gap-2 cursor-pointer mt-2">
  <input
    type="checkbox"
    checked={form.zweitspielrecht || false}
    onChange={e => onFormChange({ zweitspielrecht: e.target.checked })}
    className="w-4 h-4 accent-brand-yellow"
  />
  <span className="text-sm text-brand-text">Zweitspielrecht</span>
</label>
```

---

## Betroffene Dateien

| Datei | Änderung |
|-------|----------|
| `internal/db/migrations/035_beitragsfrei_zweitspielrecht.up.sql` | Neu |
| `internal/db/migrations/035_beitragsfrei_zweitspielrecht.down.sql` | Neu |
| `internal/members/handler.go` | Struct, GET, PUT, CSV-Import |
| `web/src/pages/MemberDetailPage.tsx` | Interface, form-State, applyMemberToForm |
| `web/src/components/admin/MemberKontaktTab.tsx` | Interface, Checkbox Beitragsfrei |
| `web/src/components/admin/MemberStammdatenTab.tsx` | Interface, Checkbox Zweitspielrecht |
