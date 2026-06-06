## Context

Die BWHV-Hallenliste ist eine CSV-Datei mit ~1000 Zeilen, die Handball-Hallen im BWHV-Verbandsgebiet enthält. Sie hat eine 3-zeilige Preamble vor den eigentlichen Daten und enthält Hallennamen mit eingebetteten Kommata (erfordert echten CSV-Parser). Die bestehende `venues`-Tabelle und `Handler`-Struct in `internal/venues/handler.go` werden erweitert.

## Goals / Non-Goals

**Goals:**
- Neue `Import`-Methode im bestehenden `venues.Handler`
- Upsert by name: neue Hallen anlegen, bestehende aktualisieren (`is_home_venue` nie überschreiben)
- Split-Button-UI auf der Venues-Seite, konsistent mit MembersPage
- Import-Ergebnis-Feedback (importiert / aktualisiert / übersprungen / Fehler)

**Non-Goals:**
- Kein neues DB-Schema (kein `bwhv_nummer`-Feld)
- Kein geplanter / automatisierter Import (manuell on-demand)
- Kein Export von Venues

## Decisions

**CSV-Parsing: `encoding/csv` statt manuellem Split**
Die BWHV-Datei hat quoted Felder mit eingebetteten Kommata (z.B. `"St.-Jakobs-Halle, Feld 1"`). Stdlib `encoding/csv` behandelt dies korrekt ohne externe Dependency.

**Preamble-Erkennung: Suche nach Header-Zeile**
Statt fixe Zeilen zu überspringen, wird die erste Zeile gesucht, deren erste Zelle `"Name"` ist. Das macht den Parser robust gegen leicht abweichende Preamble-Längen.

**Spalten-Mapping (0-indexed nach Header-Zeile):**
```
0: Name        → name        (Pflicht, Upsert-Key)
1: Nummer      → ignoriert
2: Straße      → street      (kann leer sein)
3: PLZ         → postal_code
4: Ort         → city
5: Kennzeichnung → note
6: (optional)  → an note angehängt, falls nicht leer
```
`country` wird immer auf `"DE"` gesetzt. `is_home_venue` beim Update nie überschrieben.

**Upsert-Strategie: Lookup + conditional INSERT/UPDATE**
SQLite's `INSERT OR REPLACE` würde die ID ändern und FK-Verweise brechen. Stattdessen:
```sql
SELECT id FROM venues WHERE name = ?
-- wenn gefunden: UPDATE ... WHERE id = ?
-- sonst:         INSERT INTO venues (...)
```

**Transaktion: eine Transaktion für den gesamten Import**
Bei Fehler wird alles zurückgerollt. Der `hub.Broadcast("venues")` erfolgt nur bei Commit.

**Datei-Größenlimit: 10 MB**
Über `r.ParseMultipartForm(10 << 20)` gesetzt. Die BWHV-CSV ist ~100 KB, also großzügig.

**Fehler: zeilen-weise gesammelt, nicht abgebrochen**
Zeilen ohne Namen werden übersprungen und in `errors`-Array zurückgegeben. Der Import läuft weiter.

## Risks / Trade-offs

- [Encoding] Die CSV kann BOM (UTF-8 BOM, `\xEF\xBB\xBF`) am Anfang haben → erster Zellen-Wert muss getrimmt werden (bereits beobachtet: `﻿Hallenliste...`)
- [Upsert-Key] Name-Matching ist case-sensitiv; leicht abweichende Schreibweisen erzeugen Duplikate → akzeptiert, da BWHV-Daten normiert sind
- [Transaktion + ~1000 Zeilen] SQLite WAL-Mode macht das unproblematisch, keine Performance-Bedenken bei dieser Größe

## Migration Plan

Kein DB-Schema-Change. Deploy = normaler `make deploy`.
