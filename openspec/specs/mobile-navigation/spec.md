# mobile-navigation Specification

## Purpose

Diese Spezifikation beschreibt die Capability `mobile-navigation`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Hamburger-Menü auf Mobilgeräten
Die App SHALL auf Viewports unter 640px Breite anstelle der fixen Sidebar einen Hamburger-Button (☰) in einer mobilen Kopfzeile anzeigen. Die Sidebar MUSS dabei standardmäßig ausgeblendet sein.

#### Scenario: Hamburger-Button sichtbar auf Mobile
- **WHEN** der Viewport weniger als 640px breit ist
- **THEN** ist eine mobile Kopfzeile mit Hamburger-Button (☰) und App-Name sichtbar
- **THEN** ist die Desktop-Sidebar ausgeblendet

#### Scenario: Desktop-Layout unverändert
- **WHEN** der Viewport 640px oder breiter ist
- **THEN** ist die Sidebar sichtbar und der Hamburger-Button ausgeblendet

### Requirement: Overlay-Sidebar öffnet sich bei Klick auf Hamburger
Die App SHALL beim Klick auf den Hamburger-Button eine Overlay-Sidebar öffnen, die den gesamten linken Rand überlagert. Ein halbtransparenter Backdrop MUSS den restlichen Bildschirm abdecken.

#### Scenario: Sidebar öffnet sich
- **WHEN** der Nutzer auf den Hamburger-Button (☰) tippt
- **THEN** öffnet sich die Sidebar als Overlay von links
- **THEN** ist ein Backdrop hinter der Sidebar sichtbar

#### Scenario: Sidebar schließt bei Klick auf Backdrop
- **WHEN** die Overlay-Sidebar geöffnet ist
- **WHEN** der Nutzer auf den Backdrop tippt
- **THEN** schließt sich die Sidebar

#### Scenario: Sidebar schließt nach Navigation
- **WHEN** die Overlay-Sidebar geöffnet ist
- **WHEN** der Nutzer auf einen Nav-Link tippt
- **THEN** navigiert die App zur gewählten Seite
- **THEN** schließt sich die Sidebar automatisch

#### Scenario: Schließen-Button in Sidebar
- **WHEN** die Overlay-Sidebar geöffnet ist
- **WHEN** der Nutzer auf den ✕-Button in der Sidebar tippt
- **THEN** schließt sich die Sidebar

### Requirement: Touch-freundliche Navigation
Alle interaktiven Elemente in der Navigation SHALL eine Mindesthöhe von 44px aufweisen.

#### Scenario: Nav-Links haben ausreichende Tap-Target-Größe
- **WHEN** der Nutzer die Navigation auf einem Mobilgerät nutzt
- **THEN** sind alle Nav-Links und Buttons mindestens 44px hoch

### Requirement: Responsive Hauptbereich
Der Hauptbereich (Main Content) SHALL auf Mobilgeräten die volle Viewport-Breite nutzen. Das Padding MUSS auf Mobile `px-4 py-4` betragen (statt `p-8`). Die Dekorationsklassen (`rounded-tl-3xl rounded-bl-3xl border-l-4 border-brand-yellow`) MÜSSEN auf Mobile deaktiviert sein, da sie ohne sichtbare Sidebar keinen Sinn ergeben.

#### Scenario: Kein unnötiger Whitespace auf Mobile
- **WHEN** der Viewport unter 640px ist
- **THEN** hat der Hauptbereich `px-4 py-4` statt `p-8`

#### Scenario: Keine Dekorationsklassen auf Mobile
- **WHEN** der Viewport unter 640px ist
- **THEN** hat der Hauptbereich keine abgerundeten Ecken oder gelbe linke Border

### Requirement: Sticky Suchleiste auf großen Listen
Auf Seiten mit durchsuchbaren Listen (MembersPage, AdminUsersPage) SHALL die Suchleiste auf Mobile `sticky top-0 z-10` sein, damit sie beim Scrollen durch viele Einträge stets erreichbar bleibt.

#### Scenario: Suchleiste bleibt sichtbar beim Scrollen
- **WHEN** der Nutzer auf Mobile durch eine lange Liste scrollt
- **THEN** bleibt die Suchleiste am oberen Rand des Bildschirms fixiert
