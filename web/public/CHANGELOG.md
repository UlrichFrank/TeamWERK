## 22.06.2026
- [feat] members: CSV-Import nutzt Status TeamWERK und eigene beitragsfrei-Spalten
- [feat] members: Grund für Beitragsfreiheit im Bankdaten-Tab editierbar
- [feat] members: Kassierer pflegt Beitragsfrei + Grund via bank-details
- [feat] db: Migration 007 — members.beitragsfrei_grund
- [feat] monitoring: Host- und SQLite-Metriken via Vector-Pipeline (#66)
- [fix] games: generische Events mit template_id erlauben
- [feat] games: template_id als persistente Slot-Quelle pro Event
- [feat] metrics: make metrics + metrics-gate für Code-Qualität
- [fix] auth: log.Printf in DeleteUser auf slog.Error migrieren
- [fix] web: Changelog-Modal als 3-Spalten-Grid mit Scope-Truncate

## 21.06.2026
- [feat] carpooling: One-Click-Paarung ohne vorherigen Gegen-Eintrag
- [feat] members: selektiver CSV-Import (Feld-Whitelist + Zeilen-Auswahl)
- [feat] log: Strukturiertes slog-Logging als neutrale Schnittstelle
- [feat] scheduler: Monitoring-Heartbeat nach erfolgreichem Lauf
- [feat] health: /api/healthz + /api/metrics + Panic-Recover-Middleware
- [feat] db: Migration 005 monitoring_heartbeat (Dead-Man-Datenquelle)
- [fix] auth: DeleteUser räumt member_change_drafts auf
- [feat] members: SEPA-Mandatsdatum in Kontakt-Tab editierbar
- [feat] web: recovery_email UI — Kindprofil, Konto-Tab, Admin-Override, Passwort-vergessen
- [feat] members: recovery_email lesbar im Kindprofil und eigenen Profil
- [feat] auth: persistente recovery_email — forgot-password, Doppelbestätigung, Override
- [feat] db: recovery_email auf users + field/stage auf email_change_tokens
- [feat] telemetry: anonymes Matomo-Tracking für Frontend-Nutzung
- [fix] profile: Datenschutz-Tab aktualisiert sich beim Wechsel zwischen Kindern
- [feat] profile: Datenschutz-Tab auch im Kind-Profil
- [feat] carpooling: Upsert prüft Event-Sichtbarkeit
- [fix] auth: Nutzerliste aktualisiert sich nach dem Löschen live
- [fix] auth: Impersonation und Löschen für Kinderaccounts ohne E-Mail
- [feat] games: /games-Routen filtern nach Event-Sichtbarkeit
- [feat] auth: zentraler Helper für Event-Sichtbarkeit pro User
- [feat] termine: Hinweis bei gefilterten Multi-Team-Teilnehmern
- [feat] members: Datenschutz-Tab im Admin um Sichtbarkeitstoggle ergänzen
- [feat] profile: Datenschutz-Tab mit Sichtbarkeitstoggle und DSGVO-Anzeige
- [feat] members: cross_team_visible per dediziertes Endpoint direkt setzbar
- [feat] games: /participants filtert fremde Teams bei Multi-Team-Events
- [feat] db: cross_team_visible auf members für Opt-In-Cross-Team-Sichtbarkeit
- [fix] chat: Push-Notification springt zur richtigen Unterhaltung
- [feat] games: Teilnehmer generischer Mehr-Team-Ereignisse nach Team gruppieren
- [feat] auth: Kinderaccounts ohne E-Mail mit Spielername-Login
- [fix] pwa: /manifest.json als Alias auf manifest.webmanifest

## 20.06.2026
- [feat] profile: Anleitung zum Kalender-Abo für iOS und Android
- [fix] pwa: setAppBadge-Rejection killt iOS-Pushes nicht mehr
- [feat] pwa: Android maskable Icon + Notification-Badge korrigieren (#47)
- [feat] files: Datei-Link aus 3-Punkte-Menü kopieren
- [fix] auth: doppelten Refresh-Bootstrap im StrictMode verhindern
- [feat] chat: App-Icon-Badge spiegelt ungelesene Chat-Nachrichten (#46)
- [fix] ui: Dashboard-Link, Beitragslauf-Bestätigung und Stammvereine-Mobile (#45)
- [feat] members: Mitgliedsnummer systemverwaltet und eindeutig
- [feat] notification: Neue Mitfahrgesuche lösen Notfication aus
- [fix] fee-run: Aktions-Buttons im Beitragslauf-Header oben anordnen
- [fix] beitragslauf: SEPA-XML mit FwdgAgt-Absender-BIC im GrpHdr (#41)
- [fix] teams: Eltern des erweiterten Kaders in "Mein Team" anzeigen
- [feat] dashboard: "Alle Termine"- und "Alle Dienste"-Links auf Übersicht
- [fix] beitragslauf: SEPA-XML komplett auf ASCII transliterieren (#38)
- [fix] beitragslauf: Verwendungszweck verkürzen und Vereinsnamen ergänzen (#37)
- [feat] beitragslauf: Filter für Kategorie und Hinweis (#36)
- [fix] beitragslauf: ausgetreten/honorar/anwaerter aus Preview filtern (#35)
- [fix] members: CSV-Import matcht Bestandsmitglieder ohne Geburtsdatum
- [fix] members: 2-stelliges Jahr im CSV-Import nicht in die Zukunft mappen
- [fix] members: CSV-Import-Match toleriert Timestamp-Geburtsdaten
- [fix] members: irreführendes Import-Modus-Label 'Nur ergänzen' umbenannt
- [feat] beitragslauf: nicht abbuchbare Beträge sichtbar machen
- [feat] db: Migrationen 048+049 für Stammverein-Backfill
- [feat] dashboard: einheitliches Zeilen-Raster für Termine, Dienste und Fahrt
- [feat] dashboard: partnerTreffpunkt in carpoolingConfirmed-Payload
- [feat] stammvereine: Settings-Tab CRUD + Stammverein-Auswahl im Mitglied
- [feat] members: home_club_id-Zuordnung + home_club_name in GET
- [feat] stammvereine: CRUD-Package mit Routen
- [fix] db: Migration 046 — Passiv-Satz ab Saisonstart 2026/27 gültig
- [feat] dashboard: Mitfahr-Einträge sind klickbar und springen zum Ziel-Eintrag
- [feat] calendar: Vorname an iCal-Kalendernamen anhängen
- [fix] calendar: iCal-Feed-Termine landeten im Jahr 1 (DATE als ISO-Timestamp)
- [feat] calendar: persönlicher iCal-Feed für Spiele, Trainings und Dienste (#25)
- [fix] policy: Kassierer sieht Mitgliederliste, Beitragslauf und Einstellungen

## 19.06.2026
- [feat] dashboard: Dashboard zeigt offene Mitfahr-Gesuche der eigenen Teams
- [fix] members: Eltern von Anwärtern sehen Termine und Team des Kindes
- [fix] members: Trainer sehen in der Liste nur reduzierte Mitgliedsfelder
- [fix] auth: Vorstand darf Einladungen und Beitrittsanträge verwalten
- [fix] members: Trainer und sportliche Leitung dürfen /api/members lesen
- [fix] venues: Vorstand darf GET /api/venues lesen
- [feat] members: Kassierer kann Bankdaten über /bank-details bearbeiten
- [feat] web: SEPA-Beitragslauf-UI (VereinTab, BeitraegeTab, BeitragslaufPage)
- [feat] app: Beitragslauf-/Beitragssatz-Routen und Kassierer-Gruppe verdrahten
- [feat] members: Kassierer-Lesezugriff + Bankdaten-Endpoint (Feld-Whitelist)
- [feat] beitragslauf: Kategorisierung, pain.008.001.08-XML, Saison-Protokoll
- [feat] beitragssaetze: CRUD-Handler für Beitragsmatrix (3 Kategorien)
- [feat] config: Club-API um SEPA-Stammdaten erweitert
- [feat] sepa: IBAN-Validierung (Mod-97, länderspezifische Länge)

## 18.06.2026
- [fix] permissions: Vorstand darf Dienst-Slots verwalten (manage_duties)
- [feat] permissions: zusätzliche Policy-Capabilities für Frontend-Gating
- [fix] pwa: Reload-Fallback löscht Precache- und app-shell-Cache
- [feat] pwa: Navigationen via NetworkFirst, index.html nicht mehr im Precache
- [feat] app: Cache-Control- und ETag-Header im SPA-Handler
- [feat] permissions: Policy-Package, _can-Annotationen und hasFunction-Migration
- [feat] chat: Tageswechsel-Separator im Chat-Verlauf

## 17.06.2026
- [fix] carpooling: Kurznamen statt langer Teamnamen in Mitfahrgelegenheiten anzeigen
- [fix] test: Import-Zyklus files↔testutil beheben, SQLite-In-Memory-Schema stabil machen
- [fix] members: Abwesenheits-Sichtbarkeit nur für Spieler anzeigen
- [fix] update: fix double banner
- [fix] chat: Overlay-Breite, Textselektion und iOS-Click-Bug behoben
- [feat] chat: Push-Benachrichtigungen für Nachrichten steuerbar im Profil
- [fix] chat: Mobile Overlay – Scroll-Sperre, schmäleres Menü, keine Textselektion in Buttons
- [fix] chat: Reaktions-Toggle löscht vorherige Emoji, Context-Menu-Clamp und Copy-Option
- [fix] chat: WhatsApp-Style Mobile Action Overlay bei Long-Press
- [fix] chat: Nachrichten-UX – Zeilenumbrüche, Textselektion, Links, Emoji-Regeln
- [fix] dashboard: Meine Dienste filtert offene Slots nach Audience
- [fix] trainings: Vorstand sieht alle Trainings im Kalender
- [fix] duties: Vorstand darf Dienst-Slots anlegen, ändern, löschen
- [feat] modal: Spieltag Details nun als Modal
- [fix] games: Spiele im Kalender rollenbasiert filtern
- [feat] games: can_edit-Flag in GET /api/games/{id}

## 16.06.2026
- [feat] ui: Event-Typ-Filter als Dropdown mit Checkboxen auf Mobile
- [fix] duties: Eltern-Audience-Match nur wenn Kind im Slot-Team spielt
- [feat] duties: Audience-Filter-Pille auf Dienste-Seite für Trainer und Vorstand
- [feat] duties: Trainer-Sicht und umschaltbarer Audience-Filter auf Dienstbörse
- [feat] teams: einheitliche Team-Darstellung über Kurz-/Langform-Display-Felder
- [feat] chat: mehrzeilige Nachrichten mit WhatsApp-Steuerung
- [fix] duties: Mobile-Action-Menü bei leeren Optionen ausblenden
- [feat] carpooling: Elternzugang für Mitfahrgelegenheiten

## 15.06.2026
- [feat] kalender: Spiel-Kacheln zeigen RSVP-Zähler und Dienst-Punkt in Teamname-Zeile
- [fix] games: RSVP-Zähllogik in allen Endpoints konsistent
- [feat] termine: Badges für aktive RSVP-Konfiguration in der Detailansicht
- [feat] trainings: RSVP-Konfiguration nachträglich bearbeitbar
- [feat] games: GameEditModal zeigt RSVP-Konfiguration
- [feat] trainings: RSVP-Konfiguration für Session und Series bearbeitbar
- [feat] games: RSVP-Konfiguration nachträglich änderbar
- [feat] forms: PasswordInput-Komponente mit User-typed-Erkennung
- [fix] auth: Legacy Path=/api/auth-Cookie räumen
- [fix] auth: Refresh-Token-Cookie auf Path=/ setzen
- [fix] version: Dismiss bezieht sich auf konkrete Version
- [fix] pwa: Reload wartet auf neuen SW und leert ggf. api-cache
- [fix] version: Hook reagiert auf user, DEV zeigt v dev, ?token entfernt
- [feat] version: VersionContext zentralisiert SSE-Versionserkennung
- [fix] pwa: SSE-Endpoints aus NetworkFirst-Caching ausnehmen
- [fix] icon: Icons für Dienste und Mitfahrgelegenheiten
- [fix] carpooling: Team-Dropdown nutzt /api/teams für Admin/Vorstand
- [fix] carpooling: Vorstand-Bypass via HasFunction statt System-Rolle
- [feat] duties: Dienstbörse mit Pill-Filtern, Team-Dropdown und Farbcodierung
- [feat] duties: Vorstand-Bypass und team_id/event_type in /duty-board

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
