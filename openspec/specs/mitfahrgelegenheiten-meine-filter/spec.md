# mitfahrgelegenheiten-meine-filter Specification

## Purpose

Diese Spezifikation beschreibt die Capability `mitfahrgelegenheiten-meine-filter`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Meine-Filter als Pill

Die Seite SHALL einen Pill-Button "Meine" oben im Header anzeigen, im selben visuellen Stil wie die Event-Typ-Pills (Heim, Auswärts, Sonstiges) — gleicher Border, gleiches Padding, gleicher Active-State (gelber Hintergrund mit dunklem Text). Die "Meine"-Pill verwendet ein eigenes Icon (z. B. `UserCheck`), um sie semantisch von den Typ-Pills zu unterscheiden.

Im inaktiven Zustand (Default) werden alle Spiele der eigenen Mannschaft(en) angezeigt. Im aktiven Zustand werden nur Spiele angezeigt, bei denen der eingeloggte Nutzer mindestens einen Eintrag (biete oder suche) hat oder in einer Paarung beteiligt ist (`bieteIsOwn || sucheIsOwn`).

Der Filter ist für alle Rollen sichtbar und aktiv.

#### Scenario: Standard-Ansicht zeigt Team-Spiele

- **WHEN** Nutzer die Seite öffnet
- **THEN** ist die "Meine"-Pill inaktiv und alle Spiele der eigenen Mannschaft(en) werden angezeigt

#### Scenario: Aktivierung der Meine-Pill

- **WHEN** Nutzer auf die "Meine"-Pill klickt und sie aktiv wird
- **THEN** werden nur noch Spiele angezeigt, bei denen `isOwn === true` auf mindestens einem Eintrag steht oder der Nutzer in einer Paarung (`bieteIsOwn || sucheIsOwn`) beteiligt ist

#### Scenario: Meine-Pill ohne eigene Einträge

- **WHEN** Nutzer die "Meine"-Pill aktiviert und keine eigenen Einträge oder Paarungen hat
- **THEN** ist die Liste leer und zeigt eine passende Hinweismeldung

#### Scenario: Meine-Pill kombinierbar mit Typ-Filtern

- **WHEN** Nutzer die "Meine"-Pill aktiviert und gleichzeitig "Heim" deaktiviert
- **THEN** werden nur Spiele mit `eventType ∈ {auswärts, generisch}` angezeigt, bei denen der Nutzer beteiligt ist


### Requirement: Tab-Counts spiegeln den aktiven Filter

Die Tab-Navigation SHALL durch eine einzige chronologische Liste mit Pill-Filtern ersetzt werden. Es gibt keine Tabs mehr, also auch keine Tab-Counts. Die Anzahl der sichtbaren Spiele ergibt sich implizit aus der Listendarstellung.

**Migration**: Keine Anpassung im Client nötig — der Counter war reine UI-Anzeige. Falls eine Mengenanzeige gewünscht ist, kann ein optionaler Header über der Liste die Gesamtzahl der sichtbaren Spiele anzeigen (außerhalb dieses Changes).

#### Scenario: Keine Tab-Navigation mehr vorhanden

- **WHEN** ein Nutzer die Mitfahrgelegenheiten-Seite öffnet
- **THEN** sind keine Tabs (Auswärtsspiele / Heimspiele / Events) mehr sichtbar, sondern Pill-Filter
