# Design — Massen-Regeneration der Dienst-Slots

Alle hier genannten Code-Stellen wurden vor dem Entwurf gelesen; die Aussagen über das
heutige Verhalten sind aus dem Code belegt, nicht erinnert.

## 1. Ein Treiber, keine zweite Engine

`runAutoRegen(ctx, tx, dates, seasonID)` (`internal/games/regen.go:82`) nimmt bereits eine
beliebige Datumsliste und wird vom H4A-Apply über ~146 Spiele in **einer** Transaktion
gefahren (`h4aimport_handler.go:466-488`). Der Massenlauf benutzt exakt dieselbe Funktion.
Eine zweite Regen-Implementierung wäre der sichere Weg in zwei divergierende Verhalten.

Der entscheidende Kniff, der die Engine fast unberührt lässt: **der Treiber schreibt
`games.template_id` vor dem Regen** — in derselben Transaktion:

```
   BEGIN
     ├─ UPDATE games SET template_id = ?  WHERE id IN (…)   ← Zuweisung materialisieren
     ├─ DELETE FROM duty_slots WHERE game_id IN (…purge…)   ← nur für `purge`
     ├─ runAutoRegen(tx, dates, seasonID, skip)             ← liest die NEUEN template_id
     └─ COMMIT   (bzw. ROLLBACK im Dry-Run)
```

Damit müssen **keine** Template-Overrides in die Engine gefädelt werden — sie liest
`games.template_id` wie immer (`regen.go:131-134`). Das ist auch fachlich richtig: die
Zuweisung soll persistent sein, sonst würde die nächste einzelne Spieländerung die
Massenzuweisung wieder überschreiben (es gibt seit dem Wegfall von `findTemplateForGameTx`
keinen ID-basierten Fallback mehr).

Nur zwei Dinge kann die Engine heute nicht und muss sie lernen: **Termine auslassen** (§2)
und **pro Spiel Rechenschaft ablegen** (§3).

## 2. Ausnahme ≠ Kontext

Der unangenehmste Fund der Exploration. Der Regen arbeitet **pro Tag**, nicht pro Spiel:

```
   runAutoRegen(dates) → regenSingleDay(date) → loadDayGames(date)     regen.go:343
                                                      │
                                                      └─ SELECT … FROM games WHERE date=?
                                                         ── ALLE Spiele des Tages, immer
```

Sonntag mit drei Heimspielen, der Nutzer nimmt das mittlere aus → die Engine würde es
trotzdem regenerieren. Die Ausnahme-Menge muss also bis in `loadDayGames` durchgereicht
werden.

Dieselbe Trennung gilt nicht nur für die Checkbox „ausgenommen", sondern für **alle vier
Zustände**: `loadSameDayContextTx` (`regen.go:577`) fragt für `hasPrevDay`/`hasNextDay`
ausschließlich `SELECT COUNT(*) FROM games WHERE date=… AND is_home=1`, unabhängig davon, ob
dieses Nachbarspiel selbst `template`, `none`, `purge` oder „ausgenommen" bekommt und
unabhängig davon, ob es überhaupt im gewählten Zeitraum liegt. Die Dienstoptimierung für
aufeinanderfolgende Heimspiele (an einem Tag über `allGameTimes`, an Folgetagen über
`hasPrevDay`/`hasNextDay`) hängt also **nur an der Existenz** des Nachbar-Heimspiels, nicht an
seiner Dienst-Konfiguration. Das ist bereits heute so (Einzelspiel-Regen betrifft ja auch nur
das eine geänderte Spiel), wird aber im Massenlauf zum ersten Mal beobachtbar, weil dort
bewusst mehrere Nachbartage in einem Lauf unterschiedliche Zustände bekommen. Zwei Fälle
verdienen deshalb einen eigenen Test statt nur eine Behauptung im Proposal (Task 1.4):
Nachbarspiel bekommt im selben Lauf `none`/`purge` (Reduktion muss trotzdem greifen, weil das
Spiel als `is_home=1`-Zeile bestehen bleibt) und Nachbarspiel liegt außerhalb des gewählten
Zeitraums, z. B. der Tag vor `from` (Reduktion muss trotzdem greifen, weil die Query nicht
nach `dates`/Range filtert).

Dabei sind zwei Mengen sauber zu trennen, und die Verwechslung ist der wahrscheinlichste
Folgefehler:

| Menge | Quelle | Ausgenommene Termine … |
|---|---|---|
| **Mutations-Menge** | `loadDayGames` | … sind **nicht** enthalten (ihre Slots bleiben unangetastet) |
| **Kontext-Menge** | `loadSameDayContextTx` (`regen.go:533`) → `allGameTimes`, `hasPrevDay`, `hasNextDay` | … sind **enthalten** |

Grund: `applyBehavior` reduziert oder streicht Dienstarten anhand der *gesamten*
Tageskonstellation (`same_day_behavior`, `adjacent_day_behavior`). Fällt ein ausgenommenes
Spiel aus `allGameTimes`, bekommt das Nachbarspiel plötzlich eine „Kasse" statt einer
„Kasse kurz" — eine stille Falschberechnung, ausgelöst durch eine Checkbox, die eigentlich
„fass das nicht an" bedeutet. `loadSameDayContextTx` ist eine eigene Query und bleibt
deshalb **bewusst unverändert**.

Invariante 3 der Test-Anforderungen prüft genau diesen Fall.

## 3. Live-Vorschau gehört auf den Server

Was ein Template an einem konkreten Tag erzeugt, ist keine Funktion des Templates allein:

```
   applyBehavior(item, gameTime, eventTime, allGameTimes,
                 hasPrevDay, hasNextDay, isBefore, isAfter, isBetween)
```

Ein Client-Nachbau müsste die Reduktionslogik, die Tageskonstellation, die
Team-Auffächerung (`regenGameItems` legt pro Team einen Slot an, `regen.go:457`) und die
Konflikterkennung gegen `is_custom=1`-Slots spiegeln. Er würde driften — und zwar genau
dort, wo der Nutzer sich auf die Vorschau verlässt, um eine destruktive Aktion freizugeben.

Deshalb: **serverseitiger Dry-Run**. Der Preview-Endpoint fährt die vollständige
Apply-Transaktion und schließt mit `ROLLBACK` statt `COMMIT`. Ein Codepfad, keine
Duplikation, per Konstruktion exakt.

- Frontend: Debounce ~400 ms, laufende Anfrage per `AbortController` abbrechen.
- Broadcast-Gate: `Games.PreviewBulkRegen` bekommt einen Allowlist-Eintrag mit Begründung —
  exakt der Präzedenzfall von `Games.PreviewH4AImport` (`internal/arch/broadcast_test.go:79`).
- Lastseite siehe §8.

Nebeneffekt, der die Vorschau nützlicher macht als geplant: öffnet der Nutzer das Modal ohne
Änderung, ist der Default „jeder Termin behält sein aktuelles Template". Die Vorschau zeigt
dann die **Drift** zwischen dem, was die Templates heute erzeugen würden, und dem, was in
`duty_slots` steht — also genau den Zustand, der bisher unsichtbar war (Fall 2 im Proposal).

## 4. Zuweisungs-Restore — der Vertrag

Ohne Rettung ist der Massenlauf unbrauchbar: `duty_slots` → `duty_assignments` hängt an
`ON DELETE CASCADE` (`001_initial.up.sql:151`), ein Lauf über die Restsaison leert den
kompletten künftigen Dienstplan.

Der Schlüssel liegt schon bereit. `snapshotDeletedSlots` (`regen.go:296`) liest heute
`(duty_type_id, event_time, team_id, da.user_id)` — allein, um daraus
„Dein Dienst wurde entfernt"-Pushes zu bauen. Die Query wird um `status`, `cash_amount`,
`fulfilled_at` erweitert, und der Snapshot wird nach den Inserts ein zweites Mal benutzt:

```
   Snapshot                          nach den INSERTs
   ┌──────────────────────┐          ┌──────────────────────┐
   │ duty_type_id  │  7   │          │ duty_type_id  │  7   │
   │ event_time    │17:30 │  ══════▶ │ event_time    │17:30 │  → Match: Zuweisungen zurück
   │ team_id       │  3   │  Match   │ team_id       │  3   │
   │ users [12,19] │      │          │ slots_total   │  2   │
   └──────────────────────┘          └──────────────────────┘
```

Regeln:

- **Match-Schlüssel** ist exakt `(duty_type_id, event_time, team_id)` — derselbe Dreier, den
  `customKey` (`regen.go:247`) für die Konflikterkennung benutzt. Kein Fuzzy-Matching.
- **Kapazität**: höchstens `slots_total` Zuweisungen werden zurückgeschrieben. Bei
  Schrumpfung überleben die **ältesten nach `duty_assignments.id`** — deterministisch und
  begründbar („wer zuerst da war"), nicht zufällig nach Map-Iteration.
- **`slots_filled` muss mitgezogen werden.** Der Zähler ist denormalisiert und wird von Hand
  geführt (`duties/handler.go:967`, kein Trigger). Vergisst man ihn, zeigt die Dienstbörse
  „0/4 besetzt" bei vier eingetragenen Leuten und lässt Doppelbelegung zu.
- **Nicht wiederhergestellte** Zuweisungen laufen unverändert in den bestehenden Pfad
  `buildNotificationIntents` (`regen.go:204`) → „entfernt".
- **Reduzierte Varianten matchen bewusst nicht**: wird „Kasse" zu „Kasse kurz", ist
  `duty_type_id` anders, kein Restore, und die heutige `variant_changed`-Benachrichtigung
  bleibt genau wie sie ist. Das ist richtig — es ist ein anderer Dienst.

## 5. Warum der Restore für **alle** Regen-Pfade gilt

Der Restore könnte auf den Massenlauf beschränkt werden (Flag durch die Engine). Dagegen
spricht: zwei Verhalten in derselben Funktion sind auf Dauer teurer als eines, und der
Einzelspiel-Pfad hat dasselbe Problem im Kleinen — wer heute nur den Spielort eines Spiels
korrigiert, löscht alle Dienstzuweisungen dieses Spiels und schickt allen Betroffenen einen
Push. Das ist eher ein Bug als eine Absicht.

Also: Restore immer. **Bewusst in Kauf genommenes Risiko:** die
Regen-Charakterisierungstests (`internal/games/handler_test.go`, `regen`-Suite) schreiben
das heutige Verhalten teilweise fest und müssen angepasst werden. Das ist der eine Punkt
dieses Changes, an dem ein Test-Rot **nicht** automatisch ein Fehler ist — die Anpassung
gehört begründet in denselben Commit wie die Verhaltensänderung, nicht in einen
„Tests grün machen"-Nachzügler.

## 6. Warum die Vergangenheit gesperrt ist

`duty_accounts.ist` wird beim Regen **nicht** nachgerechnet. `DeleteGame` tut es sorgfältig
(`games/handler.go:1482-1486`, mit Kommentar „so no orphan hours remain"), `regenSingleDay`
nicht — bisher unauffällig, weil ein Einzelspiel-Regen künftige Spiele trifft, die keine
`fulfilled`-Zuweisungen haben.

Ein Massenlauf, der **heute** einschließt, kann den heute Vormittag abgehakten Dienst
treffen: die Zuweisung verschwindet, die Stunden bleiben auf dem Konto stehen.

Zwei Wege: die Neuberechnung aus `DeleteGame` mitziehen, oder das Startdatum auf **morgen**
klemmen. Dieser Change nimmt die Klemme — sie ist vollständig, sofort verifizierbar und
vermeidet, ein bestehendes Loch nebenbei in fremdem Code zu flicken. Das Loch bleibt
dokumentiert (`docs/agent/06-gotchas.md`) und ist ein eigener Folge-Change wert, sobald
jemand die Vergangenheit wirklich braucht.

Konsequenz für die API: `from` ≤ heute → HTTP 400 `range_in_past`. Kein Silent-Clamping —
ein stillschweigend verschobener Zeitraum ist schlimmer als eine Fehlermeldung.

## 7. `purge` ist der einzige unumkehrbare Zustand

| Zustand | `template_id` | `is_custom=0` | `is_custom=1` | Zuweisungen |
|---|---|---|---|---|
| `template` | wird gesetzt | neu aus Template | bleiben | gerettet, wo Schlüssel gleich |
| `none` | `NULL` | gelöscht | **bleiben** | verloren (kein Slot kommt wieder) |
| `purge` | `NULL` | gelöscht | **gelöscht** | verloren |
| ausgenommen | unverändert | unangetastet | unangetastet | unangetastet |

`none` und `purge` unterscheiden sich **ausschließlich** im handgemachten Bestand — sonst
gäbe es keinen Grund für zwei Einträge. Deshalb muss die Vorschau den handgemachten Anteil
pro Zeile ausweisen (`4 Slots · 1 handgemacht`), sonst ist die Wahl zwischen beiden blind.
Der `is_custom`-Split ist heute nicht im `GET /api/games`-Response enthalten
(`handler.go:641` zählt nur `COUNT(DISTINCT ds.id)`) und kommt aus der Preview-Antwort.

Die Kombination `purge` + `notify: false` — löschen *und* schweigen — wird nicht gesperrt
(der Vorstand darf das entscheiden), aber im Modal explizit benannt.

Ebenfalls in die Vorschau gehören die **Konflikte**: fällt ein Template-Slot auf denselben
`(Dienstart, Uhrzeit, Team)` wie ein handgemachter, wird der Template-Slot nicht angelegt
(`regen.go:430-436`, gemeldet als `RegenSummary.Conflicts`). Ohne Ausweis wundert man sich
über „118 erwartet, 114 gekommen".

## 8. Last

1 GB RAM, SQLite im WAL-Modus. Der Dry-Run ist eine **Schreib**transaktion und serialisiert
gegen echte Schreiber, auch wenn er zurückgerollt wird.

Größenordnung: ~40 Termine × ~6 Template-Items ≈ 250 Deletes + Inserts, dazu pro Tag zwei
Kontext-Queries. Mit 400 ms Debounce und abgebrochenen Vorgängern ist die Blockadezeit für
andere Schreiber vernachlässigbar.

**Gemessen** (Task 13.1, `httptest`-Server + `modernc.org/sqlite`, 40 heim-Termine mit
1-Item-Template über ~200 Tage verteilt, `POST .../preview`): **~36 ms** End-to-End
(inkl. HTTP-Roundtrip innerhalb desselben Prozesses). Kein einstelliger Millisekundenbereich
wie zuerst geschätzt, aber immer noch zwei Größenordnungen unter dem 400-ms-Debounce-Fenster
— die ursprüngliche Schlussfolgerung („Blockadezeit vernachlässigbar") hält, nur die Zahl war
zu optimistisch.

## 9. API-Form

Beide Endpoints nehmen **denselben** Request-Body — der Dry-Run ist keine andere Anfrage,
nur ein anderes Ende der Transaktion.

```jsonc
POST /api/duty-slots/bulk-regen/preview      // Vorstand-Tier, Capability bulk_regen_duties
POST /api/duty-slots/bulk-regen/apply
{
  "from": "2026-08-11",          // optional; fehlt → Server liefert Default-Range zurück
  "to":   "2027-06-30",
  "defaults": {                   // Vorbelegung je Terminart
    "heim":      { "action": "template", "template_id": 3 },
    "auswärts":  { "action": "none" },
    "generisch": { "action": "purge" }
  },
  "overrides": [ { "game_id": 43, "action": "template", "template_id": 9 } ],
  "excluded_game_ids": [51, 52],
  "notify": true                  // nur von /apply ausgewertet
}
```

Antwort (beide, `apply` zusätzlich mit `"applied": true`):

```jsonc
{
  "range": { "from": "2026-08-11", "to": "2027-06-30" },   // echoed, inkl. Default
  "rows": [
    { "game_id": 42, "date": "2026-08-14", "time": "18:00",
      "event_name": "Heimspiel vs. TSV", "event_type": "heim",
      "current_template_id": 3, "effective_action": "template", "effective_template_id": 3,
      "excluded": false,
      "slots_before": { "auto": 4, "custom": 1 },
      "slots_after":  { "auto": 4, "custom": 1 },
      "created": 4, "deleted_auto": 4, "deleted_custom": 0,
      "assignments_kept": 2, "assignments_lost": 0, "conflicts": 1 }
  ],
  "totals": { "games": 42, "created": 118, "deleted": 96,
              "custom_kept": 5, "custom_deleted": 0,
              "assignments_kept": 24, "assignments_lost": 7,
              "conflicts": 3, "notified_users": 6 },
  "warnings": []
}
```

Fehlt `defaults` ganz, gilt pro Termin „behalte das aktuelle `template_id`" (bzw. `none`,
wenn es `NULL` ist) — das ist der Öffnungszustand des Modals und der Drift-Report aus §3.

`rows` ist **ungecappt** — die Zeilenliste *ist* das Produkt der Vorschau. Die bestehenden
`RegenSummary`-Listen behalten ihren `summaryCap = 20` (`regen.go:61`).

## 10. Berechtigung

Neue Capability `bulk_regen_duties`, vergeben an `IsVorstandLike` (`policy/rules.go:210`).
Bewusst **enger** als `CapManageDuties`, das auch Trainer-Personas bekommen: ein Lauf über
die Restsaison kann hunderte Slots löschen. Gleiche Begründung und gleiches Tier wie
`CapImportGames`, das aus demselben Grund vom Vorstand-Tier gehalten wird.

Im Frontend hängt das Dropdown am `+ Event`-Split-Button heute an `canImportGames`
(`KalenderPage.tsx:819`) und muss auf `canImportGames || canBulkRegenDuties` erweitert
werden, sonst verschwindet der Pfeil für Personas, die nur eine der beiden Capabilities
haben.

## 11. Bewusst nicht in diesem Change

- **Trainings.** `duty_slots.game_id` referenziert ausschließlich `games`; Mannschafts-
  trainings (`internal/trainings`) haben strukturell keine Dienste. Der Kalender zeigt beide,
  der Massenlauf betrifft nur `games` (inkl. `event_type='generisch'`).
- **Teil-Regen einzelner Dienstarten** („nur die Kasse neu, Rest stehen lassen"). Die Engine
  kennt nur „alle Template-Slots eines Spiels neu"; das wäre ein eigener Zuschnitt.
- **Nachrechnen von `duty_accounts.ist`** — siehe §6.
- **Rückgängig-machen eines Laufs.** Die Vorschau ersetzt es; ein Undo bräuchte einen
  Slot-Verlauf, den es nicht gibt.
- **Team-scoped Template-Items** (aktuell nur in Explore diskutiert, noch kein eigenes
  Proposal): Idee ist eine Team-Allowlist an `game_template_items` (Item entsteht nur für
  Teams aus der Liste, leer = alle — analog zu `audiences`, aber „für wen entsteht der Slot"
  statt „wer darf ihn übernehmen"). Trifft direkt die `for _, tid := range teamIDs`-Schleife
  in `regenGameItems` (`regen.go:456-461`) und damit die Zählung aus §9: `RegenSummary
  .Created[].Count` multipliziert heute mit `len(teamIDs)` (`regen.go:468/474`) — bei
  Team-Filterung pro Item wäre das die gefilterte Teilmenge, nicht mehr alle Spiel-Teams.
  Der Match-Schlüssel des Zuweisungs-Restores (§4, `(duty_type_id, event_time, team_id)`)
  bleibt unberührt, da er `team_id` schon enthält. Wird dieser Change zuerst gemerged, muss
  die Team-Scoping-Erweiterung die `Count`-Berechnung (und die `totals`-Aggregation aus §9)
  entsprechend anpassen.
