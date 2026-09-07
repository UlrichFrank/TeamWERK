# TeamWERK — Anleitung für Trainer

Als Trainer hast du erweiterten Zugriff auf Verwaltungsfunktionen für dein Team. Diese Anleitung beschreibt alle Funktionen, die über die Möglichkeiten eines normalen Mitglieds hinausgehen.

Alle Funktionen aus der **[Spieler-Anleitung](anleitung-spieler.md)** stehen dir ebenfalls zur Verfügung.

---

## Überblick deiner Berechtigungen

| Funktion | Spieler | Trainer |
|---|:---:|:---:|
| Eigenes Profil, Dienstbörse, Kalender | ✓ | ✓ |
| Mitgliederliste lesen | ✓ | ✓ |
| Dienst-Slots anlegen und verwalten | — | ✓ |
| Dienste als erfüllt markieren / Geldersatz buchen | — | ✓ |
| Beitrittsanträge bearbeiten | — | ✓ |
| Einladungen versenden | — | ✓ |
| Spielplan verwalten | — | ✓ |
| Kader verwalten (Stamm- und erweiterter Kader) | — | ✓ |
| Aufstellung pflegen | — | ✓ |
| Spieler dauerhaft von einer Trainingsserie abmelden | — | ✓ |
| Änderungsanträge genehmigen | — | ✓ |

---

## Mitgliederverwaltung

### Mitgliederliste

Unter **„Mitglieder"** siehst du alle Vereinsmitglieder. Als Trainer hast du zusätzlich Zugriff auf die Kader-Verwaltung (siehe unten).

### Änderungsanträge genehmigen

Mitglieder können Änderungen an ihrem Profil beantragen (z. B. neue Adresse, korrigierte Kontaktdaten). Du siehst offene Anträge in der Mitgliederdetailansicht:

1. Mitglied in der Liste anklicken
2. Bereich „Änderungsanträge" einsehen
3. Antrag **annehmen** (Daten werden übernommen) oder **ablehnen**

---

## Beitrittsanträge

Interessenten können über die öffentliche Seite einen **Beitrittsantrag** stellen. Unter **„Beitrittsanträge"** (im Admin-Bereich) siehst du alle offenen Anträge.

### Antrag bearbeiten

1. Antrag in der Liste anklicken
2. Daten des Interessenten prüfen
3. **„Genehmigen"** → der Antrag wird als genehmigt markiert; der Vorstand legt das Mitglied anschließend an und versendet die Einladung
4. **„Ablehnen"** → mit optionalem Ablehnungsgrund

---

## Einladungen versenden

Neue Mitglieder erhalten Zugang über eine Einladungs-E-Mail.

1. Im Admin-Bereich **„Einladung versenden"** aufrufen
2. E-Mail-Adresse, Team und Rolle (`spieler` oder `trainer`) eintragen
3. Einladung absenden — der Empfänger erhält einen Link, über den er sein Passwort setzt

Offene (noch nicht angenommene) Einladungen kannst du in der Einladungsliste einsehen und bei Bedarf löschen.

---

## Dienst-Slots verwalten

### Slot manuell anlegen

1. Unter **„Dienste"** → **„Neuen Slot anlegen"**
2. Diensttyp auswählen (z. B. Hallendienst, Kassendienst)
3. Datum, Uhrzeit, Bezeichnung und Anzahl der Plätze eintragen
4. Optional: Spiel verknüpfen
5. Speichern — der Slot erscheint sofort in der Dienstbörse

### Slot bearbeiten oder löschen

Klicke in der Slot-Liste auf den jeweiligen Eintrag. Du kannst alle Felder nachträglich ändern oder den Slot löschen (löscht auch alle Anmeldungen).

### Dienst als erfüllt markieren

Nach einem geleisteten Dienst:

1. Slot öffnen → Belegungsliste
2. Neben dem Mitglied **„Als erfüllt markieren"** klicken
3. Das Dienst-Konto des Mitglieds wird automatisch aktualisiert

### Geldersatz buchen

Falls ein Mitglied einen Dienst mit einer Geldleistung ablöst:

1. Slot öffnen → Belegungsliste
2. **„Geldersatz"** klicken und den Betrag eintragen
3. Der Status wechselt auf „Geldersatz geleistet"

---

## Spielplan verwalten

Unter **„Kalender"** kannst du den Spielplan deines Teams pflegen.

### Spiel anlegen

1. **„Neues Spiel"** klicken
2. Gegner, Datum, Uhrzeit, Heim/Auswärts und Team eintragen
3. Optional: Dienst-Template auswählen — dann werden die passenden Dienst-Slots automatisch generiert

### Spiel bearbeiten oder löschen

Klicke auf ein Spiel in der Kalenderansicht. Du kannst alle Felder ändern oder das Spiel löschen.

### Dienst-Slots aus Template neu generieren

Falls sich Anstoßzeit oder andere Parameter geändert haben:

1. Spiel öffnen
2. **„Dienst-Slots neu generieren"** klicken
3. Bestehende Slots werden durch neue ersetzt (Achtung: bestehende Anmeldungen gehen verloren)

---

## Kader verwalten

Unter **„Kader"** (Verwaltung) pflegst du den Kader deines Teams — immer **für eine bestimmte Saison**. Oben auf der Seite wählst du die Saison aus.

Jeder Kader hat **drei Listen**, die unabhängig voneinander gepflegt werden:

| Liste | Wer steht drin | Wozu |
|---|---|---|
| **Stammkader** | die festen Spielerinnen und Spieler der Mannschaft | Spielbetrieb, Dienstpflicht, Beiträge |
| **Erweiterter Kader** | Gelegenheitsspieler, die regelmäßig aushelfen | mitspielen und mittrainieren, ohne zur Mannschaft zu gehören |
| **Trainer** | Trainerinnen und Trainer dieses Kaders | Verwaltungsrechte für das Team, Benachrichtigungen |

### Kader initialisieren

Am Saisonstart einmalig:

1. **„+ Mannschaft"** legt den Kader für die gewählte Saison an (Altersklasse, Geschlecht, ggf. Jahrgang)
2. Mitglieder über **„Mitglied suchen…"** dem Stammkader zuweisen (die Checkbox „Jahrgang filtern" schränkt die Vorschläge ein)
3. Alternativ: **„Aus vorheriger Saison kopieren"**
4. **„Auto-Assign"** schlägt Zuordnungen anhand der Altersklassenregeln vor

> **Achtung beim Saisonwechsel:** „Aus vorheriger Saison kopieren" übernimmt **nur den Stammkader**. Erweiterter Kader und Trainer werden **nicht** mitkopiert und müssen für die neue Saison neu gesetzt werden. Solange das nicht passiert ist, sieht der erweiterte Kader keine Termine der neuen Saison und bekommt keine Meldungen mehr.

### Erweiterten Kader pflegen

Im Bereich **„Erweiterter Kader"** der Mannschafts-Karte:

- **Hinzufügen:** Feld **„Mitglied für erweiterten Kader suchen…"**, Namen eintippen, Eintrag anklicken
- **Entfernen:** das **X** am jeweiligen Eintrag

Vorgeschlagen wird jedes Vereinsmitglied, das nicht ausgetreten ist — das System schränkt **nicht** auf Altersklasse oder Geschlecht ein. Diese Auswahl ist eure fachliche Entscheidung.

#### Was an dieser Liste hängt

Wer im erweiterten Kader steht,

- sieht **alle Spiele und Trainings** der Mannschaft in der App und im Kalender-Abo,
- steht in der **Teilnehmerliste** des Termins und kann zu- und absagen,
- bekommt **alle Terminmeldungen** — neue, verschobene und abgesagte Termine, die Erinnerungen 24 h und 3 h vorher sowie Hinweise, die ihr am Termin hinterlegt (seine **Eltern ebenfalls**),
- ist in den **Team-Chatgruppen** („Spieler", seine Eltern in „Eltern"),
- erscheint in der **Anwesenheitsstatistik** in einem eigenen Block, getrennt vom Stammkader,
- sieht die **Videos** der Mannschaft und wird über neue informiert.

#### Was nicht daran hängt

- **Dienste.** Dienstbörse, Dienst-Erinnerungen und das Dienst-Konto sprechen weiterhin nur den Stammkader an. Die Dienstpflicht folgt der festen Kader-Zugehörigkeit — wer aushilft, schuldet dem Verein keine Dienststunden und wird auch nicht dazu aufgefordert.
- **Beiträge.** Der Beitragslauf richtet sich nach dem Mitgliedsdatensatz, nicht nach dieser Liste.

#### Wen ihr eintragen solltet — und wen nicht

Die Liste ist ein **Verteiler mit Wirkung**, keine Merkliste:

- **Rein gehört,** wer regelmäßig aushilft. Sonst erfährt er von einem Spiel nur, wenn ihr ihn persönlich anschreibt.
- **Raus gehört,** wer nicht mehr aushilft. Jede Karteileiche bekommt sonst zu jedem Termin eurer Mannschaft eine Meldung, inklusive beider Erinnerungen am Spieltag.
- **Mehrfachnennung ist erlaubt und normal:** dasselbe Mitglied darf im erweiterten Kader mehrerer Mannschaften stehen. Es bekommt dann die Meldungen von allen — das ist ein Grund mehr, die Listen aktuell zu halten.

### Weitere Einstellungen am Kader

- **Spiele pro Saison** („Spiele"-Feld) — Grundlage für die Dienst-Soll-Berechnung
- **Jahrgang** über den Umschalter „Gemischt" / „Dediziert"

---

## Aufstellung

Für jedes Spiel könnt ihr festhalten, wer nominiert ist. Die Aufstellung ist **eine Liste pro Spiel** — bei Terminen mit mehreren Mannschaften teilen sich diese die Liste.

1. Termin unter **„Termine"** öffnen
2. In der Teilnahme-Tabelle die Spalte **„Aufstellung"** ankreuzen
3. Jedes Häkchen wird sofort gespeichert

Setzen dürfen die Häkchen nur Trainer (und Admin). Drei Dinge, die man nicht sieht und wissen muss:

- **Die Aufstellung ist für alle sichtbar,** die den Termin sehen — Spieler, erweiterter Kader und Eltern. Sie ist keine interne Notiz.
- **Eine Änderung verschickt keine Benachrichtigung.** Wer nachträglich rein- oder herausfällt, erfährt es nur, wenn ihr es ihm sagt.
- **Eine leere Aufstellung heißt „noch nicht festgelegt"**, nicht „niemand ist aufgestellt". Einen Zustand „bewusst leer" gibt es nicht.

### Der Aufstellungsstatus im Kalender

Spieler des **erweiterten Kaders** sehen den Stand direkt im Kalender-Abo — im Titel als kurzer Zusatz hinter der Mannschaft und in der Terminbeschreibung als ganzer Satz:

| Im Titel | Bedeutung |
|---|---|
| `· erw. Kader · aufgestellt` | steht in der Aufstellung |
| `· erw. Kader · nicht aufgestellt` | Aufstellung ist gepflegt, er steht nicht drin |
| `· erw. Kader · Aufstellung offen` | für dieses Spiel wurde noch keine Aufstellung gespeichert |

Der dritte Fall ist der Grund, **die Aufstellung früh zu pflegen**: für einen Gelegenheitsspieler ist sie die verlässlichste Antwort auf die Frage, ob er am Wochenende gebraucht wird. Der Stammkader bekommt bewusst keinen Statuszusatz — dort ist die Teilnahme der Normalfall.

Zwei Eigenheiten des Kalender-Abos:

- Kalender-Apps holen den Feed nur etwa **stündlich** ab, teils seltener. Ändert ihr die Aufstellung, dauert es entsprechend, bis sie dort ankommt. Die App ist die aktuelle Quelle.
- Der Kader-Zusatz heißt seit der Umstellung kürzer `· erw. Kader` statt `- erweiterter Kader`. Bereits synchronisierte Termine ändern deshalb beim nächsten Abgleich ihre Bezeichnung — kein Fehler, kein Duplikat.

---

## Benachrichtigungen — wer bekommt was

Für **Termine** (Spiele, Trainings, generische Events) gilt eine einzige Empfängerregel. Sie umfasst:

- den **Stammkader** der betroffenen Mannschaft und dessen **Eltern**,
- den **erweiterten Kader** und dessen **Eltern**,
- die **Trainer** des Kaders.

Das gilt gleichermaßen für neu angelegte, geänderte und abgesagte Termine, für die Erinnerungen **24 Stunden und 3 Stunden** vorher und für **Hinweise**, die ihr am Termin hinterlegt.

**Ihr bekommt eure eigenen Meldungen mit.** Wer einen Termin anlegt, ändert oder absagt, erhält die Meldung ebenfalls — als Bestätigung, dass sie rausgegangen ist, und als Beleg, welchen Wortlaut die Mannschaft liest.

**Nicht dieser Regel folgen:**

- **Dienste** — Ausschreibung, 48-h-Erinnerung und „Dienst entfällt" gehen an den Stammkader und dessen Eltern (Dienstpflicht, siehe oben).
- **Mitfahrgelegenheiten** — dort werden die Eltern und Trainer angesprochen, und der Steller einer Suche bekommt seine eigene Meldung *nicht*.
- **Chat und Mitteilungen** — eigener Kanal mit eigenem Ungelesen-Zähler.

### Absagen ohne Benachrichtigung

Beim Löschen eines Termins könnt ihr einen **Grund** angeben; er steht dann in der Meldung. Die Option **„ohne Benachrichtigung"** (stumm löschen) ist dem Vorstand vorbehalten — als Trainer wird sie ignoriert, die Meldung geht in jedem Fall raus. Das Live-Update in offenen Sitzungen läuft immer, auch bei stummer Löschung.

### Wenn es zu viel wird

Jeder Nutzer kann unter **Profil → Sonstiges → Benachrichtigungen** einzelne Kategorien (Spiele, Trainings, Dienste, Chat …) abschalten. Die Meldung erscheint dann trotzdem im Nachrichten-Verlauf auf dem Dashboard — abgeschaltet wird der Zustellweg, nicht die Information.

---

## Spieler dauerhaft von einer Trainingsserie abmelden

Manche Spieler können an einer wiederkehrenden Trainingsserie **dauerhaft** oder für einen längeren Zeitraum nicht teilnehmen — z. B. weil sie fest in der A-Jugend mittrainieren, Berufsschule haben oder langfristig verletzt sind. Damit sie nicht als „keine Rückmeldung" erscheinen und die Anwesenheitsstatistik verzerren, kannst du sie **serien-gebunden abmelden**.

Die Abmeldung gilt **nur für die betroffene Serie deines Teams** — nicht für andere Kader desselben Spielers. Nur Trainer des eigenen Teams (sowie sportliche Leitung / Admin) können sie pflegen; der Spieler selbst kann nichts ändern.

### Über die Serien-Bearbeitung

1. Unter **„Termine"** → Tab **„Serien"** die betroffene Serie aufklappen
2. Im Abschnitt **„Dauerhaft abgemeldete Spieler"** auf **„Spieler abmelden"** klicken
3. Spieler auswählen, optional **Von/Bis**-Zeitraum (leer = dauerhaft) und einen **Grund** (z. B. „spielt A-Jugend") eintragen
4. Speichern — der Spieler ist ab sofort für die betroffenen Termine abgemeldet

Zum Rückgängigmachen in derselben Liste **„wieder anmelden"** klicken.

### Direkt aus dem Termin-Detail

Im Detail eines Serien-Termins kannst du neben jedem Spieler direkt **„abmelden"** (ab heute) bzw. bei bereits abgemeldeten Spielern **„wieder anmelden"** wählen.

### Auswirkungen

- **Anwesenheitsstatistik:** Betroffene Termine zählen für den Spieler in **keiner** Spalte (weder anwesend, noch fehlend, noch entschuldigt) — sie fallen komplett aus seinem Nenner. In der Detailliste erscheinen sie als **„abgemeldet"**. (Das unterscheidet die Abmeldung von einer normalen Abwesenheit, die als *entschuldigt* zählt und im Nenner bleibt.)
- **Rückmeldung/Anwesenheit:** Ein abgemeldeter Spieler kann für betroffene Termine **keine** Zu-/Absage geben, und seine Anwesenheit wird nicht erfasst. In den Terminlisten bleibt er sichtbar, mit dem Hinweis **„dauerhaft abgemeldet"**.

---

## Häufige Fragen

**Ich sehe den Admin-Bereich nicht.**
Prüfe, ob dein Account die Vereinsfunktion `trainer` zugewiesen bekommen hat. Falls nicht, wende dich an den Admin.

**Ein Mitglied hat sich für einen Dienst angemeldet, ist aber nicht erschienen.**
Markiere den Dienst nicht als erfüllt. Du kannst die Anmeldung nicht selbst löschen — wende dich an den Admin, falls der Slot bereinigt werden muss.

**Ich möchte einen Dienst-Slot für ein einzelnes Spiel anlegen, ohne das ganze Template zu nutzen.**
Lege den Slot manuell an (siehe „Dienst-Slots verwalten") und verknüpfe ihn optional mit dem Spiel.

**Das Dienst-Konto eines Mitglieds zeigt falsche Werte.**
Prüfe die Belegungsliste aller Slots und ob die Dienste korrekt als erfüllt markiert wurden. Bei Datenbankfehlern wende dich an den Admin.

**Ein Spieler des erweiterten Kaders sagt, er habe von einem Spiel nichts gewusst.**
Prüfe zuerst, ob er im erweiterten Kader **der aktuellen Saison** steht — beim Kopieren aus der Vorsaison wird diese Liste nicht übernommen. Steht er drin, prüfe seine Benachrichtigungs-Einstellungen (Profil → Sonstiges); im Nachrichten-Verlauf auf seinem Dashboard steht die Meldung auch dann, wenn Push abgeschaltet ist.

**Ich habe die Aufstellung geändert — merkt das jemand?**
Nein. Eine geänderte Aufstellung verschickt keine Benachrichtigung. Im Kalender-Abo wird der Stand beim nächsten Abgleich sichtbar (etwa stündlich), in der App sofort. Bei kurzfristigen Nachnominierungen also zusätzlich persönlich Bescheid geben.

**Ich möchte ein Mitglied vollständig anlegen oder löschen.**
Das ist eine Vorstand-Funktion. Als Trainer kannst du nur Einladungen versenden und Beitrittsanträge bearbeiten. Sprich den Vorstand an.
