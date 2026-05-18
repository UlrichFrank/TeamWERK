## ADDED Requirements

### Requirement: Navigation ist in 3 Module gegliedert
Die Sidebar-Navigation SHALL die Menüpunkte in drei Modulen gruppieren: „Mitglieder", „Dienste" und „Administration". Jedes Modul SHALL als klickbarer Header mit Pfeil-Indikator dargestellt werden, unter dem die zugehörigen Untereinträge angezeigt werden.

#### Scenario: Alle Module sichtbar für Admin
- **WHEN** ein Nutzer mit Rolle `admin` eingeloggt ist
- **THEN** zeigt die Sidebar alle 3 Module mit ihren jeweiligen Untereinträgen an

#### Scenario: Modul ohne sichtbare Einträge wird ausgeblendet
- **WHEN** ein Nutzer mit Rolle `spieler` eingeloggt ist
- **THEN** wird das Modul „Administration" nicht angezeigt, da alle seine Einträge für diese Rolle unsichtbar sind

#### Scenario: Untereinträge sind korrekt einem Modul zugeordnet
- **WHEN** die Sidebar gerendert wird
- **THEN** enthält „Mitglieder" die Einträge Mitglieder und Mein Profil, „Dienste" enthält Dienstbörse, Dienstkonten und Dienst-Planung, „Administration" enthält Beitrittsanfragen, Verein, Teams, Nutzer und Diensttypen

### Requirement: Module sind ein- und ausklappbar
Jedes Modul SHALL durch Klick auf den Modul-Header ein- oder ausgeklappt werden können. Der Klapp-Zustand SHALL im `localStorage` persistiert werden, damit er nach einem Seitenreload erhalten bleibt.

#### Scenario: Modul einklappen
- **WHEN** ein Nutzer auf den Header eines aufgeklappten Moduls klickt
- **THEN** werden die Untereinträge des Moduls ausgeblendet und der Pfeil-Indikator dreht sich

#### Scenario: Zustand nach Reload erhalten
- **WHEN** ein Nutzer ein Modul einklappt und die Seite neu lädt
- **THEN** ist das Modul weiterhin eingeklappt

#### Scenario: Beim ersten Besuch alle aufgeklappt
- **WHEN** kein `localStorage`-Eintrag für den Klapp-Zustand existiert
- **THEN** sind alle Module standardmäßig aufgeklappt

### Requirement: Aktives Modul ist visuell hervorgehoben
Wenn ein Untereintrag eines Moduls die aktive Route ist, SHALL der Modul-Header visuell hervorgehoben werden (heller Text statt gedämpftem Text).

#### Scenario: Aktiver Untereintrag hebt Modulnamen hervor
- **WHEN** die aktuelle Route zu einem Untereintrag des Moduls „Dienste" gehört
- **THEN** wird der Header „Dienste" mit vollem weißem Text dargestellt, während inaktive Modulnamen gedämpft erscheinen

### Requirement: Untereinträge sind korrekt nach Rolle gefiltert
Innerhalb eines sichtbaren Moduls SHALL jeder Untereintrag nur angezeigt werden, wenn die Rolle des eingeloggten Nutzers in der `roles`-Liste des Eintrags enthalten ist.

#### Scenario: Trainer sieht nur relevante Einträge im Modul Administration
- **WHEN** ein Nutzer mit Rolle `trainer` eingeloggt ist
- **THEN** zeigt das Modul „Administration" nur „Beitrittsanfragen" und nicht Verein, Teams, Nutzer oder Diensttypen
