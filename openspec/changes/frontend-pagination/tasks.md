## 1. Hook: usePagination

- [ ] 1.1 `web/src/lib/usePagination.ts` anlegen: generischer Hook `usePagination<T>(endpoint: string, limit = 20)` — liest `page` und `search` aus `useSearchParams()`, gibt `{ items, total, currentPage, totalPages, loading, error, setSearch }` zurück
- [ ] 1.2 Im Hook `fetchData(page, search)` implementieren: `offset = (page - 1) * limit`, GET-Request, Items werden komplett ersetzt (nicht akkumuliert)
- [ ] 1.3 Suchbegriff-Debounce (300ms) implementieren: beim Feuern `setSearchParams({ page: '1', search })` setzen
- [ ] 1.4 Ungültige Seite clampen: nach Fetch, wenn `page > totalPages && totalPages > 0` → `setSearchParams({ page: String(totalPages), search })` setzen

## 2. Komponente: Pagination

- [ ] 2.1 `web/src/components/Pagination.tsx` anlegen mit Props `{ currentPage: number; totalPages: number; onPageChange: (page: number) => void }`
- [ ] 2.2 Slot-Berechnung implementieren: Array von 7 Slot-Definitionen `{ type: 'first' | 'page' | 'last', target: number | null }` — target=null wenn außerhalb 1..totalPages
- [ ] 2.3 Slot-Rendering: navigierbarer Slot → `<button>` mit `w-10 h-10 flex items-center justify-center rounded-full bg-brand-yellow text-black text-sm font-medium transition-colors hover:bg-black hover:text-brand-yellow`; deaktivierter Slot → `<span>` gleiche Klassen plus `opacity-30 cursor-not-allowed`; aktive Seite → `bg-black text-white font-semibold` (kein Hover)
- [ ] 2.4 Icons für erste/letzte Seite: `«` und `»` (wie TYPO3-Vorlage)
- [ ] 2.5 Früher Return `null` wenn `totalPages <= 1`
- [ ] 2.6 Komponente in `<nav aria-label="Seitennavigation" className="flex justify-center items-center gap-2 mt-8 mb-4">` wrappen

## 3. MembersPage umstellen

- [ ] 3.1 Import von `usePaginatedFetch` durch `usePagination` ersetzen, `loadMore` und `items.length < total`-Check entfernen
- [ ] 3.2 `<Pagination>` importieren und unterhalb der Mitglieder-Liste/Tabelle einbinden: `currentPage={currentPage}` `totalPages={totalPages}` `onPageChange={p => setSearchParams({ page: String(p), search: currentSearch })}`
- [ ] 3.3 „Mehr laden"-Button entfernen
- [ ] 3.4 Suchfeld weiterhin mit `setSearch` verdrahten (Hook kümmert sich um URL-Update)

## 4. AdminUsersPage umstellen

- [ ] 4.1 Import von `usePaginatedFetch` durch `usePagination` ersetzen, `loadMore` und `items.length < total`-Check entfernen
- [ ] 4.2 `<Pagination>` unterhalb der Nutzer-Tabelle/Liste einbinden
- [ ] 4.3 „Mehr laden"-Button entfernen
- [ ] 4.4 Suchfeld weiterhin mit `setSearch` verdrahten

## 5. Qualitätssicherung

- [ ] 5.1 Browser-Back/Forward auf /mitglieder nach Seitenwechsel prüfen
- [ ] 5.2 URL `/mitglieder?page=3&search=müller` direkt aufrufen — korrekte Seite und Suche laden
- [ ] 5.3 Ungültige Seite (`?page=999`) aufrufen — clampt auf letzte Seite
- [ ] 5.4 Suche eingeben auf Seite 3 — springt auf Seite 1
- [ ] 5.5 Slot-Darstellung bei wenigen Seiten (2–3) prüfen — deaktivierte Slots korrekt
