## 1. Backend

- [x] 1.1 `GetProfile` in `internal/members/handler.go` erweitern: für `spieler` verknüpfte Elternteile via `family_links` abfragen und als `parents`-Feld in der Response ergänzen
- [x] 1.2 Response-Struktur von `GetProfile` auf Objekt umstellen: `{ members: [...], parents: [...] }` statt reinem Array

## 2. Frontend — Navigation

- [x] 2.1 In `AppShell.tsx` den `/mitglieder`-Eintrag auf `roles: ['admin', 'trainer']` einschränken

## 3. Frontend — Profil-Seite

- [x] 3.1 `ProfilePage.tsx` auf neue Response-Struktur (`members` + `parents`) anpassen
- [x] 3.2 Sektion „Meine Familie" implementieren: für `elternteil` verknüpfte Kinder anzeigen (aus `members`), für `spieler` verknüpfte Elternteile anzeigen (aus `parents`), read-only Karten
