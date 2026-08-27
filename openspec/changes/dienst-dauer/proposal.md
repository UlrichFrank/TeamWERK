## Why

Die Dienstbörse zeigt zu jedem Slot einen **Zeitpunkt** (`duty_slots.event_time`, gerendert
als `{s.event_time || '—'}` in `DutySlotList.tsx:164`). Wer sich einträgt, sieht wann es
losgeht, aber nicht wie lange es dauert. Genau diese Information entscheidet, ob jemand
zusagt.

Eine Dauer ist im Datenmodell längst da — sie heißt nur nicht so, wo man sie braucht.
`duty_types.hours_value` ist auf `/diensttypen` bereits als **„Dauer"** beschriftet
(`AdminDutyTypesPage.tsx:96`), nimmt Eingaben der Form `1h 30min` entgegen und wird als
`REAL` in Stunden geführt. Sie erreicht die Anzeige aber nie: keine Vorlage und kein Slot
trägt sie, und die Dienstbörse liest sie nicht.

Damit fehlt an drei Stellen dasselbe:

1. `/dienste` zeigt `8:00` statt `8:00–9:00`.
2. Das Spieltag-Detail-Modal (`/kalender`) zeigt dieselbe verkürzte Angabe.
3. „Dienst bearbeiten" und „+ Dienst hinzufügen" lassen Uhrzeit und Personenzahl
   überschreiben, aber nicht die Dauer — obwohl gerade sie am Einzeltermin abweicht
   (ein Kuchendienst an einem Vierfach-Spieltag dauert nicht so lang wie an einem
   Einzelspiel).

## What Changes

- **`hours_value` ist die Dauer.** Kein zweites Zahlenfeld, keine Umbenennung auf
  `/diensttypen`. Die Kopplung an die Anrechnung ist **gewollt**: „8:00–9:00" heißt per
  Definition „eine Stunde aufs Dienstkonto". Wer die Spanne korrigiert, korrigiert bewusst
  auch die Gutschrift.
- Die Größe bekommt zwei weitere Ebenen, jeweils `REAL` in Stunden:
  `game_template_items.hours_value` (**NEU**) und `duty_slots.hours_value` (**NEU**).
- **Copy-on-pick**, exakt das Muster von `anchor`/`offset_minutes`: der Vorlagen-Editor
  kopiert die Dauer beim Auswählen des Diensttyps in die Zeile, danach ist sie
  eigenständig. Kein `NULL` = „erben".
- `runAutoRegen` **materialisiert** die aufgelöste Dauer in den Slot — wie `event_time`,
  `slots_total` und `audiences` heute schon. Ein Slot fragt nie zur Laufzeit die Vorlage.
- **Bei `reduced` gewinnt die Variante.** Tauscht `applyBehavior` den Diensttyp gegen
  `same_day_variant_id`/`adjacent_day_variant_id`, kommt die Dauer aus dem **Varianten-Typ**,
  nicht aus der Vorlagen-Zeile. Begründung: die Variante ist ein *anderer Dienst* — die
  Vorlage bestimmt **wann und wie viele**, der Diensttyp bestimmt **welche Arbeit**, und die
  Dauer gehört zur Arbeit.
- `duty_accounts.ist` aggregiert künftig `SUM(ds.hours_value)` statt `SUM(dt.hours_value)` —
  der Slot ist die maßgebliche Quelle, weil er überschrieben sein kann.
- „+ Dienst hinzufügen" belegt Dauer **und** Uhrzeit aus dem gewählten Diensttyp vor
  (Uhrzeit aus `default_anchor` + `default_offset_minutes` gegen die Anstoßzeit des
  Termins). Heute prefillt das Modal nur die Zielgruppen; beide Felder bleiben editierbar.
- Anzeige: `8:00–9:00` (Halbgeviertstrich, ohne „Uhr"). Ein über Mitternacht laufender
  Dienst zeigt `23:30–00:30` ohne Datumszusatz.

**Bewusst akzeptierte Folge:** nach dem Deploy trägt **jeder** Dienst eine Spanne, auch
fristgebundene wie der Spielbericht-Dienst (`hours_value=0.5`, Anker Ende+24h) — der zeigt
dann `20:00–20:30`, obwohl er keine Schicht, sondern eine Frist ist. Die Alternative (Flag
„als Zeitspanne anzeigen" je Diensttyp) wurde verworfen, um kein zweites Konzept
einzuführen.

**Nicht Teil dieses Changes:** `duty_accounts.ist` wird von `Fulfill` bis heute nicht
erhöht — geschrieben wird die Spalte nur beim Anlegen (`INSERT OR IGNORE (…, 0)`,
`duties/handler.go:1136`) und in der Neuberechnung von `DeleteGame`
(`games/handler.go:1532`). Das ist ein eigenständiger Fehler, der hier nur die Quelle der
Aggregation wechselt, nicht die fehlende Buchung ergänzt.

## Capabilities

### New Capabilities

_(keine)_

### Modified Capabilities

- `duties`: Ein Dienst-Slot trägt eine eigene Dauer; die Dienstbörse zeigt eine Zeitspanne
  statt eines Zeitpunkts; Anlegen und Bearbeiten eines Slots umfassen die Dauer.

`game-deletion-cascade` bleibt unverändert — die Anforderung „Konto-Konsistenz bei
Cascade-Delete" spricht von „den Stunden dieses Dienstes", ohne die Spalte zu nennen, und
bleibt nach dem Quellenwechsel wörtlich wahr.

## Test-Anforderungen

| Route | Testname (Vorschlag) | Erwartung / Invariante |
|---|---|---|
| `GET /api/duty-board` | `TestBoard_LiefertHoursValueJeSlot` | Jeder Slot im Board-Response trägt `hours_value` aus `duty_slots`, nicht aus `duty_types`. |
| `POST /api/duty-slots` | `TestCreateSlot_UebernimmtHoursValue` | 201; der angelegte Slot trägt die mitgeschickte Dauer und `is_custom=1`. |
| `POST /api/duty-slots` | `TestCreateSlot_OhneHoursValueErbtVomTyp` | Fehlt `hours_value` im Request, wird die Dauer des `duty_type_id` eingesetzt (nicht 0). |
| `PUT /api/duty-slots/{id}` | `TestUpdateSlot_AendertHoursValue` | 204; Dauer ist überschrieben, `is_custom=1` gesetzt. |
| `PUT /api/duty-slots/{id}` | `TestUpdateSlot_HoursValueNullOderNegativ_400` | `hours_value <= 0` wird mit 400 abgelehnt statt gespeichert (sonst zeigt die Börse `8:00–8:00`). |
| `PUT /api/game-templates/{id}` | `TestUpdateTemplate_SpeichertHoursValueJeItem` | Vorlagen-Zeile trägt ihre eigene Dauer, unabhängig vom aktuellen Typ-Wert. |
| — (Regen) | `TestRegen_SlotErbtHoursValueAusVorlage` | Ein aus der Vorlage erzeugter Slot trägt die Dauer der **Vorlagen-Zeile**, auch wenn der Diensttyp inzwischen einen anderen Wert hat. |
| — (Regen) | `TestRegen_ReducedVarianteBestimmtDauer` | Greift `same_day_behavior='reduced'`, trägt der Slot die Dauer des **Varianten-Typs**, nicht die der Vorlagen-Zeile. |
| — (Regen) | `TestRegen_DauerAenderungErhaeltZusagen` | Eine geänderte Dauer verschiebt `event_time` nicht; `restoreAssignments` stellt alle Zusagen wieder her. |
| — (Konto) | `TestDeleteGame_IstNutztSlotDauer` | Nach dem Löschen eines Termins mit überschriebenem Slot rechnet `duty_accounts.ist` mit der **Slot**-Dauer, nicht mit der des Typs. |

**Garantierte Invariante:** Die auf `/dienste` angezeigte Spanne und die Stundengutschrift
für denselben Slot stammen aus **einem** Wert (`duty_slots.hours_value`) — sie können nicht
auseinanderlaufen.

## Impact

- **Migration:** `internal/db/migrations/052_dienst_dauer.{up,down}.sql` — je eine Spalte
  `hours_value REAL NOT NULL DEFAULT 1.0` auf `game_template_items` und `duty_slots`,
  Backfill per korreliertem Subselect aus `duty_types` (Copy-on-pick rückwirkend).
- `internal/games/regen.go` — `templateItemRow` + `loadTemplateItemsTx` (Z. 1169) um
  `gti.hours_value`; `insertOne` (Z. 936) schreibt die Spalte; bei `isReduce` (Z. 906) die
  Dauer der Variante nachladen statt nur den Namen.
- `internal/games/handler.go` — Template-CRUD (Z. 2020, Delete-then-Insert) trägt
  `hours_value`; `ist`-Neuberechnung (Z. 1532) wechselt auf `ds.hours_value` und verliert
  den JOIN auf `duty_types`.
- `internal/duties/handler.go` — `Board` (Z. 802) und `ListSlots` liefern `hours_value`;
  `CreateSlot`/`UpdateSlot` nehmen es entgegen und validieren `> 0`.
- `web/src/lib/` — `hoursToDisplay`/`parseHoursInput` wandern aus `AdminDutyTypesPage.tsx`
  in einen geteilten Helfer, dazu `formatTimeSpan(event_time, hours_value)` mit
  Mitternachtsüberlauf.
- `web/src/components/DutySlotList.tsx` (Z. 164) — Spanne statt Zeitpunkt.
- `web/src/components/SpieltagDetailModal.tsx` — Dauer-Feld in beiden Modalen; Add-Modal
  prefillt Dauer + Uhrzeit aus dem Diensttyp.
- `web/src/pages/AdminDutyTemplateDetailPage.tsx` (Z. 247/340) — `hours_value` in den
  bestehenden Copy-on-pick-Spread aufnehmen, Feld je Zeile.
- **Unberührt:** `restoreAssignments`/`makeCustomKey` — der Match-Dreier
  `(duty_type_id, event_time, team_id)` enthält keine Dauer, Zusagen überleben eine
  Dauer-Änderung ohne Zutun. `buildRotationPlan` ebenfalls: Rotation verteilt Kuchen, keine
  Zeiten. `DutyPage.tsx:382` bleibt — das ist die Anstoßzeit des Spiels, kein Slot.
