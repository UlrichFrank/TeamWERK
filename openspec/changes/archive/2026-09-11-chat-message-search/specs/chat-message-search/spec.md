## Purpose

Ermöglicht Nutzern, per Volltextsuche über alle ihre Chat-Konversationen und Mitteilungen hinweg nach einer Nachricht zu suchen und direkt zur Fundstelle zu springen, statt manuell durch einzelne Konversationen zu scrollen.

## ADDED Requirements

### Requirement: Übergreifende Volltextsuche
Das System MUST eine Suche bereitstellen, die den Nachrichtentext (`messages.body`) aller Konversationen, in denen der anfragende Nutzer aktives Mitglied ist (`conversation_members.left_at IS NULL`), sowie den Text (`broadcasts.body`) aller für ihn sichtbaren Mitteilungen (`broadcast_reads`-Zeile vorhanden, `hidden_at IS NULL`) nach einem Suchbegriff durchsucht. Gelöschte Nachrichten (`messages.deleted_at IS NOT NULL`) MUST von der Suche ausgeschlossen werden, unabhängig vom gespeicherten `body`-Inhalt. Die Ergebnisliste ist nach Zeitpunkt absteigend (neueste zuerst) sortiert und paginiert (`limit`/`offset`, analog zur bestehenden Mitglieder-/Nutzer-Paginierung), Antwortform `{ items: [...], total: N }`.

#### Scenario: Treffer in eigener Konversation
- **WHEN** ein Nutzer nach einem Begriff sucht, der im Text einer Nachricht in einer seiner aktiven Konversationen vorkommt
- **THEN** liefert die Suche diese Nachricht als Treffer mit Konversationsname, Absender, Zeitpunkt und einem den Treffer enthaltenden Textausschnitt

#### Scenario: Treffer in Mitteilung
- **WHEN** ein Nutzer nach einem Begriff sucht, der im Text einer für ihn sichtbaren Mitteilung vorkommt
- **THEN** liefert die Suche diese Mitteilung als Treffer mit Absender, Zeitpunkt und Textausschnitt

#### Scenario: Kein Zugriff auf fremde Konversation
- **WHEN** ein Suchbegriff nur in einer Nachricht einer Konversation vorkommt, in der der anfragende Nutzer kein Mitglied ist (auch nicht als ausgetretenes Mitglied)
- **THEN** erscheint diese Nachricht nicht in den Suchergebnissen

#### Scenario: Gelöschte Nachricht bleibt unsichtbar
- **WHEN** ein Suchbegriff im (nicht genullten) `body` einer bereits gelöschten Nachricht (`deleted_at IS NOT NULL`) vorkommt
- **THEN** erscheint diese Nachricht nicht in den Suchergebnissen

#### Scenario: Leere Suchanfrage
- **WHEN** die Suche ohne oder mit leerem `q`-Parameter aufgerufen wird
- **THEN** antwortet das System mit HTTP 400 statt eine ungefilterte Volltabelle zurückzugeben

### Requirement: Sprung zur Fundstelle im Chat
Das System MUST es erlauben, eine Konversation zentriert um eine bestimmte Nachricht zu laden (Nachrichten davor und danach in einem Fenster um die Ziel-Nachricht), damit ein Suchtreffer direkt sichtbar gemacht werden kann, ohne dass der Nutzer manuell zur passenden Stelle scrollen muss. Dieser Zugriff unterliegt denselben Sichtbarkeitsregeln wie das reguläre Laden von Nachrichten (nur Mitglieder der Konversation, gelöschte Nachrichten bleiben maskiert).

#### Scenario: Konversation wird um Treffer zentriert geöffnet
- **WHEN** ein Nutzer in den Suchergebnissen auf einen Chat-Treffer klickt
- **THEN** öffnet sich die zugehörige Konversation mit einem um die Treffer-Nachricht zentrierten Nachrichtenfenster, und die Treffer-Nachricht ist ohne weiteres Scrollen sichtbar

#### Scenario: Treffer in nicht mehr existierender Nachricht
- **WHEN** ein Nutzer auf einen Suchtreffer klickt, dessen Nachricht zwischenzeitlich gelöscht wurde
- **THEN** zeigt das System eine verständliche Rückmeldung statt eines Fehlerabsturzes

### Requirement: Sprung zur Fundstelle bei Mitteilungen
Ein Klick auf einen Mitteilungs-Treffer MUST die vollständige Mitteilung öffnen (wie beim regulären Öffnen aus der Mitteilungs-Liste).

#### Scenario: Mitteilung wird aus Suchergebnis geöffnet
- **WHEN** ein Nutzer in den Suchergebnissen auf einen Mitteilungs-Treffer klickt
- **THEN** öffnet sich die vollständige Mitteilung im Mitteilungen-Tab
