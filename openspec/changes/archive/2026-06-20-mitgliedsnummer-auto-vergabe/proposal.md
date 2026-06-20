## Why

Die Mitgliedsnummer (`members.member_number`) soll eindeutig und systemseitig vergeben sein. Heute ist sie ein frei tippbares Textfeld: die Auto-Vergabe (höchste numerische Nummer + 1) greift nur, wenn das Feld beim Anlegen leer bleibt — sonst übernimmt das Backend den getippten Wert. Dadurch können Tippfehler, nicht-numerische Werte und fehlende Nummern entstehen, und niemand sieht solche Altlasten. Wir wollen die Vergabe erzwingen und bestehende Konflikte sichtbar machen.

## What Changes

- **Auto-Vergabe beim Anlegen:** Neue Mitglieder bekommen immer automatisch die höchste vorhandene **numerische** Nummer + 1 (kein Lücken-Reuse). Vom Client mitgeschickte `member_number` wird beim Create **ignoriert**. **BREAKING** (Verhalten von `POST /api/members`): explizit gesetzte Nummern werden nicht mehr übernommen.
- **Read-only mit Admin-Override:** Die Nummer ist im Frontend nur Anzeige. Nur `role=admin` darf sie nachträglich korrigieren (Altlasten, fehlende Nummern). `PUT /api/members/{id}` akzeptiert `member_number`-Änderungen nur von Admins; für Nicht-Admins bleibt das Feld unverändert.
- **Eindeutigkeit serverseitig erzwingen:** Setzt ein Admin eine bereits vergebene Nummer, antwortet die Route mit **HTTP 409** und klarer Fehlermeldung (statt eines generischen DB-Fehlers).
- **`honorar` unverändert:** Honorar-Mitglieder behalten weiterhin keine Nummer (bestehende Lösch-Logik bleibt).
- **Konflikt-Anzeige in der Mitglieder-Übersicht (`/mitglieder`):** Das Backend liefert pro Mitglied ein Konflikt-Flag, das Frontend markiert betroffene Zeilen (lucide `AlertTriangle`, `brand-*`-Tokens). Konflikt-Typen: (a) doppelte Nummer, (b) nicht-numerischer Wert, (c) Nicht-`honorar`-Mitglied ohne Nummer.
- **Kein automatischer Backfill:** Fehlende/falsche Nummern werden nur angezeigt; ein Admin korrigiert sie über den Override.

## Capabilities

### New Capabilities
- `mitgliedsnummer-verwaltung`: Eindeutige, systemseitig vergebene Mitgliedsnummer (Auto-Vergabe höchste numerische + 1, Read-only mit Admin-Override, 409 bei Dublette) und Erkennung/Anzeige von Nummern-Konflikten in der Mitglieder-Übersicht.

### Modified Capabilities
- `members`: `POST /api/members` ignoriert Client-`member_number` (immer Auto-Vergabe); `PUT /api/members/{id}` erlaubt `member_number`-Änderung nur für Admins und prüft Eindeutigkeit (409); `GET /api/members` liefert ein Konflikt-Flag pro Item.

## Impact

- **Backend:** `internal/members/handler.go` — Create- (~Z.323-364), Update- (~Z.545-631) und List-Handler (~Z.205-218); ggf. Helper für „nächste freie Nummer" und Konflikt-Erkennung. Tests in `internal/members/*_test.go`.
- **Frontend:** `web/src/components/admin/MemberStammdatenTab.tsx` (Read-only/Override-Input je nach `user.role`), Mitglieder-Listenseite in `web/src/pages/` (Konflikt-Badge), `Member`-Interface (Konflikt-Feld).
- **DB:** Keine Schema-Änderung nötig — `UNIQUE INDEX idx_members_member_number … WHERE member_number IS NOT NULL` besteht bereits. Keine Migration.
- **SSE:** Bestehende `Broadcast`-Aufrufe der Member-Mutationen genügen; keine neuen Events.
