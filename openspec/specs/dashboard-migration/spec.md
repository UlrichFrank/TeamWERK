# dashboard-migration Specification

## Purpose

Diese Spezifikation beschreibt die Capability `dashboard-migration`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Dashboard verwendet ausschließlich Brand-Tokens
`DashboardPage.tsx` und seine Sub-Komponenten SHALL keine Raw-Tailwind-Farben (`text-black/50`, `bg-green-100`, `text-red-500`, etc.) verwenden. Alle Farbklassen MÜSSEN durch `brand-*`-Tokens ersetzt sein.

Das verbindliche Mapping:

| Aktuell | Neu |
|---|---|
| `bg-green-100 text-green-800` (Status „Erfüllt") | `bg-brand-success-light text-brand-text` |
| `bg-yellow-100 text-yellow-800` (Status „Ablöse") | `bg-brand-warning-light text-brand-text` |
| `bg-blue-100 text-blue-800` (Status „Zugesagt") | `bg-brand-info/10 text-brand-text` |
| `text-red-500` (Verletzt-Zahl in TeamStats) | `text-brand-danger` |
| `text-yellow-500` (Pausiert-Zahl in TeamStats) | `text-brand-warning` |
| `text-black/50` (Muted-Text überall) | `text-brand-text-muted` |
| `text-black/40` (Subtil-Text) | `text-brand-text-subtle` |

#### Scenario: Dienstkonto-Status-Badges verwenden Brand-Tokens
- **WHEN** ein Dienstkonto-Eintrag mit Status „Erfüllt" gerendert wird
- **THEN** hat der Badge `bg-brand-success-light` als Hintergrund

#### Scenario: Team-Stats-Verletzt-Zahl ist Karmesin
- **WHEN** die TeamStats-Karte die Anzahl verletzter Spieler anzeigt
- **THEN** hat die Zahl die Klasse `text-brand-danger` (Karmesin `#C0253A`), nicht `text-red-500`

#### Scenario: Muted-Text im Dashboard ist brand-text-muted
- **WHEN** das Dashboard sekundären Text rendert (Datum, Saison-Info, leere Zustände)
- **THEN** verwendet dieser Text `text-brand-text-muted`, nicht `text-black/50`

---

### Requirement: Dashboard-Spielplan-Emojis durch Lucide Icons ersetzt
Die Emojis `🏠` (Heimspiel) und `🚌` (Auswärtsspiel) in der `NextGamesList`-Komponente SHALL durch die Lucide-Icons `Home` und `MapPin` ersetzt werden.

#### Scenario: Heimspiel zeigt Home-Icon
- **WHEN** ein Heimspiel in der Dashboard-Spielliste gerendert wird
- **THEN** erscheint `<Home className="w-4 h-4 inline" />`, kein `🏠`-Emoji

#### Scenario: Auswärtsspiel zeigt MapPin-Icon
- **WHEN** ein Auswärtsspiel in der Dashboard-Spielliste gerendert wird
- **THEN** erscheint `<MapPin className="w-4 h-4 inline" />`, kein `🚌`-Emoji

---

### Requirement: Dashboard-Ladeanimation verwendet Brand-Tokens
Der Skeleton-Loader im Dashboard (`bg-black/5 animate-pulse`) MUST durch `bg-brand-border-subtle animate-pulse` ersetzt werden.

#### Scenario: Lade-Skeleton hat Brand-Farbe
- **WHEN** das Dashboard im Ladezustand gerendert wird
- **THEN** haben die Skeleton-Elemente den Hintergrund `bg-brand-border-subtle`

---


### Requirement: Dashboard-Sektion Fahrtgemeinschaften wird zu Link

Die bisherige `VehicleSection`-Komponente in der „Fahrtgemeinschaften"-Accordion-Sektion SHALL durch eine kompakte Link-Karte zur neuen `/mitfahrgelegenheiten`-Seite ersetzt werden.

**Vorher:** Zeigt Fahrzeuginfo (Sitzplätze) und Link zu `/profil`.

**Nachher:** Zeigt Kurzinfo zum nächsten Auswärtsspiel (Datum, Gegner, Angebots-/Gesuch-Zähler) und einen Link zu `/mitfahrgelegenheiten`.

#### Scenario: Dashboard Fahrtgemeinschaften-Sektion mit nächstem Auswärtsspiel
- **WHEN** es ein zukünftiges Auswärtsspiel gibt
- **THEN** zeigt die Sektion: Datum + Gegner + „X Angebote, Y Gesuche" + Link `[→ Mitfahrgelegenheiten]`

#### Scenario: Kein Auswärtsspiel in Zukunft
- **WHEN** kein zukünftiges Auswärtsspiel existiert
- **THEN** zeigt die Sektion: „Keine Auswärtsfahrten geplant" + Link zur Seite

#### Scenario: Klick auf Link
- **WHEN** der Nutzer auf den Link in der Fahrtgemeinschaften-Sektion klickt
- **THEN** navigiert er zu `/mitfahrgelegenheiten`
