## 1. Backend — Board-Response erweitern

- [x] 1.1 `publicAssignee`-Struct in `internal/duties/handler.go` definieren: `Name string`, `PhotoURL *string`, `Phones []phoneEntry`, `Address *string`
- [x] 1.2 `boardSlot`-Struct um `Assignees []publicAssignee` erweitern
- [x] 1.3 `GetBoard`-Query: LEFT JOIN auf `duty_assignments`, `users`, `user_visibility` ergänzen; `CASE WHEN`-Ausdrücke für `photo_url` und `address` (Straße + PLZ + Ort zusammengesetzt) hinzufügen
- [x] 1.4 Telefonnummern per `json_group_array(json_object(...))` aus `user_phones` aggregieren (nur wenn `phones_visible=1`); `phoneEntry`-Struct (Label, Number) und JSON-Deserialisierung im Handler implementieren
- [x] 1.5 Assignees in den jeweiligen `boardSlot` beim Row-Scan einsetzen

## 2. Frontend — BoardSlot Interface & Assignee-Sub-Row

- [x] 2.1 `BoardSlot`-Interface in `DutySlotList.tsx` um `assignees?: PublicAssignee[]` erweitern (`PublicAssignee`: `name`, `photo_url?`, `phones?`, `address?`)
- [x] 2.2 Sub-Row unter jeder Slot-Zeile in `DutySlotList.tsx` rendern: zeigt Assignee-Namen (+ Avatar wenn `photo_url` vorhanden), nur wenn `assignees.length > 0`
- [x] 2.3 Avatar-Rendering: `<img>`-Tag mit rundem Clip (`rounded-full w-5 h-5 object-cover`) falls `photo_url`, sonst kein Avatar

## 3. Frontend — AssigneeTooltip-Komponente

- [x] 3.1 `AssigneeTooltip`-Komponente erstellen: Props `assignee: PublicAssignee`; rendert Tooltip-Inhalt mit Name, optionalem Foto, optionalen Telefonnummern, optionaler Adresse
- [x] 3.2 Desktop-Hover: `onMouseEnter`/`onMouseLeave` auf dem Assignee-Chip schaltet Tooltip ein/aus
- [x] 3.3 Mobile-Tap: `onClick` togglet Tooltip; Außen-Klick via `useEffect` + `mousedown`-Listener schließt Tooltip
- [x] 3.4 Tooltip-Positionierung: `absolute z-50` relativ zum Chip-Container; `bottom-full` als Default, kein Viewport-Overflow-Handling nötig

## 4. Verifikation

- [x] 4.1 Lokalen Dev-Server starten, `/dienste` aufrufen: Assignee-Namen erscheinen unter belegten Slots
- [x] 4.2 Tooltip auf Desktop prüfen: Hover zeigt Kontaktdaten gemäß Freigaben
- [x] 4.3 Tooltip auf Mobile prüfen (Browser-DevTools Responsive): Tap öffnet/schließt Tooltip
- [x] 4.4 Nutzer ohne freigegebene Daten: Tooltip zeigt nur Namen (kein leerer Tooltip-Bug)
- [x] 4.5 `/kalender/:id`-Seite prüfen: `DutySlotList` zeigt dort ebenfalls Assignees (kein Extra-Aufwand, da gleiche Komponente)
