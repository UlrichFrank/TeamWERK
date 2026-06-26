## Context

Die Datenbank hält `games.template_id` bereits als nullable FK auf `game_templates(id)`. `CreateGame` persistiert die Wahl korrekt; `runAutoRegen` respektiert den Wert (siehe `regen.go:155-165`). Drei Stellen ziehen das Modell aber auseinander:

1. `UpdateGame` (`handler.go:847/851`) listet `template_id` weder im Request-Struct noch im `UPDATE ... SET ...` — der Wert ist nach dem Create eingefroren.
2. `findTemplateForGameTx` (`regen.go:522`) sucht bei `template_id IS NULL` per `ORDER BY id LIMIT 1` ein Template mit passendem `template_type`. Das verhindert, dass „kein Template" als Zustand existiert: irgendein Template ist immer da.
3. Das Frontend macht aus diesem Dilemma einen Kunstgriff: Wer keine Auto-Dienste will, wählt `event_type='generisch'`. Dadurch verschmilzt der Event-**Typ** (heim/auswärts/generisch als Klassifikation) mit der **Slot-Quelle** (Template ja/nein). Ein Heimspiel ohne Auto-Dienste ist nicht ausdrückbar.

`event_type='generisch'` hat im Auto-Regen-Pfad ohnehin schon einen Sonderzweig (`regen.go:151`): es überspringt die Template-Logik und respektiert `is_custom=1`-Slots. Das ist die heute einzige Möglichkeit, „kein Template" zu sein — bezahlt mit einem irreführenden Event-Typ.

## Goals / Non-Goals

**Goals:**
- `games.template_id` als alleinige, explizite, jederzeit änderbare Quelle der Wahrheit für die Slot-Vorlage.
- Klare Trennung von **Event-Typ** (`heim`/`auswärts`/`generisch`) und **Slot-Quelle** (Template oder NULL).
- Saubere „Kein-Template"-Semantik: NULL = keine Auto-Slots, unabhängig vom `event_type`.
- Bestehendes Verhalten für aktuelle Events nicht still kippen lassen.

**Non-Goals:**
- Neue Default-/Aktiv-Mechanik für Templates (kein `is_default` UNIQUE pro Typ — explizit Option A im Explore).
- Bulk-Template-Wechsel über mehrere Events (separate Idee, nicht hier).
- Aufräumen der `is_active`-Spalte in `game_templates` (das Dashboard nutzt es noch; ein eigener Cleanup-Change kann das später adressieren).
- Aufräumen der Doppel-Bedeutung von `event_type='generisch'` (= Event-Klasse + Slot-Trigger für `is_custom=1`). Nach diesem Change ist `event_type='generisch'` weiter der Schalter, der `req.Slots`-Custom-Persistierung beim Create freischaltet. Eine spätere Iteration kann das von `event_type` auf eine eigene Markierung umstellen.

## Decisions

### Entscheidung: NULL = „keine Auto-Dienste" (Option A)

**Gewählt:** `template_id IS NULL` bedeutet im Auto-Regen-Pfad: dieses Event wird nicht aus einer Vorlage generiert. `is_custom=1`-Slots des Events bleiben unverändert; `is_custom=0`-Slots werden bei Regen gelöscht und nicht ersetzt.

**Warum:** Saubere Semantik. „Kein Template" muss als Zustand existieren, sonst bleibt der heutige Kunstgriff über `event_type` notwendig. Komfort (Default-Vorauswahl im Anlege-Dialog) ist eine UI-Sache, kein DB-Verhalten.

**Alternative verworfen:** Default-Template pro `template_type` (`is_default` UNIQUE pro Typ). Bringt versteckte Magie zurück — derselbe Event „wechselt" Vorlage, wenn der Admin den Default umstellt. Der Explore-Dialog mit dem User hat diese Variante explizit verworfen.

### Entscheidung: `findTemplateForGameTx` ersatzlos entfernen

**Gewählt:** Die Funktion wird gelöscht; der Aufrufer in `regenSingleDay` springt bei `NOT g.TemplateID.Valid` direkt zu `continue`. Damit fällt auch der `effectiveEventDurationTx`-Fallback-Pfad für unbekannte Templates weg — wenn keine Vorlage da ist, gibt es keinen Slot, also auch keine Dauer-Berechnung.

**Warum:** Solange der Fallback existiert, ist „NULL" nicht wirklich NULL — irgendein Template wird trotzdem aufgelöst. Das ist die zentrale Verwirrung.

### Entscheidung: `PUT /api/admin/games/{id}` — Partial-Update für `template_id`

**Gewählt:** Das Request-Struct bekommt `TemplateID json.RawMessage` (oder ein Wrapper-Typ). Drei Fälle:

| Body-Inhalt                  | Verhalten                          |
|------------------------------|------------------------------------|
| Feld fehlt                   | `template_id` bleibt unverändert   |
| `"template_id": null`        | `template_id := NULL`              |
| `"template_id": 7`           | `template_id := 7` (FK-Check)      |

**Warum:** PUT in diesem Codebase ist semantisch ein gemischter Full/Partial-Update (siehe `rsvp_opt_out`, `rsvp_require_reason` — beide bereits Partial). Bestands-Clients, die `template_id` nicht senden, dürfen nicht versehentlich NULL setzen. Gleichzeitig muss explizites `null` möglich sein, sonst kann der User die Vorlage nicht entfernen.

**Implementierungshinweis:** `json.RawMessage` ist im Codebase noch nicht etabliert; alternativ ein Sentinel-`*int` mit einer zusätzlichen `template_id_set bool`-Flag aus einem manuellen `UnmarshalJSON`. Entscheidung an den Implementierer — die Anforderung steht im Spec.

### Entscheidung: Backfill-Migration für Bestands-Events

**Gewählt:** Neue Migration `006_template_id_backfill.up.sql` setzt für jedes Game mit `template_id IS NULL` AND `event_type IN ('heim','auswärts')` den Wert auf das Ergebnis der heutigen Fallback-Logik (kleinste passende `game_templates.id`).

```sql
UPDATE games
SET template_id = (
    SELECT id FROM game_templates
    WHERE template_type = games.event_type
    ORDER BY id LIMIT 1
)
WHERE template_id IS NULL
  AND event_type IN ('heim','auswärts');
```

**Warum:** Ohne Backfill würden Bestands-Events nach Deploy beim nächsten Auto-Regen ihre Slots verlieren. Die Migration konserviert exakt den Status quo — danach ist NULL ein bewusster Zustand. `event_type='generisch'`-Events haben heute schon `NULL` und sollen das behalten.

**Down-Migration:** Setzt `template_id` für die backfill-betroffenen Zeilen zurück auf NULL. Realistisch ist das nicht von NULL unterscheidbar — die Down-Migration ist „best effort" und im Kommentar dokumentiert.

### Entscheidung: Frontend-Form-Verhalten

**Gewählt:** 
- Beim Anlegen: Dropdown zeigt **„— Keine Vorlage (keine Auto-Dienste) —"** als erste Option mit Wert `null`. Default-Selektion: erste Vorlage mit passendem `template_type`, falls vorhanden — purer UI-Komfort, der DB-Default bleibt NULL.
- Beim Bearbeiten: Dropdown zeigt den aktuell gesetzten Wert. Wechsel auf „Keine Vorlage" sendet `template_id: null`.
- Das frühere „Ohne Dienste"-Toggle entfällt. Wer das Verhalten will, wählt „Keine Vorlage".

**Hinweistext** in `AdminDutyTemplatesPage.tsx` „Vorlagen werden nur initial verwendet" wird entfernt.

## Risks / Trade-offs

- **Bestands-Events verlieren ihren impliziten Fallback.** Mitigiert durch Backfill-Migration. Risiko-Rest: wenn nach der Migration ein Admin manuell `template_id = NULL` setzt und annimmt, das System fängt's mit dem Fallback auf — passiert nicht mehr. Akzeptiert.
- **`event_type='generisch'` bleibt überladen.** Es ist weiter Event-Klasse UND der Schalter, der `req.Slots`-Custom-Persistierung beim Create freischaltet. Mit diesem Change ist die Slot-Quelle auch für `generisch` über `template_id` ausdrückbar — generische Templates (mit eigenem `duration_minutes`) waren bisher praktisch ungenutzt und dienen genau diesem Workflow. Bei `event_type='generisch'` UND `template_id != null` koexistieren `req.Slots`-Custom-Slots (`is_custom=1`) und template-basierte Auto-Slots (`is_custom=0`); die bestehende Konfliktlogik in `regen.go` (customSlots-Map → ConflictEntry) verhindert Doppel-Inserts und meldet sie im `regen_summary`.
- **PUT mit explizitem `null` ist im Codebase neu.** Implementierungs-Aufwand für `UnmarshalJSON` oder `json.RawMessage`. Klein, aber nicht null.
- **Down-Migration nicht verlustfrei.** Akzeptiert.

## Test-Anforderungen

| Route / Verhalten | Testname | Erwarteter Status / Invariante |
|---|---|---|
| `POST /api/admin/games` mit `template_id=null`, `event_type=heim` | `TestCreateGame_HomeWithoutTemplate` | 201; `games.template_id IS NULL`; nach Regen keine `is_custom=0`-Slots am Event |
| `POST /api/admin/games` mit `template_id=null`, `event_type=generisch`, `slots[]` befüllt | `TestCreateGame_GenericWithCustomSlots` | 201; alle `slots[]` als `is_custom=1` persistiert; keine Template-Slots |
| `POST /api/admin/games` mit `template_id=<gen-tpl>`, `event_type=generisch` | `TestCreateGame_GenericWithTemplate` | 201; Auto-Slots aus generisch-Template (Dauer aus `duration_minutes`) |
| `PUT /api/admin/games/{id}` ohne `template_id`-Feld im Body | `TestUpdateGame_TemplateIDFieldOmitted_Preserves` | 200; `template_id` unverändert |
| `PUT /api/admin/games/{id}` mit `template_id: null` | `TestUpdateGame_TemplateIDExplicitNull_SetsNull` | 200; `games.template_id IS NULL`; Auto-Regen löscht `is_custom=0`-Slots |
| `PUT /api/admin/games/{id}` mit `template_id: 7` (Wechsel) | `TestUpdateGame_TemplateIDChange_RegeneratesSlots` | 200; alte Slots weg, neue aus Template 7 vorhanden |
| `runAutoRegen` für Event mit `template_id IS NULL` | `TestAutoRegen_NullTemplate_NoSlotsGenerated` | keine `is_custom=0`-Slots am Event; `is_custom=1`-Slots erhalten |
| Backfill-Migration | `TestMigration006_BackfillsHeimAuswaertsNull` | nach Up-Migration haben alle `heim`/`auswärts`-Events mit vorher NULL nun ein Template (kleinste passende ID); `generisch`-Events bleiben NULL |
