## Context

Motivation: siehe proposal.md. Ist-Stand im Code:

- `game_lineup (game_id, member_id)` hängt am **Spiel**, nicht an der Mannschaft.
  Geschrieben wird nur über `POST /api/games/{id}/lineup` (Diff-Upsert, broadcastet
  `games`, meldet Aufnahme/Herausnahme per `notifyLineupChange`).
- Die dreiwertige Regel existiert genau einmal: `resolveLineupState` in
  `internal/calendar/handler.go`, gespeist aus zwei `EXISTS`-Spalten, eingeschränkt auf
  den erweiterten Kader.
- `GET /api/games/{id}/participants` liefert nur `in_lineup: bool`. Die Detailseite
  baut daraus eine `lineupMap` und zeigt Häkchen/Strich — „offen" und „nicht
  aufgestellt" sehen gleich aus.
- Spieler bekommen von `participants` nicht alle Zeilen (`hidden_team_ids`, Footer
  „Weitere Mitglieder nicht sichtbar"). Aus den ausgelieferten Zeilen lässt sich „ist
  überhaupt jemand aufgestellt?" deshalb clientseitig **nicht** sicher beantworten.
- `GET /api/games/my` (Liste) und `GET /api/teams/{id}/rsvp-matrix` (Tabelle,
  `internal/attendance/matrix.go`) kennen die Aufstellung gar nicht.
- Im iCal-Feed führt die Hilfsabfrage `kaderMembership` Trainer und Stammkader beide als
  `is_extended = 0` — sie sind dort bisher nicht unterscheidbar, weil der Status für
  beide entfiel.
- `internal/db` (Import-Alias `appdb`) ist Foundation und hält bereits geteilte
  SQL-Fragmente (`UserTeamsSQL`, `TeamDisplayShort`, …). `games`, `attendance` und
  `calendar` sind Domänen und dürfen sich nicht gegenseitig importieren.

## Goals / Non-Goals

**Goals:**
- Eine Ableitungsregel, eine Stelle; jede Lese-Route nutzt sie.
- Eine Bezeichnungs-Quelle im Frontend; Liste, Tabelle und Detailseite importieren sie.
- Additive API-Felder, keine Migration, kein neuer Endpunkt.

**Non-Goals:**
- Kein expliziter Veröffentlichungs-Zustand der Aufstellung (siehe proposal).
- Keine Änderung an Schreibpfad, Broadcast oder Benachrichtigung der Aufstellung.

## Decisions

### 1. Ableitung als SQL-Fragment in `internal/db`

`appdb.LineupStateSQL(eventTypeExpr, gameIDExpr, memberIDExpr string) string` liefert

```sql
CASE WHEN <eventType> NOT IN ('heim','auswärts') THEN NULL
     WHEN EXISTS (SELECT 1 FROM game_lineup WHERE game_id=<game> AND member_id=<member>) THEN 'in'
     WHEN EXISTS (SELECT 1 FROM game_lineup WHERE game_id=<game>) THEN 'out'
     ELSE 'open' END
```

dazu die Konstanten `appdb.LineupIn/LineupOut/LineupOpen`. Beide `EXISTS` treffen den
PK `(game_id, member_id)` bzw. dessen Präfix — kein Scan.

Die **Rollen**-Einschränkung (Spieler ja, Trainer nein) bleibt beim Aufrufer: jede
Route weiß selbst, ob eine Zeile ein Trainer ist (`is_trainer` in participants,
`inTrainerKader` in `ListMyGames`, Trainer sind in der Matrix gar keine Zeile). Das
Fragment kann das nicht wissen, ohne die Kader-Joins nachzubauen.

*Alternativen:* (a) Go-Funktion über zwei `EXISTS`-Spalten wie heute im Kalender — dann
müssten drei Domänen je zwei Spalten selektieren und dieselbe `switch`-Logik aufrufen;
die Regel „leer ≠ nicht aufgestellt" steht dann faktisch an vier Stellen im SQL.
(b) Frontend leitet ab — scheitert an den verborgenen participants-Zeilen (Context).

`calendar.resolveLineupState` wird auf das Fragment umgestellt und mappt nur noch
Code → Kennwort/Satz; der Kommentar „die Regel lebt allein hier" wandert an das
Fragment.

### 2. Participants liefert `lineup` und `lineup_count`

Pro Zeile `lineup` (über das Fragment), einmal je Antwort `lineup_count`
(`SELECT COUNT(*) FROM game_lineup WHERE game_id=?`). `in_lineup` bleibt für
Abwärtskompatibilität und als Checkbox-Wert des Trainers.

Die Detailseite leitet den Anzeigezustand einer Zeile beim **Trainer** weiterhin aus
dem optimistisch geführten `lineupMap` ab (sonst hinkte die Anzeige nach dem Klick
hinterher): `open`, solange kein Eintrag `true` ist, sonst `in`/`out`. Beim Trainer ist
das sicher, weil er alle Zeilen seines Teams sieht. Für Spieler/Eltern gilt das
Server-Feld `lineup` unverändert. Der Kartenkopf nutzt `lineup_count` bzw. beim Trainer
die Zahl der `true`-Einträge nach optimistischem Update.

### 3. Liste: `my_lineup` und `children_rsvp[].lineup`

`ListMyGames` selektiert das Fragment mit `memberID` und setzt `my_lineup` nur, wenn
`inRegularKader || inExtendedKader` — ein reiner Trainer bekommt kein Feld.
`attachChildrenRSVPToGames` ergänzt beide `UNION`-Zweige um das Fragment. Anzeige:
`LineupBadge` rechts (`ml-auto`) in der Zeile der Zu-/Absage-Knöpfe, je Person.

### 4. Darstellung (entschieden am 27.09.2026, Entwurf: https://claude.ai/artifact/GksFknrPG1sw7i4guFcUt2)

Ein Schema für **alle** Flächen — Liste, Tabelle, Detailseite. Es gibt genau eine
Quelle (`lib/lineup.ts`), und alle Elemente folgen denselben Regeln:

| Regel | Wert |
|---|---|
| Farben | ein Grün `brand-green` (voll), ein Grau `bg-brand-border`, eine Strichfarbe `border-brand-text-subtle` |
| Auf Grün | Schrift, Rückmeldesymbole und Haken **weiß** (`text-white`; Voreinstellung `text-white opacity-60`) |
| Rahmen | immer 1 px (`border`); gefüllte Zustände `border-transparent` (gleiche Außenmaße), offen `border-dashed` |
| Offen | **innen vollständig transparent** (`bg-transparent`), nur der gestrichelte Rahmen |
| Ecken | immer `rounded-md` (wie die Zu-/Absage-Knöpfe) |
| Schrift | immer `text-xs font-medium`, keine Symbole in Kennzeichen |
| Kennzeichen | `w-32`; in der Knopfzeile `self-stretch` (= Knopfhöhe, 26 px Desktop / 30 px mobil), frei stehend `h-[26px]` |
| Tabellenzelle | Box `h-8`, volle Zellbreite, Rückmeldesymbol mittig |
| Checkbox | eigenes Element `role="checkbox"`, `w-5 h-5`, dieselben Flächen, Haken `Check` weiß |
| Grauer Kreis | auf grüner Fläche weiß, auf grauer Fläche `brand-text-muted` statt `brand-text-subtle` |

| Zustand | Fläche | Schrift/Symbole |
|---|---|---|
| `in` · aufgestellt | `bg-brand-green border-transparent` | weiß |
| `out` · nicht aufgestellt | `bg-brand-border border-transparent` | `brand-text`, Symbole farbig |
| `open` · Aufstellung offen | `bg-transparent border-dashed border-brand-text-subtle` | `brand-text`, Symbole farbig |

**Grün der Rückmeldung auf `/termine`:** das bisherige Roh-Grün `green-600` (aktiver
„Zusagen“-Knopf in Liste und Tabellen-Dialog, Zusage-Zähler auf den Karten, Zusage-Haken
in der Rückmelde-Spalte der Detailseite) wird durch `brand-green` ersetzt — ein Grün für
Zusage und Aufstellung. Andere Seiten mit `green-600` bleiben unberührt.

Entschieden am 27.09.2026 als „Variante 3" des Entwurfs; die beiden anderen Varianten
(Markengrün 30 % mit dunkler Schrift, `green-600`) sind verworfen.

- **Liste:** Kennzeichen rechts (`ml-auto`) in der Zeile der Zu-/Absage-Knöpfe.
- **Tabelle:** jede Spielzelle (heim/auswärts) trägt die Box; Trainings und generische
  Termine eine unsichtbare Box derselben Geometrie. „Offen" ist eine Zelleneigenschaft,
  kein Spaltenkopf-Symbol. Die gelbe Antippbar-Markierung der eigenen Zeile bleibt am
  `td`. Legende: drei Farbfelder. Zelltitel/`aria-label`: „<Rückmeldung> · <Aufstellung>".
- **Detailseite:** Checkbox je Spieler; für Spieler/Eltern `aria-disabled`, für Trainer
  bedienbar (Hover/Fokus `ring-2 ring-brand-yellow` wie in der Matrix). Kartenkopf:
  Kennzeichen „Aufstellung offen" bzw. „N aufgestellt".

**„Sonstiges" (generische Termine):** die Detailseite behält die Spalte
„Aufstellung" (Entscheidung 27.09.2026 — Trainer pflegen dort teils eine Aufstellung).
`GetParticipants` leitet den Status deshalb über `appdb.LineupFromFacts` ohne Typ-Gate
ab; Liste, Tabelle und Kalender-Abo nutzen `LineupStateSQL`/`ResolveLineupState` mit
Gate und zeigen bei „Sonstiges" nichts — sonst stünde an jeder Vereinsfeier
„Aufstellung offen".

*Warum eine eigene Checkbox statt `<input type="checkbox">`:* `@tailwindcss/forms` füllt
eine angehakte Checkbox vollfarbig mit weißem Haken — das wäre ein zweites, kräftigeres
Grün neben dem 30-%-Grün der Kennzeichen. Das eigene Element braucht Tastaturbedienung
(Leertaste) und `aria-checked`.

*Warum eine Fläche statt Symbol:* auf einen Blick lesbar, auch neben den farbigen
Rückmeldesymbolen; das Grau des Zeilen-Hovers (`brand-table-select`) liegt am `td`, die
Aufstellungsbox darüber bleibt unterscheidbar.

### 5. Eine Frontend-Quelle für Bezeichnung und Darstellung

`web/src/lib/lineup.ts`: Typ `LineupState = 'in' | 'out' | 'open'`, `LINEUP_LABEL`
(`aufgestellt` / `nicht aufgestellt` / `Aufstellung offen`), `LINEUP_SURFACE`
(Flächenklassen oben), `LINEUP_SHAPE` und `LINEUP_TEXT`; `LineupCheckbox.tsx` nutzt dieselben Klassen.
`web/src/components/LineupBadge.tsx` rendert das Text-Kennzeichen (Liste, Kopf der
Detailseite). Keine Icons für die Aufstellung.

### 6. iCal: Rolle statt `is_extended`

`kaderMembership` bekommt statt `is_extended` eine Spalte `kind` (0 = Stammkader,
1 = Trainer, 2 = erweitert). Die bestehende `ORDER BY`-Regel „regulär schlägt
erweitert" bleibt erhalten, und die bisherige Reihenfolge Trainer-vor-erweitert
ebenfalls. Status gibt es für `kind IN (0, 2)`; `kaderLabel` hängt das Kennwort in
beiden Fällen an, `erw. Kader` nur bei `kind = 2`.

### 7. Live-Updates

Nichts Neues: `SaveLineup` broadcastet `games`; `/termine` lädt bei `games` bereits Liste
bzw. Matrix neu, die Detailseite ebenfalls. Kein Eintrag in Broadcast-Allowlists.

## Risks / Trade-offs

- [Kalender-Abos aller Stammspieler ändern ihre Titel; bei nicht gepflegter Aufstellung
  steht überall „Aufstellung offen"] → Bewusste Entscheidung des Nutzers (Umfang „alle
  Kaderspieler"). Im Release-Hinweis/Doku erwähnen; der Satz im `DESCRIPTION`
  („steht noch nicht fest") ist neutral formuliert.
- [Erstes Häkchen des Trainers macht sofort alle anderen zu „nicht aufgestellt", auch in
  Liste und Kalender] → bestehende Semantik (Non-Goal); in der Trainer-Doku klarstellen,
  die Aufstellung am Stück zu setzen. Ein Veröffentlichungs-Schalter wäre der
  Folge-Change.
- [Spiel mit mehreren Mannschaften: Aufstellung der einen macht die Spieler der anderen
  zu „nicht aufgestellt"] → bestehende Semantik der geteilten Liste (so auch in
  `docs/anleitung-trainer.md` beschrieben); Szenario in der Spec festgehalten.
- [Matrix-Ableitung in Go statt über das Fragment = zweite Form der Regel] →
  Deckungsgleichheits-Test (Decision 4), analog `TestAushilfePraedikat_…`.
- [Eigene Checkbox statt nativer: Tastatur und Screenreader müssen nachgebaut werden]
  → `role="checkbox"`, `aria-checked`, `tabindex=0`, Leertaste/Enter toggeln; Vitest
  prüft Rolle, Zustand und Tastatur.
- [Weiß auf `brand-green` hat nur 2,6 : 1 — unter 4,5 : 1 (Schrift) und 3 : 1 (Symbole);
  ebenso der `brand-green`-Haken/Zähler auf hellem Grund mit 2,4 : 1] → bewusste
  Entscheidung für die Markenfarbe (Entwurf mit Kontrasttabelle vorgelegt). Die
  Information hängt nicht allein an der Farbe: Kennzeichen tragen Text, Zellen und
  Checkboxen ein `aria-label`/`title`. Nachschärfen wäre ein dunkleres Marken-Grün als
  neues Token (eigener Change).
- [Daumen hoch `brand-green` (1,7 : 1) und „vielleicht“ `brand-warning` (1,5 : 1) auf
  der grauen Fläche] → bekannt, nicht Teil dieses Changes; der Kreis wird dort auf
  `brand-text-muted` (3,3 : 1) angehoben.

## Migration Plan

Keine DB-Migration. Deploy wie üblich; Rollback per `make deploy-rollback` ohne
Datenfolgen (nur Lesepfade und additive JSON-Felder). Ältere Frontends ignorieren die
neuen Felder.
