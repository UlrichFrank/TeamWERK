# Tasks

## 1. Migration

- [x] 1.1 `internal/db/migrations/054_dienst_abloesung.up.sql` — `end_at_next_duty INTEGER
      NOT NULL DEFAULT 0 CHECK(end_at_next_duty IN (0,1))` auf `duty_types` **und**
      `game_template_items`. Kommentar: rein additiv, Default `0` = Bestandsverhalten,
      kein Backfill.
- [x] 1.2 `054_dienst_abloesung.down.sql` — beide Spalten per `DROP COLUMN` (Reihenfolge
      wie in `053` gespiegelt).

## 2. Backend: Persistenz und API

- [x] 2.1 `internal/duties/handler.go` — `end_at_next_duty` in `CreateType`, `UpdateType`
      und `ListTypes` (SELECT, Scan, JSON-Feld). Keine neue Validierung; im Modus
      `absolut` wird der Wert gespeichert, aber nicht angewendet.
- [x] 2.2 `internal/games/handler.go` — Feld je Item in `UpdateTemplate` und beim Lesen
      der Vorlage. Copy-on-pick vom Diensttyp beim Hinzufügen einer Zeile.
- [x] 2.3 Tests: `TestCreateType_AbloesungWirdPersistiert`,
      `TestUpdateType_AbloesungWirdPersistiert`,
      `TestUpdateType_AbloesungUnauthentifiziert`,
      `TestListTypes_AbloesungImResponse`,
      `TestUpdateTemplate_AbloesungJeItemWirdPersistiert`,
      `TestUpdateTemplate_AbloesungOhneVorstandsrecht`.

## 3. Regen: Kandidaten sammeln (Phase 1)

- [x] 3.1 `internal/games/regen.go` — `templateItemRow` und `dutyTypeDuration`
      (`lookupDutyTypeTx`) um `EndAtNextDuty bool` erweitern; bei Varianten-Reduktion
      gewinnt der Wert des Varianten-Diensttyps (wie Modus/End-Anker/End-Versatz).
- [x] 3.2 `insertOne` liefert die neue Slot-ID (`LastInsertId`) statt nur `bool`.
- [x] 3.3 `regenGameItems` sammelt je erzeugtem Slot mit gesetztem Kennzeichen einen
      Eintrag `{SlotID, DutyTypeID: resultDutyTypeID, StartTime: eventTime}` und gibt
      die Liste zusätzlich zurück; `regenSingleDay` akkumuliert sie über die Spiele.

## 4. Regen: Kappung (Phase 2)

- [x] 4.1 `internal/games/regen.go` — `applyChainCaps(ctx, tx, date, seasonID, candidates)`,
      aufgerufen in `regenSingleDay` **nach** der Pro-Spiel-Schleife:
      ein `SELECT duty_type_id, event_time FROM duty_slots WHERE event_date=? AND
      season_id=?`, je Kandidat die früheste `event_time` desselben `duty_type_id` mit
      `event_time > StartTime`, dann `UPDATE duty_slots SET hours_value=?` nur wenn die
      Ablösung vor dem bestehenden Ende liegt.
- [x] 4.2 Doc-Kommentar an `applyChainCaps`: warum ein Nachlauf über reale Slots statt
      eines Prepass (die sechs Gates), und die Invariante „kappt nur, löscht nie"
      (design.md §3/§4).
- [x] 4.3 Tests: `TestRegen_DienstEndetBeiAbloesung`,
      `TestRegen_LetzterDienstDerKetteBehaeltDenDeckel`,
      `TestRegen_NachfolgerNachDemDeckelKapptNicht`,
      `TestRegen_NachfolgerVorDemEigenenStartWirdIgnoriert`,
      `TestRegen_AbloesungNurDurchDenselbenDiensttyp`,
      `TestRegen_VarianteBestimmtDasKennzeichen`,
      `TestRegen_ManuellerSlotLoestAbAberWirdNichtGekappt`,
      `TestRegen_AusgenommenerTerminLoestTrotzdemAb`,
      `TestRegen_KappungErzeugtNieEineNichtpositiveDauer`,
      `TestRegen_OhneKennzeichenUnveraendert`.

## 5. Frontend: Maske `/diensttypen`

- [x] 5.1 `web/src/pages/AdminDutyTypesPage.tsx` — Häkchen „Endet spätestens bei Ablösung
      durch den nächsten gleichartigen Dienst am selben Tag" unter End-Anker/-Versatz,
      nur im Modus „Startzeit + Endzeit" sichtbar; Wert im Payload.
- [x] 5.2 Vitest `AdminDutyTypesPage.abloesung.test.tsx`: Sichtbarkeit an den Modus
      gekoppelt, Wert bleibt beim Moduswechsel hin und zurück erhalten, wird gesendet.

## 6. Frontend: Maske `/dienstplan-vorlagen`

- [x] 6.1 `web/src/pages/AdminDutyTemplatesPage.tsx` — dasselbe Häkchen je Item.
- [x] 6.2 `web/src/lib/dutyTemplateItems.ts` — Copy-on-pick übernimmt `end_at_next_duty`
      vom gewählten Diensttyp.
- [x] 6.3 Vitest `AdminDutyTemplatesPage.abloesung.test.tsx` + Ergänzung in
      `dutyTemplateItems.refresh.test.ts`.

## 7. Abschluss

- [x] 7.1 `docs/agent/06-gotchas.md` — Absatz „Dienst-Zeitmodus" um die Ablösung
      erweitern: Kappung statt drittem Modus, Nachlauf über reale Slots, Invariante
      „kappt nur, löscht nie", Vorschau zeigt den Deckel.
- [x] 7.2 `/verify-change` (Build/Test/Lint + Invarianten). Archivierung steht noch aus.
