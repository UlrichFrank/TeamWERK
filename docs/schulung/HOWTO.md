# Schulungsfolien für Funktionäre — Howto

Die Präsentation für Vorstand, Trainer und Kassierer: Motivation, der Kern
(Saison → Kader → Team, Nutzer vs. Mitglied, Vereinsfunktionen) und je eine Folie
pro Funktion mit Absicht, Weg im Tool und echtem Screenshot.

```
docs/schulung/
├── HOWTO.md            diese Anleitung
├── folien.txt          die Texte aller Folien — HIER wird editiert
├── shots.txt           Liste der Screenshots (eine Zeile = ein Bild)
├── folien/
│   ├── index.html      fertige Präsentation, von `make folien` erzeugt (nicht von Hand ändern)
│   ├── logo.svg        Vereinslogo (Kopie aus web/public, ohne DOCTYPE)
│   └── img/            Screenshots, von `make schulung` erzeugt (nicht eingecheckt)
├── tools/
│   ├── folien.py       Generator: folien.txt + vorlage.html → folien/index.html
│   ├── vorlage.html    Aussehen: CSS, Präsentations-JS, Rahmen (nur für Design-Änderungen)
│   ├── build.sh        Ablauf hinter `make schulung`
│   ├── anon.py         anonymisiert die DB-Kopie
│   └── shots.mjs       nimmt die Screenshots per Playwright auf
└── .build/             Demo-DB, Binary, Server-Log (Wegwerf, nicht eingecheckt)
```

## Texte ändern: `folien.txt`

Alle Texte stehen in **`folien.txt`**, einer schlichten Textdatei ohne HTML. Nach
dem Ändern:

```bash
make folien            # dauert unter einer Sekunde
```

und `folien/index.html` im Browser neu laden. Die HTML-Datei ist generiert; Änderungen
dort gehen beim nächsten `make folien` verloren.

### Die Schreibweise

```text
---                                   ← neue Folie
typ: funktion                         ← Folientyp (siehe Tabelle)
weg: Spielbetrieb › Termine           ← Feld: name: wert
titel: Termine
absicht: Wer kommt?
- Zusagen und Absagen auf einen Blick ← Punkt
- Eltern sagen für ihr Kind zu
rollen: Alle
bild: vs-termine | /termine           ← Bilddatei | Text in der Adressleiste
hinweis: Beispieldaten
```

- `---` allein auf einer Zeile beginnt eine Folie. Reihenfolge in der Datei = Reihenfolge
  der Folien; die Nummern zählt die Präsentation selbst.
- `name: wert` setzt ein Feld. **Eingerückte Zeilen darunter** gehören zum Feld
  (für mehrzeilige Felder wie `vorher:` oder `fuss:`).
- `- text` ist ein Punkt. **Eingerückte Zeilen darunter** gehören zum Punkt: je nach
  Folientyp als Beschreibung, als Tabellenzeile (`Name = Detail`) oder als Zusatzangabe
  (`weg: …`, `warn: …`).
- `**fett**` hebt hervor, `&shy;` markiert eine erlaubte Silbentrennung in langen
  Wörtern (`Nutzer&shy;verwaltung`).
- `#` am Zeilenanfang ist ein Kommentar und erscheint nicht auf den Folien.

Tippfehler meldet `make folien` mit Zeilennummer, etwa
`folien.txt:118: Feld(er) titl passen nicht zu typ „funktion“ — erlaubt: …`.

### Folientypen

| `typ:`            | Wofür                                   | Felder                                                              | Punkte (`-`)                                                         |
|-------------------|-----------------------------------------|---------------------------------------------------------------------|----------------------------------------------------------------------|
| `titel`           | Titel- und Schlussfolie (gelb)          | `eyebrow`, `titel`, `untertitel`, `chips` (Komma-Liste), `fuss`     | —                                                                    |
| `kapitel`         | Kapiteltrenner (schwarz)                | `teil`, `titel`, `text`                                             | —                                                                    |
| `funktion`        | Funktion mit Screenshot                 | `weg`, `titel`, `absicht`, `rollen`, `rollen-text`, `rollen-zusatz`, `bild`, `bild-ganz`, `alt`, `hinweis` | Stichpunkte (höchstens drei)                 |
| `handy`           | wie `funktion`, mit zwei Handybildern   | wie `funktion`, aber `bilder: datei1, datei2` statt `bild`          | Stichpunkte                                                          |
| `vorher-nachher`  | Bisher/Jetzt                            | `titel`, `vorher` (eine Zeile je Punkt), `nachher` (letzte Zeile gelb), `text` | —                                                         |
| `aussagen`        | drei Aussagen nebeneinander             | `titel`                                                             | `- Überschrift`, eingerückt der Text                                 |
| `modell`          | Schaubild mit Pfeilen                   | `titel`, `notiz`                                                    | `- Kategorie: Name`, eingerückt Text oder `Name = Detail`-Zeilen; der erste Kasten ist gelb |
| `vergleich`       | zwei Kästen und Beispiele               | `titel`, `verbindung`                                               | die ersten zwei = Kästen (`- Kategorie: Name` + Text), alle weiteren = Beispiele |
| `raster`          | Kachelraster mit Rollen                 | `titel`, `fuss` (eine Zeile je Hinweis)                             | `- Rolle`, eingerückt der Text                                       |
| `liste`           | Kurzliste ohne Bilder                   | `titel`                                                             | `- Name`, eingerückt `weg: …` und der Text                           |
| `schritte`        | nummerierte Schrittfolge                | `titel`                                                             | `- Schritt`, eingerückt `weg: …` und/oder `warn: …` (gelb)           |

Felder der Funktionsfolie im Einzelnen:

| Feld            | Bedeutung                                                                          |
|-----------------|------------------------------------------------------------------------------------|
| `weg`           | Navigation im Tool, Teile mit `›` getrennt; der letzte Teil wird fett              |
| `absicht`       | ein Satz: wozu gibt es die Funktion (gelb markiert)                                |
| `rollen`        | getrennt durch Komma oder `/`: `Alle`, `Spieler`, `Trainer`, `Sportliche Leitung`/`Sportl. Leitung`, `Vorstand`, `Kassierer`, `Medien` |
| `rollen-text`   | Wort vor den Rollen, Standard `für` (z. B. `anlegen`, `verwalten`)                 |
| `rollen-zusatz` | kleiner Text nach den Rollen (z. B. `je nach Ordnerrecht`)                         |
| `bild`          | `datei | adresse` — Datei aus `folien/img/` ohne `.jpg`, dahinter der Text der Adressleiste |
| `bild-ganz`     | `ja` zeigt den Screenshot mit Tool-Seitenleiste (sonst wird sie abgeschnitten)     |
| `alt`           | Beschreibung des Bilds für Screenreader (Standard: „Screenshot: Titel“)            |
| `hinweis`       | Kleingedrucktes unten links, Standard `Beispieldaten · Namen anonymisiert`         |

Faustregel für den Stil: ein Satz Absicht, höchstens drei kurze Punkte, lieber das
Bild sprechen lassen. Eine neue Folie legst du am einfachsten an, indem du eine
vorhandene gleichen Typs kopierst.

### Aussehen ändern

Farben, Schriften und Layout stehen in `tools/vorlage.html` (CSS + das kleine
Präsentations-Skript). Der Platzhalter `<!-- FOLIEN -->` markiert, wo `folien.py` die
Folien einsetzt. Einen neuen Folientyp legst du in `tools/folien.py` an: Eintrag in
`TYPES` (erlaubte Felder), eine `r_…`-Funktion, die das HTML baut, und ein Eintrag in
`RENDER`.

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
3. In `folien.txt` an der Folie `bild: <datei> | <adresse>` setzen, dann `make folien`.
   Steht ein Bild in `folien.txt`, aber nicht in `shots.txt`, weist `make folien` darauf hin.

Die Tool-Seitenleiste wird auf den Funktionsfolien abgeschnitten (der Weg steht ja
links auf der Folie). Soll ein Bild ganz sichtbar sein: `bild-ganz: ja`.

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
