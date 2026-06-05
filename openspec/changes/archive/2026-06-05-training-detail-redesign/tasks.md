## 1. Session-Header: Stat-Badges

- [x] 1.1 In der Session-Info-Karte unterhalb von Uhrzeit/Ort eine Badge-Zeile ergänzen: drei Badges `bg-green-100 text-green-700` (✓), `bg-brand-danger-light text-brand-danger` (✗), `bg-brand-border-subtle text-brand-text-muted` (?); Werte aus `session.confirmed_count`, `session.declined_count`, `session.maybe_count`
- [x] 1.2 Trainer-only: viertes Badge „– N" (No-RSVP) ergänzen; Wert = `attendances.length - (confirmed + declined + maybe)`; erst rendern wenn `attendances` geladen
- [x] 1.3 Stats aus dem Rückmeldungen-Kartenheader entfernen (werden dort nicht mehr gebraucht)

## 2. Vereinte Teilnahme-Tabelle

- [x] 2.1 Bestehende Rückmeldungen-Karte und Anwesenheits-Karte entfernen; neue Karte „Teilnahme" mit `<table>` einführen (Card-Tabellen-Container-Klasse: `bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden`)
- [x] 2.2 Tabellen-Header: Spalten „Mitglied" / „Rückmeldung" / „Anwesend" — Anwesend-Spalte nur rendern wenn `isTrainer && isPast`
- [x] 2.3 Trainer-Datenquelle: Tabelle rendert aus `attendances` (jede Zeile = ein `AttendanceItem`); RSVP-Status kommt aus `a.rsvp_status`; Attendances-Call ohne `isPast`-Guard (immer laden für Trainer)
- [x] 2.4 Nicht-Trainer-Datenquelle: Tabelle rendert aus `session.responses` (nur Responder); keine Anwesend-Spalte
- [x] 2.5 RSVP-Zelle: Icon (Check/X/HelpCircle) für `confirmed/declined/maybe`; Strich `–` für kein RSVP; wenn `reason` vorhanden: `MessageCircle w-3 h-3 text-brand-text-muted ml-1` als Indikator

## 3. Kommentar-Tooltip

- [x] 3.1 State `showReasonId: number | null` ergänzen (für Mobile-Tap)
- [x] 3.2 RSVP-Zelle: `MessageCircle`-Icon in `<button onClick={() => setShowReasonId(...)}>` wrappen; RSVP-Zelle zusätzlich mit `group relative` für Desktop-Hover
- [x] 3.3 Tooltip-Div: `className="hidden group-hover:block sm:block absolute left-0 top-full z-10 mt-1 max-w-xs rounded-md bg-brand-text px-2 py-1 text-xs text-white shadow-lg"` — nur rendern wenn `reason` vorhanden
- [x] 3.4 Mobile-Fallback: unterhalb der Zeile `{showReasonId === a.member_id && reason && <tr><td colSpan={3} className="px-4 pb-2 text-xs text-brand-text-muted">{reason}</td></tr>}` (zusätzliche Tabellenzeile)

## 4. Auto-save Anwesenheit

- [x] 4.1 `saveAttendances`-Funktion: `attendanceSaving`-State und Speichern-Button entfernen; stattdessen direkt in `onChange` der Checkbox aufrufen
- [x] 4.2 State `attendanceError: string | null` ergänzen
- [x] 4.3 Bei Save-Fehler: Checkbox-State zurücksetzen (`setAttendanceMap(prev => ({ ...prev, [memberId]: !newValue }))`), `setAttendanceError('Fehler beim Speichern. Bitte nochmal versuchen.')` setzen
- [x] 4.4 Fehler-Banner am Fuß der Tabelle: `{attendanceError && <div className="px-4 py-2 text-xs text-brand-danger bg-brand-danger-light border-t border-brand-danger/20">{attendanceError} <button onClick={() => setAttendanceError(null)}>✕</button></div>}`
- [x] 4.5 Bei erfolgreichem Save: `setAttendanceError(null)` (Banner wegräumen falls es vorher da war)
