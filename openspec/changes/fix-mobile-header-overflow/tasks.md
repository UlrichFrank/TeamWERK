## 1. AdminUsersPage Header-Fix

- [ ] 1.1 Header-div von `flex items-center justify-between gap-3` auf `flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 sm:gap-0` ändern
- [ ] 1.2 Controls-div (`flex gap-2` mit Suchfeld + Button) sicherstellen, dass er auf Mobile korrekt umbrochen wird — `flex-wrap` ergänzen falls nötig

## 2. AdminDutyTypesPage Header-Fix

- [ ] 2.1 Header-div auf `flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 sm:gap-0` umstellen
- [ ] 2.2 Prüfen ob der „+ Neu"-Button auf Mobile volle Breite oder Auto-Breite haben soll (analog MembersPage)

## 3. AdminDutyTemplatesPage Header-Fix

- [ ] 3.1 Header-div auf `flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 sm:gap-0` umstellen
- [ ] 3.2 Prüfen ob der „+ Neue Vorlage"-Button auf Mobile volle Breite oder Auto-Breite haben soll

## 4. Verifikation

- [ ] 4.1 Alle drei Seiten im Browser-DevTools bei 375px (iPhone SE) und 390px (iPhone 15 Pro) prüfen — kein horizontaler Overflow
- [ ] 4.2 Desktop-Darstellung (> 640px) unverändert sicherstellen
