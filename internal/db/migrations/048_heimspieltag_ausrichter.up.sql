-- Ausrichter: der Verein, der einen Heim-Spieltag organisiert (Halle stellt,
-- Bewirtung/Kuchen verantwortet). Struktur bewusst am Vorbild `stammvereine`
-- (001_initial.up.sql) orientiert: freie Namensliste, aktiv/inaktiv statt
-- Löschen im Regelfall, eigene sort_order für die UI.
CREATE TABLE ausrichter (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    aktiv       INTEGER NOT NULL DEFAULT 1,
    is_default  INTEGER NOT NULL DEFAULT 0,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Die Auflösung "Ausrichter eines Tages" ist TOTAL: fehlt ein expliziter
-- Eintrag, gilt der Default (design.md Decision 2). Das setzt voraus, dass es
-- nie mehr als einen Default geben kann — dieser Partial-Index ist die
-- eigentliche Garantie dafür, nicht der Anwendungscode. Handler, die den
-- Default wechseln, dürfen sich also nicht allein darauf verlassen, in der
-- richtigen Reihenfolge "alten Default aus, neuen Default an" zu schreiben;
-- verletzt eine Race das, bricht der zweite INSERT/UPDATE hier hart ab, statt
-- still zwei Defaults nebeneinander bestehen zu lassen.
CREATE UNIQUE INDEX idx_ausrichter_default ON ausrichter(is_default) WHERE is_default = 1;

-- Ausrichter je Spieltag. season_id gehört bewusst mit in den Primärschlüssel
-- (design.md Decision 1): auch `games` ist saisongebunden, und
-- regenSingleDay(date, seasonID) führt beide Werte immer zusammen. Ohne
-- season_id im Schlüssel würde ein Datum aus einer archivierten Saison mit
-- demselben Kalendertag der aktiven Saison kollidieren.
--
-- ausrichter_id verweist ON DELETE SET NULL, nicht CASCADE: ein gelöschter
-- Ausrichter soll den Spieltag nicht mitreißen, sondern ihn auf "kein
-- expliziter Wert" zurückfallen lassen. Weil die Auflösung ein NULL genauso
-- behandelt wie eine fehlende Zeile (Decision 2), fällt der Tag dadurch
-- automatisch auf den Default zurück — ohne Aufräumschritt im Handler.
CREATE TABLE spieltag_ausrichter (
    date          DATE NOT NULL,
    season_id     INTEGER NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    ausrichter_id INTEGER REFERENCES ausrichter(id) ON DELETE SET NULL,
    updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by    INTEGER REFERENCES users(id),
    PRIMARY KEY (date, season_id)
);

-- Optionale Bindung einer Vorlagen-Zeile an einen Ausrichter: die Zeile
-- erzeugt an einem Heim-Spieltag nur dann Slots, wenn der aufgelöste
-- Tages-Ausrichter übereinstimmt (Gate in internal/games/regen.go).
--
-- Bewusst RESTRICT statt SET NULL (design.md Decision 6): SET NULL würde eine
-- an einen Ausrichter gebundene Zeile beim Löschen dieses Ausrichters still
-- auf "gilt immer" heben — die Zeile erzeugte danach an MEHR Spieltagen
-- Dienste als vorher, genau umgekehrt zur Absicht eines Löschens. RESTRICT
-- verhindert den stillen Pfad; die Kaskade (gebundene Vorlagen-Zeilen aktiv
-- mitlöschen) führt stattdessen der Handler explizit in einer Transaktion aus,
-- nachdem GET /api/ausrichter/{id}/usage die Betroffenen benannt hat.
ALTER TABLE game_template_items ADD COLUMN ausrichter_id INTEGER REFERENCES ausrichter(id) ON DELETE RESTRICT;

-- Seed: genau eine Default-Zeile, ohne die wäre die Auflösung aus Decision 2
-- nicht total. Der Name ist NICHT hartcodiert (Entbrandung, siehe
-- openspec/changes/opensource-2-entbranding) — er wird aus der ersten
-- Vereins-Stammdatenzeile (`clubs.name`) abgeleitet, falls zu diesem
-- Zeitpunkt bereits eine existiert, sonst greift ein neutraler Platzhalter.
-- Der Name bleibt in der UI jederzeit umbenennbar.
--
-- Idempotent über UNIQUE(name): ein zweiter Lauf mit unverändertem clubs.name
-- (bzw. unverändertem Fallback) trifft dieselbe Kollision und wird von
-- INSERT OR IGNORE verworfen — es entsteht nie eine zweite Default-Zeile.
INSERT OR IGNORE INTO ausrichter (name, is_default)
SELECT COALESCE((SELECT name FROM clubs ORDER BY id LIMIT 1), 'Eigener Verein'), 1;
