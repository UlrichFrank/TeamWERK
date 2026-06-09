## 1. Datenbank

- [x] 1.1 Migration `033_games_end_date.up.sql` anlegen: `ALTER TABLE games ADD COLUMN end_date DATE`
- [x] 1.2 Migration `033_games_end_date.down.sql` anlegen (leer, da SQLite kein DROP COLUMN unterstützt)

## 2. Backend

- [x] 2.1 `CreateGame`: `end_date` aus Request-Body lesen, Validierung `end_date >= date`, in INSERT schreiben
- [x] 2.2 `UpdateGame`: `end_date` aus Request-Body lesen, Validierung, in UPDATE schreiben (auch NULL-Setzen ermöglichen)
- [x] 2.3 `ListGames` und `GetGame`: `end_date` aus DB lesen und in Response-Struct aufnehmen (`EndDate *string`)

## 3. Frontend — Kalender-Rendering

- [x] 3.1 `Game`-Interface in `KalenderPage.tsx` um `end_date?: string | null` erweitern
- [x] 3.2 Filterlogik für `dayGames` anpassen: Event anzeigen wenn `date <= dateStr <= end_date` (oder `end_date` null und `date === dateStr`)

## 4. Frontend — Event-Wizard

- [x] 4.1 Im Event-Wizard-Formular für `event_type === 'generisch'` ein optionales Enddatum-Feld hinzufügen
- [x] 4.2 Client-seitige Validierung: Enddatum muss >= Startdatum sein
- [x] 4.3 `end_date` beim Submit an `POST /api/kalender` mitsenden (oder weglassen wenn leer)

## 5. Frontend — Event-Edit-Modal

- [x] 5.1 Im Edit-Modal (`GameEditModal` oder analoges Formular) `end_date`-Feld für alle Event-Typen ergänzen
- [x] 5.2 `end_date` beim Submit an `PUT /api/kalender/{id}` mitsenden
