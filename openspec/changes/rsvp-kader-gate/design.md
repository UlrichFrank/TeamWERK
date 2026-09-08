## Context

Fund aus dem Change `uebungsgruppen` (dort `design.md — Bekannter Rest`). Ausgangslage im
Code (`internal/trainings/handler.go`, `Respond`):

- Der **Fremd-Zweig** (`member_id` gesetzt und ≠ eigenes Mitglied) ist abgesichert: Staff
  (`admin`/`vorstand`/`IsTrainerLike`) oder Elternteil des Ziels. Der Kommentar dort
  dokumentiert einen früheren Fix gegen „broken access control".
- Der **Selbst-Zweig** prüft ausschließlich, ob der Nutzer überhaupt ein Mitglied ist
  (`ownMemberID != 0`). Danach folgen Absence-Lock, Serien-Abmeldung und RSVP-Cutoff —
  alles Prüfungen am *Termin*, keine an der *Zugehörigkeit*.
- `ListSessions` beantwortet die Zugehörigkeitsfrage bereits, als
  `am_i_participant`: drei `EXISTS`-Zweige über `kader_members`,
  `kader_extended_members` und `kader_trainers` gegen `ts.kader_id`.

Gemessen (Wegwerf-Probe gegen den Testserver): Fremd-Zusage → `204`, danach sieht der
Trainer `confirmed_count = 1`. Der Join, der `confirmed_count` bildet, schließt nur
Trainer aus, nicht Nicht-Kadermitglieder.

## Goals / Non-Goals

**Goals**

- Eine `training_responses`-Zeile existiert nur für Beteiligte des Termins.
- Lese- und Schreibpfad beantworten die Zugehörigkeitsfrage mit **einer** Funktion.

**Non-Goals**

- **Bereinigung des Bestands.** Vorhandene Zeilen von Nicht-Kadermitgliedern bleiben
  stehen. Ein Löschlauf über `training_responses` hätte kein fachliches Ziel (die Zeilen
  sind nicht schädlich, nur sinnlos) und träfe im Zweifel Zeilen, die durch einen
  Kaderwechsel *nachträglich* fremd wurden — das ist ein anderer Sachverhalt.
- **Kaderwechsel-Semantik.** Wer den Kader verlässt, behält seine bereits gegebene Antwort.
  Die Prüfung greift beim Schreiben, nicht rückwirkend.
- **Frontend.** Die Oberfläche bietet die Zusage nur auf sichtbaren Terminen an; ein 403
  ist von dort nicht erreichbar. Keine neue Fehlermeldung nötig.

## Entscheidung 1 — Ein Helfer für beide Pfade, nicht zwei Prüfungen

Die Zugehörigkeitsfrage wird an zwei Stellen gebraucht: als `bool` in `Respond` und als
SQL-Ausdruck in `ListSessions` (`am_i_participant`). Die naheliegende Abkürzung wäre, in
`Respond` schnell ein `EXISTS` hinzuschreiben.

**Verworfen.** Genau so entstehen zwei Antworten auf dieselbe Frage, die auseinander
driften: ein Termin, der als „du bist dabei" angezeigt wird, dessen Zusage aber 403 liefert
— oder umgekehrt. Diese Fehlerklasse ist im Repo dokumentiert (Video-Sichtbarkeit an drei
Stellen, `docs/agent/06-gotchas.md`), dort mit der ausdrücklichen Zusage, dass die drei
Formen deckungsgleich bleiben müssen. Hier sind es nur zwei, und sie lassen sich
zusammenlegen.

Gewählt: ein Helfer `isKaderParticipant(ctx, kaderID, memberID) (bool, error)` mit den drei
Zweigen. `Respond` ruft ihn direkt. `ListSessions` behält seinen SQL-Ausdruck, solange die
Umstellung dort einen Query-Umbau erzwingen würde — dann aber mit einem Kommentar an
**beiden** Stellen, der auf die jeweils andere zeigt, und einem Test, der die
Deckungsgleichheit prüft (Szenario „Anzeige und Antwortrecht stimmen überein"). Die
Zusammenlegung ist das Ziel, der Test ist die Absicherung, falls sie nur teilweise gelingt.

## Entscheidung 2 — 403, nicht 404

Der Termin existiert; nur die Person gehört nicht dazu. Das ist derselbe Fall wie beim
Gate gegen den erweiterten Kader in `uebungsgruppen` (dort Entscheidung 3) und dieselbe
Antwort. Ein 404 wäre hier zusätzlich irreführend, weil der Nutzer den Termin in
Einzelfällen durchaus sehen kann.

## Entscheidung 3 — Auch Staff darf keine Termin-Fremden eintragen

Der Fremd-Zweig prüft heute „darf ich für **diese Person** antworten?", nicht „gehört
diese Person zu **diesem Termin**?". Beide Fragen sind nötig: ein Vorstand darf für ein
Mitglied antworten — aber nicht auf einem Termin, mit dem dieses Mitglied nichts zu tun
hat. Die Kaderprüfung gilt deshalb für das **Ziel**-Mitglied, unabhängig davon, wer
schreibt.

Das ist eine echte Verschärfung für Staff. Sie ist gewollt: der praktische Fall
„Trainer trägt für einen Spieler nach" betrifft immer einen Spieler seines Kaders.

## Risiken

**Verhaltensänderung an einer viel genutzten Route.** Die Prüfung lehnt ab, was heute
durchgeht. Das Risiko liegt nicht bei Fremden (die gibt es im Alltag kaum), sondern bei
Konstellationen, die *unabsichtlich* außerhalb des Kaders stehen — etwa ein Spieler, dessen
Kaderzuordnung zur neuen Saison noch fehlt. Er bekäme statt einer Zusage ein 403. Die
Sichtbarkeit ist davon aber schon heute gleich betroffen (`ListSessions` filtert über
dieselben drei Zweige): er sieht den Termin ohnehin nicht. Die Änderung macht einen
bestehenden Zustand konsistent, sie erzeugt ihn nicht.

**Bestandstests, die die Lücke nutzen.** `TestRespond_CreatesRSVP` und
`TestRespond_UpdatesExistingRSVP` legen ihr Mitglied in keinen Kader. Sie werden angepasst,
nicht gestrichen — die Zugehörigkeit ist das, was sie fachlich immer gemeint haben. Beim
Anpassen ist darauf zu achten, dass sie danach noch dasselbe prüfen (Zusage wird
gespeichert / überschreibt statt zu duplizieren) und nicht versehentlich zum Gate-Test
werden.
