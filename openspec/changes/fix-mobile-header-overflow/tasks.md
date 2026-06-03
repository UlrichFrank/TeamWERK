## 1. AppShell Root-Fix

- [x] 1.1 In `AppShell.tsx`: `<div className="flex-1 flex flex-col min-h-0">` → `min-w-0` ergänzen

## 2. AdminUsersPage

- [x] 2.1 Header-div (Zeile mit `h1 + Suchfeld + Button`) auf `flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 sm:gap-0` umstellen
- [x] 2.2 Controls-div `flex-wrap` ergänzen damit Suchfeld + Button auf Mobile umbrechen
- [x] 2.3 Beiden Tabellen-Containern `overflow-x-auto` als Klasse hinzufügen

## 3. AdminDutyTypesPage

- [x] 3.1 Header-div auf `flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 sm:gap-0` umstellen

## 4. AdminDutyTemplatesPage

- [x] 4.1 Header-div auf `flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 sm:gap-0` umstellen

## 5. KalenderPage

- [x] 5.1 Header-div (Zeile mit `h1 Kalender` + „Event anlegen"-Button) auf `flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 sm:gap-0` umstellen

## 6. Verifikation

- [ ] 6.1 Alle betroffenen Seiten in DevTools bei 375px prüfen — kein horizontaler Overflow, korrekte Stapelung
- [ ] 6.2 Desktop (> 640px) unverändert sicherstellen
