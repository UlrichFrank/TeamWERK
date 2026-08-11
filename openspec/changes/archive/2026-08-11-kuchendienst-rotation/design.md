## Context

Der Auto-Regen (`internal/games/regen.go`) erzeugt Duty-Slots pro Spiel. `regenSingleDay` lädt die Spiele eines Tages (`loadDayGames`) und ruft für jedes Spiel einzeln `regenGameItems` auf; dort läuft für jedes Vorlagen-Item (`game_template_items`) eine Schleife über `teamIDs` — die Teams **dieses einen Spiels** (`loadGameTeamIDsTx`) — und legt pro Team einen Slot an. `team_id` auf einem Slot ist damit heute immer „das Team, das an diesem Spiel beteiligt ist", niemals abgeleitet.

Ein zweiter bestehender Mechanismus, `team_ids` auf `game_template_items` (additive Spalte aus dem Change `duty-template-team-scope`, Migration `044`), filtert bereits **welche** dieser Spiel-Teams einen Slot bekommen — bleibt aber auf derselben 1:1-Beziehung „Team spielt mit → Team bekommt Slot" aufgebaut.

Kuchendienst (`duty_types.id=11`, Item `game_template_items.id=135` auf dem Heimspiel-Template) braucht ein grundsätzlich anderes Modell: die Kuchen eines Spieltags werden reihum unter den Mannschaften verteilt, unabhängig davon, wer welches konkrete Spiel bestreitet. Das erfordert eine **tagesweite** Vorausberechnung, bevor die bestehende Pro-Spiel-Schleife läuft.

`restoreAssignments`/`snapshotDeletedSlots`/`loadNewAutoSlotsKeyed` matchen Zusagen über den Dreier-Schlüssel `customKey{DutyTypeID, EventTime, TeamID, HasTeam}` (`regen.go:314-527`). Dieser Schlüssel ist die Grundlage für „welche Zusage überlebt einen Regen" — für Rotations-Items ist `TeamID` aber kein stabiles Merkmal mehr (siehe Decision 5).

## Goals / Non-Goals

**Goals:**
- Pro Spieltag eine Team-Warteschlange aufbauen (Reihenfolge = chronologischer Anpfiff, jedes Team einmalig bei seinem ersten Heimspiel des Tages) und Kuchen-Slots greedy mit Cap darauf verteilen.
- Konfigurierbares Spiele→Kuchen-Verhältnis (vereinsweit) und Cap pro Team (pro Vorlagen-Item), additiv zum Bestandsschema.
- Bestehendes Verhalten für alle Items ohne gesetzten Cap bleibt exakt unverändert (Rückwärtskompatibilität, wie schon bei `team_ids`/Migration 044 demonstriert).

**Non-Goals:**
- Keine Interaktion mit `same_day_behavior`/`adjacent_day_behavior` (Varianten-Wechsel je Spielposition am Tag). Ein Item mit gesetztem Rotations-Cap MUSS `same_day_behavior='normal'` und `adjacent_day_behavior='normal'` haben — serverseitig erzwungen (siehe Decision 4). Das deckt sich mit dem Bestandswert des Kuchen-Items (`same_day_behavior=normal`).
- Kein saisonweiter/persistenter Rotationszustand — jeder Spieltag startet die Warteschlange bei Position 1 (Nutzer-Entscheidung aus dem Explore-Gespräch).
- Kein Mehrfach-Kuchen-Bedarf pro einzelnem Spiel — pro Spiel entsteht höchstens ein Rotations-Slot (siehe Decision 2 zur Verhältnis>1-Begrenzung).
- Keine rückwirkende Neuberechnung von `duty_accounts.ist` — folgt demselben bekannten Rest wie beim Massen-Regen (`duty-bulk-regen`).

## Decisions

### 1. Tagesweite Vorausberechnung als neuer Schritt vor der Pro-Spiel-Schleife

`regenSingleDay` bekommt einen neuen Schritt zwischen `loadDayGames` und der bestehenden Pro-Spiel-Schleife: `buildRotationPlan(ctx, tx, dayGames, seasonID)`. Diese Funktion:

1. Filtert `dayGames` auf `EventType == "heim"` (auswärts/generisch tragen keine eigene `team_id`-Zuordnung und zählen nicht mit).
2. Lädt für jedes verbleibende Spiel dessen Template-Items (bestehende `loadTemplateItemsTx`) und dessen Team (`loadGameTeamIDsTx`, erstes Element — mehrere Teams an einem eigenen Heimspiel sind der seltene Fall gemeinsamer Ansetzungen; jedes davon tritt unabhängig in die Warteschlange ein, sobald es zuerst auftaucht).
3. Gruppiert nach `duty_type_id` der rotations-aktivierten Items (`rotation_max_per_team IS NOT NULL`) — typischerweise nur Kuchen, das Modell bleibt aber generisch für weitere Bewirtungs-Dienste.
4. Baut je Gruppe die Team-Warteschlange (distinct, erstes Auftreten) und berechnet `kuchenBedarf = min(homeGameCount, ceil(homeGameCount * verhältnis))` (Deckelung, siehe Decision 2).
5. Weist die ersten `kuchenBedarf` Heimspiele (chronologisch) je einen Slot zu, Team via Greedy-Cap-Zuteilung; überzählige Slots (Warteschlange erschöpft) bekommen `team_id = NULL`.

Ergebnis: `map[dutyTypeID]map[gameID]rotationAssignment{hasSlot bool, teamID sql.NullInt64}`, das an `regenGameItems` durchgereicht wird.

**Alternative verworfen:** Rotation innerhalb der bestehenden Pro-Spiel-Schleife berechnen (z. B. mit einem `*rotationState`-Zeiger, der zwischen den Iterationen weitergereicht wird). Verworfen, weil `kuchenBedarf` von der **Gesamtzahl** der Heimspiele des Tages abhängt — die Schleife müsste die Tagesliste vorher kennen, was faktisch dieselbe Vorausberechnung ist, nur unsauber über Closures verteilt statt als benannter, testbarer Schritt.

### 2. Verhältnis > 1 wirkt nur bis zur Spieleanzahl

Pro Design entsteht **höchstens ein** Rotations-Slot pro Spiel (kein Mehrfachbedarf innerhalb eines einzelnen Spiels). Ein Verhältnis > 1.0 hat deshalb ab dem Punkt, wo `Bedarf > Anzahl Heimspiele`, keine zusätzliche Wirkung — der Bedarf wird auf die Spieleanzahl gedeckelt (`min(...)` in Decision 1, Schritt 4). Das Einstellungs-UI validiert das nicht gesondert (kein Fehler bei Werten > 1), dokumentiert die Deckelung aber im Hilfetext.

**Alternative verworfen:** mehrere Kuchen-Slots pro Spiel bei Verhältnis > 1 (z. B. 1.5 → jedes zweite Spiel bekommt zwei Slots). Verworfen als Over-Engineering für einen Fall, der im Explore-Gespräch nicht verlangt wurde — das Verhältnis-Feature zielt auf „weniger Kuchen als Spiele", nicht „mehr".

### 3. Bei Verhältnis < 1: die ersten N Heimspiele bekommen den Slot, nicht gleichmäßig verteilt

Wenn `kuchenBedarf < homeGameCount`, bekommen die **chronologisch ersten** `kuchenBedarf` Heimspiele des Tages je einen Rotations-Slot; die übrigen Heimspiele bekommen für dieses Item gar keinen Slot (kein `Skipped`-Eintrag nötig — analog zu einem Item, dessen Team-Allowlist kein Team des Spiels trifft, `regen.go:702-709`).

**Alternative verworfen:** gleichmäßige Verteilung über den Tag (z. B. jedes zweite Spiel). Verworfen, weil das für die Team-Warteschlange keinen Unterschied macht (sie zählt Teams, nicht Slot-Positionen) und „die ersten N" deterministisch und ohne zusätzlichen Konfigurationsbedarf ist.

### 4. Rotations-Cap und `same_day_behavior`/`adjacent_day_behavior` schließen sich aus

`PUT /api/admin/duty-templates/{id}` (Item-Validierung, analog zur bestehenden `team_ids`-Existenzprüfung, `handler.go:1863ff`) lehnt ein Item mit gesetztem `rotation_max_per_team` UND einem `duty_types`-Verhalten `same_day_behavior != 'normal'` oder `adjacent_day_behavior != 'normal'` mit `400 rotation_requires_normal_behavior` ab.

**Warum:** Ohne diese Einschränkung müsste die Rotation zusätzlich mit variantenwechselndem `resultDutyTypeID` (`applyBehavior`, `regen.go:625-626`) umgehen — die Warteschlange würde dann teils für Duty-Type A, teils für dessen Variante B geführt, mit unklarer Bedeutung von „Cap pro Team". Das bestehende Kuchen-Item hat bereits `same_day_behavior=normal`, die Einschränkung kostet also im Ist-Zustand nichts.

### 5. Restore-Matching für Rotations-Items ignoriert `team_id`

`snapshotCustomSlots`, `snapshotDeletedSlots`/`restoreAssignments`, `loadNewAutoSlotsKeyed` bekommen einen zusätzlichen Parameter `rotationTypes map[int]bool` (Menge der `duty_type_id`, die für dieses Spiel rotations-aktiv sind — abgeleitet aus den bereits geladenen `items` vor dem Lösch-Schritt in `regenSingleDay`). Beim Aufbau von `customKey` wird `TeamID`/`HasTeam` **nicht gesetzt**, wenn `rotationTypes[dutyTypeID]` wahr ist — unabhängig davon, ob die Slot-Zeile selbst einen `team_id`-Wert trägt:

```go
k := customKey{DutyTypeID: ds.DutyTypeID, EventTime: ds.EventTime}
if ds.TeamID.Valid && !rotationTypes[ds.DutyTypeID] {
    k.TeamID = ds.TeamID.Int64
    k.HasTeam = true
}
```

**Warum:** `team_id` ist bei Rotations-Items aus der tagesweiten Reihenfolge abgeleitet, keine stabile Spiel-Eigenschaft. Verschiebt sich die Reihenfolge (neues Spiel eingefügt, Zeit geändert), verschiebt sich potenziell die Team-Zuordnung aller nachfolgenden Slots des Tages — ein Match über `team_id` würde bestehende Zusagen grundlos als „verloren" melden, obwohl derselbe Spiel-Zeitpunkt weiterhin einen Kuchen braucht. Die Zusage bleibt so am Termin hängen, auch wenn sich die Team-Zurechnung durch den Regen verschiebt.

**Trade-off, bewusst akzeptiert:** eine Person könnte nach einem Regen „ihrem" ursprünglichen Team nicht mehr zugeordnet sein, obwohl ihre Zusage für den Termin bestehen bleibt. Das ist der geringere Schaden gegenüber einer grundlos verworfenen Zusage (siehe Risks).

**Alternative verworfen:** bei jeder Spielplanänderung an einem Rotations-Tag alle Kuchen-Zusagen des Tages verwerfen und neu anfragen. Verworfen als zu aggressiv für den Normalfall „ein Spiel verschiebt sich um 15 Minuten" — dort ändert sich die Zusage-Relevanz überhaupt nicht.

### 6. Konfigurationsspeicher: einfacher Direkt-Read statt Store-Cache

Die neue Einstellung „Spiele-zu-Kuchen-Verhältnis" wird **nicht** nach dem `Store`/`atomic.Bool`/Poll-Loop-Muster von `internal/settings/store.go` (Wartungsmodus) gebaut. Dieses Muster existiert dort, weil die Mitte-Middleware auf jedem Request ohne DB-Roundtrip antworten muss. Das Verhältnis wird nur **einmal pro Regen-Lauf** gelesen (kein Hot-Path) — ein einfacher `SELECT value FROM system_settings WHERE key='bewirtung_verhaeltnis'` innerhalb derselben `tx`, mit der auch `regenSingleDay` arbeitet, reicht aus und vermeidet einen zusätzlichen In-Memory-Cache, der mit der Transaktion nicht konsistent gehalten werden müsste.

Row-Anlage (`INSERT OR IGNORE ... 'bewirtung_verhaeltnis', '1'`) läuft idempotent in der neuen Migration, analog zum Default-Row-Vorbild von `maintenance_mode`.

### 7. Neue `RegenSummary`-Kategorie „Unassigned" für Team-lose Kuchen-Slots

`RegenSummary` bekommt ein additives Feld `Unassigned []UnassignedEntry{Date, DutyType, GameID}` (gleiches Cap-Verhalten wie `Skipped`/`Created`, `capSummary` ergänzt). Ein Slot landet dort, wenn die Team-Warteschlange erschöpft ist, aber laut `kuchenBedarf` trotzdem ein Kuchen für dieses Spiel nötig ist (Decision 1, Schritt 5). Der Slot selbst entsteht ganz normal (`team_id=NULL`, offen für alle berechtigten Nutzer über die bestehende Audience-Filterung) — die neue Kategorie ist reine Sichtbarkeit im Regen-Ergebnis, damit der Vorstand den Engpass bemerkt, ohne die Dienstbörse durchsuchen zu müssen.

## Risks / Trade-offs

- **[Risiko] Team-Zurechnung „springt" bei Spielplanänderungen.** Siehe Decision 5 — eine Person kann nach einem Regen einem anderen Team zugerechnet erscheinen als bei ihrer Zusage. → **Mitigation:** keine im Backend; Frontend zeigt die *aktuelle* Team-Zuordnung des Slots, nicht die zum Zusage-Zeitpunkt — das ist bereits das bestehende Anzeigeverhalten für alle Slots (keine Historisierung).
- **[Risiko] Mehrere Teams an einem eigenen Heimspiel.** Trägt ein Spiel mehrere Club-Teams (`game_teams`), treten alle unabhängig in die Warteschlange ein — ihre relative Reihenfolge untereinander ist die DB-Rückgabereihenfolge von `loadGameTeamIDsTx`, nicht fachlich definiert. → **Mitigation:** keine; seltener Randfall (gemeinsame Ansetzungen), bestehendes Verhalten der Team-Schleife hat dieselbe Uneindeutigkeit bereits heute.
- **[Risiko] Item-Umkonfiguration zwischen zwei Regen-Läufen.** Wird der Rotations-Cap eines Items nachträglich gesetzt/entfernt, nutzt Decision 5 beim nächsten Regen die *aktuelle* Konfiguration für Alt-Slots, die unter der alten Konfiguration entstanden. → **Mitigation:** keine; Konfigurationsänderungen an Vorlagen-Items sind grundsätzlich nicht rückwirkend versioniert (bestehendes Verhalten für `team_ids`, `slots_count` etc.).
- **[Trade-off] Verhältnis > 1 ohne Wirkung über die Spieleanzahl hinaus** (Decision 2) — bewusst vereinfacht, kein Use-Case aus dem Explore-Gespräch verlangt echte Mehrfach-Slots pro Spiel.

## Migration Plan

Rein additiv, keine Datenmigration, Migration `045_bewirtungsrotation` (nächste freie Nummer — vor Task-Umsetzung `ls internal/db/migrations/ | sort -V | tail -1` erneut prüfen):

1. `ALTER TABLE game_template_items ADD COLUMN rotation_max_per_team INTEGER;` — nullable, kein Default nötig (`NULL` = Rotation deaktiviert).
2. `INSERT OR IGNORE INTO system_settings (key, value) VALUES ('bewirtung_verhaeltnis', '1');` — idempotenter Default, analog `maintenance_mode`.
3. Down: `ALTER TABLE game_template_items DROP COLUMN rotation_max_per_team;` / `DELETE FROM system_settings WHERE key='bewirtung_verhaeltnis';`. Nach Rollback verlieren bereits gepflegte Caps ihre Wirkung (Spalte weg) — Items fallen zurück auf das bestehende Pro-Team-Verhalten, kein Datenverlust an anderer Stelle.

Kein Backfill, kein Downtime-Fenster.

## Open Questions

- Soll das Verhältnis-Setting perspektivisch pro Saison statt vereinsweit gelten (falls sich die Spieltag-Länge saisonabhängig ändert)? Nicht Teil dieses Change — `system_settings` ist heute grundsätzlich saisonunabhängig (wie `maintenance_mode`); eine Saison-Bindung wäre ein eigener Folge-Change, falls der Bedarf entsteht.
- Soll ein unzugeordneter Kuchen-Slot (Decision 7) eine eigene Benachrichtigung an den Vorstand auslösen (Push/E-Mail), statt nur im Regen-Summary sichtbar zu sein? Nicht spezifiziert — für v1 reicht die Sichtbarkeit im Preview/Apply-Response, analog zu `Conflicts`.
