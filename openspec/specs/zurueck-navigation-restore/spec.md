# zurueck-navigation-restore Specification

## Purpose

Sorgt dafür, dass der globale „Zurück"-Button auf den Listen-/Kalenderseiten der App an die zuvor betrachtete Position zurückführt, statt die Ansicht zurückzusetzen.

## Requirements

### Requirement: Zurück-Navigation zu Termine fokussiert die zuvor geöffnete Karte

Wenn ein Nutzer auf `/termine` eine Termin- oder Trainingskarte anklickt und zur Detailseite navigiert, SHALL das Frontend den Fokus auf diese Karte in der URL von `/termine` hinterlegen, bevor es zur Detailseite wechselt. Kehrt der Nutzer über den globalen Zurück-Button zurück, SHALL `/termine` automatisch zu dieser Karte scrollen und sie optisch hervorheben.

#### Scenario: Zurück zu einem angeklickten Termin

- **WHEN** ein Nutzer auf `/termine` eine Trainings- oder Spielkarte anklickt und danach den globalen Zurück-Button betätigt
- **THEN** landet der Nutzer wieder auf `/termine`
- **AND** die zuvor angeklickte Karte ist sichtbar in den Viewport gescrollt und optisch hervorgehoben

### Requirement: Zurück-Navigation zu Dienste fokussiert den zuvor geöffneten Slot

Wenn ein Nutzer auf `/dienste` die Anleitung eines Dienst-Slots öffnet (`/dienste/anleitung/:typeId`), SHALL das Frontend den Fokus auf diesen Slot in der URL von `/dienste` hinterlegen, bevor es zur Anleitungsseite wechselt. Kehrt der Nutzer über den globalen Zurück-Button zurück, SHALL `/dienste` automatisch zu dieser Slot-Zeile scrollen und sie optisch hervorheben.

#### Scenario: Zurück von der Anleitung zum ursprünglichen Slot

- **WHEN** ein Nutzer auf `/dienste` bei einem Dienst-Slot die Anleitung öffnet und danach den globalen Zurück-Button betätigt
- **THEN** landet der Nutzer wieder auf `/dienste`
- **AND** die Zeile des Slots, von dem aus die Anleitung geöffnet wurde, ist sichtbar in den Viewport gescrollt und optisch hervorgehoben

### Requirement: Zurück-Navigation zu Kalender erhält den zuletzt betrachteten Monat

`/kalender` SHALL den aktuell angezeigten Monat als Teil seiner URL führen und diese bei jedem Monatswechsel (vor, zurück, „Heute") aktualisieren. Verlässt ein Nutzer `/kalender`, nachdem er zu einem anderen Monat als dem aktuellen navigiert hat, und kehrt über den globalen Zurück-Button zurück, SHALL `/kalender` wieder den zuletzt betrachteten Monat anzeigen statt des aktuellen Monats.

#### Scenario: Zurück zu einem durchblätterten Monat

- **WHEN** ein Nutzer auf `/kalender` mehrere Monate weiterblättert, danach zu einer anderen Seite wechselt und den globalen Zurück-Button betätigt
- **THEN** zeigt `/kalender` wieder den zuletzt betrachteten Monat an, nicht den aktuellen Kalendermonat
