## 1. Backend — DELETE family-links Endpoint

- [x] 1.1 `DeleteFamilyLink`-Handler in `internal/members/handler.go` implementieren: JSON-Body mit `parent_user_id` und `member_id` lesen, `DELETE FROM family_links WHERE parent_user_id=? AND member_id=?` ausführen, 204 bei Erfolg, 404 wenn keine Zeile betroffen
- [x] 1.2 Route in `cmd/teamwerk/main.go` registrieren: `r.Delete("/api/admin/family-links", membH.DeleteFamilyLink)` in der Admin-Gruppe

## 2. Backend — Limit in CreateFamilyLink

- [x] 2.1 In `CreateFamilyLink` vor dem INSERT prüfen: `SELECT COUNT(*) FROM family_links WHERE member_id=?`; wenn >= 2 → HTTP 409 zurückgeben

## 3. Frontend — Mannschafts-Sektion entfernen

- [x] 3.1 Den gesamten „Mannschaft zuweisen"-Block aus `MemberDetailPage.tsx` entfernen (inkl. `selectedTeam`, `selectedSeason`, `isPrimary` State, `handleAssignTeam`, die Select-Felder und den Button)
- [x] 3.2 `api.get('/admin/teams')` und das Season-Laden in `useEffect` entfernen, sofern nicht anderweitig benötigt; `teams`-State entfernen

## 4. Frontend — Erziehungsberechtigte umbenennen und Dropdown öffnen

- [x] 4.1 Überschrift „Elternteile" → „Erziehungsberechtigte" in `MemberDetailPage.tsx`
- [x] 4.2 Button-Label „Verknüpfen" / Beschriftungen konsistent auf „Erziehungsberechtigten hinzufügen" anpassen
- [x] 4.3 Dropdown-Filter anpassen: `users.filter(u => u.role === 'elternteil' && ...)` → `users.filter(u => !linkedParents.some(p => p.id === u.id))`

## 5. Frontend — Entfernen-Button und Max-2-Sperre

- [x] 5.1 Neben jedem verknüpften Erziehungsberechtigten einen Entfernen-Button rendern
- [x] 5.2 Klick auf Entfernen: `DELETE /api/admin/family-links` mit `{parent_user_id: p.id, member_id: id}` aufrufen, danach `loadLinkedParents()` aufrufen; bei Fehler Toast anzeigen
- [x] 5.3 Hinzufügen-Button und Dropdown deaktivieren/ausblenden wenn `linkedParents.length >= 2`

## 6. Abschluss

- [ ] 6.1 Manuell testen: Erziehungsberechtigten hinzufügen (mit Nutzer aller Rollen), entfernen, dritten Eintrag verhindern
- [x] 6.2 Sicherstellen dass die Stammdaten-Sektion und Nutzer-Verknüpfungs-Sektion keine Regression haben
