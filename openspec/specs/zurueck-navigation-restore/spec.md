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

### Requirement: Kalender-Verweis „In Diensten öffnen" fokussiert das zugehörige Spiel

Wenn ein Nutzer im Kalender-Event-Info-Modal eines Spiels auf „In Diensten öffnen" klickt, SHALL das Frontend zu `/dienste` mit einem Fokus-Marker auf das Spiel navigieren, statt zu einer ungefilterten `/dienste`-Ansicht. `/dienste` SHALL bei vorhandenem Spiel-Fokus zur Dienst-Gruppe dieses Spiels scrollen und sie optisch hervorheben, auch wenn sie durch aktive Typ-/Team-/Zeitraum-Filter sonst ausgeblendet wäre.

#### Scenario: Von „In Diensten öffnen" zur richtigen Spiel-Gruppe

- **WHEN** ein Nutzer im Kalender-Modal eines Spiels mit Dienst-Slots auf „In Diensten öffnen" klickt
- **THEN** landet der Nutzer auf `/dienste`
- **AND** die Dienst-Gruppe dieses Spiels ist sichtbar in den Viewport gescrollt und optisch hervorgehoben

#### Scenario: Button ist deaktiviert ohne Dienst-Slots

- **WHEN** das Spiel im Kalender-Modal keine Dienst-Slots hat
- **THEN** ist der „In Diensten öffnen"-Button deaktiviert

### Requirement: Zurück-Navigation stellt die Scroll-Position wieder her

Die App SHALL die Scroll-Position ihres Inhaltsbereichs pro History-Eintrag sichern und bei einer Zurück-/Vorwärts-Navigation (POP) wieder herstellen, unabhängig davon, ob die Zielseite einen Fokus-Marker in der URL trägt. Weil der Inhalt der Zielseite erst per API nachgeladen wird, SHALL die Wiederherstellung nachfassen, bis die Seite hoch genug ist, und bei einer Nutzereingabe (Scrollen, Tastatur) abbrechen. Trägt die Ziel-URL einen `focus`-Parameter, SHALL der Fokus-Mechanismus Vorrang haben. Eine Navigation auf einen anderen Pfad (PUSH/REPLACE) SHALL oben beginnen; eine reine Query-Änderung auf derselben Seite (Filter) SHALL die Position unverändert lassen.

#### Scenario: Zurück nach reinem Scrollen

- **WHEN** ein Nutzer eine lange Liste (z. B. `/termine`, `/dienste`) scrollt, ohne eine Karte anzuklicken, danach auf eine andere Seite wechselt und den Zurück-Button betätigt
- **THEN** zeigt die Liste wieder den zuvor betrachteten Ausschnitt, auch wenn ihr Inhalt erst nach der Navigation geladen ist

#### Scenario: Nutzer scrollt während der Wiederherstellung selbst

- **WHEN** der Nutzer nach dem Zurück selbst scrollt, bevor die Zielposition erreicht ist
- **THEN** bricht die Wiederherstellung ab und die vom Nutzer gewählte Position bleibt stehen

### Requirement: Zurück-Navigation zu Kalender erhält den zuletzt betrachteten Monat

`/kalender` SHALL den aktuell angezeigten Monat als Teil seiner URL führen und diese bei jedem Monatswechsel (vor, zurück, „Heute") aktualisieren. Verlässt ein Nutzer `/kalender`, nachdem er zu einem anderen Monat als dem aktuellen navigiert hat, und kehrt über den globalen Zurück-Button zurück, SHALL `/kalender` wieder den zuletzt betrachteten Monat anzeigen statt des aktuellen Monats.

#### Scenario: Zurück zu einem durchblätterten Monat

- **WHEN** ein Nutzer auf `/kalender` mehrere Monate weiterblättert, danach zu einer anderen Seite wechselt und den globalen Zurück-Button betätigt
- **THEN** zeigt `/kalender` wieder den zuletzt betrachteten Monat an, nicht den aktuellen Kalendermonat
