## Context

Siehe `proposal.md — Why`. Technisch stehen im Videos-Paket drei Auflösungen derselben
Frage („gehört dieser Nutzer zu diesem Team?"), die sich heute schon einig sind und es
bleiben sollen:

- `access.go: userBelongsToTeam` — Detailabruf, Stream, `CanViewVideo`
- `crud.go: visibilityFilter` — SQL-Fragment für die Liste (`v.team_id IN (…)`)
- `worker.go: pushRecipients` — Empfänger der Ready-Meldung

Der Kommentar an `visibilityFilter` sagt es ausdrücklich: „spiegelt exakt
`userBelongsToTeam`". Die drei sind bewusst identisch formuliert, aber technisch getrennt:
eine liefert `bool`, eine ein SQL-Fragment für eine fremde Query, eine eine Nutzerliste.

## Goals / Non-Goals

**Goals:**

- Der erweiterte Kader und dessen Eltern sehen die Videos ihrer Mannschaft und werden über
  neue benachrichtigt.
- Die drei Auflösungen bleiben deckungsgleich.

**Non-Goals:**

- **Keine Änderung an Schreibrechten.** Hochladen, Metadaten, Löschen bleiben bei
  Trainer/Vorstand/Admin.
- **Keine Zusammenlegung der drei Auflösungen.** Verlockend, aber sie haben drei
  verschiedene Formen (bool, SQL-Fragment, `[]int`); ein gemeinsamer Helfer müsste alle
  drei bedienen und wäre umständlicher als die Duplizierung. Siehe Decision 3.
- **Keine Ausweitung auf `notify.TeamAudience`.** Die Video-Menge ist eine andere: sie
  hängt an der Saison **des Videos**, filtert auf den Mitgliedsstatus und trägt den
  Hochladenden. Sie bleibt in der `audienceAllowlist` des Arch-Gates.

## Decisions

### 1. Statusfilter der neuen Zweige: `<> 'ausgetreten'`, nicht `= 'aktiv'`

Die bestehenden Zweige filtern `members.status = 'aktiv'`. Für den erweiterten Kader wäre
das ein stiller Selbstwiderspruch: **Förderkinder tragen `status = 'foerderkind'`**
(Migration `034`) und sind die typische Besetzung dieser Liste. Ein `= 'aktiv'` hätte die
Änderung für den Hauptfall wirkungslos gemacht — und zwar unsichtbar, weil die Query
weiterhin Zeilen liefert, nur eben nicht die gemeinten.

Die neuen Zweige filtern deshalb `status <> 'ausgetreten'`. Das ist die Aussage, die
gemeint war: „wer noch im Verein ist". Ein Test hält den Förderkind-Fall fest, damit ein
späteres Vereinheitlichen der Filter nicht unbemerkt zurückdreht.

*Alternative:* auch die Stammkader-Zweige auf `<> 'ausgetreten'` ziehen. Verworfen — das
änderte das Verhalten für `passiv`, `anwaerter` und `extern` mit, ohne dass jemand danach
gefragt hat. Die Uneinheitlichkeit ist die kleinere Überraschung und steht im Kommentar.

### 2. `userBelongsToTeam` deckt den Stream mit ab

Der Stream läuft über einen signierten `?st=`-Token, der aus `Play` stammt; `Play` prüft
`CanViewVideo`, das auf `userBelongsToTeam` steht. Die Erweiterung an dieser einen Stelle
öffnet damit Liste, Detail **und** Stream — es gibt keinen vierten Pfad, an dem die alte
Enge zurückbliebe.

### 3. Drei Stellen bleiben drei Stellen — abgesichert durch einen Test

Die drei Auflösungen werden parallel geändert und nicht zusammengelegt (Non-Goal). Der
Schutz dagegen, dass sie später auseinanderlaufen, ist die in `video-transcode`
festgeschriebene Zusage „niemand wird über ein Video benachrichtigt, das er nicht öffnen
kann" plus ein Test, der beide Seiten am selben Fixture prüft.

*Alternative:* ein Arch-Gate wie bei `notify.TeamAudience`. Verworfen: dort ging es um vier
Domänen und eine über Jahre gewachsene Divergenz; hier stehen drei Funktionen in einem
Paket, die man beim Ändern ohnehin nebeneinander sieht.

## Risks / Trade-offs

- **Mehr Zuschauer für Spielaufnahmen.** Videos zeigen Minderjährige; der Kreis wächst um
  die Gelegenheitsspieler und deren Eltern. → Der Kreis bleibt team- und saisongebunden und
  wächst genau um die Personen, die auf den Aufnahmen zu sehen sind. Die Einwilligung
  (`members.foto_veroeffentlichung`) ist davon unberührt — sie regelt die Veröffentlichung
  nach außen, nicht die vereinsinterne Sichtbarkeit.
- **Uneinheitliche Statusfilter in einer Query** (Decision 1). → Kommentar an beiden
  Stellen; ein Test hält den Grund fest.
- **Mehr Push-Volumen** um die Größe des erweiterten Kaders. → Kategorie `sonstiges`, per
  Präferenz abschaltbar; unverändert.

## Migration Plan

Keine Migration. Wirksam mit dem Deploy — Bestandsvideos werden für den erweiterten Kader
schlagartig sichtbar, ohne Nachlauf. Rollback ist ein Redeploy der Vorversion.
