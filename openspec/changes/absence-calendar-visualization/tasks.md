## 1. Frontend — Kalender-Zelle umstrukturieren

- [x] 1.1 Zell-Div in `KalenderPage.tsx`: `relative` hinzufügen, `p-1.5` beibehalten
- [x] 1.2 Abwesenheitsbalken aus dem Dokumentfluss nehmen: als `absolute top-[2px] left-[2px] right-[2px] h-5` vor den Content-Elementen rendern
- [x] 1.3 Content-Wrapper `<div className="relative z-10">` um Tag-Kopfzeile und Event-Pills legen

## 2. Frontend — Balken-Styling

- [x] 2.1 Typ-Farben einsetzen: `bg-brand-yellow/20 border-brand-yellow/40` für `vacation`, `bg-red-400/20 border-red-400/40` für `injury`
- [x] 2.2 Radius-Logik: `rounded` (isFirst && isLast), `rounded-l` (isFirst), `rounded-r` (isLast), kein Radius (Mitteltag)
- [x] 2.3 Alten Inline-Stil entfernen: `h-1.5 mb-1 bg-brand-yellow/40 border border-brand-yellow` und `-mx-1.5`-Logik

## 3. Backend — Overlap-Validierung

- [x] 3.1 In `internal/absences/handler.go`, `Create`-Handler: vor dem INSERT COUNT-Query ausführen
  ```sql
  SELECT COUNT(*) FROM member_absences
  WHERE member_id = ? AND type = ?
    AND start_date <= ? AND end_date >= ?
  ```
- [x] 3.2 Bei Count > 0: HTTP 409 mit `{"error":"overlap"}` zurückgeben

## 4. Frontend — Fehlerbehandlung 409

- [x] 4.1 In `doSaveAbsence` (KalenderPage.tsx): HTTP-409-Antwort erkennen und spezifische Fehlermeldung zeigen: „Eine Abwesenheit dieses Typs überschneidet sich bereits mit diesem Zeitraum."
