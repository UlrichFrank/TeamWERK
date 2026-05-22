## Why

Die Mitglieder-Detailseite hat vier Usability-Probleme: die Mannschafts-Zuweisung gehört seit der Kader-Einführung nicht mehr hierhin, die Erziehungsberechtigten-Sektion hat falsche Beschränkungen (nur Rolle `elternteil`), erlaubt keine Entfernung von Verlinkungen und begrenzt die Anzahl nicht. Das muss bereinigt werden.

## What Changes

- **Mannschafts-Zuweisung entfernen**: Die Sektion „Mannschaft zuweisen" wird aus `MemberDetailPage` entfernt — Team-Zuordnung erfolgt künftig ausschließlich über die Kaderplanung
- **Erziehungsberechtigte statt Elternteile**: Abschnitt wird von „Elternteile" in „Erziehungsberechtigte" umbenannt (Label, Überschrift, API-intern bleibt `family_links`)
- **Alle Nutzer verknüpfbar**: Das Dropdown für Erziehungsberechtigte zeigt alle System-Nutzer, nicht nur solche mit Rolle `elternteil`
- **Verlinkung entfernbar**: Jeder Erziehungsberechtigter erhält einen Entfernen-Button; neuer Backend-Endpoint `DELETE /api/admin/family-links` wird implementiert
- **Maximum zwei Erziehungsberechtigte**: Frontend deaktiviert „Hinzufügen" ab 2 Einträgen; Backend lehnt weitere Einträge mit 409 ab

## Capabilities

### New Capabilities

- `erziehungsberechtigte-verwaltung`: Erziehungsberechtigte zu einem Mitglied verknüpfen und wieder entfernen, mit Begrenzung auf maximal zwei Einträge und ohne Einschränkung auf eine bestimmte Nutzer-Rolle

### Modified Capabilities

- `member-team-assignment`: Mannschafts-Zuweisung wird aus der Mitglieder-Detailseite entfernt (kein API-Endpoint entfernt, nur UI)

## Impact

- **Frontend**: `web/src/pages/MemberDetailPage.tsx`
- **Backend**: `internal/members/handler.go` — neuer Handler `DeleteFamilyLink`, Limit-Prüfung in `CreateFamilyLink`
- **Routing**: `cmd/teamwerk/main.go` — neuer Route `DELETE /api/admin/family-links`
- Keine Datenbankänderung; `family_links`-Tabelle und -Schema bleiben unverändert
- Keine Breaking Changes: der vorhandene POST-Endpoint und GET-Endpoint bleiben erhalten
