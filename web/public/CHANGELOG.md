## 10.06.2026
- [fix] termine: Generische Events nutzen Route /termine/ereignis/:id
- [feat] termine: Ort als Maps-Link bei Spielen und generischen Events
- [fix] termine: Ort als anklickbarer Maps-Link mit Icon
- [fix] upload: SEPA-Upload auf 2 MB begrenzen mit klarer Fehlermeldung
- [feat] members: Vorstand sieht alle Tabs in der Mitglied-Detailseite
- [feat] members: Vorstand erhält vollständigen Lese- und Schreibzugriff auf Mitgliederdaten
- [fix] teams: Mannschaftsnummern anhand aller Kader-Einträge bestimmen
- [feat] chat: Emoji-Reaktionen auf Nachrichten (WhatsApp-Style)
- [fix] termine: Ort und Mannschaft in Terminliste und Trainingsdetail anzeigen
- [fix] kalender: Team-Filter auf eigene/Kinder-Teams für Spieler und Elternteile beschränken
- [fix] teams: Mannschaftsnamen konsistent aus Kader-Nummerierung berechnen
- [fix] kalender: Mannschaftsliste auf aktiven Kader beschränken + Bearbeiten
- [fix] kalender: Detailanzeige zeigt Datumsrange, Event-Name und Teams

## 09.06.2026
- [feat] chat: Verlassen und Erneeutes Beitreten von Chats
- [feat] members: Checkboxen Beitragsfrei (Bankdaten) + Zweitspielrecht (Stammdaten)
- [feat] members: CSV-Import leitet beitragsfrei aus Status-Feld ab
- [feat] members: Felder beitragsfrei + zweitspielrecht – GET/PUT
- [fix] chat: Anzahl an ungelesenen Nachrichten zurücksetzen
- [feat] tooltip: Chat direkt aus Tooltip
- [feat] notify: keine Pushnotifications für sender
- [feat] upload: SEPA-Mandat sicher hochladen, öffnen und löschen
- [feat] members: WhatsApp-Sichtbarkeit im Profil und Nutzer-Tooltip
- [feat] version: Versionshistorie-Modal mit CHANGELOG.md
- [feat] sse: Vollständige SSE-Abdeckung für alle Seiten und Handler
- [fix] duties: Sonstige Dienste nach Datum und Event-Namen gruppiert
- [feat] chat: Cross-Device SSE-Sync und System-Nachricht beim Gruppenaustritt
- [feat] games: mehrtägige Events mit end_date für Turniere und Trainingslager

## 08.06.2026
- [feat] members: CSV-Import mit Adresse, IBAN-Validierung, Email-Klassifizierung und Preview-Modus
- [fix] absences: Kalender lädt Trainings nach Abwesenheits-Save neu
- [feat] members: Proxy-Accounts für Kinder ohne eigenen Login
- [feat] absences: Modellierung von Verletzung und Abwesenheit
- [feat] absences: Abwesenheitsbalken klickbar mit Info-Modal und Inline-Edit
- [feat] absences: Mitglied-Abwesenheiten implementieren
- [fix] api: kaputte Template-Literals in Frontend nach Route-Refactoring reparieren

## 07.06.2026
- [fix] chat: horizontales Scrollen im Nachrichtenverlauf unterbinden
- [fix] chat: Kontextmenü auf Mobile via Long-Press aktivieren
- [feat] chat: Nachrichten beantworten, bearbeiten und löschen
