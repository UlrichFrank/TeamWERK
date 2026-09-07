## Context

Siehe `proposal.md — Why` für die Motivation und die Tabelle der drei divergierenden
Empfängermengen. Technisch relevant ist der Ist-Zustand:

- Drei Kopien derselben Funktion: `games.Handler.teamMembersAndParents([]int)`,
  `trainings.Handler.teamMembersAndParents(int)` und
  `Scheduler.teamMembersAndParents(int)` (letztere zusätzlich verpackt in
  `teamMembersAndParentsMulti` für die Termin-Hinweise). Alle drei bauen dieselbe
  `UNION`-Query aus `player_memberships` + `family_links` gegen die **aktive** Saison; nur
  die Trainings-Kopie hat irgendwann einen dritten `UNION`-Zweig auf
  `kader_extended_members` bekommen — ohne den zugehörigen Eltern-Zweig.
- `player_memberships` ist eine **View über `kader_members`**. Der erweiterte Kader ist
  darin per Definition nicht enthalten (`openspec/specs/erweiterter-kader`), und das soll
  auch so bleiben: an der View hängen Dienst-Soll und Beitragslogik.
- `notify.Send` ist bereits der einzige Fan-out-Punkt (Event-Log, Push-Präferenzen,
  E-Mail) und wird von allen betroffenen Stellen aufgerufen. Das Push-Fan-out-Gate
  (`internal/arch/pushfanout_test.go`) hält das mechanisch.
- Die Sichtbarkeit (`auth.UserCanSeeGame`) kennt den erweiterten Kader längst. Wer heute
  keine Meldung bekommt, sieht den Termin trotzdem — die Lücke ist ausschließlich die
  aktive Zustellung.

## Goals / Non-Goals

**Goals:**

- Eine Auflösung, ein Fundort. Wer die Empfängermenge ändern will, ändert sie für alle
  Terminmeldungen gleichzeitig.
- Die Regel ist mechanisch gegen Rückfall geschützt, nicht nur dokumentiert.
- Kein Verhalten außer der Empfängermenge ändert sich: gleiche Texte, gleiche Kategorien,
  gleiche `url`, gleiches `silent`, gleiche Idempotenz.

**Non-Goals:**

- **Kein Opt-out pro Nutzer.** Es gibt keinen Schalter „nur Meldungen für meinen
  Stammkader" und keinen „ohne meine eigenen Änderungen" (siehe Decision 7).
- **Keine Rückwirkung.** Für Termine, die vor dem Deploy angelegt oder geändert wurden,
  wird nichts nachgemeldet.
- **Keine neue Präferenz.** Es gibt keinen Schalter „nur Meldungen für meinen Stammkader".
  Wer im erweiterten Kader steht, bekommt die Meldungen dieser Mannschaft — die
  Steuerung ist die Kader-Pflege durch den Trainer, nicht eine weitere Einstellung.

## Decisions

### 1. Der Helfer liegt in `internal/notify`, nicht in einem neuen Package

`notify` ist Foundation, wird von allen drei betroffenen Paketen bereits importiert und
ist der Ort, an dem die Frage „wer bekommt das?" ohnehin beantwortet wird (Event-Log,
Präferenzfilter). Der Aufruf steht damit direkt neben dem Versand:

```go
notify.Send(h.db, h.cfg, notify.TeamAudience(h.db, teamIDs...), "games", …)
```

*Alternative:* ein neues Foundation-Package `internal/audience`. Konzeptionell sauberer
getrennt, aber es müsste in `internal/arch/arch_test.go` klassifiziert werden und stünde
für Aufrufer an einer Stelle, die man erst suchen muss. Der Gewinn wäre benannt, nicht
strukturell — `notify` importiert dadurch keine neue Schicht, nur eine weitere Query.

*Alternative:* `internal/policy`. Verworfen: `policy` beantwortet „darf jemand?", nicht
„wen betrifft es?". Beides zu vermischen wäre die Voraussetzung dafür, dass später jemand
die Empfängermenge versehentlich als Berechtigungsprüfung benutzt (in der Spec deshalb
ausdrücklich abgegrenzt).

### 2. Eine Query mit `team_id IN (…)`, kein Aufruf pro Mannschaft

`TeamAudience(db, teamIDs ...int) []int` nimmt beliebig viele Mannschaften und löst sie in
**einem** Statement auf: vier `UNION`-Zweige (Stammkader-Mitglied, dessen Elternteil,
erweitertes Mitglied, dessen Elternteil), `DISTINCT` über das Ergebnis. Die Deduplizierung
macht SQLite, nicht eine Map in Go — der bisherige `teamMembersAndParentsMulti`-Wrapper
entfällt ersatzlos.

Leere Team-Liste → leeres Ergebnis ohne Query (der heutige Guard in `games`).

### 3. Query-Fehler werden geloggt, nicht verschluckt

Die drei Bestandskopien geben bei einem Query-Fehler `nil` zurück — die Meldung entfällt
dann lautlos, und niemand erfährt davon. Der neue Helfer behält die Signatur ohne `error`
(die Aufrufstellen sollen nicht neun Fehlerpfade bekommen, und eine fehlgeschlagene
Benachrichtigung darf nie die Mutation kippen), loggt den Fehler aber über `slog.Error`.
Ein stiller Ausfall wird damit zu einem sichtbaren.

*Alternative:* `([]int, error)` und Behandlung an jeder Aufrufstelle. Verworfen: alle neun
Stellen würden den Fehler ohnehin nur loggen und weitermachen — dieselbe Entscheidung,
neunmal abgeschrieben.

### 4. Bezugsgröße bleibt die **aktive** Saison

Alle drei Bestandskopien lösen gegen `seasons.is_active = 1` auf; die Sichtbarkeit
(`UserCanSeeGame`) dagegen gegen die Saison **des Termins**. Der Helfer übernimmt die
aktive Saison unverändert.

Das ist bewusst kein Nebenbei-Fix: Termine außerhalb der aktiven Saison erzeugen heute
schon keine Meldung, ein Wechsel auf die Termin-Saison wäre also eine zweite
Verhaltensänderung in derselben Auslieferung — mit einer eigenen Frage im Gepäck (soll ein
Termin der nächsten Saison Meldungen an den dann gültigen Kader schicken, bevor die Saison
aktiv ist?). Steht als bekannter Rest in „Risks".

### 4b. Trainer gehören in die Menge, ohne Eltern-Zweig

`kader_trainers` kommt als fünfter `UNION`-Zweig dazu. Dass die Trainer bisher fehlten, ist
derselbe Fehler wie beim erweiterten Kader, nur vollständiger: sie standen in **keiner** der
drei Kopien. Ein Co-Trainer erfuhr die Verlegung des eigenen Spiels nur, wenn er zufällig
auch Spieler war.

**Ohne Eltern-Zweig:** ein Trainer steht in seiner Funktion in der Menge, nicht als Kind.
Ein `family_link` auf einen Trainer — bei einem jungen Übungsleiter durchaus real — darf
nicht dazu führen, dass dessen Mutter die Mannschaftsmeldungen bekommt. Der Test
`TestTeamAudience_TrainerOhneElternZweig` hält das fest.

### 5. Ein Arch-Gate gegen die nächste Kopie

Ein Test in `internal/arch` (stdlib, Teil von `make test`, im Stil von
`broadcast_test.go` / `pushfanout_test.go`) scannt `internal/` nach Funktionen, die die
Empfängermenge selbst auflösen, und lässt sie außerhalb von `internal/notify`
fehlschlagen. Erkennungsmerkmal ist die Kombination aus **Rückgabetyp `[]int`** und einem
Funktionsrumpf, der `family_links` mit einer Kader-Tabelle (`player_memberships`,
`kader_members`, `kader_extended_members`) verbindet. Der Rückgabetyp ist der
entscheidende Teil des Filters: Sichtbarkeits- und Berechtigungsprüfungen fragen dieselben
Tabellen ab, liefern aber `bool`, Zeilen oder Team-IDs — ohne diese Einschränkung wäre die
Allowlist voller Einträge, die mit Benachrichtigungen nichts zu tun haben, und das Gate
sagte nichts mehr aus.

Ausnahmen stehen mit Begründung in einer Allowlist; ein verwaister Eintrag lässt den Test
ebenfalls fehlschlagen. Der Lauf hat neben den Dienst-Pfaden zwei weitere Fundstellen
freigelegt, die vorher niemand als Empfängerauflösung geführt hat und die bewusst
eigenständig bleiben: `carpooling.kaderRecipients` (adressiert nur Eltern + Trainer und
schließt den Steller aus — „wer kann fahren?") und `videos.pushRecipients` (Hochladender +
aktive Spieler + Trainer, gebunden an die Saison des Videos). Dass der erweiterte Kader in
der Video-Meldung fehlt, ist eine eigene Frage der Video-Sichtbarkeit, kein Rest dieses
Changes.

Ohne dieses Gate ist die Ausgangslage in zwei Jahren wieder da: der Anlass dieses Changes
ist keine falsche Entscheidung, sondern eine Kopie, die niemand mitgezogen hat.

*Alternative:* nur ein Kommentar an den Aufrufstellen. Verworfen — genau das war der
bisherige Zustand.

### 6. Duty-Meldungen bleiben auf `player_memberships`

`internal/duties` und die Dienst-Erinnerung im Scheduler lösen ihre Empfänger über
`player_memberships` + `family_links` auf, aber mit anderer Bedeutung: sie fragen „wer ist
dienstpflichtig?", nicht „wen betrifft der Termin?". Diese Stellen werden **nicht**
umgestellt und stehen als begründete Ausnahmen in der Allowlist aus Decision 5. Die
Abgrenzung steht als Requirement in der Spec, damit sie nicht als vergessener Rest gelesen
wird.

### 7. Der Auslöser bleibt in der Menge

Wer einen Termin anlegt, verschiebt oder absagt, bekommt die eigene Meldung mit. Das ist
die Kontrolle über den eigenen Versand: die Bestätigung, dass die Meldung rausgegangen
ist, und der Beleg, welchen Wortlaut die Mannschaft gelesen hat — inklusive der
generierten Teile (Zeitpunkt, Aktor-Satz, Grund), die der Auslöser sonst nirgends zu
sehen bekommt.

Erst mit Decision 4b wird das spürbar: bis dahin war der anlegende Trainer meist gar nicht
in der Menge, die Frage stellte sich nicht.

*Alternative:* den Auslöser herausfiltern, wie es `carpooling.kaderRecipients` mit dem
Steller einer Suche tut. Verworfen — dort geht es um eine Anfrage an andere („wer kann
fahren?"), hier um eine Tatsachenmeldung über einen gemeinsamen Termin. Der Preis ist eine
Push wenige Sekunden nach dem eigenen Speichern; das ist der übliche Einwand gegen
Selbst-Benachrichtigung und in „Risks" festgehalten. Die Gegenmaßnahme wäre ein
`excludeUserID`-Parameter an einer einzigen Stelle — bewusst nicht vorab gebaut, weil sie
die Zusage still aushebeln würde, sobald sie existiert.

## Risks / Trade-offs

- **Mehr Meldungen für Mehrfach-Zugehörige.** Wer im erweiterten Kader von drei
  Mannschaften steht, bekommt ab sofort deren sämtliche Spiel- und Trainingsmeldungen
  inklusive Erinnerungen. → Mitigation: keine technische. Der erweiterte Kader ist eine
  vom Trainer gepflegte Liste; die Ankündigung an die Trainer muss die Bitte enthalten,
  Karteileichen zu entfernen. Ein Filter pro Nutzer wäre die Gegenmaßnahme, falls sich das
  als Belastung erweist — bewusst nicht vorab gebaut.
- **Eltern erhalten Meldungen, die sie bisher nicht kannten.** Für Eltern von Kindern im
  erweiterten Kader ist der Kanal neu. → Mitigation: die bestehenden
  Benachrichtigungs-Präferenzen greifen unverändert pro Kategorie; das Event-Log zeigt die
  Meldung auch bei abgeschaltetem Push.
- **Volumen.** Push-Versand und `user_events`-Zeilen wachsen um die Größe des erweiterten
  Kaders (Größenordnung: einstellige Prozent der Mitgliederzahl). → Für den VPS
  unkritisch; die Retention des Event-Logs bleibt unverändert.
- **Bekannter Rest: aktive Saison statt Termin-Saison** (Decision 4). Ein Termin in einer
  nicht-aktiven Saison ist sichtbar, erzeugt aber weiterhin keine Meldung. → Mitigation:
  keine; unverändertes Bestandsverhalten, eigener Change falls es auffällt.
- **Der Auslöser bekommt eine Push auf das eigene Gerät**, Sekunden nach dem Speichern
  (Decision 7). Für Trainer, die einen Spielplan am Stück pflegen, ist das der spürbarste
  Teil dieser Änderung. → Mitigation: die Benachrichtigungs-Präferenzen greifen pro
  Kategorie, das Event-Log bleibt in jedem Fall vollständig. Wenn es stört, ist die
  Gegenmaßnahme ein `excludeUserID` an einer Stelle — dann aber als bewusste Rücknahme
  der Zusage, nicht nebenbei.
- **Trainer mehrerer Mannschaften bekommen jetzt alle deren Termin- und
  Erinnerungsmeldungen.** → Dieselbe Mitigation wie beim erweiterten Kader: Präferenzen
  pro Kategorie, ansonsten bewusst akzeptiert.

## Migration Plan

Keine Migration, kein Schema, kein Datenrückbau. Wirksam mit dem Deploy; Rollback ist ein
Redeploy der Vorversion (die Empfängermenge wird bei jedem Versand frisch aufgelöst, es
bleibt kein Zustand zurück).

Vor der Auslieferung: Ankündigung an die Trainer, dass ab sofort auch der erweiterte Kader
und dessen Eltern Termin- und Erinnerungsmeldungen erhalten — verbunden mit der Bitte, die
erweiterten Kader durchzusehen.
