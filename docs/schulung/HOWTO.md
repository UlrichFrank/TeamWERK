# Schulungsfolien für Funktionäre — Howto

Die Präsentation für Vorstand, Trainer und Kassierer: Motivation, der Kern
(Saison → Kader → Team, Nutzer vs. Mitglied, Vereinsfunktionen) und je eine Folie
pro Funktion mit Absicht, Weg im Tool und echtem Screenshot.

```
docs/schulung/
├── HOWTO.md            diese Anleitung
├── shots.txt           Liste der Screenshots (eine Zeile = ein Bild)
├── folien/
│   ├── index.html      die Präsentation — reiner Text, hier wird editiert
│   ├── logo.svg        Vereinslogo (Kopie aus web/public, ohne DOCTYPE)
│   └── img/            Screenshots, von `make schulung` erzeugt (nicht eingecheckt)
├── tools/
│   ├── build.sh        Ablauf hinter `make schulung`
│   ├── anon.py         anonymisiert die DB-Kopie
│   └── shots.mjs       nimmt die Screenshots per Playwright auf
└── .build/             Demo-DB, Binary, Server-Log (Wegwerf, nicht eingecheckt)
```

## Die Basis: eine HTML-Datei

Die Folien sind **eine einzige Textdatei**, `folien/index.html`. Es gibt kein
Framework, keinen Build-Schritt und kein PowerPoint: Datei im Editor ändern,
speichern, im Browser neu laden. Schrift (Hanken Grotesk, IBM Plex Mono) kommt von
Google Fonts. Offline fällt sie auf Systemschriften zurück.

Jede Folie ist ein Block zwischen `<section class="slide-wrap">` und `</section>`,
davor steht ein Kommentar zur Orientierung (`<!-- 16 Termine -->`). Die
Foliennummer unten rechts zählt CSS automatisch, die Nummer im Kommentar ist nur
eine Suchhilfe.

### Text ändern

Eine Funktionsfolie sieht so aus — jede Zeile ist ein Baustein:

```html
<section class="slide-wrap"><div class="slide s-feature">
  <div class="copy">
    <p class="eyebrow">Spielbetrieb › <b>Termine</b></p>          <!-- Weg im Tool -->
    <h2>Termine</h2>                                              <!-- Titel -->
    <p class="intent"><mark>Wer kommt?</mark></p>                 <!-- Absicht, gelb markiert -->
    <ul class="points">                                           <!-- höchstens drei Punkte -->
      <li>Zusagen, Absagen und offene Rückmeldungen auf einen Blick</li>
      <li>Eltern sagen für ihr Kind zu</li>
    </ul>
    <div class="roles-row"><span class="lbl">für</span><span class="chip c-all">Alle</span></div>
  </div>
  <figure class="shot"><div class="urlbar">/termine</div><img src="img/vs-termine.jpg" alt="…"></figure>
  <p class="demo">Beispieldaten</p>
</div></section>
```

Rollen-Kennzeichen (`chip`):

| Klasse        | Aussehen          | für                            |
|---------------|-------------------|--------------------------------|
| `c-all`       | grau              | Alle                           |
| `c-trainer`   | blau              | Trainer, Sportliche Leitung    |
| `c-vorstand`  | schwarz/gelb      | Vorstand                       |
| `c-kasse`     | Umriss            | Kassierer                      |
| `c-medien`    | grün              | Medien                         |

Faustregel für den Stil: ein Satz Absicht, höchstens drei kurze Punkte, lieber das
Bild sprechen lassen.

### Folien hinzufügen, entfernen, umsortieren

Einen `<section class="slide-wrap">…</section>`-Block kopieren, löschen oder
verschieben. Mehr ist nicht nötig.

| Folientyp (Klasse am `slide`-div) | Wofür                                   | Vorlage im Deck |
|-----------------------------------|-----------------------------------------|-----------------|
| `s-yellow s-title`                | Titel und Schluss                       | Folie 1, 33     |
| `s-dark s-section`                | Kapiteltrenner                          | Folie 4, 13     |
| `s-feature`                       | Funktion mit Screenshot                 | Folie 6–29      |
| `s-text` + `before-after`         | Vorher/Nachher                          | Folie 2         |
| `s-text` + `gains`                | drei Aussagen nebeneinander             | Folie 3         |
| `s-text` + `model-wrap`           | Schaubild mit Pfeilen                   | Folie 5         |
| `s-text` + `pair` + `cases`       | Gegenüberstellung mit Beispielen        | Folie 8         |
| `s-text` + `funcs`                | Kachelraster                            | Folie 11        |
| `s-phones`                        | zwei Handy-Screenshots                  | Folie 30        |
| `s-text` + `more`                 | Liste ohne Bilder                       | Folie 31        |
| `s-text` + `steps`                | nummerierte Schrittfolge                | Folie 32        |

## Screenshots neu erzeugen: `make schulung`

```bash
make schulung
```

Das Target erzeugt alle Bilder in `folien/img/` neu, immer aus dem aktuellen Stand
des Tools. Ablauf (`tools/build.sh`):

1. Konsistente Kopie der **lokalen** `teamwerk.db` nach `.build/demo.db`
   (die Prod-DB auf dem VPS wird nie angefasst; ein Pfad unter `/var/lib/teamwerk` wird verweigert).
2. **Anonymisieren** (`anon.py`): Namen, E-Mails, Telefonnummern, Adressen, Geburtstage
   (Jahr bleibt), Chat-Texte, Kommentare und Gründe werden ersetzt, Übungsgruppen
   umbenannt. Steht danach noch ein echter Voll-Name in einem Freitext, bricht das Skript ab.
3. Frontend und Binary bauen, Demo-Server auf der Kopie starten — leere Medienordner
   (keine echten Fotos), kein Mailversand, kein Push.
4. Screenshots laut `shots.txt` aufnehmen (Desktop 1600 px, Handy 600 px breit, JPEG).

Dauer: etwa 3–4 Minuten (mit `SKIP_WEB_BUILD=1` etwas weniger). Der Server wird
danach beendet; Demo-DB und Server-Log bleiben bis zum nächsten Lauf in `.build/`.

**Voraussetzungen** (einmalig): eine lokale `teamwerk.db` (z. B. `make pull-db`),
`pnpm -C web install`, Playwright-Chromium (`pnpm -C web exec playwright install chromium`),
`sqlite3`, `python3`.

**Optionen:**

| Variable                | Wirkung                                                       |
|-------------------------|---------------------------------------------------------------|
| `SRC_DB=pfad/zur.db`    | andere Quell-DB als `./teamwerk.db`                           |
| `SKIP_WEB_BUILD=1`      | vorhandenes Frontend-Build nutzen (schneller)                 |
| `KEEP=1`                | Demo-Server danach laufen lassen, um selbst durchzuklicken    |
| `SCHULUNG_VORSTAND=10`  | Nutzer-ID, die zur Persona `vorstand@beispiel.de` wird        |
| `SCHULUNG_TRAINER=11`   | Nutzer-ID, die zur Persona `trainer@beispiel.de` wird         |
| `SCHULUNG_PORT=18090`   | Port des Demo-Servers                                         |

Beispiel: `make schulung KEEP=1 SKIP_WEB_BUILD=1` — danach unter
`http://127.0.0.1:18090` mit `vorstand@beispiel.de` oder `trainer@beispiel.de`,
Passwort `Schulung2026!`, anmelden.

Die Personas müssen passende Vereinsfunktionen haben (Vorstand bzw. Trainer eines
Teams in der aktiven Saison). `anon.py` meldet sie beim Lauf und warnt, wenn eine
Funktion fehlt.

### Einen Screenshot hinzufügen oder ändern

1. Zeile in `shots.txt` ergänzen:

   ```
   persona | route | datei | klick | scrollen
   trainer | /termine/spiel/{trainer_game} | tr-termin_detail
   vorstand | /mitglieder/{vorstand_member} | vs-mitglied_stammdaten | Stammdaten | Vereinsfunktion
   ```

   - `persona`: `vorstand`, `trainer`, `vorstand-mobil`, `trainer-mobil`
   - `route`: Pfad im Tool; Platzhalter `{vorstand_member}`, `{trainer_team}`,
     `{trainer_game}` füllt `anon.py` passend zur DB (IDs ändern sich mit jeder neuen DB-Kopie)
   - `klick` (optional): Beschriftung eines Tabs/Buttons, der vorher geklickt wird
   - `scrollen` (optional): Text, der an den oberen Bildrand gescrollt wird
2. `make schulung SKIP_WEB_BUILD=1`
3. Das Bild in `index.html` einbinden: `<img src="img/<datei>.jpg" alt="…">`

Die Tool-Seitenleiste wird auf den Funktionsfolien per CSS abgeschnitten (der Weg steht
ja links auf der Folie). Soll ein Bild ganz sichtbar sein, am `<img>` das Attribut
`style="width:100%;margin-left:0"` setzen.

## Präsentieren und als PDF speichern

- **Öffnen:** `open docs/schulung/folien/index.html` (oder per Doppelklick).
- **Präsentieren:** Button „Präsentieren“ oder Taste `P` → Vollbild. Blättern mit
  Pfeiltasten, Leertaste oder Klick (rechte Hälfte vor, linke zurück), `Esc` beendet.
- **PDF:** Im Browser drucken (`Cmd+P`) → „Als PDF sichern“. Jede Folie wird eine
  Seite im 16:9-Format. In Chrome unter „Weitere Einstellungen“ **Hintergrundgrafiken**
  aktivieren und Ränder auf „Keine“ stellen, sonst fehlen gelbe und schwarze Flächen.
- **Handy/Tablet:** Auf schmalen Bildschirmen stapeln sich Text und Bild untereinander.

## Online teilen

Die Präsentation ist als privates Claude-Artefakt veröffentlicht:
<https://claude.ai/artifact/Xjz7ziCLVGBRVzHJFDKUcB>

Nach Änderungen in Claude Code aktualisieren, z. B.:

> Veröffentliche `docs/schulung/folien/index.html` samt `logo.svg` und `img/*.jpg`
> als Update von https://claude.ai/artifact/Xjz7ziCLVGBRVzHJFDKUcB

Freigeben kannst du sie über das Teilen-Menü der Seite. Das Logo darf keine
`<!DOCTYPE>`-Zeile enthalten, sonst lehnt die Veröffentlichung es ab — deshalb liegt
hier eine bereinigte Kopie statt eines Verweises auf `web/public/logo.svg`.

## Datenschutz — vor dem Teilen

Die Screenshots zeigen das **echte Tool mit echter Struktur**, aber **ersetzten Personen**.

- **Ersetzt:** Namen (auch in Meldungs- und Termintexten), E-Mails, Logins,
  Telefonnummern, Adressen, Geburtstag und -monat, Chat- und Mitteilungstexte,
  Kommentare, Absage- und Abwesenheitsgründe, Namen von Übungsgruppen.
- **Nicht ersetzt:** Team-, Hallen- und Gegnernamen, Vereinsname und -adresse,
  Hinweistexte an Terminen, Namen von Chat-Gruppen, Zahlen (Zusagen, Dienste, Beiträge).
  Stehen dort Spitznamen oder persönliche Angaben, erkennt das Skript sie nicht.
- **Deshalb:** Jeden neuen Screenshot vor dem Teilen einmal ansehen. Taucht etwas
  Persönliches auf, `anon.py` erweitern (Liste `TEXT` für Namen in Freitexten oder
  eine eigene `UPDATE`-Zeile) und neu erzeugen.
- `folien/img/` und `.build/` sind in `.gitignore` — Demo-DB und Bilder landen nicht im Repo.
