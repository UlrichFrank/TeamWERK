## Context

TeamWERK nutzt aktuell `usePaginatedFetch` mit einem akkumulierenden „Mehr laden"-Muster (limit=50). Beide betroffenen Seiten (`MembersPage`, `AdminUsersPage`) verwenden diesen Hook identisch. Das Backend liefert bereits `{ items, total }` mit `limit`/`offset`-Parametern — keine Backend-Änderungen nötig.

Die TYPO3-Pagination-Vorlage des Projekts definiert das exakte visuelle Muster: 7 feste Slots in einer `<nav>`, kreisförmige Buttons (w-10 h-10), Gelb für navigierbare Seiten, Schwarz für die aktive Seite, opacity-30 + cursor-not-allowed für deaktivierte Slots.

## Goals / Non-Goals

**Goals:**
- URL-State (`?page=N&search=X`) via React Router `useSearchParams`
- Exaktes TYPO3-Slot-Muster: « (page-3) (page-1) [page] (page+1) (page+3) »
- Geteilte, wiederverwendbare `<Pagination>`-Komponente
- limit=20 als neuer Standard für die beiden Listen

**Non-Goals:**
- Backend-Änderungen
- Mobile-spezifisches Pagination-Layout
- `usePaginatedFetch` anfassen oder entfernen
- Weitere Seiten über MembersPage + AdminUsersPage hinaus

## Decisions

### 1. URL-State via useSearchParams (React Router)
`useSearchParams()` aus `react-router-dom` liest und schreibt `?page=N&search=X` direkt in der URL, ohne manuelles `history.pushState`. Seitenänderung via `setSearchParams({ page: String(n), search })` — React Router behandelt das als Navigation, Browser-History wird korrekt befüllt.

Beim Suchen: `setSearchParams({ search: val, page: '1' })` — immer auf Seite 1 zurücksetzen.

### 2. Neuer Hook usePagination (nicht usePaginatedFetch erweitern)
`usePaginatedFetch` hat akkumulierende Semantik (items wachsen). Das umzubauen würde bestehende Nutzer brechen. Stattdessen: neuer schlanker Hook mit klarer Seiten-Semantik:

```
usePagination<T>(endpoint: string, limit = 20)
  → { items, total, currentPage, totalPages, loading, error }
  liest page/search aus URL, fetchData ersetzt items komplett
```

Keine `loadMore`-Funktion — das Konzept existiert in diesem Hook nicht.

### 3. Pagination-Komponente: reine Darstellung
`<Pagination currentPage totalPages onPageChange>` ist vollständig stateless. Die Slot-Berechnung:

```
slot 1: «   → page 1 (deaktiviert wenn currentPage === 1)
slot 2: –3  → currentPage - 3 (deaktiviert wenn < 1)
slot 3: –1  → currentPage - 1 (deaktiviert wenn < 1)
slot 4:     → currentPage (immer aktiv, schwarz)
slot 5: +1  → currentPage + 1 (deaktiviert wenn > totalPages)
slot 6: +3  → currentPage + 3 (deaktiviert wenn > totalPages)
slot 7: »   → totalPages (deaktiviert wenn currentPage === totalPages)
```

Deaktivierte Slots: `<span>` statt `<button>`, gleiche Größe, opacity-30, cursor-not-allowed. Aktive Seite: `bg-black text-white`. Navigierbare Seiten: `bg-brand-yellow text-black hover:bg-black hover:text-brand-yellow`.

Komponente rendert `null` wenn `totalPages <= 1` (kein Pagination-Bar nötig).

### 4. Suchfeld-Debounce bleibt
300ms Debounce aus `usePaginatedFetch` wird in `usePagination` übernommen. Beim Feuern des Debounce wird immer auf page=1 zurückgesetzt.

## Risks / Trade-offs

- **URL-Schreibhäufigkeit**: Jeder Seitenklick schreibt die URL — normal für SPAs, kein Problem.
- **Initialer Load**: Wenn jemand `/mitglieder?page=5` öffnet und nur 60 Mitglieder da sind (3 Seiten), wird page=5 als ungültig behandelt → clamp auf totalPages nach dem ersten Fetch.
- **Slot-Muster bei wenigen Seiten**: Bei 2 Seiten zeigt das Muster viele deaktivierte Slots (–3, –1, +3 alle aus). Das entspricht exakt dem TYPO3-Verhalten und ist gewünscht.
