## 1. Backend: game_id Filter

- [x] 1.1 In `internal/duties/handler.go` → `Board()`: Nach dem `view=mine`-Block einen `?game_id=`-Query-Parameter auslesen und bei Vorhandensein `AND ds.game_id = ?` an `whereParts` anhängen sowie den Wert an `args` appenden

## 2. Frontend: DutySlotList extrahieren

- [x] 2.1 Datei `web/src/components/DutySlotList.tsx` anlegen mit Props: `slots: BoardSlot[]`, `isPast: boolean`, `canEdit: boolean`, `onReload: () => void`, `onSlotDeleted?: (id: number) => void`
- [x] 2.2 Slot-Rendering-Logik (Tabelle mit Claim/Unclaim-Buttons, Expand-Row, Assignment-Tabelle mit Fulfill/Geldersatz, Delete mit Bestätigungsdialog) aus `DutyPage.tsx` in `DutySlotList.tsx` verschieben — inkl. `StatusBadge`, interner State für `expanded`, `assignments`, `cashAmount`, `deleteConfirm`
- [x] 2.3 `BoardSlot`-Interface und `StatusBadge`-Komponente in `DutySlotList.tsx` definieren (oder separates `types/duty.ts` — aber nur wenn es auch `DutyPage` sauber hält)

## 3. DutyPage auf DutySlotList umstellen

- [x] 3.1 In `DutyPage.tsx` die extrahierte Slot-Rendering-Logik durch `<DutySlotList>` ersetzen — ein `<DutySlotList>`-Element pro `BoardGroup`, `isPast={g.past}`, `onReload={load}`
- [x] 3.2 Sicherstellen dass `BoardGroup`-Interface und `formatDate` in `DutyPage` verbleiben (sie sind seitenspezifisch)

## 4. SpieltagDetailPage: Board-Daten laden

- [x] 4.1 In `SpieltagDetailPage.tsx` State für Board-Daten hinzufügen: `boardSlots: BoardSlot[]`, `boardLoading: boolean`
- [x] 4.2 `BoardSlot`-Interface aus `DutySlotList` importieren
- [x] 4.3 `loadBoard()`-Funktion implementieren: `api.get('/duty-board?game_id=' + gameId)` → erste Gruppe nehmen (oder leeres Array), `setBoardSlots`
- [x] 4.4 `loadBoard()` parallel zu `loadGame()` im initialen `useEffect` aufrufen
- [x] 4.5 SSE-`useLiveUpdates`-Hook für `'duties'`-Events ergänzen (falls noch nicht vorhanden): ruft `loadBoard()` auf
- [x] 4.6 Nach Slot-Mutation (Add, Delete) auch `loadBoard()` aufrufen

## 5. SpieltagDetailPage: Slot-Darstellung ersetzen

- [x] 5.1 Die bestehende Slot-Liste (ProgressBar + Edit/Delete-Buttons im `div.divide-y`) durch `<DutySlotList slots={boardSlots} isPast={...} canEdit={canEdit} onReload={loadBoard} onSlotDeleted={...}>` ersetzen
- [x] 5.2 `isPast` aus dem Spieldatum berechnen: `new Date(game.date.slice(0,10) + 'T23:59:59') < new Date()`
- [x] 5.3 `ProgressBar`-Komponente und `SlotDetail`-Interface aus `SpieltagDetailPage.tsx` entfernen (SlotDetail wird nur noch für Add/Edit-Modals genutzt — prüfen ob Interface noch nötig ist)
- [ ] 5.4 Manuelles Testen: Eintragen/Austragen auf `/kalender/{id}` funktioniert; `/dienste` verhält sich unverändert
