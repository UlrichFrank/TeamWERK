## 14.06.2026
- [feat] carpooling: team_ids und time in Mitfahrgelegenheiten-Response
- [fix] web: Legacy-Role-Checks durch hasFunction ersetzt
- [fix] absences: Phantom-Vereinsfunktion sportvorstand und trainer-Role-Check entfernt
- [fix] scheduler: Reminder-Empfänger über Vereinsfunktion und family_links
- [fix] duties: eligibleDutyUsers über member_club_functions statt users.role
- [fix] members: SEPA-Dokument-Löschen setzt Mandat-Flag zurück
- [fix] carpooling: Mitfahrgelegenheiten-Bugfixes und Team-Filter
- [feat] mailer: MAILER_DISABLED-Flag zum Deaktivieren des E-Mail-Versands
- [fix] trainings: sportliche_leitung kann Termine für alle Mannschaften anlegen und sehen
- [feat] absences: Mannschaftsabwesenheiten im Kalender für Trainer
- [feat] ui: globaler Zurück-Button in AppShell
- [feat] members: Kartendienst-Präferenz im Profil einstellbar
- [feat] dashboard: erweiterter Kader einheitlich als Badge kennzeichnen
- [feat] dashboard: erweiterter Kader im Dashboard kennzeichnen
- [fix] trainings: rsvp_opt_out gilt nicht für erweiterten Kader
- [fix] trainings: erweiterter Kader in Trainings-Anwesenheitsliste korrekt ausweisen
- [feat] trainings: erweiterter Kader sieht Trainings, Anwesenheitsliste und Benachrichtigungen
- [feat] members: Anwärter-Status für neue Spieler ohne Vereinsmitgliedschaft
- [feat] kader: erweiterter Kader sichtbar auf MeinTeamPage und in Spielliste
- [fix] duties: Trainer-Lesezugriff auf Dienst-Typen und Templates
- [feat] members,auth: Verknüpfungsstatus-Filter und Beitrittsantrag-Deeplink

## 13.06.2026
- [feat] ux: Zurück-Button auf MeinTeamPage bei gefilterter Team-Ansicht
- [fix] security: Security- und Datenintegritäts-Hardening (13 Bugs)
- [feat] files: PermissionsModal zeigt Nutzernamen statt User-ID
- [feat] auth: GET /api/users/picker — team-scoped Nutzerliste
- [fix] files: resolveAccess nearest-ancestor-wins + family context
- [feat] members: CSV-Import-Modus "enrich" – nur leere Felder ergänzen

## 12.06.2026
- [feat] team: Tab-Ansicht bei Teams
- [fix] mailer: Logo via CID einbetten statt externer URL
- [fix] mailer: Logo via CID einbetten statt externer URL
- [fix] mailer: Precedence- und X-Mailer-Header für bessere Zustellbarkeit
- [fix] notify: mailer.New-Aufruf um baseURL-Parameter ergänzen
- [fix] mailer: Exakte App-Kachel – border-top statt div, Tailwind shadow
- [fix] mailer: Hintergrund weiß, Kachel grau (brand-surface-card)
- [fix] mailer: Brand-Design – gelber Hintergrund, weiße Kachel wie in App
- [fix] mailer: Logo links, Titel rechts; gleiches Format in allen E-Mails
- [feat] mailer: Team-Stuttgart-Logo in E-Mail-Header einbinden
- [fix] mailer: Action-Links als CTA-Button rendern statt nackte URL
- [fix] mailer: HTML-Part ergänzen und Spam-Einstufung reduzieren
- [fix] teams: Kurznamen für alle Rollen konsistent über GET /api/teams/names
- [fix] games: RegenSummaryCard gegen null-Arrays absichern
- [fix] games: TestDeleteGame durch echten Grenzfall-Regen-Test ersetzen
- [fix] termine: Zurück-Button nutzt Browser-History statt hardcoded URL
- [feat] scheduler: Reminder-Links auf konkreten Termin
- [feat] trainings: Push-Link auf konkretes Training
- [feat] games: Push-Link auf konkretes Spiel
- [feat] termine: URL-driven Filter und Focus-Param
- [feat] dashboard: Datum in "Meine Termine" als eigene Spalte anzeigen

## 11.06.2026
- [feat] kalender: Heute-Button zum Springen auf aktuellen Monat
- [fix] trainings: Anwesenheitserfassung Datum SQL-seitig vergleichen
- [feat] user: Direktes Erzeugen von Nutzeraccounts
- [fix] trainings: sportliche_leitung kann Anwesenheiten erfassen
- [fix] auth: Beitrittsanfrage-Name und Reload nach Genehmigung
- [feat] abwesenheiten: Familienurlaub für mehrere Kinder gleichzeitig
- [feat] chat: Teilnehmer-Verwaltung für Gruppen-Konversationen
- [feat] notify: Kategorie-Fassade für Push+Email und Dienst-Notify bei Event-Löschung

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
