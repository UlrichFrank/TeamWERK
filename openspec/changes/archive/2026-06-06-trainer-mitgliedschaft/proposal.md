## Why

Nicht alle Trainer von Team Stuttgart sind Vereinsmitglieder — externe Honorartrainer haben keine Mitgliedschaft, müssen aber einem Team zugewiesen werden und ins System passen. Das bestehende `members`-Modell kennt nur aktive Vereinsmitglieder und bietet keine saubere Kategorie für diese Personengruppe.

## What Changes

- Neuer Mitgliedsstatus `honorar` in `members.status` für externe/Honorar-Trainer und andere assoziierte Personen ohne vollständige Vereinsmitgliedschaft
- Honorar-Mitglieder können (wie reguläre Mitglieder) ein `users`-Login-Konto mit `role=standard` und die Vereinsfunktion `trainer` in `member_club_functions` erhalten und werden über `team_trainers` Mannschaften zugewiesen
- Honorar-Trainer werden in allen Trainer-Funktionen (Kader, Slots, Anfragen) identisch zu regulären Trainer-Mitgliedern behandelt
- Honorar-Mitglieder erscheinen **nicht** in aktiven Mitglieder-Zählungen, Dienst-Konten und RSVP-Pflicht-Übersichten (kein Soll, keine Vereinspflichten)
- Admin-UI zeigt Honorar-Mitglieder sichtbar markiert in Trainer-Zuweisungs-Ansichten

## Capabilities

### New Capabilities

_(keine neuen Specs nötig — reine Erweiterung bestehender Capability)_

### Modified Capabilities

- `members`: Status-Enum um `honorar` erweitern; Filterbedingung "aktive Mitglieder" schließt `honorar` aus; Honorar-Mitglieder haben keinen Dienst-Soll und kein RSVP

## Impact

- **DB-Migration**: `members.status` CHECK-Constraint um `'honorar'` erweitern (1 Migration)
- **Backend**: `internal/members/` — alle Queries, die auf `status != 'ausgetreten'` filtern, müssen `honorar` explizit ausschließen oder einschließen je nach Kontext
- **Frontend**: `MembersPage` + `AdminUsersPage` — Status-Dropdown/-Filter um `honorar` ergänzen; Badge/Label in Trainer-Listen
- **Kein Breaking Change** an bestehenden API-Endpunkten; bestehende Trainer-Accounts bleiben unberührt
