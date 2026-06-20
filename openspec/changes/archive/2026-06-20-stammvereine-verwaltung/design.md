# Design: Stammvereine-Verwaltung

## 1. Datenmodell

### 1.1 Neue Tabelle `stammvereine`

```sql
CREATE TABLE stammvereine (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    aktiv       INTEGER NOT NULL DEFAULT 1,   -- Soft-Delete
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

Seed = die 8 Vereine aus `internal/beitragslauf/compute.go` `Mitgliedsvereine[]`.

### 1.2 `members.home_club_id`

```sql
ALTER TABLE members ADD COLUMN home_club_id INTEGER REFERENCES stammvereine(id);
```

Nullable: `NULL` = „kein Stammverein" = Kategorie `aktiv_ohne`. Bewusst **keine** `NOT NULL`-Pflicht, da „ohne Stammverein" ein gültiger fachlicher Zustand ist (Entscheidung des Vereins).

### 1.3 Warum FK statt Freitext-Dropdown

| Aspekt | FK (`home_club_id`) | Freitext + Dropdown |
|---|---|---|
| Determinismus Beitrag | exakt (`IS NOT NULL`) | Stringvergleich, bricht bei Umbenennung |
| Umbenennen eines Vereins | 1 UPDATE, alle Mitglieder folgen | bricht Zuordnung aller Mitglieder |
| Referenzielle Integrität | DB-garantiert | keine |
| Migration | einmaliger Mapping-Schritt nötig | trivial |

Der einmalige Migrationsaufwand ist vertretbar; die Integrität wiegt schwerer.

## 2. Daten-Migration (home_club → home_club_id) — reviewbar

**Grundsatz:** Die automatisch laufende Schema-Migration (047) verändert **keine** Mitglieder-Daten. Der Backfill `home_club` → `home_club_id` ist ein **getrennter, vorab reviewbarer Schritt**, weil `make deploy` Migrationen ungefragt anwendet — eine eingebettete `UPDATE`-Migration würde das Mapping ohne Kontrolle festschreiben.

### 2.1 Schritt A — Schema (Migration 047, automatisch)
Legt nur `stammvereine` (inkl. Seed) und die Spalte `members.home_club_id` (alle `NULL`) an. Keine Datenmutation.

### 2.2 Schritt B — Mapping-Preview (read-only, Review durch Vorstand/Kassierer)
Eine reine SELECT-Abfrage erzeugt eine Review-Tabelle: jeder distinct `home_club`-Freitext, Anzahl betroffener Mitglieder, vorgeschlagene Zuordnung per **exakt-normalisiertem** Abgleich (lowercase, `.`/`-`/`/` entfernt), und ein Status `exakt` | `UNMATCHED`.

```sql
-- Mapping-VORSCHAU (verändert nichts). Gegen die Produktiv-DB ausführen.
SELECT
    m.home_club                                   AS freitext,
    COUNT(*)                                      AS anzahl_mitglieder,
    s.id                                          AS vorgeschlagene_id,
    s.name                                        AS vorgeschlagener_verein,
    CASE WHEN s.id IS NULL THEN 'UNMATCHED' ELSE 'exakt' END AS status
FROM members m
LEFT JOIN stammvereine s
    ON lower(replace(replace(replace(s.name,'.',''),'-',''),'/','')) =
       lower(replace(replace(replace(m.home_club,'.',''),'-',''),'/',''))
WHERE TRIM(COALESCE(m.home_club,'')) <> ''
GROUP BY m.home_club, s.id, s.name
ORDER BY status, anzahl_mitglieder DESC;
```

Der Reviewer entscheidet pro `UNMATCHED`-Zeile: manuell zuweisen (Mitglied-Edit) oder bewusst `NULL` lassen (= `aktiv_ohne`). Die SQL-Normalisierung bildet die Go-`NormalizeClubName`-Logik ab; den einzigen Unterschied (Zusammenfassen mehrfacher Innen-Leerzeichen) deckt die Preview konservativ ab, indem solche Fälle als `UNMATCHED` zur manuellen Prüfung erscheinen.

### 2.3 Schritt C — Apply (erst nach Freigabe)
Nach Review wird genau das exakte Mapping angewendet — als separat ausführbares SQL (nicht Teil der Auto-Migration), z. B. via `make migrate-remote`-analogem One-off oder im Deploy-Skript dokumentiert:

```sql
-- Apply: NUR exakte Treffer. UNMATCHED bleibt bewusst NULL.
UPDATE members
   SET home_club_id = (
       SELECT s.id FROM stammvereine s
       WHERE lower(replace(replace(replace(s.name,'.',''),'-',''),'/','')) =
             lower(replace(replace(replace(members.home_club,'.',''),'-',''),'/',''))
   )
 WHERE TRIM(COALESCE(home_club,'')) <> ''
   AND home_club_id IS NULL;
```

**Kein** automatischer Fuzzy-Schritt — bewusst, um keine falschen Zuordnungen festzuschreiben. `MatchHomeClub` bleibt als deprecated im Code (kein Aufruf mehr im Lauf), bleibt aber für ein optionales Hilfsskript verfügbar, das dem Reviewer Fuzzy-*Vorschläge* zu den `UNMATCHED`-Zeilen liefern könnte.

## 3. Beitragslauf-Anpassung

`internal/beitragslauf/compute.go`:
- `AktivKategorie(mitStammverein bool)` bleibt unverändert.
- Aufrufseite in `handler.go`/`query.go`: statt `MatchHomeClub(m.HomeClub).Matched` jetzt `m.HomeClubID != nil`.

`LoadMembersForLauf` (`query.go`): SELECT um `home_club_id` ergänzen, `home_club`-Freitext kann für Anzeige/Debug erhalten bleiben.

Wegfall der Warnung `home_club_unklar` (`handler.go`): Es gibt keine unsichere Zuordnung mehr. Bestehende Tests, die diese Warnung prüfen, anpassen.

## 4. Soft-Delete-Verhalten

`DELETE /api/stammvereine/{id}` setzt `aktiv=0`. Begründung: Mitglieder können den Verein referenzieren; ein Hard-Delete würde den FK verletzen bzw. (bei `ON DELETE SET NULL`) stillschweigend ihre Beitragskategorie auf `aktiv_ohne` kippen — das soll der Vorstand bewusst tun, nicht als Nebeneffekt.

Dropdown im Mitglied: zeigt aktive Vereine. Ist einem Mitglied ein inaktiver Verein zugeordnet, wird dieser zusätzlich (markiert) angezeigt, damit er nicht still verschwindet.

## 5. Router-Einordnung

Neue Routen-Gruppe in `internal/app/router.go`:
- `GET /api/stammvereine` → **Authenticated** (jeder Eingeloggte braucht die Liste fürs Mitglied-Dropdown).
- `POST/PUT/DELETE /api/stammvereine` → **Vorstand** (`auth.RequireClubFunction("vorstand")`, admin umgeht).

## 6. SSE

Alle Mutationen broadcasten `"stammvereine"`. Frontend: Settings-Tab und ggf. Mitglied-Edit abonnieren `useLiveUpdates(e => { if (e === 'stammvereine') reload() })`.
