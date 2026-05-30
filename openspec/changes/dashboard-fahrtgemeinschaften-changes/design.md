## Context

Das Dashboard zeigt pro User einen personalisierten Überblick. Die Sektion „Fahrtgemeinschaften" wird durch `queryCarpoolingHint` im `internal/dashboard/handler.go` befüllt und gibt bisher nur das nächste Auswärtsspiel mit aggregierten Zählern zurück.

Paarungsanfragen, Zusagen und Ablehnungen sind in `mitfahrt_paarungen` mit `status` und `updated_at` persistent. Gelöschte Einträge in `mitfahrgelegenheiten` lösen jedoch `ON DELETE CASCADE` auf `mitfahrt_paarungen` aus — Paarungen zu einem gelöschten Eintrag verschwinden spurlos. Push Notifications für alle diese Ereignisse sind bereits implementiert und laufen korrekt.

Der Push-Notification-Change (push-notifications-mitfahrgelegenheiten) ist vollständig implementiert. Die Delete-Handler senden bereits vor dem DELETE einen Push an betroffene User.

## Goals / Non-Goals

**Goals:**
- Dashboard zeigt personalisierten Status: eigener Eintrag (biete/suche), aktuell bestätigte Paarungen, Ereignisse der letzten 48 h
- Alle relevanten Carpooling-Ereignisse werden in `carpooling_events` persistiert (vollständiger Log)
- Dashboard-Query vereinfacht: „was hat sich für mich geändert?" kommt aus einer einzigen Tabelle
- Kein weiterer Endpunkt nötig — Daten kommen über die bestehende `GET /api/dashboard`-Route

**Non-Goals:**
- Kein persistentes „gelesen"-Marking für Events
- Keine Push-Änderungen (die laufen bereits korrekt)
- Kein Anzeigen von Events für vergangene Spiele
- Kein globaler Audit-Log (nur Events, die den jeweiligen User betreffen)

## Decisions

### 1. Vollständiger Event-Log statt Live-Abfrage auf Paarungen

**Entscheidung:** `carpooling_events` speichert alle Ereignisse, die einen User betreffen: neue Einträge anderer (`biete_created`, `suche_created`), Paarungsaktionen (`pairing_requested`, `pairing_confirmed`, `pairing_rejected`, `pairing_cancelled`) und Löschungen (`biete_deleted`, `suche_deleted`).

**Warum:** Eine einheitliche Quelle vereinfacht den Dashboard-Query erheblich — statt `mitfahrt_paarungen` mit 48-h-Fenster-Logik separat abzufragen, liest man nur `carpooling_events WHERE user_id = ? AND game_id = ? AND created_at >= now - 48h`. Außerdem ist der Log robuster: Löschungen hinterlassen bei CASCADE-DELETE keine Spur in `mitfahrt_paarungen`, aber der Event ist bereits geschrieben.

**Alternative:** Nur Löschungen loggen, Rest aus Live-Daten. Abgelehnt: zwei unterschiedliche Abfragelogiken im Dashboard-Handler, 48-h-Fenster-Filter auf `mitfahrt_paarungen.updated_at` nötig, schwerer nachvollziehbar.

**Write-Points:**

| Handler | Trigger | Event-Typ | Empfänger |
|---|---|---|---|
| `Upsert` | neuer biete-Eintrag | `biete_created` | alle User mit suche für dasselbe Spiel |
| `Upsert` | neuer suche-Eintrag | `suche_created` | alle User mit biete für dasselbe Spiel |
| `RequestPairing` | Anfrage gestellt | `pairing_requested` | Gegenseite |
| `ConfirmPairing` | Anfrage bestätigt | `pairing_confirmed` | Initiator |
| `RejectPairing` | pending abgelehnt | `pairing_rejected` | Initiator |
| `RejectPairing` | confirmed storniert | `pairing_cancelled` | Gegenseite |
| `Delete` | biete gelöscht (vor CASCADE) | `biete_deleted` | User mit pending/confirmed Paarung |
| `Delete` | suche gelöscht (vor CASCADE) | `suche_deleted` | Biete-User mit pending/confirmed Paarung |

### 2. Dashboard zeigt confirmed-Paarungen separat vom Event-Feed

**Entscheidung:** `paarungen` im Dashboard-Response enthält ausschließlich aktuell `confirmed` Paarungen (unabhängig vom Alter). `recentEvents` zeigt den 48-h-Log.

**Warum:** Eine bestätigte Mitfahrt ist bindende Information, die dauerhaft sichtbar sein muss — auch wenn die Bestätigung vor mehr als 48 h kam. Der Event-Feed ist dagegen für kurzfristige Änderungen gedacht.

### 3. Event-Cleanup implizit via Spielfilter

**Entscheidung:** Kein Cron-Job. Events zu vergangenen Spielen werden im Dashboard-Query durch `DATE(g.date) >= DATE('now')` herausgefiltert.

**Warum:** Einfachste Lösung ohne zusätzliche Infrastruktur. Die DB wächst minimal (wenige Events pro User pro Saison).

### 4. Write-Point im Delete-Handler: Transaktion vor CASCADE

**Entscheidung:** Events werden *vor* dem `DELETE FROM mitfahrgelegenheiten` in derselben Transaktion geschrieben.

**Warum:** Nach dem DELETE sind Paarungsdaten durch CASCADE weg. Transaktion sichert Konsistenz.

### 5. Dashboard-Antwortstruktur erweitern (kein neuer Endpunkt)

**Entscheidung:** `CarpoolingHint`-Struct um `MyEntry`, `Paarungen` (nur confirmed) und `RecentEvents` erweitern. Kein separater API-Endpunkt.

**Warum:** Das Dashboard lädt alles in einem einzigen `GET /api/dashboard`-Request. Ein separater Endpunkt würde einen zweiten Netzwerk-Request erzwingen.

### 6. SSE-Integration für Live-Aktualisierung des Dashboards

**Entscheidung:** `DashboardPage.tsx` abonniert den bestehenden `useLiveUpdates`-Hook (aus dem Change `global-sse-live-updates`) und lädt die Dashboard-Daten still neu bei SSE-Events vom Typ `"mitfahrgelegenheiten"`.

**Warum:** Alle Carpooling-Mutations (Upsert, Delete, Paarungsanfragen, Confirm, Reject) broadcasten bereits `hub.Broadcast("mitfahrgelegenheiten")`. Kein neuer Backend-Code nötig — das Dashboard muss nur zuhören. Nach einem silent Reload enthält `carpoolingHint` sofort die aktuellen `recentEvents` und `paarungen`.

**Kein eigener SSE-Event-Typ:** Ein separater `"dashboard"`-Typ wäre nötig, wenn das Dashboard auf Mutations reagieren müsste, die keine anderen Seiten betreffen. Das ist nicht der Fall — Mitfahrgelegenheiten-Mutations sind der einzige relevante Trigger für den Fahrtgemeinschaften-Bereich.

**Scope:** Nur `"mitfahrgelegenheiten"`-Events triggern einen Reload der Dashboard-Daten. Andere SSE-Events (`"duties"`, `"games"`, `"members"`) betreffen andere Dashboard-Sektionen und sind im Scope dieses Changes nicht berücksichtigt.

## Risks / Trade-offs

- **Transaktion im Delete-Handler:** Muss auf `db.BeginTx` umgestellt werden. → Standardmuster in Go, gut handhabbar.
- **Doppelte Writes in Paarungs-Handlern:** Event-Log-Write kommt zusätzlich zu bestehendem Push-Notification-Goroutine. → Beide bleiben unabhängig; kein Konflikt.
- **48-h-Fenster für recentEvents:** Wer länger nicht schaut, sieht ältere Ereignisse nicht mehr. → Akzeptiert; Push-Notification kompensiert den Real-Time-Fall. Bestätigte Paarungen bleiben dauerhaft über `paarungen`-Feld sichtbar.
- **Kein „ungelesen"-Indikator:** Events sehen immer gleich aus, egal ob gerade eingetreten oder 47 h alt. → Bewusst einfach gehalten; Zeitstempel im Widget schafft ausreichend Kontext.

## Migration Plan

1. Migration `018_carpooling_events.up.sql` + `.down.sql` anlegen
2. Backend deployen (Binary enthält Migration) → `make deploy` führt `migrate up` automatisch aus
3. Kein Rollback-Risiko: nur neue Tabelle und erweitertes Dashboard-JSON, bestehende Features unberührt
