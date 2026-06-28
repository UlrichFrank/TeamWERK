# Exploration: System-Gruppenchats pro aktivem Kader

Stand der Erkundung — **keine Entscheidung, keine Implementierung**. Dokumentiert die
Form des Problems, bevor ein OpenSpec-Proposal angelegt wird.

## Anforderung (Originalton)

> Bei /chat soll es vom System angelegte, nicht löschbare Gruppenchats geben.
> Passend zu meiner KaderTeam-Zuordnung (als Eltern, Kind, Trainer) aber auch in
> meiner Vereinsfunktion Vorstand, Sportlicher Leiter oder admin möchte ich Gruppen
> für die Teams (Spieler, Trainer, Eltern) jeweils getrennt, alle zusammen oder
> Spieler-Trainer, Trainer-Eltern und Spieler-Eltern sehen. Sie existieren jeweils
> für aktive Kader und die Mitgliedschaft wird darüber definiert.

## Lesart der Anforderung

Pro **aktivem Kader** entstehen automatisch bis zu 7 Räume — die nicht-leeren
Teilmengen aus `{Spieler, Trainer, Eltern}`:

```
   audiences = {S, T, E}
   nicht-leere Teilmengen = 2³ − 1 = 7

   ┌──────────────┬──────────────┬─────────────────────┐
   │ {S}          │ {T}          │ {E}                 │
   │ {S,T}        │ {T,E}        │ {S,E}               │
   │ {S,T,E}                                            │
   └────────────────────────────────────────────────────┘
```

Ich (= der eingeloggte User) sehe einen Chat genau dann, wenn ich zu mindestens
einer seiner Audiences im Kader gehöre. Vorstand / Sportliche Leitung / Admin
sehen alle Räume aller aktiven Kader.

## Ist-Zustand im Code

```
        ┌────────────────────────────────────────────────┐
        │                  KADER (pro Saison)            │
        │   age_class · gender · team_number             │
        └──────────────────┬─────────────────────────────┘
              ┌────────────┼─────────────┬────────────┐
              ▼            ▼             ▼            ▼
        kader_members   kader_trainers   kader_extended_members
        (Spieler)       (Trainer)        (erweitert)
              │
              │   family_links
              ▼   (parent_user_id ↔ member_id)
           Eltern (auf User-Ebene)

        conversations(type='direct'|'group', name, created_by)
        conversation_members(user_id, joined_at, left_at)
        messages / message_reads / message_reactions
```

Konsequenzen aus dem Schema:

- Chat-Mitgliedschaft ist **per `user_id`** modelliert.
  → Spieler ohne User-Account können nicht Chat-Member sein (Edge: muss
  abgefangen werden — entweder „nur Spieler mit User" oder Stummschalter).
- Eltern stehen schon als User in `family_links`.
- Trainer = `kader_trainers.member_id` → `members.user_id`.
- Das bestehende Feld `conversation_members.left_at` reicht für „war drin,
  sieht den Verlauf, kann nicht mehr posten".
- Heute kann jede Gruppe gelöscht werden (`DELETE /api/chat/conversations/{id}`).
  System-Chats brauchen ein **`is_system=1`-Flag**, das Delete/Rename/Leave
  ablehnt (403/409).

## Mitgliedschaft ist derivable

```
   System-Chat für Kader K mit Audience-Set A ⊆ {S,T,E}:

     S ∈ A → users hinter kader_members(K)               [Spieler]
     T ∈ A → users hinter kader_trainers(K)              [Trainer]
     E ∈ A → users in family_links(member ∈ kader_members(K)) [Eltern]

   Ein Eltern-User mit Kindern in 3 Kadern ist in 3 Eltern-Chats Member.
```

Daraus folgt:
- **Keine manuelle Member-Pflege.** Jede Mutation an
  `kader_members` / `kader_trainers` / `family_links` triggert ein
  Membership-Refresh des betroffenen Kaders.
- SSE-Broadcast (`hub.Broadcast("chat:conv-updated:<id>")`) ist bereits etabliert.
- `is_system=1`-Chats müssen den Member-CRUD-Endpunkten verboten sein —
  Membership ist Funktion des Datenstands, nicht editierbar.

## Drei offene Designentscheidungen

### 1. Maßstab — wirklich 7 Räume pro Kader?

Bei 6 aktiven Kadern wären das **42 System-Chats** in der Sidebar von
Vorstand / Sportlicher Leitung / Admin.

```
   Variante A  (Maximalist)
     Alle 7 Audience-Kombis fest pro Kader.
     + maximaler Aufschlag, klare Mental-Map („immer die gleichen 7")
     − Listen-Overload für Vorstand, viele leere Räume

   Variante B  (Minimalist)
     Nur die häufigen Muster:  {S}, {T}, {E}, {S,T,E}    (= 4)
     + WhatsApp-Realität in Vereinen
     − User-Wunsch fordert ausdrücklich auch {S,T}/{T,E}/{S,E}

   Variante C  (Konfigurierbar)
     Vorstand/Trainer entscheidet pro Kader, welche der 7 Räume
     tatsächlich existieren.
     + Aufschlag nur dort, wo er gebraucht wird
     − mehr UI / Settings-Aufwand, neue Frage: „warum gibt es Kader X
       den E-Chat nicht?"
```

→ **Frage an User offen.**

### 2. Wer darf schreiben?

```
   Naiv:   Alle Member dürfen posten   (entspricht dem heutigen Gruppenmodell)
   Anders: Eltern-Chat = nur Trainer „Aushänge" + Eltern lesen
           (verschiebt das Modell Richtung Broadcasts)
```

Aktuell hat TeamWERK zwei Modi: Chat (alle Member schreiben) und Broadcasts
(nur ausgewählte Sender). System-Kader-Chats als **echte Chats** liegt näher
am Wunsch („Gruppen" statt „Mitteilungen").

→ **Empfehlung Default:** alle Audience-Member dürfen posten. Sonderregeln
sind ein Folge-Feature.

### 3. Saisonwechsel & Verlauf

```
   Saison 25/26 aktiv              Saison 26/27 wird aktiv
   ─────────────────               ──────────────────────
   Kader A → 7 System-Chats        Kader A' (neue Kader-Zeile)
   Mitglieder lesen + schreiben       → neue 7 System-Chats
                                      → alte werden read-only
                                         (alle Member behalten left_at,
                                          sehen den Verlauf weiter)
```

Begründung: Saison-Zugehörigkeit hängt am Kader, nicht am Chat. Ein neuer
Kader-Eintrag = neuer Kontext = neuer Raum. Verlauf der alten Saison bleibt
sichtbar (datenschutzkonform, weil nur die damaligen Member ihn sehen).

Offene Frage: passiert das **automatisch beim Saisonwechsel** oder
**lazy beim ersten Aktivieren des neuen Kaders**?

→ Lazy klingt einfacher (kein Saison-Wechsel-Hook nötig).

## Spannungen / Risiken

| Bereich | Risiko | Mitigation |
|---|---|---|
| Delete-Pfad | bestehendes `DELETE /conversations/{id}` löscht alles | `is_system=1` → 403 |
| Member-CRUD | `add/remove/leave/rename` würde Derivation kaputtmachen | für `is_system=1` ablehnen |
| Spieler ohne User | können nicht Member werden | dokumentieren, oder „User anlegen" als Voraussetzung |
| Listen-Overload Vorstand | 6 Kader × 7 = 42 Räume | Sidebar-Gruppierung „nach Kader" + Filter |
| Schreib-Cascade | jede Kader-Mutation refresht 7 Chat-Memberships | batchen, nur Diff schreiben |
| Naming-Konflikt | viele Chats heißen „mJSB1 · Spieler" o. ä. | deterministisch generiert, nicht editierbar |
| Backfill | beim Rollout 1× Erst-Erzeugung für alle aktiven Kader | Migration / Job, idempotent |

## Was vor einem Proposal noch geklärt sein muss

1. **Maßstab** — Variante A / B / C? (siehe oben)
2. **Posting-Rechte** — Default „alle Audience-Member"?
3. **Saisonwechsel** — lazy beim neuen Kader, nicht beim Saison-Flip?
4. **Sidebar-UX für Vorstand** — flache Liste oder Gruppierung nach Kader?
5. **Spieler ohne User-Account** — ignorieren (nicht in der Liste) oder UI-Hinweis?
6. **Eltern-Definition** — `family_links` ist die einzige Quelle (bestätigen)?
7. **Erweiterter Kader** (`kader_extended_members`) — gehört er auch in die
   Spieler-Audience oder nicht? (User hat nur „Spieler" gesagt — interpretiert
   als `kader_members`, nicht `extended`.)

## Skizze: API-Oberfläche (nur als Diskussionsanker)

```
GET  /api/chat/conversations          (unverändert, liefert System- + User-Chats)
                                       Response-Feld pro Conv: isSystem: boolean

POST /api/chat/conversations          → 400 wenn Klient is_system=1 mitschickt
DELETE /api/chat/conversations/{id}   → 403 wenn is_system=1
PUT  /api/chat/conversations/{id}     → 403 wenn is_system=1 (rename)
DELETE /api/chat/conversations/{id}/members/{x}  → 403 wenn is_system=1

Interner Trigger (nach jeder Kader-/Trainer-/Family-Mutation):
  chat.SyncSystemConversations(db, kaderID)
  → erzeugt fehlende System-Convs für aktive Kader, gleicht Member ab
    (setzt left_at für entfernte, joined_at für neue)
  → ein Broadcast „chat:conv-updated:<id>" pro betroffener Conv
```

## Schema-Skizze (nur als Diskussionsanker)

```sql
ALTER TABLE conversations ADD COLUMN is_system     BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE conversations ADD COLUMN kader_id      INTEGER REFERENCES kader(id);
ALTER TABLE conversations ADD COLUMN audience_mask INTEGER;  -- Bitmask: 1=S, 2=T, 4=E

-- Eindeutigkeit: pro (kader_id, audience_mask) genau ein System-Chat.
CREATE UNIQUE INDEX idx_conv_kader_audience
  ON conversations(kader_id, audience_mask)
  WHERE is_system = 1;
```

Bitmask statt drei Bool-Spalten, weil sie die Audience-Logik (Match per
`(audience_mask & user_audience_bits) != 0`) in einer einzigen SQL-Bedingung
ausdrückbar macht.

---

**Nächster Schritt:** Antworten auf die 7 offenen Fragen → dann
`openspec change propose chat-system-kader-gruppen`.
