## Why

Listen-Seiten wie `/mitglieder` und `/admin/nutzer` laden derzeit 50 Einträge auf einmal und bieten nur einen „Mehr laden"-Button. Das macht es unmöglich, direkt auf eine bestimmte Seite zu springen, behindert das Teilen von Links auf gefilterte Ergebnisse und skaliert schlecht wenn die Mitgliederzahl wächst. Eine echte Seitennavigation mit URL-State löst all das.

## What Changes

- Neuer Hook `web/src/lib/usePagination.ts`: seitenbasierte Variante von `usePaginatedFetch`, liest `?page` und `?search` aus der URL, ersetzt Items komplett beim Seitenwechsel
- Neue Komponente `web/src/components/Pagination.tsx`: 7 feste Slots exakt wie die TYPO3-Vorlage (`«  –3  –1  [aktiv]  +1  +3  »`), inaktive Slots als deaktivierte Platzhalter
- `MembersPage.tsx`: Umstieg auf `usePagination`, limit=20, „Mehr laden" entfernt, `<Pagination>` eingebunden
- `AdminUsersPage.tsx`: gleiche Umstellung

## Capabilities

### New Capabilities
- `paginated-list-navigation`: Seitennavigation mit URL-State für Listen-Seiten

### Modified Capabilities

## Impact

- Nur Frontend-Änderungen — Backend unterstützt `limit`+`offset` bereits
- `usePaginatedFetch` bleibt unverändert erhalten
- Browser-Back/Forward funktioniert korrekt nach der Umstellung
- Direkte URLs wie `/mitglieder?page=3&search=müller` sind teilbar und bookmarkbar
