# event-log Specification

## Purpose
TBD - created by archiving change event-log. Update Purpose after archive.
## Requirements
### Requirement: Vollständiger Event-Log je Nutzer

Das System SHALL jede über `notify.Send` versandte Meldung als Zeile in `user_events` festhalten — eine Zeile je Empfänger, geschrieben aus der **ungefilterten** Empfängerliste, bevor Push- oder Email-Präferenzen ausgewertet werden.

Der Log SHALL für jede Kategorie außer `chat` eine **Obermenge** der tatsächlich zugestellten Pushes sein. Ein Nutzer, der Push für eine Kategorie abgeschaltet hat, keine Push-Subscription besitzt oder dessen Push am Transport scheitert, SHALL die Meldung dennoch im Log vorfinden.

Eine Zeile SHALL `category`, `title`, `body`, `url`, `created_at` und `seen_at` tragen. `category` SHALL auf die acht Nicht-Chat-Kategorien beschränkt sein (`games`, `trainings`, `duties`, `duty_reminders`, `carpooling`, `membership`, `operativ`, `sonstiges`).

Der Log SHALL **keine** Fremdschlüssel auf Domänen-Objekte tragen. Die Meldung wird zum Sendezeitpunkt eingefroren, weil die referenzierten Objekte (gelöschte Termine, entfernte Dienst-Slots) danach nicht mehr existieren — genau die Fälle, in denen Nachlesen am wichtigsten ist. `url` ist ein Sprungziel und DARF ins Leere zeigen.

#### Scenario: Meldung erreicht alle Empfänger im Log

- **WHEN** ein Trainer einen Termin ändert, der 12 Team-Mitglieder betrifft
- **THEN** entstehen 12 `user_events`-Zeilen, je eine pro Mitglied
- **THEN** tragen alle dieselbe `category`, `title`, `body` und `url`

#### Scenario: Push abgeschaltet, Log trotzdem vollständig

- **WHEN** ein Empfänger `push_enabled=0` für die Kategorie hat
- **THEN** erhält er keine Push Notification
- **THEN** existiert dennoch seine `user_events`-Zeile

#### Scenario: Keine Push-Subscription, Log trotzdem vollständig

- **WHEN** ein Empfänger keine Zeile in `push_subscriptions` hat (Berechtigung nie erteilt, iOS nicht als PWA installiert)
- **THEN** existiert dennoch seine `user_events`-Zeile

#### Scenario: Chat schreibt nicht in den Log

- **WHEN** eine Chat-Nachricht oder eine Mitteilung versandt wird
- **THEN** entsteht keine `user_events`-Zeile — Chat läuft über `push.SendToUserWithBadge` und hat eigene Ungelesen-Zähler

### Requirement: Empfängermenge wird beim Schreiben eingefroren

Das System SHALL die Sichtbarkeit einer Log-Zeile ausschließlich über `user_events.user_id` bestimmen. Beim Lesen SHALL **nicht** gegen den aktuellen Kader, die aktuelle Vereinsfunktion oder eine sonstige aktuelle Zugehörigkeit nachgefiltert werden.

Der Log protokolliert, was gesendet wurde — nicht, was heute gelten würde. Eine Nachfilterung beim Lesen würde in beide Richtungen falsche Aussagen erzeugen: sie verstecke Meldungen vor Nutzern, die sie nachweislich erhalten haben, und zeige neu hinzugekommenen Nutzern rückwirkend Meldungen, die nie an sie gingen.

#### Scenario: Nutzer verlässt das Team

- **WHEN** ein Nutzer nach Erhalt einer Team-Meldung aus dem Kader entfernt wird
- **THEN** bleibt seine Log-Zeile bis zur Retention sichtbar

#### Scenario: Nutzer kommt neu ins Team

- **WHEN** ein Nutzer neu in einen Kader aufgenommen wird
- **THEN** sieht er keine Log-Zeilen zu Meldungen, die vor seiner Aufnahme versandt wurden

#### Scenario: Fremde Log-Zeilen bleiben unsichtbar

- **WHEN** ein Nutzer das Dashboard abruft
- **THEN** enthält die Antwort ausschließlich Zeilen mit seiner eigenen `user_id`

### Requirement: Auslieferung stempelt `seen_at`

`GET /api/dashboard` SHALL die jüngsten Log-Zeilen des aufrufenden Nutzers zurückgeben (absteigend nach `created_at`, gedeckelt auf 30) und im selben Vorgang `seen_at` auf **genau den zurückgegebenen Zeilen** setzen, sofern dort `seen_at IS NULL` ist.

Die Stempelung SHALL sich auf die IDs der ausgelieferten Zeilen beziehen, nicht auf alle Zeilen des Nutzers — sonst verfielen Zeilen jenseits des Caps, ohne je angezeigt worden zu sein.

Ein bereits gesetztes `seen_at` SHALL bei weiteren Abrufen unverändert bleiben; die Retention-Uhr DARF NICHT durch erneutes Laden verschoben werden.

#### Scenario: Erster Abruf stempelt

- **WHEN** ein Nutzer mit drei ungesehenen Zeilen das Dashboard abruft
- **THEN** enthält die Antwort alle drei
- **THEN** tragen alle drei anschließend ein `seen_at`

#### Scenario: Zeilen jenseits des Caps bleiben ungesehen

- **WHEN** ein Nutzer 35 ungesehene Zeilen hat und das Dashboard abruft
- **THEN** enthält die Antwort die 30 jüngsten
- **THEN** tragen genau diese 30 ein `seen_at`; die restlichen 5 behalten `seen_at IS NULL`

#### Scenario: Zweiter Abruf verschiebt die Uhr nicht

- **WHEN** ein Nutzer das Dashboard zweimal im Abstand von einem Tag abruft
- **THEN** trägt eine beim ersten Abruf gestempelte Zeile weiterhin den Zeitpunkt des ersten Abrufs

#### Scenario: Ohne Authentifizierung

- **WHEN** `GET /api/dashboard` ohne gültigen Access-Token aufgerufen wird
- **THEN** antwortet das System mit HTTP 401

### Requirement: Retention drei Tage nach Ansicht

Das System SHALL Log-Zeilen löschen, deren `seen_at` länger als drei Tage zurückliegt. Zeilen mit `seen_at IS NULL` SHALL erhalten bleiben — wer nicht in der App war, DARF keine Meldung verlieren.

Als Betriebs-Sicherung SHALL das System ungesehene Zeilen dennoch nach 90 Tagen löschen. Diese Kappe verhindert unbegrenztes Wachstum durch Accounts, die nie wieder eingeloggt werden (ausgetretene Mitglieder, verwaiste Kinder-Accounts), und liegt weit oberhalb jeder plausiblen Abwesenheit.

Die Bereinigung SHALL im bestehenden Scheduler laufen und ohne Idempotenzschutz auskommen — ein `DELETE` ist wiederholbar.

#### Scenario: Gesehene Zeile verfällt

- **WHEN** eine Zeile ein `seen_at` von vor vier Tagen trägt
- **THEN** löscht der Retention-Lauf sie

#### Scenario: Frisch gesehene Zeile bleibt

- **WHEN** eine Zeile ein `seen_at` von vor zwei Tagen trägt
- **THEN** bleibt sie erhalten

#### Scenario: Ungesehene Zeile überlebt den Urlaub

- **WHEN** eine Zeile 30 Tage alt ist und `seen_at IS NULL` trägt
- **THEN** bleibt sie erhalten

#### Scenario: Sicherheitskappe greift

- **WHEN** eine Zeile 91 Tage alt ist und `seen_at IS NULL` trägt
- **THEN** löscht der Retention-Lauf sie

### Requirement: Dashboard-Section „Ereignisse"

Das Dashboard SHALL eine Section „Ereignisse" als kollabierbares `Accordion` in derselben Card-Optik wie die bestehenden Sections anzeigen (`bg-brand-surface-card`, `border-t-4 border-brand-yellow`). Sie listet die Log-Einträge des Nutzers, neueste zuerst, mit Titel, Text, relativer Zeitangabe und — sofern `url` gesetzt ist — Sprung ins Ziel.

Die Section SHALL **nicht** „Benachrichtigungen" heißen. Sie steht neben der bestehenden Section „Nachrichten" (Chat, ungelesen-basiert) und muss von ihr unterscheidbar bleiben: „Nachrichten" = jemand spricht mich an, „Ereignisse" = die Terminlage bewegt sich.

Der Event-Log SHALL **nicht** in den App-Icon-Badge (`navigator.setAppBadge`) einzahlen; dieser bleibt Chat-only.

Die Section SHALL sich über die bestehende `useLiveUpdates`-Verdrahtung der Dashboard-Seite aktualisieren. Es SHALL kein neuer Hub-Event-Name und keine Hub-Verdrahtung in `notify.Send` entstehen.

#### Scenario: Einträge vorhanden

- **WHEN** ein Nutzer mit Log-Einträgen das Dashboard öffnet
- **THEN** zeigt die Section „Ereignisse" die Einträge, neueste zuerst
- **THEN** führt ein Klick auf einen Eintrag mit gesetzter `url` an das dort genannte Ziel

#### Scenario: Eintrag ohne Sprungziel

- **WHEN** ein Eintrag eine leere `url` trägt (z. B. eine Termin-Absage, deren Termin nicht mehr existiert)
- **THEN** wird er dargestellt, ist aber nicht anklickbar

#### Scenario: Leerzustand

- **WHEN** ein Nutzer keine Log-Einträge hat
- **THEN** zeigt die Section einen dezenten Leerzustand

#### Scenario: Live-Aktualisierung

- **WHEN** während geöffnetem Dashboard ein `games`-, `trainings`-, `duties`- oder `mitfahrgelegenheiten`-Event über SSE eintrifft
- **THEN** aktualisiert die Section ihre Liste ohne manuelles Neuladen

### Requirement: Absagegründe leben im Log statt nur im Zustellkanal

Das System SHALL den bei einer Absage angegebenen Freitext-Grund als Teil des Meldungstexts in `user_events.body` festhalten. Er SHALL weiterhin **nicht** an einem Domänen-Objekt persistiert werden (es gibt kein `games.status='cancelled'`); seine Lebensdauer SHALL ausschließlich durch die Retention des Event-Logs bestimmt sein.

Dies kehrt die bisherige Festlegung um, dass der Grund nirgends persistiert wird. Die Begründung dafür war, dass Löschen wirklich löschen soll — mit der benannten Folge, dass wer die Push wegwischt, den Grund nirgends wiederfindet. Der Event-Log löst genau diese Folge auf; die drei Tage Retention treten an die Stelle von „gar nicht gespeichert".

Betroffen sind vier Meldungstypen: Termin abgesagt, Training abgesagt, Trainingsserie beendet, Dienst entfällt.

#### Scenario: Grund ist nachlesbar

- **WHEN** ein Trainer ein Spiel mit dem Grund „Halle gesperrt" löscht
- **THEN** enthält der `body` der entstehenden `user_events`-Zeilen den Grund
- **THEN** findet ein Empfänger ihn im Dashboard, auch wenn er die Push weggewischt hat

#### Scenario: Kein Domänen-Objekt trägt den Grund

- **WHEN** ein Termin mit Grund gelöscht wurde
- **THEN** enthält keine Tabelle außer `user_events` den Grund-Text

#### Scenario: Stumme Löschung schreibt keinen Log

- **WHEN** ein Berechtigter mit der Capability `suppress_event_notification` `silent: true` setzt
- **THEN** entsteht keine Benachrichtigung und keine `user_events`-Zeile

