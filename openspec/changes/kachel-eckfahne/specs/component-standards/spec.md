## MODIFIED Requirements

### Requirement: Verbindlicher Card-Klassen-String
Alle Panel-Container SHALL die Kachelform „Eckfahne“ in folgenden Varianten verwenden:

**Standard:** `bg-brand-surface-card rounded-xs shadow-sm border-t-2 eckfahne border-brand-yellow transform-gpu p-6`
**Kompakt:** `bg-brand-surface-card rounded-xs shadow-sm border-t-2 eckfahne border-brand-yellow transform-gpu p-4`
**Tabellen-Container:** `bg-brand-surface-card rounded-xs shadow-sm border-t-2 eckfahne border-brand-yellow transform-gpu overflow-hidden`

Modals erhalten ebenfalls `rounded-xs border-t-2 eckfahne border-brand-yellow`. Karten mit Statusfarbe tragen die Fahne in derselben Farbe wie die Oberkante (`border-<farbe> fahne-<farbe>`).

#### Scenario: Karte hat oberen Farbstreifen
- **WHEN** eine Standard-Karte gerendert wird
- **THEN** hat sie eine 2-px-Oberkante in `brand-yellow`, 2 px Eckenradius und oben links ein gelbes Dreieck

#### Scenario: Modal hat oberen Farbstreifen
- **WHEN** ein Modal geöffnet wird
- **THEN** hat auch das Modal `border-t-2 eckfahne border-brand-yellow`

#### Scenario: Statuskarte trägt ihre Farbe in der Fahne
- **WHEN** eine Trainingskarte auf `/termine` gerendert wird
- **THEN** sind Oberkante und Dreieck grün (`brand-green`)

#### Scenario: Alte Kachelform wird abgelehnt
- **WHEN** eine Datei in `pages/` oder `components/` `border-t-4` oder eine 2-px-Oberkante ohne `eckfahne` enthält
- **THEN** schlägt `eckfahne.gate.test.ts` fehl

### Requirement: Verbindlicher Tabellen-Klassen-String
Alle `<table>`-Strukturen SHALL folgende Klassen verwenden:

**Container:** `bg-brand-surface-card rounded-xs shadow-sm border-t-2 eckfahne border-brand-yellow transform-gpu overflow-hidden`
**Header-TH:** `bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left`
**Row-TR:** `hover:bg-brand-table-select transition-colors`
**Data-TD:** `px-4 py-3 text-sm text-brand-text`

#### Scenario: Row-Hover ist brand-table-select
- **WHEN** der Cursor über eine Tabellenzeile bewegt wird
- **THEN** wird die Zeile mit `bg-brand-table-select` hinterlegt

#### Scenario: Fahne liegt über dem Tabellenkopf
- **WHEN** ein Tabellen-Container mit hinterlegtem Kopf gerendert wird
- **THEN** ist das Dreieck über dem Kopf sichtbar und die Kopfschrift darunter lesbar
