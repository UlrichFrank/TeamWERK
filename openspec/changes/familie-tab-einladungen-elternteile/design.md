## Context

`invitation_tokens` hat bereits ein `member_id`-Feld, das den eigenen Account eines eingeladenen Nutzers mit einem Mitglied verknüpft. Bei Registrierung setzt der Handler `members.user_id = newUserID`. Für Erziehungsberechtigte brauchen wir eine analoge Verknüpfung, die aber in `family_links` statt in `members` landet.

Der Admin Tab in `MemberAdminTab.tsx` zeigt bereits beide — registrierte Nutzer und ausstehende Einladungen — in einem gemeinsamen Dropdown. Dieselbe Pattern wird im Familie Tab repliziert.

## Goals / Non-Goals

**Goals:**
- Einladungen als Erziehungsberechtigte vormerken können (aus dem Familie Tab)
- Automatische `family_links`-Erstellung bei Registrierung
- Gemeinsame Liste (registrierte + pending) im Familie Tab, max. 2 gesamt

**Non-Goals:**
- Einladungen direkt aus dem Familie Tab versenden (bleibt in der Nutzerverwaltung)
- Gleichzeitig `member_id` und `parent_member_id` auf derselben Einladung setzen ist erlaubt (zwei verschiedene Mitglieder möglich), wird aber nicht aktiv verhindert

## Decisions

**1. Neues Feld `parent_member_id` statt Wiederverwendung von `member_id`**

`member_id` bedeutet „diese Person ist das Mitglied selbst". Ein Elternteil kann sowohl ein eigenes Mitgliedsprofil haben als auch Erziehungsberechtigter sein — das sind semantisch getrennte Konzepte. Ein neues Feld ist eindeutiger und vermeider Konflikte.

**2. Neuer dedizierter Endpoint `PUT /admin/invitations/{id}/parent-member`**

Analog zum bestehenden `PUT /admin/invitations/{id}/member`. Klare Trennung der Semantik. Alternative wäre ein generischer PATCH-Endpoint, aber das würde das bestehende Muster brechen.

**3. Familie Tab: Dropdown aus bestehenden Einladungen (kein Inline-Invite)**

Einladungen werden zentral in der Nutzerverwaltung erstellt. Ein zweites Invite-Formular im Familie Tab würde Logik duplizieren und könnte zu Inkonsistenzen führen (z.B. Einladung ohne team_id/role). Der Admin erstellt zuerst die Einladung, dann verknüpft er sie.

**4. Pending-Einladungen in gemeinsamer Liste mit Badge**

Registrierte Elternteile und pending Einladungen werden in einer Liste zusammengeführt (max. 2 gesamt). Pending-Einträge zeigen E-Mail + „Einladung ausstehend"-Badge statt Name. Entfernen funktioniert in beiden Fällen (DELETE family_links vs. `parent_member_id = null`).

## Risks / Trade-offs

- **Race condition bei Doppelregistrierung**: Wenn zwei Einladungen auf dieselbe E-Mail und dasselbe Mitglied zeigen (parent_member_id) und beide gleichzeitig registriert werden → doppelter INSERT in `family_links` schlägt durch UNIQUE-Constraint fehl → kein Datenverlust, zweite Registrierung bekommt 409 → akzeptabel.
- **Einladung läuft ab mit gesetztem `parent_member_id`**: Familie Tab zeigt eine ausstehende Einladung, die nie eingelöst wird. Admin muss manuell entfernen. → Kein kritischer Fehler, nur UX-Rauschen. Scheduler könnte abgelaufene Einladungen bereinigen (bereits vorhanden).

## Migration Plan

1. Migration anlegen: `ALTER TABLE invitation_tokens ADD COLUMN parent_member_id INTEGER REFERENCES members(id) ON DELETE SET NULL`
2. Kein Datenmigrationsbedarf — alle bestehenden Einladungen bleiben unverändert (NULL-Default)
3. Backend deployen vor Frontend — neues Feld ist optional, alter Code ignoriert es
4. Rollback: `parent_member_id`-Feld ignorieren reicht; die Spalte kann stehen bleiben
