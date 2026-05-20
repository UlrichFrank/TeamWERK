## ADDED Requirements

### Requirement: Seitennavigation zeigt 7 feste Slots
Die `<Pagination>`-Komponente SHALL exakt 7 Slots rendern: `«`, `page-3`, `page-1`, `[aktiv]`, `page+1`, `page+3`, `»`. Slots außerhalb des gültigen Seitenbereichs (< 1 oder > totalPages) werden als deaktivierter Platzhalter gerendert (`–`, opacity-30, cursor-not-allowed). Die aktive Seite wird mit schwarzem Hintergrund und weißem Text dargestellt. Navigierbare Seiten haben gelben Hintergrund.

#### Scenario: Slot außerhalb des Bereichs wird deaktiviert
- **WHEN** currentPage=2 und totalPages=4 → slot `page-3` = -1 (ungültig)
- **THEN** Slot zeigt `–` als `<span>` mit opacity-30, kein klickbarer Button

#### Scenario: Erste-Seite-Slot deaktiviert auf Seite 1
- **WHEN** currentPage=1
- **THEN** Slot `«` ist deaktiviert (span, opacity-30)

#### Scenario: Letzte-Seite-Slot deaktiviert auf letzter Seite
- **WHEN** currentPage === totalPages
- **THEN** Slot `»` ist deaktiviert

#### Scenario: Aktive Seite visuell hervorgehoben
- **WHEN** currentPage=3
- **THEN** Slot 4 zeigt `3` mit bg-black text-white, kein Hover-Effekt

### Requirement: Pagination rendert nichts bei einer Seite
Die Komponente SHALL `null` zurückgeben wenn `totalPages <= 1`.

#### Scenario: Zu wenige Einträge für Pagination
- **WHEN** total=15, limit=20 → totalPages=1
- **THEN** kein Pagination-Element im DOM

### Requirement: Seitenzustand liegt in der URL
Der `usePagination`-Hook SHALL `?page=N` und `?search=X` aus `useSearchParams` lesen. Seitenänderungen SHALL via `setSearchParams` in die URL geschrieben werden, sodass Browser-Back/Forward funktioniert.

#### Scenario: Direktaufruf einer Seite per URL
- **WHEN** Nutzer öffnet `/mitglieder?page=3`
- **THEN** Hook lädt sofort Seite 3 (offset=40 bei limit=20), Pagination zeigt Seite 3 aktiv

#### Scenario: Browser-Back nach Seitennavigation
- **WHEN** Nutzer navigiert von Seite 1 → 3 → Browser-Back
- **THEN** URL springt auf `?page=1`, Hook lädt Seite 1 neu

#### Scenario: Teilbarer Link
- **WHEN** Nutzer befindet sich auf `/mitglieder?page=2&search=müller`
- **THEN** dieser URL in neuem Tab öffnet dieselbe Seite mit gleichen Ergebnissen

### Requirement: Neue Suche setzt Seite auf 1 zurück
Wenn der Suchbegriff im `usePagination`-Hook ändert, SHALL die Seite auf 1 zurückgesetzt werden.

#### Scenario: Suchbegriff eingeben setzt Seite zurück
- **WHEN** Nutzer ist auf Seite 3, gibt neuen Suchbegriff ein (nach 300ms Debounce)
- **THEN** URL wird zu `?page=1&search=neuerbegriff`, Ergebnisse von Seite 1

### Requirement: Ungültige Seite wird geclampt
Wenn die URL `?page=N` enthält und N > totalPages (z.B. durch alten Link), SHALL die letzte vorhandene Seite geladen werden.

#### Scenario: page=99 bei 3 Seiten
- **WHEN** URL enthält `?page=99`, Backend liefert total=45 (3 Seiten bei limit=20)
- **THEN** Hook setzt currentPage auf 3 und lädt Seite 3

### Requirement: MembersPage und AdminUsersPage nutzen usePagination mit limit=20
Beide Seiten SHALL `usePagination` statt `usePaginatedFetch` verwenden. Der „Mehr laden"-Button SHALL entfernt werden. `<Pagination>` SHALL unterhalb der Tabelle/Liste platziert werden.

#### Scenario: MembersPage zeigt maximal 20 Einträge
- **WHEN** 60 Mitglieder in der DB, Nutzer ist auf Seite 1
- **THEN** genau 20 Mitglieder sichtbar, Pagination zeigt 3 Seiten

#### Scenario: AdminUsersPage zeigt maximal 20 Nutzer
- **WHEN** 35 Nutzer in der DB
- **THEN** Seite 1 zeigt 20, Seite 2 zeigt 15, kein „Mehr laden"-Button
