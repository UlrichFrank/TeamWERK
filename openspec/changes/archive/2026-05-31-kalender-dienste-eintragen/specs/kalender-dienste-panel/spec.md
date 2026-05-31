## ADDED Requirements

### Requirement: DutySlotList Shared Component
Das System SHALL eine gemeinsame React-Komponente `DutySlotList` bereitstellen, die eine Liste von `BoardSlot`-Objekten darstellt und folgende Interaktionen kapselt: Eintragen/Austragen (Claim/Unclaim), Zuteilungen aufklappen (Admin/Trainer), Erfüllen und Geldersatz buchen (Admin/Trainer), Slot löschen mit Bestätigungsdialog (Admin/Trainer).

#### Scenario: Nutzer trägt sich ein
- **WHEN** ein eingeloggter Nutzer auf „Eintragen" klickt und freie Plätze vorhanden sind
- **THEN** wird `POST /duty-board/{slotId}/claim` aufgerufen und die Slot-Liste aktualisiert sich

#### Scenario: Nutzer trägt sich aus
- **WHEN** ein Nutzer mit `claimed_by_me = true` auf „Austragen" klickt
- **THEN** wird `DELETE /duty-board/{slotId}/claim` aufgerufen und die Slot-Liste aktualisiert sich

#### Scenario: Kein Eintragen bei vergangenen Slots
- **WHEN** ein Slot als `isPast = true` übergeben wird
- **THEN** sind Eintragen- und Austragen-Buttons nicht sichtbar

#### Scenario: Admin sieht Zuteilungen
- **WHEN** `canEdit = true` und der Nutzer „Zuteilungen" klickt
- **THEN** wird `GET /duty-slots/{id}/assignments` geladen und die Zuteilungsliste eingeblendet

### Requirement: Dienst-Panel auf Kalender-Detailseite
Die `SpieltagDetailPage` SHALL die Slots des jeweiligen Spiels über `GET /duty-board?game_id={id}` laden und mit `DutySlotList` darstellen. Eintragen/Austragen ist für alle authentifizierten Nutzer möglich. Die bisherige ProgressBar-Darstellung der Slots entfällt.

#### Scenario: Regulärer Nutzer trägt sich auf Kalenderseite ein
- **WHEN** ein Spieler oder Elternteil die Kalender-Detailseite eines Spiels aufruft
- **THEN** sieht er die Dienst-Slots mit Eintragen-Button für freie Plätze

#### Scenario: Admin-Slot-Management bleibt erhalten
- **WHEN** ein Admin oder Trainer die Kalender-Detailseite aufruft
- **THEN** sind „+ Dienst hinzufügen", Bearbeiten und Löschen weiterhin verfügbar

#### Scenario: Reload nach Slot-Mutation
- **WHEN** ein Nutzer einen Slot hinzufügt oder löscht
- **THEN** werden sowohl die Spieldaten (`GET /kalender/{id}`) als auch die Board-Daten (`GET /duty-board?game_id={id}`) neu geladen

#### Scenario: Leerer Zustand
- **WHEN** das Spiel keine Slots hat
- **THEN** zeigt das Panel den Text „Keine Dienste für dieses Spiel angelegt"

### Requirement: DutyPage verwendet DutySlotList
`DutyPage` SHALL die gemeinsame `DutySlotList`-Komponente für die Slot-Darstellung verwenden. Das Verhalten der Seite ändert sich für den Nutzer nicht.

#### Scenario: DutyPage verhält sich unverändert
- **WHEN** ein Nutzer `/dienste` aufruft
- **THEN** sieht er dieselbe Darstellung und Funktionalität wie vor der Umstellung
