## Context

Die Mitglieder- und Nutzerverwaltung hat aktuell begrenzte Filtermöglichkeiten. Offene Änderungsanträge sind als kleine Icons sichtbar, aber nicht filterbar. Mitglieder ohne App-Account und User ohne Mitgliedsverknüpfung sind nur durch manuelles Durchscrollen findbar. Außerdem hat der bestehende Code einen Bug: `ListUsers` prüft `family_links` nicht, sodass Eltern-User fälschlich als "ohne Mitglied" eingestuft werden.

## Goals / Non-Goals

**Goals:**
- Vier gezielte Filter über bestehende Daten (keine neuen Tabellen, keine Migration)
- Bug-Fix family_links in ListUsers
- Push-Notification mit Deeplink auf spezifischen Beitrittsantrag

**Non-Goals:**
- Änderung der Datenhaltung oder des Draft-Workflows
- Neue Rollen oder Berechtigungen
- Filterpersistenz über Seitenreloads hinaus

## Decisions

### Filter als Query-Params statt clientseitig
Die Filter `unlinked_user`, `has_draft` und `unlinked` werden serverseitig über Query-Params implementiert, nicht clientseitig nach dem Laden. Die Mitgliederliste ist paginiert (50 pro Seite) — clientseitiges Filtern würde nur die aktuelle Seite filtern und wäre irreführend.

**Alternative:** Clientseitig filtern — verworfen, da bei 500+ Mitgliedern nur die erste Seite gefiltert würde.

### has_family_link als Response-Feld statt separater Endpoint
Das Feld `has_family_link` wird direkt in den `GET /api/users`-Response eingebettet via LEFT JOIN auf `family_links`. Das vermeidet einen zweiten API-Call im Frontend.

**SQL-Ansatz:**
```sql
SELECT u.id, ..., m.id, (fl.parent_user_id IS NOT NULL) AS has_family_link
FROM users u
LEFT JOIN members m ON m.user_id = u.id
LEFT JOIN (SELECT DISTINCT parent_user_id FROM family_links) fl ON fl.parent_user_id = u.id
```

### SQLite RETURNING nicht verwenden
Für das Auslesen der neuen Membership-Request-ID wird `result.LastInsertId()` nach dem INSERT verwendet (nicht RETURNING), da SQLite < 3.35 kein RETURNING unterstützt und die Produktionsdatenbank möglicherweise älter ist.

### Highlight via React useEffect + setTimeout
Das Scroll-/Highlight-Verhalten in MembershipRequestsPage wird über `useEffect` implementiert:
1. Nach dem Laden der Requests `?id`-Param aus `useSearchParams` lesen
2. Per `document.getElementById` zur Karte scrollen
3. Highlight-Klasse setzen, nach 2000ms wieder entfernen

Die Karten-IDs werden als `id="request-{id}"` auf dem DOM-Element gesetzt.

## Risks / Trade-offs

- **Filterperformance bei großen Listen**: Die EXISTS-Subqueries auf `member_change_drafts` und `family_links` sind ohne spezielle Indizes auf kleinen SQLite-Datenbanken (<1000 Mitglieder) vernachlässigbar. Bestehende Indizes (`idx_member_change_drafts_member_id`) sind bereits vorhanden.
- **Kombination mehrerer Filter**: `unlinked_user=1&has_draft=1` kann zu leeren Resultaten führen (logisch korrekt). Frontend zeigt keinen expliziten Hinweis — Paginierung zeigt "0 Einträge".
