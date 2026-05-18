## Context

Die Nutzerverwaltungsseite hat bereits eine Tabelle registrierter Nutzer (`GET /api/admin/users`). Einladungen liegen in `invitation_tokens` (Felder: id, email, role, team_id, expires_at, used_at), Beitrittsanfragen in `membership_requests` (Felder: id, name, email, team_id, status, created_at). Beide Tabellen haben kein eigenes Listing-Endpunkt für den Admin (außer `GET /api/admin/membership-requests`, der nur pending zurückgibt).

## Goals / Non-Goals

**Goals:**
- Unified Table: eine einzige Tabelle zeigt registrierte Nutzer, offene Einladungen und offene Beitrittsanfragen — je mit passendem Status-Badge
- `GET /api/admin/invitations` liefert aktive Einladungen (used_at IS NULL, expires_at > now)
- `DELETE /api/admin/invitations/{id}` löscht einen Einladungstoken
- `DELETE /api/admin/membership-requests/{id}` löscht eine Beitrittsanfrage aus der DB
- Frontend integriert alle drei Datenquellen in `AdminUsersPage.tsx`

**Non-Goals:**
- E-Mail-Benachrichtigung beim Widerruf einer Einladung
- Einladungen erneut versenden
- Abgelaufene Einladungen anzeigen (nur aktive)
- Angenommene/abgelehnte Anfragen im Verlauf behalten

## Decisions

**1. Unified Table vs. separate Sektionen**

→ **Entscheidung:** Eine gemeinsame Tabelle mit Typ-unterscheidung via Status-Badge (`Einladung` / `Anfrage` / Rolle). Sortierung: Anfragen und Einladungen oben (pending), Nutzer darunter. Kein Tab-Switching nötig.

**2. Einladungs-ID für DELETE**

`invitation_tokens` hat eine `id`-Spalte (INTEGER PRIMARY KEY). `GET /api/admin/invitations` gibt diese ID mit zurück, `DELETE /api/admin/invitations/{id}` löscht direkt per PK.

**3. Membership-Request DELETE vs. REJECT**

Löschen entfernt den Datensatz komplett. Die bestehende Reject-Funktion setzt nur `status='rejected'` und behält den Eintrag. Beides bleibt erhalten — Delete ist für Datenmüll (Spam, Duplikate), Reject für den dokumentierten Ablehnungsfall.

**4. Datenmischung im Frontend**

Drei separate API-Calls beim Mount (`/admin/users`, `/admin/invitations`, `/admin/membership-requests`). Zusammenführung client-seitig in einem gemischten Array mit Typ-Diskriminator (`type: 'user' | 'invitation' | 'request'`). Sortierung: requests + invitations zuerst, dann users alphabetisch.

**5. Aktionen je Zeile**

| Typ | Aktionen |
|---|---|
| `user` | Löschen (wie bisher) |
| `invitation` | Löschen (widerruft Token) |
| `request` | Genehmigen · Ablehnen · Löschen |

## Risks / Trade-offs

- **Race condition beim Löschen einer genutzten Einladung:** Eine Einladung könnte genau dann genutzt werden, wenn der Admin sie löscht. Da `used_at IS NULL` in der Listequery geprüft wird, ist das Risiko minimal und kein Datenverlust entsteht.
- **Kein Undo:** Delete ist permanent. Bestätigungsdialog im Frontend mitigiert.

## Migration Plan

1. Backend: `ListInvitations` + `DeleteInvitation` + `DeleteMembershipRequest` Handler
2. Backend: Routen in `main.go` ergänzen
3. Frontend: Typen zusammenführen, Tabelle erweitern
4. Kein DB-Schema-Change
