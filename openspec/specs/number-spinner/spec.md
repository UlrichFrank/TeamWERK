# number-spinner Specification

## Purpose

Diese Spezifikation beschreibt die Capability `number-spinner`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: NumberSpinner rendert ein Zahlenfeld mit gestylten Chevron-Buttons

Die Komponente SHALL ein `<input type="number">` mit zwei absolut positionierten Chevron-Buttons (▲ oben, ▼ unten) rechts im Eingabefeld rendern. Die nativen Browser-Spinner-Pfeile SHALL ausgeblendet werden. Die Buttons SHALL in Markenfarben gelb/schwarz gestaltet sein (`bg-brand-yellow text-brand-black`, Hover: `bg-brand-black text-brand-yellow`).

#### Scenario: Komponente rendert korrekt

- **WHEN** `<NumberSpinner value={5} onChange={fn} />` gerendert wird
- **THEN** ist ein Textfeld mit zwei Chevron-Buttons rechts im Feld sichtbar, native Browser-Pfeile sind nicht sichtbar

### Requirement: Schrittweite per Prop konfigurierbar

Die Komponente SHALL eine `step`-Prop akzeptieren (Default: `1`). Beim Klick auf ▲ SHALL der Wert um `step` erhöht werden, beim Klick auf ▼ um `step` verringert werden.

#### Scenario: Klick auf ▲ mit step=5

- **WHEN** der Wert `20` ist und `step={5}` gesetzt ist und auf ▲ geklickt wird
- **THEN** wird `onChange(25)` aufgerufen

#### Scenario: Klick auf ▼ mit step=5

- **WHEN** der Wert `20` ist und `step={5}` gesetzt ist und auf ▼ geklickt wird
- **THEN** wird `onChange(15)` aufgerufen

### Requirement: Min/Max-Grenzen werden beim Button-Klick eingehalten

Die Komponente SHALL `min`- und `max`-Props akzeptieren. Beim Button-Klick SHALL der Wert nicht unter `min` bzw. nicht über `max` fallen. Der ▼-Button SHALL bei Erreichen von `min` disabled sein, der ▲-Button bei `max`.

#### Scenario: ▼ an der Untergrenze

- **WHEN** `value={1}`, `min={1}` und auf ▼ geklickt wird
- **THEN** wird `onChange` nicht aufgerufen und der ▼-Button ist disabled

#### Scenario: ▲ an der Obergrenze

- **WHEN** `value={10}`, `max={10}` und auf ▲ geklickt wird
- **THEN** wird `onChange` nicht aufgerufen und der ▲-Button ist disabled

### Requirement: Direktes Eintippen ist möglich

Der Nutzer SHALL Werte direkt in das Eingabefeld tippen können. Beim `onChange`-Event des Inputs SHALL `onChange(parseInt(e.target.value))` aufgerufen werden. Ein Step-Raster DARF beim direkten Tippen nicht erzwungen werden.

#### Scenario: Direktes Eintippen eines Wertes

- **WHEN** der Nutzer `17` in das Feld tippt
- **THEN** wird `onChange(17)` aufgerufen

### Requirement: Komponente wird an allen drei Einsatzorten verwendet

Die Komponente SHALL in `ProfileMiscTab.tsx` (Sitzplätze), `MitfahrgelegenheitenPage.tsx` (Freie Plätze) und `AdminSettingsPage.tsx` (Halbzeit und Pause) eingesetzt werden.

#### Scenario: Einsatz mit step=5 bei Altersklassen

- **WHEN** `<NumberSpinner value={20} min={1} step={5} onChange={fn} />` in der Altersklassen-Tabelle gerendert wird
- **THEN** erhöht/verringert ein Button-Klick den Wert um 5
