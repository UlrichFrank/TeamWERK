# Design

## 1. Vereinigung statt Rangfolge

Die Zweige in `ListTeamsForUser` waren als **Rangfolge** gebaut: erst prüfen, was der Nutzer
„ist", dann die dazu passende Menge liefern. Das trägt, solange jeder Nutzer genau eine Rolle
hat. Sobald jemand zwei hat — Trainer *und* Elternteil —, entscheidet die Reihenfolge der
`else if` darüber, welche seiner Zugehörigkeiten verschwinden. Sie ist damit eine stille
Priorisierung, die niemand so entschieden hat.

Die Frage, die der Filter stellt, ist aber keine nach der Rolle, sondern nach der Menge: *an
welchen Mannschaften hängt dieser Nutzer?* Darauf gibt es genau eine Antwort, und sie ist
eine Vereinigung. `user_accessible_teams` bildet sie bereits vollständig ab — inklusive des
Trainer-Wegs, weshalb kein zusätzliches `UNION` nötig ist.

## 2. Warum `?scope=duties` trotzdem bleibt

Der Dienst-Filter ist die eine Stelle, an der die Antwort *nicht* die Vereinigung ist: der
erweiterte Kader schuldet keine Dienststunden (`duties.eligibleDutyRecipients`, mit
Begründung in der `audienceAllowlist`), eine Filter-Option dafür bliebe immer leer.

Der Unterschied ist damit auf eine Zeile geschrumpft und steht sichtbar nebeneinander statt
in zwei getrennten Zweigen:

- `?scope=duties` → Stammkader (Selbst + Kinder) ∪ Trainer-Teams
- sonst → `user_accessible_teams`

Der `∪ Trainer-Teams`-Teil braucht **keinen** eigenen Trainer-Zweig: für einen Nutzer ohne
Eintrag in `kader_trainers` ist die Teilmenge leer, die Bedingung gilt deshalb unverändert
für alle. Genau diese Beobachtung fehlte beim vorigen Fix — dort wurde sie als zusätzlicher
Zweig modelliert, was den scope-losen Zweig danebenstehen ließ.

## 3. Die Statistik-Scopes bleiben eine echte Ausnahme

`?scope=attendance-stats` und `?scope=diary-stats` liefern weiterhin **nur** Trainer-Teams,
und das ist keine Inkonsistenz: die beiden Statistikseiten navigieren beim Laden auf das
erste zurückgegebene Team und prüfen es dann gegen `attendance.canSeeTeamStats` bzw.
`trainingdiary.canSeeTeamDiary`. Eine weitere Menge hieße dort: Selektor zeigt ein Team,
dessen Abruf mit 403 endet. Der Filter darf hier enger sein als die Sichtbarkeit, weil er
eine *Navigation* speist und keine Filterung einer bereits geladenen Liste.

Der Unterschied ist also nicht „mal so, mal so", sondern: **filternde** Scopes folgen der
Sichtbarkeit, **navigierende** Scopes folgen dem Zugriffsrecht der Zielseite.

## 4. Keine Ausweitung der Sichtbarkeit

Die neue Menge ist eine Teilmenge dessen, was `GET /api/teams/my` demselben Nutzer schon
heute ausgibt (Nav, Dashboard, „Mein Team"). Die Inhalts-Endpunkte bleiben unverändert:
`/games/my`, `/training-sessions` und `/duty-board` entscheiden weiterhin selbst, welche
Termine der Nutzer sieht. Der Filter kann also nichts sichtbar machen, was die Liste nicht
ohnehin liefert — er kann es künftig nur noch ein- und ausblenden.

## 5. Der Test, der den Fehler konserviert hat

`TestListTeamsForUser_ScopeDutiesTrainerAlsoParent` enthielt die Zeile

```go
// Without scope: unchanged existing behavior, only the trainer's own team.
if len(plainTeams) != 1 { … }
```

Der Kommentar hat den ungeprüften Teil des Verhaltens zur Zusage erklärt. „Unverändert" ist
keine Aussage über Richtigkeit — und eine Assertion, die einen nicht untersuchten Zustand
festschreibt, macht aus einem Bug eine Spezifikation. Dieselbe Person hat denselben Fehler
zwei Tage später für die nächste Oberfläche gemeldet.

Der neue Test `TestListTeamsForUser_TrainerElternteilErweiterterKader` bildet deshalb die
vollständige gemeldete Konstellation ab — Trainer, Kind im Stammkader, Kind im erweiterten
Kader — und prüft beide Aufrufe gegeneinander: die erweiterte Mannschaft gehört in den
scope-losen Filter und nicht in den Dienst-Filter. Ein Test, der nur den Stammkader kennt,
kann diesen Unterschied nicht zeigen.
