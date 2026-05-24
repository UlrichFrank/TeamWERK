## Context

Die bestehende `vehicle_info`-Tabelle speichert allgemeine Fahrzeugdaten eines Nutzers (Sitze, Notiz) — unabhängig von einem konkreten Spiel. Für Mitfahrkoordination brauchen wir spielspezifische Einträge: Wer fährt wohin, wie viele Plätze, wo ist der Treffpunkt.

Auswärtsspiele sind in `games` als `is_home = 0` markiert. Diese Tabelle ist die Basis für die neue Seite.

## Goals / Non-Goals

**Goals:**
- Pro Auswärtsspiel: Fahrer und Mitfahrer sichtbar machen
- Einfaches Eintragen (1-2 Klicks): biete / suche Mitfahrt
- Rückzug möglich (eigenen Eintrag löschen)
- Kurzinfo im Dashboard

**Non-Goals:**
- Automatisches Matching / Zuweisung von Mitfahrern zu Fahrern
- Bestätigungs-Workflow (kein Accept/Reject)
- Benachrichtigungen per E-Mail
- Vergangene Spiele (nur zukünftige Auswärtsspiele)
- Heimspiele

## Decisions

### D1: Eigenes Schema statt `vehicle_info` erweitern

`vehicle_info` ist spielunabhängig (Stammdaten). Ein spielspezifischer Eintrag in einer separaten Tabelle ist klarer trennbar und vermeidet NULL-Felder für nicht-fahrende Nutzer.

```sql
CREATE TABLE mitfahrgelegenheiten (
  id            INTEGER  PRIMARY KEY AUTOINCREMENT,
  game_id       INTEGER  NOT NULL REFERENCES games(id)  ON DELETE CASCADE,
  user_id       INTEGER  NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
  typ           TEXT     NOT NULL CHECK(typ IN ('biete','suche')),
  plaetze       INTEGER,          -- nur wenn typ='biete'; NULL wenn suche
  treffpunkt    TEXT,             -- optional
  notiz         TEXT,             -- optional
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(game_id, user_id)        -- 1 Eintrag pro Spiel pro Nutzer
)
```

### D2: Kein separater "Matching"-Schritt

Variante B bedeutet volle Transparenz: Alle Einträge sichtbar, jeder sieht wen er direkt kontaktieren kann. Kein Request/Confirm-Workflow hält die Komplexität gering und ist für die Vereinsgröße ausreichend.

### D3: GET /api/mitfahrgelegenheiten — aggregierte Response

Der Endpoint liefert alle zukünftigen Auswärtsspiele mit je zwei Listen:
```json
[
  {
    "game": { "id": 1, "date": "...", "opponent": "...", "team": "..." },
    "biete": [
      { "id": 1, "user_name": "Maria K.", "plaetze": 3, "treffpunkt": "Halle 9:00", "notiz": "", "is_own": true }
    ],
    "suche": [
      { "id": 2, "user_name": "Peter S.", "notiz": "2 Personen", "is_own": false }
    ]
  }
]
```

`is_own: true` markiert den eigenen Eintrag (zum Bearbeiten/Löschen). Namen anderer Nutzer werden angezeigt (kein Datenschutzproblem — im Vereinskontext bewusst).

### D4: POST zum Anlegen/Aktualisieren (Upsert)

Da nur 1 Eintrag pro Nutzer pro Spiel erlaubt ist, behandelt `POST /api/mitfahrgelegenheiten` einen bestehenden Eintrag als Update (INSERT OR REPLACE). Kein separater PUT nötig.

### D5: Dashboard-Kurzinfo via bestehendem `/api/dashboard`

Der Dashboard-Handler gibt im `vehicleInfo`-Feld (oder einem neuen `carpoolingHint`-Feld) eine Kurzinfo: nächstes Auswärtsspiel + Zählerstände. Kein zweiter API-Call im Frontend.

## Risks / Trade-offs

- **Datenschutz Namen**: Nutzernamen (nicht E-Mails) werden in der Liste angezeigt. Im Vereinskontext vertretbar — alle kennen sich. Mitigation: Nur `name` aus `users`, keine E-Mail.
- **Keine Benachrichtigung**: Ein Fahrer weiß nicht, wer sein Angebot genutzt hat. Mitigation: klare UI-Kommunikation „Kontakt direkt im Verein klären".
- **Mobile-Komplexität**: Zwei Listen pro Spiel könnten auf kleinem Screen unübersichtlich sein. Mitigation: kompaktes Card-Layout, Tabs/Toggle für biete/suche pro Spielkarte.

## Migration Plan

1. Migration 013 deployen
2. Backend + Frontend deployen
3. Nav-Eintrag erscheint sofort; Seite ist leer bis erste Einträge
