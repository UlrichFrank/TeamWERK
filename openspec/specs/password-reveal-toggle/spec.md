## ADDED Requirements

### Requirement: Toggle nur bei selbst getipptem Passwort

Ein Reveal-Toggle (Auge-Icon) im Passwort-Feld SHALL nur dann sichtbar sein, wenn der aktuelle Wert des Feldes vom Nutzer selbst getippt (oder per Tastatur/Kontextmenü eingefügt) wurde und das Feld nicht leer ist. Wurde der Wert ganz oder teilweise durch Browser-Autofill, Passwort-Manager-Extension oder anderweitige programmatische Wertsetzung eingetragen, SHALL der Toggle NICHT erscheinen.

#### Scenario: Frisches Feld, kein Eingabe — kein Auge

- **WHEN** ein Passwort-Feld initial gerendert wird und keinen Wert enthält
- **THEN** ist kein Reveal-Toggle sichtbar

#### Scenario: Nutzer tippt ein Zeichen — Auge erscheint

- **WHEN** der Nutzer in ein leeres Passwort-Feld ein Zeichen tippt
- **THEN** erscheint der Reveal-Toggle rechts im Feld

#### Scenario: Browser-Autofill — kein Auge

- **WHEN** der Browser das Passwort-Feld via Autofill mit einem gespeicherten Passwort befüllt (kein vorausgehender `keydown`/`paste`/`cut`-Event)
- **THEN** erscheint KEIN Reveal-Toggle, obwohl das Feld einen Wert hat

#### Scenario: Passwort-Manager-Extension — kein Auge

- **WHEN** eine Browser-Extension (z. B. 1Password, Bitwarden) das Feld programmatisch befüllt
- **THEN** erscheint KEIN Reveal-Toggle

#### Scenario: Paste via Strg+V — Auge erscheint

- **WHEN** der Nutzer aus der Zwischenablage per `Ctrl+V` in das Feld einfügt
- **THEN** erscheint der Reveal-Toggle (Paste zählt als User-Aktion)

#### Scenario: Paste via Kontextmenü — Auge erscheint

- **WHEN** der Nutzer per Rechtsklick → „Einfügen" einen Wert in das Feld einsetzt
- **THEN** erscheint der Reveal-Toggle

---

### Requirement: Strenger Tainted-Modus

Sobald der Wert eines Passwort-Feldes mindestens einmal durch ein Injection-Ereignis (Autofill, Extension) verändert wurde, SHALL der Reveal-Toggle versteckt bleiben — auch wenn der Nutzer danach weitere Zeichen tippt. Erst wenn das Feld vollständig geleert wird (Wert wird `""`), SHALL der Tainted-Status zurückgesetzt werden, sodass anschließendes Tippen den Toggle wieder freischaltet.

#### Scenario: Autofill + nachträgliches Tippen — kein Auge

- **WHEN** ein Passwort-Feld via Browser-Autofill befüllt wurde, danach der Nutzer ein zusätzliches Zeichen tippt
- **THEN** bleibt der Reveal-Toggle versteckt

#### Scenario: Feld komplett geleert, dann neu getippt — Auge erscheint wieder

- **WHEN** ein autofilled-Feld komplett geleert wird (Wert `""`) und der Nutzer danach mindestens ein Zeichen tippt
- **THEN** erscheint der Reveal-Toggle wieder

---

### Requirement: Toggle-Aktion macht Klartext sichtbar

Bei Klick auf den sichtbaren Reveal-Toggle SHALL das Feld zwischen `type="password"` (maskiert) und `type="text"` (Klartext) umschalten. Das Icon SHALL entsprechend zwischen Lucide `Eye` (aktuell maskiert, Klick zeigt) und Lucide `EyeOff` (aktuell sichtbar, Klick verbirgt) wechseln.

#### Scenario: Klick zeigt Klartext

- **WHEN** der Nutzer auf den Reveal-Toggle klickt während das Feld maskiert ist
- **THEN** wird der Wert im Klartext angezeigt und das Icon wechselt zu `EyeOff`

#### Scenario: Erneuter Klick maskiert wieder

- **WHEN** der Nutzer auf den Reveal-Toggle klickt während das Feld den Klartext zeigt
- **THEN** wird der Wert wieder maskiert und das Icon wechselt zu `Eye`

---

### Requirement: Auto-Hide bei Fokus-Verlust

Verliert das Passwort-Feld den Fokus (`onBlur`), SHALL die Klartext-Anzeige automatisch beendet werden und das Feld zurück auf `type="password"` schalten. Der Tainted-/UserTyped-Status SHALL dabei NICHT zurückgesetzt werden — der Toggle bleibt beim erneuten Fokus weiterhin verfügbar, falls er vorher es war.

#### Scenario: Wegklicken maskiert sofort

- **WHEN** das Passwort im Klartext angezeigt wird und der Nutzer das Feld verlässt (Tab, Klick außerhalb)
- **THEN** wird der Wert sofort wieder maskiert

#### Scenario: Zurückkehren behält Toggle-Berechtigung

- **WHEN** der Nutzer nach einem Blur in das selbe Feld zurückkehrt
- **THEN** ist der Toggle weiterhin sichtbar (falls vorher schon erlaubt), der Wert aber maskiert

---

### Requirement: Kein Timeout-basiertes Auto-Hide

Die Klartext-Anzeige SHALL NICHT durch einen Inaktivitäts-Timer beendet werden. Nur ein expliziter erneuter Klick auf den Toggle oder ein `onBlur`-Ereignis SHALL die Maskierung wiederherstellen.

#### Scenario: Klartext bleibt bei Inaktivität

- **WHEN** der Nutzer den Reveal aktiviert und das Feld fokussiert lässt, ohne weiter zu tippen
- **THEN** bleibt der Klartext sichtbar, ohne nach einer Zeitdauer automatisch zu verschwinden

---

### Requirement: Tastatur- und Screenreader-Bedienbarkeit

Der Reveal-Toggle SHALL als `<button type="button">` implementiert sein (verhindert Form-Submit), in der Tab-Reihenfolge nach dem Passwort-Input liegen, und folgende ARIA-Attribute tragen:

- `aria-label="Passwort anzeigen"` (wenn maskiert) bzw. `"Passwort verbergen"` (wenn sichtbar)
- `aria-pressed="true"` (wenn sichtbar) bzw. `"false"` (wenn maskiert)

#### Scenario: Toggle ist per Tab erreichbar

- **WHEN** der Nutzer mit Tab durch das Login-Formular navigiert
- **THEN** ist der Reveal-Toggle nach dem Passwort-Input fokussierbar

#### Scenario: Enter/Space togglet, ohne Form abzuschicken

- **WHEN** der Reveal-Toggle den Fokus hat und der Nutzer Enter oder Space drückt
- **THEN** wechselt die Maskierung, und das Formular wird NICHT abgeschickt

#### Scenario: ARIA-Label spiegelt aktuellen Zustand

- **WHEN** das Feld maskiert ist
- **THEN** trägt der Toggle `aria-label="Passwort anzeigen"` und `aria-pressed="false"`

- **WHEN** das Feld den Klartext zeigt
- **THEN** trägt der Toggle `aria-label="Passwort verbergen"` und `aria-pressed="true"`

---

### Requirement: Mobile Touch-Target

Der Reveal-Toggle SHALL auf Mobile (`< 640px`) mindestens 44 × 44 px Touch-Target-Fläche haben (CLAUDE.md Mobile-Konvention).

#### Scenario: Touch-Target erfüllt 44px

- **WHEN** das Passwort-Feld auf Mobile gerendert wird
- **THEN** ist der Toggle-Button mindestens 44 px hoch und 44 px breit klickbar (inkl. Padding)
