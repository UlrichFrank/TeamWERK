## Context

Das Chat-Modul (`internal/chat/`) speichert Konversationen mit konkreten User-Mitgliedern in `conversation_members`. Broadcasts (`broadcasts`-Tabelle) haben ein eigenes Targeting-Modell (`team`/`role`/`all`), das auf `user_accessible_teams` aufsetzt. Die View `user_accessible_teams` (in Migration `001`) vereinigt vier Quellen: `kader_members`, `family_links` zu `kader_members`, `kader_trainers`, `kader_extended_members` und `family_links` zu `kader_extended_members`.

## Goals / Non-Goals

**Goals:**
- Bulk-Auswahl typischer Team×Rolle-Gruppen im „Neues Gespräch"-Modal.
- Keine neue Konversations-Klasse, kein neues Mitgliedschafts-Modell.
- Konsistente Sichtbarkeits-Regel mit dem bestehenden Berechtigungs-Modell.

**Non-Goals:**
- Lebende Gruppen (Mitglieder, die später ins Team kommen, werden NICHT automatisch in laufende Konversationen aufgenommen).
- Dedup auf Konversations-Ebene („gleiche Mitgliederliste vorhanden → reuse"). Group-Gespräche werden weiterhin immer neu erstellt.
- Tags als Metadaten am Conversation-Record (woher kam die Auswahl). Die Auflösung ist rein client-seitig und nicht persistiert.

## Decisions

### D1: Auflösung im Client, Endpoint liefert User-IDs

Der Client ruft beim Klick auf einen Tag `GET /api/chat/team-groups/{teamId}/{kind}/members` auf, mergt die zurückgelieferten User in den bestehenden `selected[]`-State und schickt am Ende eine flache `memberIds[]`-Liste an den unveränderten `POST /api/chat/conversations`. Die bestehende `createGroup`-Berechtigungsprüfung (`canContactUser` pro ID) bleibt die einzige Sicherheitsgrenze — der Endpoint wendet dieselbe Sichtbarkeitsregel an, damit ein bösartiger Client nichts dazugewinnt.

**Alternative:** Tags im Conversation-Payload (`groupTags: [...]`) und Server-seitige Auflösung. Verworfen, weil es ein eigenes Mitgliedschafts-Modell andeuten würde, das wir explizit nicht wollen (Snapshot-Semantik).

### D2: Saison-Filter im Picker

`GET /api/chat/team-groups` (Liste sichtbarer Tags) filtert auf `seasons.is_active = 1`. Begründung: sonst sieht ein Trainer, der seit fünf Jahren coacht, jedes Jahrgangs-Team aus jeder Saison. Konsequenz: Alt-Saison-Trainer/Spieler/Eltern können weiterhin via Personen-Suche kontaktiert werden, nur eben nicht über den Tag.

`GET /api/chat/team-groups/{teamId}/{kind}/members` filtert ebenfalls auf die aktive Saison (Kader müssen `seasons.is_active = 1` haben), damit beide Seiten konsistent bleiben.

### D3: Sichtbarkeitsregel

```
Tags sichtbar/abrufbar  =  Caller hat role='admin'
                        OR Caller hat ClubFunction 'vorstand'
                        OR Caller hat ClubFunction 'sportliche_leitung'
                        OR Caller ist in user_accessible_teams für team_id
                           (in der aktiven Saison)
```

Damit gilt dieselbe Regel für Auflisten (`/team-groups`) und Auflösen (`/team-groups/{teamId}/{kind}/members`). Verstößt ein Caller gegen die Regel beim Auflösen, antwortet der Server mit 403.

### D4: Caller-Filter beim Auflösen

Sowohl `/team-groups/{teamId}/{kind}/members` als auch `/team-groups` filtern den Caller aus der Mitgliederliste bzw. dem Count heraus — er ist bei `createGroup` ohnehin automatisch dabei. Sonst sähe man sich selbst im Picker als Chip.

### D5: Kind-Definition

```
kind='trainer' →  kader_trainers JOIN members
kind='spieler' →  kader_members  JOIN members
              ∪   kader_extended_members JOIN members
kind='eltern'  →  family_links für (kader_members ∪ kader_extended_members),
                  parent_user_id
```

Die Spieler-Gruppe enthält bewusst sowohl Stamm- als auch erweiterten Kader; spiegelnd dazu enthält die Eltern-Gruppe die Eltern beider Gruppen. So bleibt die Definition konsistent mit der `user_accessible_teams`-View.

### D6: Counts in der Picker-Liste

`GET /api/chat/team-groups` liefert pro Tag ein `count`-Feld (Anzahl Mitglieder *ohne* Caller). Bei Count `0` wird der Tag weggelassen — sonst stehen leere Eltern-Gruppen im Picker.

## Risks / Trade-offs

- **„Vergessene" Trainer:** Beim Klick auf „Trainer U16m1" fügt der Client die zum Zeitpunkt aktuell aktiven Trainer hinzu. Kommt nach der Konversations-Erstellung ein neuer Trainer dazu, ist er nicht in der Gruppe. Bewusst akzeptiert (Snapshot-Modell).
- **Inkonsistenz mit Broadcast:** Broadcast `team`-Target nutzt `user_accessible_teams` ohne Saison-Filter. Hier filtern wir auf aktive Saison. Die Inkonsistenz ist akzeptabel — Broadcast adressiert „alle, die jemals dazugehörten", der Picker dagegen „aktuelle Saison".
- **Größenexplosion bei großen Gruppen:** Eine Eltern-Gruppe kann 20+ Personen enthalten. Der Client zeigt 20+ Chips. Akzeptabel; bei Bedarf könnte später ein Aggregat-Tag eingebaut werden.

## Migration Plan

Keine Migration nötig. Reine Read-only Endpoints, kein Schema-Eingriff, kein Datenbestand.
