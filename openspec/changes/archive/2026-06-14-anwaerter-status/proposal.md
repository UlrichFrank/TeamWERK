## Why

Neue Spieler, die sich noch in einer Anwärterphase befinden (kein vollwertiges Vereinsmitglied, kein System-Account), können bisher nicht im System erfasst werden und damit auch nicht am Spielbetrieb teilnehmen. Der Vorstand braucht eine Möglichkeit, sie mit minimalem Aufwand anzulegen und über den erweiterten Kader in die Spieltag-Teilnehmerlisten einzubinden.

## What Changes

- Neuer Member-Status `anwaerter` in der `members`-Tabelle (analog zu `honorar`)
- Migration die den CHECK-Constraint erweitert
- `normalizeStatus`-Funktion im CSV-Import kennt den neuen Wert
- Status-Dropdown in der Vorstand-UI zeigt "Anwärter" als Option
- Kader-Ansicht zeigt ein visuelles Badge für Mitglieder mit Status `anwaerter`
- `PUT /api/members/:id/status` akzeptiert `anwaerter` als gültigen Wert

## Capabilities

### New Capabilities
- `anwaerter-member-status`: Neuer Lifecycle-Status für Spieler in der Anwärterphase — minimale Daten (Name + GebDatum), kein Account erforderlich, nur über erweiterten Kader einbindbar

### Modified Capabilities

_(keine bestehenden Spec-Anforderungen ändern sich)_

## Impact

- **DB:** `members.status` CHECK-Constraint um `anwaerter` erweitern (1 Migration, gleicher Ansatz wie Migration 018 für `honorar`)
- **Backend:** `internal/members/handler.go` — `normalizeStatus`-Funktion + Status-Validierung in `PUT /api/members/:id/status`
- **Frontend:** `web/src/pages/MembersPage.tsx` oder Mitglied-Formular — Dropdown-Option; Kader-Seite — Badge-Anzeige
- **Keine neuen API-Endpunkte**, keine neuen Abhängigkeiten
