## Context

Der `/duty-board`-Endpunkt liefert aktuell pro Slot nur aggregierte Daten (`slots_total`, `vacancies`, `claimed_by_me`) — keine Assignee-Namen. Die Assignee-Liste ist nur für Admins/Trainer via `GET /duty-slots/:id/assignments` abrufbar und enthält keine Privacy-Felder.

Das Datenschutz-Modell ist bereits vorhanden: `user_visibility` (phones_visible, address_visible, photo_visible), `user_phones` für Telefonnummern, `users.photo_path`/`street`/`zip`/`city` für Kontaktdaten.

## Goals / Non-Goals

**Goals:**
- Assignee-Namen (+ optional Avatar, Telefon, Adresse) im Board-Response einbetten
- Privacy-Filterung serverseitig — der Client bekommt nur freigegebene Felder
- Kein Extra-Request für die Tooltip-Anzeige (alles in einem Response)
- Kein Bruch der bestehenden Admin-Zuteilungs-Ansicht

**Non-Goals:**
- Neue DB-Tabellen oder neue API-Routen
- Änderung der Admin-Zuteilungsansicht (Status-Badges, Geldersatz)
- Echtzeit-Updates der Assignee-Liste (SSE-Events für `duties` reloaden den Board ohnehin)

## Decisions

### Entscheidung 1: Assignee-Daten inline im Board-Response

**Gewählt:** `boardSlot` um `assignees []publicAssignee` erweitern; alle Daten in einem Query.

**Alternativen:**
- *Lazy per Tooltip*: separater Endpoint `GET /duty-slots/:id/assignees`, nur on hover fetchen — sauberer, aber Latenz + Loading-Spinner im Tooltip
- *Hybrid* (Namen inline, Kontaktdaten lazy): zwei Requests, mehr Komplexität ohne messbaren Vorteil

**Begründung:** Typisches Board hat 10–30 Slots mit je 0–3 Assignees. Die Zusatzlast ist minimal (Namen + Foto-URL + optional Telefon/Adresse in Textform). Kein Round-Trip auf Tooltip-Hover macht die UX flüssiger.

### Entscheidung 2: Privacy-Filterung im SQL-Query

**Gewählt:** JOIN auf `user_visibility`; `CASE WHEN uv.photo_visible=1 THEN u.photo_path END` etc. direkt im Query.

**Alternative:** Alle Daten holen, in Go filtern. Gleichwertig, aber unnötig Daten transferieren.

**Begründung:** SQLite-seitige Filterung ist idiomatisch für dieses Projekt und hält den Go-Code schlank.

### Entscheidung 3: Subquery für Telefonnummern

`user_phones` ist eine 1:n-Tabelle. Da SQLite kein `ARRAY_AGG` als JSON kennt (außer `json_group_array`, verfügbar ab SQLite 3.38), wird ein separater Query pro Slot für Telefonnummern ausgeführt — oder `json_group_array` verwendet falls die SQLite-Version es unterstützt.

**Gewählt:** Telefonnummern mit `json_group_array(json_object('label', p.label, 'number', p.number))` aggregieren — modernc.org/sqlite bündelt SQLite ≥ 3.45, also kein Kompatibilitätsproblem.

### Entscheidung 4: Tooltip — JS-gesteuert, keine externe Library

CSS-only `:hover` würde auf Mobile nicht funktionieren. Statt einer Tooltip-Library (Floating UI etc.) wird ein einfacher `useState`-Toggle verwendet: click öffnet/schließt, `onMouseEnter`/`onMouseLeave` auf Desktop.

**Alternative:** `@floating-ui/react` für Positionierung. Overkill für diesen Use-Case; manuelles Positioning mit `absolute`/`z-50` reicht.

## Risks / Trade-offs

- **N+1 bei großen Boards** → Der Telefonnummern-JOIN ist in einem Query via `json_group_array` gelöst; kein N+1
- **Privacy-Leak bei Bug** → Unit-testbar: `CASE WHEN uv.photo_visible=1` ist deterministisch; kein Application-Layer-Routing
- **Tooltip-Positionierung am Rand** → Einfaches `right-0`/`left-0` je nach Position; kein Viewport-Overflow-Handling. Akzeptables Trade-off für die Komplexität
- **Board-Response-Größe** → Bei vollbesetztem Board mit 30 Slots à 3 Personen + Telefon ~5 KB Overhead. Vernachlässigbar
