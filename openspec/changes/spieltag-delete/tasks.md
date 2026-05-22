# Tasks: Spieltag löschen

## 1. Frontend — SpieltagDetailPage.tsx

- [ ] 1.1 `isAdmin`-Check auf `canEdit` umbenennen: `user?.role === 'admin' || user?.role === 'vorstand' || user?.role === 'trainer'`
- [ ] 1.2 State `showDeleteGame` (boolean) und `deletingGame` (boolean) hinzufügen
- [ ] 1.3 Handler `handleDeleteGame`: `DELETE /api/admin/games/{id}` aufrufen, bei Erfolg `navigate('/spielplan')`
- [ ] 1.4 „Event löschen"-Button in der Header-Zeile einfügen (nur wenn `canEdit`), rot, neben dem Regenerieren-Button
- [ ] 1.5 Bestätigungs-Dialog implementieren: Eventname anzeigen, Hinweis auf mitgelöschte Dienste, Buttons „Abbrechen" und „Endgültig löschen"
- [ ] 1.6 Alle bisherigen `isAdmin`-Vorkommen durch `canEdit` ersetzen

## 2. Verifikation

- [ ] 2.1 Als Admin: Löschen-Button sichtbar, Dialog erscheint, Löschen leitet zu `/spielplan` weiter
- [ ] 2.2 Als Spieler: kein Löschen-Button sichtbar
- [ ] 2.3 Gelöschtes Event erscheint nicht mehr im Kalender
