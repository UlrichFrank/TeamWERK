## 1. AppShell — excludeRoles-Mechanismus

- [ ] 1.1 `NavItem`-Typ um optionales `excludeRoles?: string[]` erweitern
- [ ] 1.2 Filter-Logik in `visibleItems` um `excludeRoles`-Check ergänzen

## 2. Navigation — Sichtbarkeitsregeln anpassen

- [ ] 2.1 „Mein Profil": `roles` auf `[]` setzen, `excludeRoles: ['admin']` hinzufügen
- [ ] 2.2 „Mitglieder": `roles` auf `['admin', 'vorstand']` ändern (trainer entfernen)
- [ ] 2.3 „Kader": `roles` auf `['admin', 'vorstand', 'trainer']` erweitern

## 3. Backend — Kader-API für Trainer freischalten

- [ ] 3.1 In `main.go` die Kader-Routen in die `RequireRole("admin", "vorstand", "trainer")`-Gruppe verschieben
