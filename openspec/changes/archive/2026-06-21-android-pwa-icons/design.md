# Design — android-pwa-icons

## Kontext: Wie Android PWA-Icons rendert

Zwei voneinander unabhängige Android-Mechanismen sind betroffen, beide mit nicht-offensichtlichen Regeln:

### 1. Notification-Badge (Status-Bar) — Alpha-only, monochrom

`ServiceWorkerRegistration.showNotification(title, { badge })` liefert das kleine Symbol in der Status-Bar und neben dem App-Namen. Android **verwirft die Farbe** und füllt sämtliche opaken Pixel flächig weiß (getönt mit der System-Akzentfarbe). Nur der **Alpha-Kanal** zählt.

```
   Quelle (icon-192, vollfarbig)     Android-Ergebnis (Alpha → Weiß)
   ┌───────────┐                     ┌───────────┐
   │ ████████  │   nur Alpha         │           │  transparente Ecken
   │█ vollfarb █│  ───────────►       │   █████   │  opaker Kreis = solide WEISS
   │ ████████  │                     │           │
   └───────────┘                     └───────────┘
```

→ Ein farbiges Mini-Icon ist **nicht erreichbar**. Lösung: dedizierte Silhouette mit transparentem Hintergrund, deren *Form* (das aufbäumende Pferd im Ball) auch in reinem Weiß erkennbar bleibt. Das ist `Handball.svg`.

### 2. Maskable Icon — 80 % Safe Zone

`purpose: "maskable"` erlaubt dem Launcher eine beliebige Maske (Kreis, Squircle, Rechteck). Garantiert sichtbar ist nur die zentrale **Safe Zone = Kreis mit 80 % Kantenlänge**; die äußeren ~10 % je Seite dürfen weggeschnitten werden. Das heutige randlose Logo verliert dort den Schriftring.

`IconAndroid.svg` löst das: gelber Kreis mit ~8 % transparentem Rand, Logo-Inhalt sicher innen.

## Entscheidungen

### E1 — Maskable: weißer Hintergrund statt transparent oder gelb

`IconAndroid.svg` hat transparente Ecken. Auf Kreis-Masken (Pixel/Stock-Android) ideal, auf Squircle-Masken (z. B. Samsung) blitzen die Ecken durch (Launcher-Hintergrund).

| Option | Kreis-Maske | Squircle | Bewertung |
|---|---|---|---|
| transparent (so lassen) | ✅ | ⚠ Ecken leer | nur Pixel/Stock sauber |
| gelb füllen | ✅ | ✅ | Kreisrand verschwimmt mit Grund |
| **weiß füllen** ← gewählt | ✅ | ✅ | gelber Kreis bleibt klar abgegrenzt; konventioneller PWA-Look; passt zu `background_color: #FFFFFF` |

Umsetzung ohne SVG-Änderung: Render mit `rsvg-convert -b '#FFFFFF'` füllt die transparenten Ecken weiß.

### E2 — Manifest auf VitePWA konsolidieren

Heute existieren **zwei** Manifeste, die driften können:
- statisch `web/public/manifest.json`, verlinkt in `index.html` (`<link rel="manifest" href="/manifest.json">`)
- generiert von VitePWA aus `vite.config.ts` (`manifest.webmanifest`, Link wird automatisch injiziert)

Browser verwenden den **ersten** `<link rel="manifest">` → vermutlich die statische Datei, d. h. Manifest-Änderungen am VitePWA-Block greifen evtl. gar nicht. Lösung: statische Datei + manuellen Link entfernen, **einzige Quelle = `vite.config.ts`**. VitePWA injiziert den Manifest-Link selbst.

`<link rel="apple-touch-icon">` bleibt erhalten — iOS braucht ihn (siehe E4).

### E3 — Purposes trennen statt mischen

`"any maskable"` an einem Bild ist der Grund für die Beschneidung: dasselbe randlose Bild wird sowohl unmaskiert als auch maskiert verwendet. Künftig:

```js
icons: [
  { src: '/icons/icon-192.png',          sizes: '192x192', type: 'image/png', purpose: 'any' },
  { src: '/icons/icon-512.png',          sizes: '512x512', type: 'image/png', purpose: 'any' },
  { src: '/icons/icon-maskable-512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
]
```

### E4 — iOS bewusst außen vor

iOS spielt nach eigenen Regeln und ist von den Android-Fixes nicht betroffen:

| Aspekt | Android | iOS |
|---|---|---|
| Homescreen-Icon-Quelle | manifest `icons[]` | `<link apple-touch-icon>` (manifest ignoriert) |
| Icon-Beschnitt | Safe Zone 80 % | nur leichte Eckenrundung |
| Transparenz | erlaubt | wird mit **Schwarz** gefüllt |
| Notification-Icon | `badge:` → mono weiß | nutzt automatisch das farbige App-Icon |

Daraus folgt: maskable-Eintrag und Badge wirken auf iOS **nicht** (schaden aber auch nicht). Das iOS-Pendant-Problem (transparente Ecken des `apple-touch-icon.png` → schwarze Ecken) ist real, aber Status quo und **nicht Teil dieses Change** (Nutzerentscheidung). Ein späterer Zweizeiler kann es bei Bedarf beheben.

### E5 — Assets generieren statt nur committen

Es gibt heute keine Icon-Pipeline (nur committете PNGs). Wir behalten committете PNGs (kein Build-Time-Tool-Zwang, funktioniert in CI/Deploy ohne rsvg-convert) **und** legen `scripts/gen-icons.sh` für Reproduzierbarkeit dazu. Quell-SVGs ziehen nach `web/icon-src/` — außerhalb von `web/public/`, damit die großen SVGs nicht unnötig ausgeliefert werden.

```bash
# scripts/gen-icons.sh (Auszug)
rsvg-convert -w 512 -h 512 -b '#FFFFFF' web/icon-src/IconAndroid.svg -o web/public/icons/icon-maskable-512.png
rsvg-convert -w 96  -h 96               web/icon-src/Handball.svg    -o web/public/icons/badge-96.png
```

## Verhältnis zu PR #46 (chat-unread-app-badge)

PR #46 ist bereits in `origin/main` gemerged (`be850be`) und fasst denselben `push`-Handler in `web/src/sw.ts` an. **Es gibt keine semantische Kollision, aber eine Begriffsüberladung auf denselben Zeilen** — beide Änderungen müssen koexistieren:

```
web/src/sw.ts  push-Handler (nach beiden Changes)
│
├─ showNotification(title, { icon: '...', badge: '...' })
│     • icon:  '/icons/icon-192.png'   (bunte Vorschau, unverändert)
│     • badge: '/icons/badge-96.png'   ← DIESER Change: Notification-Badge-ICON (URL)
│
└─ navigator.setAppBadge(data.badge)   ← PR #46: App-Icon-Badge = ungelesen-ZAHL (number)
```

- **Notification-Badge-Icon** (dieser Change): das monochrome Status-Bar-Symbol, übergeben als `badge:`-**URL** an `showNotification`. PR #46 ließ diese Zeile auf `/icons/icon-192.png` stehen — dieser Change ändert sie auf `/icons/badge-96.png`.
- **App-Icon-Badge** (PR #46): die ungelesen-**Zahl** auf dem App-Icon, gesetzt über `navigator.setAppBadge(data.badge)` aus dem Payload-Feld `badge: number`. Bleibt hier unangetastet.

Konsequenzen für die Umsetzung:
1. **Erst `origin/main` pullen** (lokaler `main` war 1 Commit hinter), damit der Edit auf dem PR-#46-Handler aufsetzt — sonst Merge-Konflikt genau an der `badge:`-Zeile.
2. Der Edit ist nur die `badge:`-URL innerhalb von `showNotification`; die `setAppBadge`-Logik von PR #46 bleibt erhalten.
3. Empfehlung: kurzer Kommentar im SW, der die zwei Bedeutungen von „badge" trennt (Icon vs. Zahl), damit künftige Leser sie nicht verwechseln.

PR #46 berührt sonst nur Backend (`internal/chat`, `internal/push`), `AppShell.tsx` und `CLAUDE.md` — **keine** Überschneidung mit Manifest, `vite.config.ts`, `index.html` oder den Icon-Assets dieses Change.

## Verifikation

Kein automatischer Test kann das tatsächliche Android-Rendering prüfen — der Kern ist **manuelle Geräteverifikation**:
- Push auslösen → Status-Bar zeigt Pferd-Silhouette (nicht weißer Kreis).
- App zum Homescreen hinzufügen → Logo vollständig sichtbar, kein abgeschnittener Rand.

Automatisch absicherbar sind die **Regressions-Invarianten** (siehe Test-Anforderungen): Asset-Dateien existieren, Manifest hat genau einen maskable-Eintrag und keine doppelte Quelle mehr, Service Worker referenziert das Badge.

## Test-Anforderungen

Dieser Change fügt **keine HTTP-Route und keine Geschäftslogik** hinzu — der Standard „Route → Happy-Path + Fehlerfall" ist nicht anwendbar. Stattdessen leichte Frontend-Invarianten-Tests (Vitest) plus manuelle Gerätekontrolle.

| Invariante | Test | Erwartung |
|---|---|---|
| Maskable-Asset existiert | `icons.test.ts` | `web/public/icons/icon-maskable-512.png` vorhanden, > 1 KB |
| Badge-Asset existiert | `icons.test.ts` | `web/public/icons/badge-96.png` vorhanden, > 1 KB |
| Keine doppelte Manifest-Quelle | `icons.test.ts` | `web/public/manifest.json` existiert NICHT; `index.html` enthält kein `rel="manifest"` |
| Purposes getrennt | `icons.test.ts` | `vite.config.ts` deklariert genau einen Eintrag mit `purpose: 'maskable'` und keinen mit `'any maskable'` |
| SW nutzt Badge | `icons.test.ts` | `web/src/sw.ts` referenziert `/icons/badge-96.png` als `badge` |

Garantierte Invariante: Solange diese Tests grün sind, kann das gemischte `any maskable` nicht versehentlich zurückkehren, das Badge nicht auf das Vollbild-Icon zurückfallen und die Manifest-Redundanz nicht wieder einziehen.
