## Why

Dienstplan-Vorlagen (`game_templates`/`game_template_items`, Frontend `/dienstplan-vorlagen/{id}`)
sind heute team-agnostisch: eine Vorlage wie „Heimspiel Standard" wird von **jedem** Heimspiel
geteilt, unabhängig davon, welches Kaderteam spielt. Bei der Regeneration (`regenGameItems`,
`internal/games/regen.go:377-462`) entsteht jedes Item für **jedes** Team, das über `game_teams`
am Spiel hängt — es gibt keinen Weg, ein einzelnes Item nur für bestimmte Teams verpflichtend zu
machen (Beispiel: Kamera-Dienst soll nur bei mA1- und mB1-Spielen entstehen, bei mC1 nicht).

Aktuell bleibt nur der Umweg über eine eigene Vorlage pro Team-Kombination — das verdoppelt
jedes gemeinsame Item (Kasse, Aufbau, …) in mehreren Vorlagen und lässt sie unweigerlich
auseinanderdriften, sobald ein gemeinsames Item geändert wird.

## What Changes

- **Neue optionale Team-Zuordnung pro Vorlagen-Item** (`game_template_items`): ein Item kann auf
  eine Liste von `teams.id` eingeschränkt werden. Leer/NULL = gilt weiterhin für alle Teams des
  Spiels (heutiges Verhalten, keine Migration von Bestandsdaten nötig — reiner Additiv-Default).
- **Regen-Filter**: bei der Slot-Erzeugung (`regenGameItems`) wird die Team-Schleife pro Item
  gegen dessen Team-Liste gefiltert. Bei einem Mehrteam-Spiel (`game_teams` mit >1 Eintrag)
  entsteht der Slot **nur** für die in der Item-Liste gelisteten Teams, nicht für alle Teams des
  Spiels.
- **Vorlagen-Editor** (`AdminDutyTemplateDetailPage.tsx`): neues Auswahlfeld „Kaderteams" pro
  Item, analog zur bestehenden `audiences`-Checkbox-Zeile, aber mit umgekehrter Leer-Semantik
  (leer = **alle** Teams, nicht „keine"). Auswahloptionen sind die Kaderteams der **aktiven**
  Saison (über `kader.team_id`), nicht die vollständige `teams`-Tabelle.
- **Kein Migrations-Bruch**: bestehende Items ohne Team-Zuordnung verhalten sich exakt wie
  bisher (alle Teams des Spiels).

## Capabilities

### New Capabilities

- `duty-template-team-scope`: Team-Zuordnung auf Ebene einzelner Dienstplan-Vorlagen-Items —
  ein Item kann auf eine explizite Teilmenge der Kaderteams eingeschränkt werden, sodass es bei
  der Auto-Regeneration nur für diese Teams Dienst-Slots erzeugt, während andere Items derselben
  Vorlage weiterhin für alle Teams gelten.

### Modified Capabilities

(keine — die Regen-Engine selbst ändert nur ihr internes Filterverhalten, ihre öffentlich
dokumentierten Verträge aus `games-regen-refactor` (Zerlegung, `regen_summary`-Struktur) und
`duties` (Duty-Board-Contract) bleiben unverändert.)

## Impact

- **Schema**: neue Spalte auf `game_template_items` (JSON-Array von `teams.id`, analog zur
  bestehenden `audiences`-Spalte), neue Migration mit der nächsten freien Nummer.
- **Backend**: `internal/games/regen.go` (`templateItemRow`, `loadTemplateItemsTx`,
  `regenGameItems`), `internal/games/handler.go` (`GET`/`PUT /api/duty-templates/{id}`
  Payload-Erweiterung, Zeilen ~1713/~1734 laden/schreiben die Items).
- **Frontend**: `web/src/pages/AdminDutyTemplateDetailPage.tsx` (Item-Zeile Mobile + Desktop),
  neue Datenquelle für aktive Kaderteams im Editor (Kurzname via bestehende
  `internal/db/team_display_short.go`-Logik).
- **Tests**: Regen-Charakterisierungssuite (`internal/games/handler_test.go`) — Default-Verhalten
  (kein Team-Filter) bleibt unverändert, neuer Test für gefilterte Slot-Erzeugung bei
  Mehrteam-Spielen. Neue/erweiterte Route-Tests für `PUT /api/duty-templates/{id}`.
- **Kein Fremdeinfluss** auf den parallel laufenden, noch unfertigen Change
  `duty-bulk-regen` (eigener Massenlauf-Treiber über dieselbe Regen-Engine) — dessen
  `design.md` §11 verweist bereits auf diese Erweiterung und die davon betroffene
  `RegenSummary.Created[].Count`-Zählung; dieser Change führt keine Abhängigkeit dorthin ein.

## Test-Anforderungen

| Route | Fall | Erwartung |
|---|---|---|
| `PUT /api/duty-templates/{id}` | Vorstand, Item mit `team_ids: [5,7]` | 200, Item wird mit Team-Liste gespeichert |
| | Vorstand, Item mit `team_ids: null`/fehlend | 200, Item gilt weiterhin für alle Teams (Default) |
| | Vorstand, `team_ids` enthält unbekannte `teams.id` | 400 `invalid_team` |
| | Standard-Nutzer ohne `vorstand` | 403 |
| `GET /api/duty-templates/{id}` | beliebiger authentifizierter Nutzer | 200, `team_ids` je Item im Response (leer-Array bei Default) |

**Garantierte Invarianten:**

1. **Rückwärtskompatibel.** Ein Item ohne `team_ids` (bzw. leeres Array) erzeugt bei der
   Regeneration weiterhin für **jedes** Team des Spiels einen Slot — bit-identisches Verhalten
   zu vor diesem Change (Regen-Charakterisierungstest).
2. **Team-Filter greift bei Mehrteam-Spielen.** Ein Spiel mit `game_teams = [mA1, mC1]` und
   einem Item mit `team_ids: [mA1]` erzeugt den Slot **ausschließlich** für mA1 — mC1 bekommt
   diesen Slot nicht, obwohl es am selben Spiel teilnimmt.
3. **Andere Items derselben Vorlage bleiben unbeeinflusst.** Ein Item ohne `team_ids` in
   derselben Vorlage erzeugt weiterhin Slots für alle Teams des Spiels, auch wenn ein anderes
   Item der Vorlage team-eingeschränkt ist.
4. **Zählung stimmt.** `RegenSummary.Created[].Count` für ein team-eingeschränktes Item
   entspricht der Anzahl der tatsächlich getroffenen Teams (nicht `len(teamIDs)` des Spiels).
