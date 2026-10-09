## MODIFIED Requirements

### Requirement: Responsive Hauptbereich
Der Hauptbereich (Main Content) SHALL auf Mobilgeräten die volle Viewport-Breite nutzen. Das Padding MUSS auf Mobile `px-4 py-4` betragen (statt `p-8`). Auf dem Desktop trägt der Hauptbereich links einen 3-px-Streifen in `brand-yellow` ohne Rundung und oben links die große Eckfahne (`sm:border-l-[3px] sm:border-brand-yellow`, dazu die große Fahne als eigenes, nicht mitscrollendes Element über dem Inhalt). Diese Dekorationsklassen MÜSSEN auf Mobile deaktiviert sein, da sie ohne sichtbare Sidebar keinen Sinn ergeben.

#### Scenario: Kein unnötiger Whitespace auf Mobile
- **WHEN** der Viewport unter 640px ist
- **THEN** hat der Hauptbereich `px-4 py-4` statt `p-8`

#### Scenario: Keine Dekorationsklassen auf Mobile
- **WHEN** der Viewport unter 640px ist
- **THEN** hat der Hauptbereich weder gelbe linke Border noch Eckfahne

#### Scenario: Desktop-Hauptbereich mit Eckfahne
- **WHEN** der Viewport mindestens 640px breit ist
- **THEN** hat der Hauptbereich einen 3-px-Streifen links ohne abgerundete Ecken und oben links ein 44-px-Dreieck, das beim Scrollen stehen bleibt und über dem Inhalt liegt
