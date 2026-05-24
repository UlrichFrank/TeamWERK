## Context

Aktuell gibt es drei separate Seiten für die Dienstdomäne:

- `/dienstboerse` (`DutyBoardPage`): Mitglieder sehen Slots ihrer Teams und können sich ein-/austragen. Filter basiert auf `team_memberships` — einer Tabelle, die nie per Frontend befüllt wird. Folge: alle Nutzer sehen eine leere Liste.
- `/dienste` (`DutySlotsPage`): Admins/Trainer sehen alle Slots ungefiltert, können Zuteilungen auf „Erfüllt" oder „Geldersatz" setzen. Kein Delete im UI, obwohl `DELETE /api/duty-slots/:id` im Backend existiert.
- `/dienstkonten` (`DutyAccountsPage`): Stundenkonto-Übersicht — wird in einem späteren Change (dashboard-home) integriert.

Der Bug in der Filterlogik (`team_memberships` nie befüllt) ist der eigentliche Grund, warum `DutySlotsPage` als ungefilterten Workaround entstand.

Die Kader-Zuweisung (Spieler via `kader_members`, Trainer via `kader_trainers`) funktioniert bereits korrekt über die KaderPage und bleibt unverändert.

## Goals / Non-Goals

**Goals:**
- Eine Seite `/dienste` für alle Rollen, mit rollenabhängigen Aktionen
- `team_memberships` wird zur SQL-VIEW über den Kader — Board-Query bleibt unverändert, Daten sind automatisch korrekt
- Admin/Trainer können Slots löschen (mit Bestätigung), erfüllen und Geldersatz buchen
- Toggle „Meine" / „Alle" für Admin+Trainer (Meine = eigene Zuteilungen)
- `POST /api/members/{id}/team-assignment` und `AssignTeam`-Handler entfernen

**Non-Goals:**
- Dienstkonten-Integration (kommt in dashboard-home)
- Änderungen an der Duty-Slot-Erstellung oder den Diensttypen
- Änderungen an der Kader-Verwaltung

## Decisions

### 1. team_memberships als SQL-VIEW über den Kader

Statt den Board-Query umzuschreiben wird `team_memberships` per Migration von einer Tabelle zur VIEW umgewandelt:

```sql
DROP TABLE team_memberships;

CREATE VIEW team_memberships AS
SELECT km.id, km.member_id, k.team_id, k.season_id, 0 AS is_primary
FROM kader_members km
JOIN kader k ON k.id = km.kader_id
WHERE k.team_id IS NOT NULL

UNION

SELECT kt.kader_id * 100000 + kt.member_id, kt.member_id, k.team_id, k.season_id, 0
FROM kader_trainers kt
JOIN kader k ON k.id = kt.kader_id
WHERE k.team_id IS NOT NULL;
```

Effekt: Der bestehende Board-Query (`WHERE ds.team_id IN (SELECT team_id FROM team_memberships WHERE member_id IN (...))`) funktioniert unverändert für Spieler, Elternteile **und** Trainer. Die VIEW enthält sowohl `kader_members` als auch `kader_trainers`, sodass Trainer automatisch die Slots ihrer Teams sehen. Alle anderen Stellen im Code, die `team_memberships` lesen (z.B. Mitglieder-Listenfilter), profitieren ohne Anpassung.

*Alternative: Board-Query direkt auf kader_members umschreiben* — abgelehnt, weil mehr Codefläche betroffen und andere Lesestellen separat angepasst werden müssten.

*Alternative: team_memberships synchron mit kader halten* — abgelehnt, weil es zwei Schreibpfade einführt und das Problem nicht löst, sondern nur verlagert.

Da `team_memberships` nun eine VIEW ist, schlägt jeder INSERT fehl — das macht den `AssignTeam`-Endpoint zwingend obsolet.

### 2. Admin: Eigener Query-Zweig ohne Team-Filter

Für Admins greift der team_memberships-Filter nicht (sie haben kein Mitgliedsprofil). Der Board-Handler bekommt einen expliziten Admin-Zweig: kein WHERE auf team_id, alle Slots der aktiven Saison.

### 3. Meine/Alle-Toggle: Query-Parameter am bestehenden Endpoint

`GET /api/duty-board?view=mine` filtert zusätzlich auf Slots mit aktiver Zuteilung des anfragenden Nutzers (`duty_assignments.user_id = current_user`). Kein neuer Endpoint — der Toggle ist ein Darstellungsfilter, kein strukturell anderer Datenabruf.

*Alternative: Clientseitiger Filter* — abgelehnt, weil bei großen Saisons alle Slots übertragen werden müssten, auch wenn der Nutzer nur seine eigenen sehen will.

### 4. Zuteilungen: Lazy-Loading per Klick (wie bisher)

Assignments werden weiterhin per `GET /api/duty-slots/{id}/assignments` nachgeladen wenn ein Slot aufgeklappt wird. Kein Inline-Embed in den Board-Response, um die Payload-Größe klein zu halten.

*Alternative: Assignments inline im Board-Response* — abgelehnt wegen Payload-Größe bei vielen Slots + Assignments.

### 5. Delete: Immer für Admin+Trainer, mit Confirm-Dialog

`DELETE /api/duty-slots/:id` existiert bereits im Backend ohne Schutz für belegte Slots. Im Frontend erscheint für Admin+Trainer immer ein 🗑-Button. Bei `slots_filled > 0` öffnet sich ein Confirm-Dialog mit Warnung. Der Backend-Endpoint bleibt unverändert.

### 6. Navigation: Ein Eintrag „Dienste"

AppShell-Nav verliert „Dienstbörse" und „Dienst-Planung", erhält einen Eintrag „Dienste" → `/dienste`. Der Eintrag ist für alle eingeloggten Rollen sichtbar.

### 7. Reguläre Mitglieder: Kein Toggle

Spieler und Elternteile sehen immer alle Teamslots (kader-gefiltert) ohne Toggle. „Meine" für Mitglieder wäre ein leeres Interface für Nutzer die noch nichts beansprucht haben.

## Risks / Trade-offs

- **Trainer ohne Kader-Eintrag**: Wenn kein `kader_trainers`-Eintrag existiert (Kader noch nicht für die Saison angelegt), sieht der Trainer keine Slots. Mitigation: Hinweis im UI.
- **VIEW-ID-Kollision**: Die synthetische ID `kader_id * 100000 + member_id` im Trainer-Zweig der VIEW kann bei sehr großen IDs kollidieren. Da die `id`-Spalte der VIEW nirgends als Fremdschlüssel oder Primary Key genutzt wird (Board-Query liest nur `team_id` und `member_id`), ist das unkritisch.
- **Bestehende team_memberships-Daten**: Falls einzelne Zeilen manuell eingefügt wurden, gehen sie beim DROP verloren. Da die Tabelle nachweislich nie per Frontend befüllt wurde, ist kein Datenverlust zu erwarten.
- **AssignTeam-Endpoint-Entfernung**: Theoretisch könnten externe Skripte diesen Endpoint nutzen. Da er nie dokumentiert und nie per Frontend aufgerufen wurde, ist das Risiko vernachlässigbar.
