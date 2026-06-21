## Why

Auf Android hat die installierte PWA zwei sichtbare Icon-Mängel:

1. **Push-Notifications zeigen in der Status-Bar nur einen weißen Kreis** statt eines erkennbaren App-Symbols. Ursache: Android rendert das Status-Bar-Icon (`badge`) **ausschließlich aus dem Alpha-Kanal** und füllt alle opaken Pixel flächig weiß — Farbe wird verworfen. Das aktuell verwendete `icon-192.png` ist eine vollflächig opake Kreisfläche → daraus wird ein solider weißer Kreis. Ein farbiges Mini-App-Icon ist auf Android technisch nicht möglich; die Lösung ist eine eigens gestaltete **monochrome Silhouette**.

2. **Das App-Icon wird am Rand zu stark beschnitten.** Das Manifest deklariert `icon-512.png` als `purpose: "any maskable"`, das Bild füllt aber randlos bis zur Kante (Schriftring „TEAM STUTTGART / HANDBALL" berührt den Rand). Android garantiert für maskable Icons nur eine **Safe Zone von 80 % Durchmesser** — alles außerhalb wird je nach Launcher-Maske weggeschnitten, daher fehlt der äußere Ring.

Der Nutzer hat zwei passende Quell-SVGs erstellt: `IconAndroid.svg` (gelber Kreis mit Schutzrand) und `Handball.svg` (Pferd-im-Ball als Silhouette).

## What Changes

- **Neues maskable Icon** `icon-maskable-512.png`, gerendert aus `IconAndroid.svg` mit **weißem Hintergrund** (`-b '#FFFFFF'`), damit auch Squircle-/Rechteck-Launcher keine transparenten Ecken zeigen. Das Logo sitzt innerhalb der 80 %-Safe-Zone.
- **Neues Badge-Icon** `badge-96.png`, gerendert aus `Handball.svg` mit **transparentem Hintergrund** (96×96). Der Service Worker nutzt es als `badge:` → Android zeigt die weiße Pferd-Silhouette statt eines weißen Klecks.
- **Manifest auf eine Quelle konsolidiert:** Die statische `web/public/manifest.json` und ihr `<link rel="manifest">` in `index.html` werden entfernt; einzige Quelle ist das von VitePWA generierte Manifest aus `vite.config.ts`. Dort wird das gemischte `purpose: "any maskable"` in **getrennte Einträge** aufgeteilt (`any` für die bestehenden Vollbild-Icons, `maskable` für das neue gepaddete Icon).
- **Reproduzierbares Generierungsskript** `scripts/gen-icons.sh` (rsvg-convert) erzeugt beide PNGs aus den Quell-SVGs; die fertigen PNGs werden zusätzlich eingecheckt (wie bisher). Quell-SVGs ziehen nach `web/icon-src/` (nicht ausgeliefert).

### Bewusste Nicht-Änderungen

- **iOS bleibt unangetastet.** iOS ignoriert das Web-Manifest für das Homescreen-Icon (nutzt `<link rel="apple-touch-icon">`) und zeigt bei Push automatisch das farbige App-Icon — der weiße-Klecks-Effekt existiert dort nicht. Der heutige `apple-touch-icon.png` bleibt wie er ist (transparente Ecken → iOS-typische schwarze Ecken bleiben Status quo; bewusst nicht in diesem Change behoben).
- Das `icon:`-Feld der Notification (große, farbige Vorschau) bleibt das bunte `icon-192.png`.
- Keine Änderung an Push-Versandlogik, Payload oder Backend.

## Capabilities

### New Capabilities

- `android-pwa-icons`: Korrekt beschnittenes maskable App-Icon und erkennbares monochromes Notification-Badge für die Android-PWA, gespeist aus reproduzierbar generierten Assets bei einer einzigen Manifest-Quelle.

## Impact

- `web/public/icons/icon-maskable-512.png` — NEU (maskable, weißer Grund)
- `web/public/icons/badge-96.png` — NEU (transparente Silhouette)
- `web/icon-src/IconAndroid.svg`, `web/icon-src/Handball.svg` — NEU (Quell-SVGs, aus Repo-Root verschoben)
- `scripts/gen-icons.sh` — NEU (rsvg-convert-Pipeline)
- `web/vite.config.ts` — Manifest-`icons[]` auf drei Einträge mit getrennten Purposes
- `web/index.html` — `<link rel="manifest" href="/manifest.json">` entfernt (apple-touch-icon bleibt)
- `web/public/manifest.json` — GELÖSCHT (Konsolidierung)
- `web/src/sw.ts` — `badge:` zeigt auf `/icons/badge-96.png`
- Kein Backend-, Schema- oder Routen-Change; keine neue Migration
