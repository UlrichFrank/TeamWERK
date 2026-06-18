## Context

Aktuell existieren drei Teamname-Formate parallel:

```
┌──────────────┬───────────────────┬──────────────────────────────────────┐
│ Format       │ Beispiel          │ Quelle                               │
├──────────────┼───────────────────┼──────────────────────────────────────┤
│ Kurzform     │ mA · mA1 · wB2    │ web/src/lib/teamName.ts              │
│              │                   │   buildTeamShortNames() (clientseitig)│
│ Langform     │ B-Jugend männlich │ internal/db/team_display_name.go     │
│              │ B-Jugend 2 männl. │   TeamDisplayName(alias) SQL         │
│ Rohname      │ TS Stuttgart B…   │ teams.name (direkt aus DB)           │
└──────────────┴───────────────────┴──────────────────────────────────────┘
```

`buildTeamShortNames` baut den Kurznamen aus `gender[0]` (m→m, f→w, mixed→g) + `age_class[0]` + optionalem `team_number` (nur wenn `group_count > 1`). Die SQL-Langform nutzt `kader.team_number` und Gender-Mapping in lesbarer Form.

Inkonsistenzen heute (Auszug):
- `KalenderPage.tsx:847,857`: Kurzform, aber String `"Mehrere"`/`"Mehrere Teams"` bei >1 Team
- `TerminePage.tsx:492–496`: Roh-`t.name` (GROUP_CONCAT) + `"Mehrere Teams"`
- `TerminePage.tsx:402`: Langform via SQL `TeamDisplayName`
- `DashboardPage.tsx:160` + `dashboard/handler.go:176`: Langform via SQL, aber **`MIN(...)`** — Bug bei Doppelheimspielen
- `SpieltagDetailPage.tsx:217`: `team_name` existiert nicht im API-Response → rendert leer (Bug)
- `DutyPage.tsx:240` + `duties/handler.go:429`: Langform
- `MeinTeamPage.tsx:36`: Rohname

## Goals / Non-Goals

**Goals:**
- Eindeutige Regel pro UI-Ort, welches Format gezeigt wird
- Server als Single Source of Truth für Display-Strings (vermeidet Drift)
- Vollständige Auflistung statt „Mehrere" überall außer im Kalender
- Bestehende Frontend-Logik in `buildTeamShortNames` bleibt als Fallback/Spiegel zum Server, aber Standardpfad ist Server-Wert

**Non-Goals:**
- Keine Änderung am Datenmodell (`teams`, `kader`)
- Keine Migrationen
- Keine Umbenennung von `teams.name` (das bleibt der „Rohname")
- Keine neue Übersetzungsschicht für age_class-Labels — die SQL-Logik wird 1:1 für die Kurzform repliziert

## Decisions

### Regelwerk pro UI-Ort

```
┌────────────────────────────┬────────────┬───────────────────────┐
│ Ort                        │ Format     │ Multi-Team            │
├────────────────────────────┼────────────┼───────────────────────┤
│ Kalender-Tile (Spiel)      │ Kurzform   │ "Mehrere" (Ausnahme!) │
│ EventInfoModal             │ Kurzform   │ komma-getrennt        │
│ Termine-Seite Spiel/Train. │ Kurzform   │ komma-getrennt        │
│ DutyPage Gruppen-Header    │ Kurzform   │ —                     │
│ Mitfahrten, Chat, Admin    │ Kurzform   │ komma-getrennt        │
│ SpieltagDetailPage         │ Langform   │ komma-getrennt        │
│ TermineDetailPage Training │ Langform   │ —                     │
│ MeinTeam                   │ Langform   │ —                     │
│ Dashboard „Nächste Termine"│ Kurzform   │ komma-getrennt        │
└────────────────────────────┴────────────┴───────────────────────┘
```

### Server-Helper `TeamDisplayShort(alias)`

Analog zu `TeamDisplayName`, produziert aber den Kurznamen:

```sql
(
  SELECT
    CASE k_dn.gender WHEN 'm' THEN 'm' WHEN 'f' THEN 'w' ELSE 'g' END
    || SUBSTR(k_dn.age_class, 1, 1)
    || CASE
         WHEN (SELECT COUNT(*) FROM kader k_cnt
               WHERE k_cnt.season_id = (SELECT id FROM seasons WHERE is_active=1 LIMIT 1)
                 AND k_cnt.age_class = k_dn.age_class
                 AND k_cnt.gender = k_dn.gender) > 1
         THEN CAST(k_dn.team_number AS TEXT)
         ELSE ''
       END
  FROM kader k_dn
  WHERE k_dn.team_id = <alias>.id
    AND k_dn.season_id = (SELECT id FROM seasons WHERE is_active=1 LIMIT 1)
  LIMIT 1
)
```

Parity zu `buildTeamShortNames`:
- `gender m/f/mixed → m/w/g`
- `age_class` → erstes Zeichen (für nicht-A–F-Klassen ebenfalls erstes Zeichen)
- Suffix `team_number` nur wenn mehrere Teams gleicher `age_class+gender` in aktiver Saison

### API-Felder

Pro Endpoint-Item:
- Single-Team-Items (DutyBoard-Group, TrainingSession): `team_display_short`, `team_display_long`
- Multi-Team-Items (Game-List): `team_display_short_csv`, `team_display_long_csv` (komma-getrennt, alphabetisch sortiert) plus existierende `team_ids[]` und `teams[]`
- Bestehende Felder (`team_names`, `team_name`) bleiben unverändert für Rückwärtskompatibilität — werden in einem Folge-Cleanup entfernt, sobald keine Stelle mehr sie liest.

### Frontend-Helper

```ts
type Mode = 'short' | 'long' | 'kalender'

interface TeamRef { id: number; display_short?: string; display_long?: string; name?: string }

function formatTeamList(teams: TeamRef[], mode: Mode): string
//  mode='kalender' & teams.length > 1 → 'Mehrere'
//  mode='kalender' & teams.length === 1 → teams[0].display_short
//  mode='short' → teams.map(display_short).join(', ')
//  mode='long' → teams.map(display_long).join(', ')
//  Fallback wenn display_* fehlt: buildTeamShortNames-Map oder name
```

### Dashboard-Fix

`dashboard/handler.go:176` `MIN(COALESCE(TeamDisplayName(t), t.name))` → `GROUP_CONCAT(COALESCE(TeamDisplayShort(t), t.name), ', ')` mit `ORDER BY t.id` für deterministische Reihenfolge.

### SpieltagDetailPage-Fix

Interface `team_name` entfernen, stattdessen `teams[]` aus dem bestehenden `/api/games/{id}`-Response nutzen und mit `formatTeamList(teams, 'long')` rendern.

## Risks / Trade-offs

- **SQL-Logik-Drift**: `TeamDisplayShort` muss exakt mit `buildTeamShortNames` übereinstimmen, sonst sehen User je nach Quelle leicht unterschiedliche Namen. Mitigation: Parity-Test, der mehrere Beispiel-Datensätze durch beide Implementierungen laufen lässt und Gleichheit prüft.
- **Doppelte Felder in API**: solange `team_name`/`team_names` parallel bleiben, ist die Payload etwas größer. Akzeptabel; Cleanup in Folge-Change.
- **Performance**: pro Game zwei zusätzliche Subqueries pro Team-Row. Bei der aktiven Saison mit ~10 Teams und üblicher Game-Liste vernachlässigbar (SQLite, alles im Index).
- **DutyPage-Wechsel auf Kurzform**: User sehen in vertrauter Stelle plötzlich „mA1" statt „B-Jugend 2 männlich". Bewusst akzeptiert — die Kurzform ist im Listen-Kontext schneller scanbar.
- **Generische Events (event_type='generisch')** im Kalender können mehrere Teams referenzieren oder gar keins — „Mehrere" gilt auch hier, kein Team → kein Label.
