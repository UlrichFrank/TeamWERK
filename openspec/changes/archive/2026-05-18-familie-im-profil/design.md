## Context

Die Navigation zeigt „Mitglieder" aktuell für alle Rollen inkl. `spieler` und `elternteil`. Für diese beiden Rollen liefert `GET /api/members` jedoch eine leere Liste (die Query filtert auf Teams wo der User Trainer ist — was für spieler/elternteil nie zutrifft).

`GET /api/profile/me` liefert bereits:
- Eigenes Mitgliedsprofil (via `members.user_id`)
- Verknüpfte Kinder für `elternteil` (via `family_links`)

Nicht vorhanden: verknüpfte Elternteile für `spieler`.

AppShell nutzt bereits ein `roles`-Array pro Nav-Eintrag, das per `filter` ausgewertet wird.

## Goals / Non-Goals

**Goals:**
- Mitglieder-Nav ausblenden für `spieler` und `elternteil`
- Profil-Seite zeigt Familien-Sektion für betroffene Rollen
- `GET /api/profile/me` gibt für `spieler` auch verknüpfte Elternteile zurück

**Non-Goals:**
- Keine Änderung an `GET /api/members` (bleibt für admin/trainer)
- Kein Editieren von Mitgliedsdaten über das Profil
- Keine Änderung an MemberDetailPage (admin-seitige family-links Verwaltung ist separates Thema)

## Decisions

**Backend: Erweiterung von `GetProfile` statt neuer Endpoint**
`GetProfile` wird um eine Abfrage ergänzt: Falls der User ein `spieler` ist und ein Mitglied via `user_id` verknüpft hat, werden die zugehörigen Elternteile aus `family_links` geladen.

Query-Logik:
```sql
SELECT u.id, u.name, u.email
FROM users u
JOIN family_links fl ON fl.parent_user_id = u.id
JOIN members m ON m.id = fl.member_id
WHERE m.user_id = ?  -- aktueller spieler
```

Das Ergebnis wird als neues Feld `parents` neben dem bestehenden Array in der Response ergeben. Die Response-Struktur bleibt ein Array von `Member`-Objekten für Abwärtskompatibilität — Eltern werden als separates Feld im Response-Objekt zurückgegeben.

**Response-Struktur (neu):**
```json
{
  "members": [...],
  "parents": [
    { "id": 1, "name": "Maria Muster", "email": "..." }
  ]
}
```

Das bricht die bestehende API (bisher reines Array) — aber `ProfilePage` ist der einzige Consumer, daher kein Breaking Change nach außen.

**Frontend: Neue Sektion in ProfilePage**
Unterhalb der bestehenden Mitgliedskarten eine bedingte Sektion „Meine Familie":
- Nur rendern wenn `members.parents` vorhanden und nicht leer (spieler) oder `members` > 1 Eintrag hat mit Kindern (elternteil)
- Read-only Karten analog zur bestehenden Mitgliedskarte

**Navigation: roles-Array anpassen**
`/mitglieder` roles: `['admin', 'trainer', 'elternteil', 'spieler']` → `['admin', 'trainer']`

## Risks / Trade-offs

**Response-Format-Änderung** → Mitigation: ProfilePage ist der einzige Consumer, kein externer Client.

**spieler ohne Mitgliedsprofil** (user_id nicht in members): parents-Query gibt korrekt leeres Array zurück, Sektion wird nicht gerendert. Kein Fehlerfall.
