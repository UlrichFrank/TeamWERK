## ADDED Requirements

### Requirement: Maskable App-Icon respektiert die Android Safe Zone

Das Web-App-Manifest SHALL einen eigenen Icon-Eintrag mit `purpose: "maskable"` führen, dessen Bild den sichtbaren Logo-Inhalt vollständig innerhalb der zentralen 80 %-Safe-Zone platziert und dessen gesamte Fläche (inklusive Ecken) opak mit Weiß (`#FFFFFF`) gefüllt ist. Die bestehenden Vollbild-Icons SHALL ausschließlich `purpose: "any"` tragen; ein kombiniertes `"any maskable"` SHALL nicht mehr vorkommen.

#### Scenario: Launcher mit Kreis-Maske
- **WHEN** Android die installierte PWA mit einer kreisförmigen Icon-Maske darstellt
- **THEN** ist der vollständige Logo-Inhalt (Schriftring „TEAM STUTTGART / HANDBALL" und Pferd) sichtbar und nicht am Rand abgeschnitten

#### Scenario: Launcher mit Squircle-/Rechteck-Maske
- **WHEN** Android die installierte PWA mit einer Squircle- oder abgerundet-rechteckigen Maske darstellt
- **THEN** erscheinen keine transparenten Ecken; der Hintergrund außerhalb des gelben Kreises ist weiß

#### Scenario: Kein gemischter Purpose
- **WHEN** das generierte Manifest (`manifest.webmanifest`) gelesen wird
- **THEN** existiert genau ein Eintrag mit `purpose: "maskable"` und kein Eintrag mit `purpose: "any maskable"`

### Requirement: Notification-Badge ist eine monochromfähige Silhouette

Der Service Worker SHALL beim Anzeigen einer Push-Notification als `badge` ein eigenes Bild (`/icons/badge-96.png`) verwenden, das einen transparenten Hintergrund und eine als Form erkennbare Silhouette (Pferd im Ball) besitzt. Das `icon`-Feld (große, farbige Vorschau) SHALL weiterhin das farbige App-Icon (`/icons/icon-192.png`) verwenden.

#### Scenario: Push erscheint in der Android Status-Bar
- **WHEN** auf Android eine Push-Notification eintrifft und Android das Badge-Icon monochrom aus dem Alpha-Kanal rendert
- **THEN** erscheint die weiße Pferd-Silhouette und nicht eine vollflächig weiße Kreisfläche

#### Scenario: Aufgeklappte Notification
- **WHEN** der Nutzer die Notification aufklappt
- **THEN** zeigt die große Vorschau (`icon`) das farbige App-Icon

### Requirement: Eine einzige Manifest-Quelle

Das Projekt SHALL das Web-App-Manifest ausschließlich aus der VitePWA-Konfiguration (`web/vite.config.ts`) generieren. Eine statische `web/public/manifest.json` SHALL nicht existieren, und `web/index.html` SHALL keinen manuellen `<link rel="manifest">` enthalten. Der `<link rel="apple-touch-icon">` SHALL erhalten bleiben.

#### Scenario: Gebautes Dokument hat genau einen Manifest-Link
- **WHEN** `pnpm -C web build` ausgeführt und das gebaute `index.html` inspiziert wird
- **THEN** existiert genau ein `<link rel="manifest">` (von VitePWA injiziert) und keine statische `manifest.json` im Output

#### Scenario: Manifest-Änderungen greifen verlässlich
- **WHEN** die Icon-Liste in `web/vite.config.ts` geändert wird
- **THEN** spiegelt das ausgelieferte Manifest diese Änderung wider, ohne dass eine zweite Quelle sie überschreibt

### Requirement: iOS-Verhalten bleibt unverändert

Dieser Change SHALL das iOS-Homescreen- und Push-Verhalten nicht verändern. Der bestehende `web/public/icons/apple-touch-icon.png` SHALL unangetastet bleiben.

#### Scenario: iOS-Push nach der Änderung
- **WHEN** auf einer installierten iOS-PWA (ab iOS 16.4) eine Push-Notification eintrifft
- **THEN** zeigt iOS weiterhin automatisch das farbige App-Icon, unbeeinflusst vom neuen `badge`-Asset

#### Scenario: iOS-Homescreen-Icon nach der Änderung
- **WHEN** die PWA auf iOS zum Homescreen hinzugefügt wird
- **THEN** wird weiterhin der unveränderte `apple-touch-icon.png` verwendet (das maskable Manifest-Icon hat keinen Effekt auf iOS)
