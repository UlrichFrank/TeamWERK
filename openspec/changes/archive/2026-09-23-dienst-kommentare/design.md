## Context

`internal/duties/handler.go` implementiert bereits Claim/Unclaim
(`Claim`/`Unclaim`, inkl. Proxy-Child-Eigentümerprüfung über `family_links`)
und das Duty-Board (`GET /api/duty-board`, schlanker Response nach dem Muster
„nur Namen inline, Details on-demand" — siehe `PublicAssignee`/`PersonChip`).
`web/src/components/DutySlotList.tsx` rendert die Slot-Zeilen virtualisiert
(`useWindowedList`, feste `estimatedRowHeight: 52`) und hat seit Commit
`3e6fe417` (PR #215) keinen Löschpfad mehr — Löschen läuft ausschließlich über
den Bearbeiten-Dialog im Kalender-Modal (`SpieltagDetailModal`). Das bestehende
Mobile-`ActionMenu` (`sm:hidden`) enthält aktuell Eintragen/Austragen,
Bearbeiten und — bewusst redundant zum Inline-Icon — „Anleitung"; Desktop hat
kein Menü, nur Einzelbuttons. Siehe proposal.md für die Motivation.

## Goals / Non-Goals

**Goals:**
- Ein Kommentarfeld, das strukturell an die Zuteilung (nicht an die Person
  oder den Slot) gebunden ist, damit „genau ein Kommentar" und „räumt sich
  beim Austragen automatisch auf" ohne Zusatzlogik in mehreren Handlern gelten.
- Board bleibt schlank (nur Zählung), Volltext nur on-demand.
- ⋮-Menü auch auf Desktop, damit weitere Zeilen-Aktionen (jetzt: Kommentieren)
  nicht als N-ter Einzelbutton neben Eintragen/Austragen wachsen.

**Non-Goals:**
- Keine Moderation/Admin-Löschrecht für fremde Kommentare (siehe proposal.md).
- Kein Markdown/Rich-Text — reiner Text, React-Escaping reicht.
- Kein Opt-in pro Dienst-Typ — das Feature ist generisch, das Icon erscheint
  ohnehin nur bei tatsächlichem Inhalt.
- Keine Bearbeitungs-Historie (`edited_at`) — Upsert überschreibt kommentarlos.

## Decisions

### Kommentar hängt an `duty_assignments.id`, nicht an `(duty_slot_id, user_id)`

`duty_assignments` trägt bereits `UNIQUE(duty_slot_id, user_id)`; ein
alternatives Design mit `duty_assignment_comments.duty_slot_id` +
`duty_assignment_comments.user_id` + eigenem `UNIQUE` wäre möglich, verlangt
aber in **jedem** Pfad, der eine Zuteilung entfernt, einen manuellen
zusätzlichen `DELETE FROM duty_assignment_comments WHERE …` — und davon gibt
es mehrere unabhängig gewachsene: `Unclaim` (Austragen), `restoreAssignments`/
`snapshotDeletedSlots` (Massen-Regen), `DeleteGame`, Slot-Löschung über das
Kalender-Modal. Ein Foreign Key `assignment_id INTEGER NOT NULL UNIQUE
REFERENCES duty_assignments(id) ON DELETE CASCADE` verschiebt diese Garantie
in die Datenbank: „Kommentar existiert höchstens so lange wie seine
Zuteilung" gilt dann für jeden aktuellen und jeden künftigen Löschpfad ohne
Codeänderung an anderer Stelle. Die `UNIQUE`-Constraint auf `assignment_id`
liefert „genau ein Kommentar" als Nebenprodukt derselben Spalte.

### Upsert statt Verlauf

`PUT` überschreibt den bestehenden Kommentar direkt. Alternative wäre ein
`edited_at`-Feld analog `messages.edited_at` im Chat — für ein kurzes
Koordinations-Freitextfeld (was bringe ich mit) ist eine Änderungshistorie
kein erkennbarer Nutzen, nur zusätzliches Schema/UI. Löschen bleibt ein
eigener `DELETE`-Endpoint (nicht „leerer PUT"), damit ein leerer Body eindeutig
ein Validierungsfehler ist und nicht zwei Bedeutungen (löschen vs. ungültig)
teilen muss.

### Lesen: eigener On-Demand-Endpoint statt Board-Inline

`GET /api/duty-slots/{id}/comments` liefert den Volltext erst beim Öffnen des
Kommentar-Modals — konsistent mit dem bestehenden Muster in dieser Datei
(`PersonChip` lädt Kontakt on-demand, das Board selbst liefert laut
Kommentar im Code bewusst „nur Namen inline"). Der Board-Response bekommt
stattdessen nur `comment_count`, damit ein Slot mit vielen langen Kommentaren
nicht jeden Board-Request verteuert, den die meisten Nutzer nie öffnen.

### Anzeige als Modal, nicht als Inline-Akkordeon

Die Slot-Liste ist virtualisiert mit fester `estimatedRowHeight: 52`. Ein
Ausklappen der Kommentare unter der Zeile würde diese Schätzung pro Zeile
variabel machen und die Windowing-Logik (`useWindowedList`) verkomplizieren.
Ein Modal (wie die bereits vorhandenen `claimDialog`/`noInstructionOpen`-
Dialoge in derselben Datei) umgeht das vollständig und braucht keinen
Scroll-Anker — Klick auf das Kommentar-Icon öffnet direkt lesend, Klick auf
„Kommentieren" im ⋮-Menü öffnet dasselbe Modal mit vorbefüllter Textarea für
den eigenen Kommentar.

### ⋮-Menü auch auf Desktop

Mit Kommentieren als vierter potenzieller Zeilen-Aktion (nach
Eintragen/Austragen, Bearbeiten, Anleitung) würde die Desktop-Button-Reihe
enger, je mehr Features dazukommen. Die Konsolidierung überträgt das
bestehende Mobile-Muster (`ActionMenu`) auf beide Breakpoints; Eintragen/
Austragen bleiben als eigene, sofort sichtbare Buttons (häufigste Aktionen,
nicht hinter einem Menü versteckt).

### Eigentümer-Prüfung wiederverwendet die Claim-Logik

Die Proxy-Child-Prüfung in `Handler.Claim` (`family_links` JOIN auf
`can_login = 0`) wird für die Kommentar-Endpunkte dupliziert bzw. in einen
gemeinsamen Helfer gezogen (Implementierungsdetail für tasks.md) — dieselbe
Regel „wer darf für wen handeln" soll nicht zweimal unterschiedlich
formuliert werden.

## Risks / Trade-offs

- **[Risk]** Ein Kommentar zu einer Zuteilung, die durch Massen-Regen kurz
  gelöscht und mit identischen Merkmalen neu angelegt wird (`restoreAssignments`
  matcht über `(duty_type_id, event_time, team_id)`), verliert seinen
  Kommentar, obwohl die Person „gefühlt" dieselbe Zuteilung behält — die neue
  `duty_assignments`-Zeile hat eine neue `id`.
  → **Mitigation**: Akzeptiert. `restoreAssignments` restauriert nur
  `status`/`cash_amount`/`fulfilled_at`, keine abgeleiteten Inhalte; ein
  Kommentar-Restore wäre ein Sonderfall nur für dieses eine Feld und stünde
  im Widerspruch zur bestehenden Restore-Semantik (die auch andere pro-
  Zuteilung-Daten nicht kennt). Für den Hauptfall (Kuchen-Rotation am
  eigenen Heimspiel) ist Regen selten genug, dass ein gelegentliches erneutes
  Kommentieren zumutbar ist.
- **[Risk]** `comment_count` im Board-Response erfordert einen zusätzlichen
  Join/Subquery pro Board-Request.
  → **Mitigation**: Aggregation über einen Index auf
  `duty_assignment_comments(assignment_id)` (durch `UNIQUE` bereits
  vorhanden) plus den bestehenden Index auf `duty_assignments.duty_slot_id`;
  bei der Datenmenge eines Vereins (Hunderte, nicht Zehntausende offene
  Slots) nicht spürbar.

## Migration Plan

Additive Migration `062_duty_assignment_comments.up.sql` (neue Tabelle, keine
Änderung an Bestandstabellen) — folgt der Projekt-Konvention „Migrationen sind
additiv, kein Rollback nötig" (`docs/agent/10-deployment.md`). `down.sql`
droppt die Tabelle. Kein Datenbestand zu migrieren (neues Feature, keine
Altdaten).
