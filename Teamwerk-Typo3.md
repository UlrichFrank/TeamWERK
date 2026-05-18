# TeamWERK — Realisierung mit TYPO3 14 + fe_login

Explorations-Zusammenfassung: Machbarkeit des TeamWERK-Konzepts auf dem bestehenden Stack (TYPO3 14, Tailwind CSS, Mittwald Webhosting L).

---

## Hosting-Constraint: Mittwald Webhosting L

Persistente Prozesse (Node.js-Server, Python-API, Laravel Queue Worker) sind auf Webhosting L **nicht möglich**. Der Plan unterstützt ausschließlich PHP-Apps (CGI/FPM-Modell).

| Feature | Webhosting L | Space Server / vServer |
|---|---|---|
| PHP-Apps (TYPO3, Laravel klassisch) | ✅ | ✅ |
| SSH-Zugang | ✅ | ✅ |
| Cronjobs (unbegrenzt, min. 1 min) | ✅ | ✅ |
| Node.js als persistenter Server | ❌ | ✅ |
| Python-App (Flask, FastAPI...) | ❌ | ✅ |
| PHP Workers / Laravel Queue Worker | ❌ | ✅ |
| Container / eigene Binaries (Go etc.) | ❌ | ✅ |

**Konsequenz:** Entweder alles in TYPO3/PHP (Webhosting L reicht), oder Upgrade auf Space Server (~21 €/Monat) für einen zweiten Stack.

---

## fe_login — was es liefert

`fe_login` verwaltet Frontend-Nutzer (`fe_users`) und Gruppen (`fe_groups`). TYPO3 sperrt Seiten und Content-Elemente nativ per Gruppe.

```
Konzept-Rolle     →  fe_group-Mapping
─────────────────────────────────────────
Vereinsadmin      →  fe_group: admin
Jugendleiter      →  fe_group: jugendleiter
Teamleiter        →  fe_group: teamleiter_m1, teamleiter_m2, …
Trainer           →  fe_group: trainer_m1, trainer_m2, …
Elternteil        →  fe_group: eltern
Spieler (>14)     →  fe_group: spieler
```

**Wichtig:** Seitenebene-Zugangskontrolle ist nativ. **Datenisolation auf Datensatzebene** (Eltern sehen nur eigene Kinder) muss die Extbase-Logik selbst durchsetzen — das ist kein TYPO3-Feature.

---

## Fit-Assessment je Modul (Überblick)

| Modul | TYPO3-Fit | Begründung |
|---|---|---|
| CORE — Administration | ✅ gut | fe_users + fe_groups + Extbase, Cronjob für Scheduler |
| MEMBERS — Mitglieder | ⚠️ mittel | Custom Table + Extbase-Repository; CSV-Import und Lizenz-Tracking = Extra-Aufwand |
| SCHEDULE — Termine | ⚠️ mittel | Custom doktype (wie Spielberichte), aber Zu-/Absagen brauchen eigenen Extbase-Controller |
| DUTIES — Dienste | ⚠️ mittel | Großteils machbar; Rotationsvorschläge und Diensttausch aufwändig |
| TRANSPORT — Fahrdienst | ⚠️ mittel | Anmeldung + E-Mail machbar; visuelle Zuordnung braucht JS |
| HALLS — Hallenzeiten | ⚠️ mittel | Kalenderanzeige machbar, Buchungs-/Genehmigungs-Workflow aufwändig |
| EVENTS — Turniere | ❌ schwer | Spielplan-Generator und Live-Tabelle nicht sinnvoll in TYPO3 |
| COMMS — Kommunikation | ⚠️ mittel | E-Mail über TYPO3 Mail API ✅; Web Push ❌ kein nativer Support |
| DOCS — Dokumente | ✅ gut | FAL für Uploads, Workflows, Zeitstempel — alles in Extbase machbar |
| SEPA — Mandate | ⚠️ mittel | Datenpflege + XML-Export per PHP machbar; Lastschriftläufe komplex |
| FINANCE — Beiträge | ⚠️ mittel | Konten und manuelle Zahlungen machbar; automatische Sollstellung per Cronjob |

---

## Detailbewertung je Funktion

Legende: **FE** = TYPO3 Frontend (fe_login + Extbase Plugin + Fluid) · **BE** = TYPO3 Backend (Extbase BE-Modul / List-Modul / Scheduler) · **❌** = Nicht ohne separates Backend realisierbar

### CORE — Administration

| Funktion | FE | BE | ❌ |
|---|---|---|---|
| Vereinsstammdaten, Logo, Saison-Konfiguration | | ✅ Custom Records im List-Modul | |
| Teams und Altersklassen anlegen | | ✅ Custom Records | |
| Diensttypen definieren (Name, Stundenwert, Ersatzbetrag) | | ✅ Custom Records | |
| Hallen mit Adressen und Zeitfenstern pflegen | | ✅ Custom Records | |
| Benutzerrollen zuweisen | | ✅ fe_users/fe_groups im BE | |
| Einladungen versenden | ✅ Token-Link per E-Mail, Extbase | | |
| DSGVO-Einwilligungstexte verwalten | | ✅ Custom Records / TYPO3-Seiten | |
| Systemweite Benachrichtigungsregeln konfigurieren | | ⚠️ Einfache Config machbar; komplexe Regelmaschine aufwändig | |

### MEMBERS — Mitgliederverwaltung

| Funktion | FE | BE | ❌ |
|---|---|---|---|
| Spielerprofile: Stammdaten, Passnummer, Trikotnummer, Position | ✅ Selbstauskunft-Formular | ✅ Custom Table + BE-Liste | |
| Eltern/Kind-Verknüpfung | | ✅ TCA-Relation (MM) | |
| Fahrzeuginformationen hinterlegen | ✅ FE-Formular | ✅ BE-Liste | |
| Lizenzen hochladen mit Ablaufdatum | ✅ FAL-Upload | ✅ BE-Übersicht | |
| Ablaufdatum-Erinnerung | | ✅ TYPO3 Scheduler Cronjob + Mail | |
| Mitgliedsstatus verwalten | | ✅ Select-Feld in BE | |
| Mehrfach-Mannschaftszugehörigkeit | | ✅ MM-Relation | |
| CSV-Import | | ⚠️ Machbar mit PHPSpreadsheet, eigener Extbase-Command | |
| CSV/XLSX-Export | | ⚠️ Machbar mit PHPSpreadsheet | |

### SCHEDULE — Terminplanung

| Funktion | FE | BE | ❌ |
|---|---|---|---|
| Trainingstermine anlegen (manuell) | | ✅ Custom Records | |
| Wiederkehrende Termine (RRULE) | | ⚠️ Keine native RRULE-Logik; manuelle Einzeleinträge oder eigene PHP-Logik | |
| Spielplan per iCal importieren | | ✅ Extbase Scheduler Command + iCal-PHP-Library | |
| Treffpunkte und Abfahrtszeiten | | ✅ Felder am Termin-Record | |
| Zu-/Absagen mit konfigurierbarer Deadline | ✅ Extbase-Controller + eigene Tabelle | | |
| Erinnerungsmail vor Meldefrist | | ✅ Scheduler Cronjob + Mail | |
| Anwesenheitserfassung durch Trainer | ✅ Einfaches FE-Formular | | |
| Anwesenheitserfassung offline-fähig | | | ❌ Braucht PWA/Service Worker — nicht in TYPO3 |
| Kaderaufstellung und Spielbericht | ✅ Extbase-Plugin | | |
| iCal-Export (Google/Apple Calendar) | ✅ PHP iCal-Generierung per Extbase | | |

### DUTIES — Dienste

| Funktion | FE | BE | ❌ |
|---|---|---|---|
| Dienste an Spiele knüpfen | | ✅ Relation-Record | |
| Dienstrollen und Anzahl je Dienst | | ✅ Custom Felder | |
| Automatische Rotationsvorschläge | | ⚠️ PHP-Logik möglich, aber komplex (Dienstkontoabfrage + Sortierung) | |
| Dienstbörse: offene Dienste anzeigen | ✅ Extbase-Plugin | | |
| Dienst selbst eintragen | ✅ Extbase-Controller + DB-Update | | |
| Diensttausch anfragen | ✅ Extbase-Workflow + E-Mail | | |
| Diensttausch bestätigen | ✅ Token-Link oder FE-Formular | | |
| Dienstkonto Soll/Ist-Stand je Familie | ✅ Aggregation aus DB, FE-Ansicht | ✅ BE-Übersicht | |
| Ersatzzahlung erfassen | ✅ FE-Formular | ✅ BE-Eingabe | |
| Export-Report für Kassenwart | | ✅ CSV-Export per BE-Modul | |

### TRANSPORT — Fahrdienst

| Funktion | FE | BE | ❌ |
|---|---|---|---|
| Fahrer anmelden mit Fahrzeug und Sitzplätzen | ✅ FE-Formular | | |
| Kinder zu Fahrzeugen zuweisen (visuell, Drag & Drop) | | | ❌ Interaktive Zuordnungsansicht braucht JS-Framework (Alpine.js wäre Minimalansatz) |
| Kinder zu Fahrzeugen zuweisen (Formular-basiert) | ✅ Select-Formular als Workaround | | |
| Automatische Benachrichtigung mit Fahrerkontakt | | ✅ Scheduler + Mail | |
| Rückfahrt separat planbar | ✅ Zweiter Eintrag oder Flag | | |
| Kilometererfassung | ✅ FE-Eingabeformular | | |

### HALLS — Hallenzeiten

| Funktion | FE | BE | ❌ |
|---|---|---|---|
| Wochenkalender: gebuchte/freie Zeitfenster anzeigen | ✅ Fluid-Kalenderansicht aus DB | | |
| Buchungsanfragen stellen | ✅ FE-Formular | | |
| Genehmigung durch Hallenwart | ✅ FE-Admin-Bereich oder | ✅ BE-Statusfeld | |
| Konflikterkennung bei Doppelbuchungen | | ✅ PHP-Logik bei Speichern (Extbase) | |
| Kosten je Zeitfenster | | ✅ Felder am Buchungs-Record | |

### EVENTS — Turniere & Sonderveranstaltungen

| Funktion | FE | BE | ❌ |
|---|---|---|---|
| Heimturniere anlegen | | ✅ Custom Records | |
| Externe Teams einladen per E-Mail | ✅ FE-Formular + Mail | | |
| Anmeldeformular für externe Teams | ✅ Extbase-Controller | | |
| Automatischer Spielplan-Generator | | | ❌ Gruppen-/K.O.-Algorithmus — PHP möglich, aber komplexer Algorithmus ohne dedizierte Logikschicht sehr aufwändig |
| Live-Tabelle / Anzeigetafel-Modus | | | ❌ Braucht Echtzeit-Updates (WebSocket oder aggressives JS-Polling) |
| Ergebniserfassung | ✅ FE-Formular | ✅ BE-Eingabe | |
| Helferkoordination (via Duties) | ✅ Über DUTIES-FE-Plugin | | |

### COMMS — Kommunikation

| Funktion | FE | BE | ❌ |
|---|---|---|---|
| Ankündigungen erstellen und anzeigen | ✅ Extbase-Plugin | ✅ Custom Records im BE | |
| Gruppen-Nachricht an Teams / alle | | ✅ Massenmail per Scheduler + Mail-API | |
| Lesebestätigung | ✅ FE-Flag in DB beim Aufruf | | |
| E-Mail-Fallback (Benachrichtigung) | | ✅ TYPO3 Mail API + Scheduler | |
| Automatische Systemnachrichten | | ✅ Scheduler Tasks (Erinnerungen, Lizenz, Frist) | |
| Push-Benachrichtigungen (Web Push / VAPID) | | | ❌ Kein nativer Service-Worker-Support in TYPO3; würde eigenes JS + externen Push-Dienst brauchen |

### DOCS — Einverständniserklärungen & Dokumente

| Funktion | FE | BE | ❌ |
|---|---|---|---|
| Dokumentvorlagen anlegen mit Versionierung | | ✅ FAL + Custom Records mit Version-Feld | |
| Zuordnung: Vorlage → Rolle / Altersgruppe | | ✅ TCA-Relation | |
| Pflichtdokument-Kennzeichnung | | ✅ Checkbox-Feld + Extbase-Prüfung | |
| Digitale Unterzeichnung (Checkbox + Zeitstempel + IP) | ✅ Extbase-Controller schreibt in DB | | |
| PDF-Upload (handschriftlich) | ✅ FAL-Upload-Formular | | |
| Anforderung per E-Mail mit Token-Link (ohne Login) | ✅ Extbase + Token-generierung + Mail | | |
| Erinnerungsmail an Ausstehende | | ✅ Scheduler Cronjob | |
| Statusübersicht Ampel je Dokument/Team | ✅ FE-Admin-Plugin | ✅ BE-Modul | |
| Export fehlender Einwilligungen | | ✅ CSV-Export | |
| Unveränderliche Ablage mit Zeitstempel | ✅ Schreibgeschützter DB-Record bei Unterzeichnung | | |
| Widerruf dokumentieren | ✅ FE-Formular + Status-Update | | |
| DSGVO-Anonymisierung bei Austritt | | ✅ BE-Aktion + PHP-Logik | |
| Zugriffsprotokoll | | ✅ Log-Tabelle per Extbase | |

### SEPA — Mandate & Lastschriften

| Funktion | FE | BE | ❌ |
|---|---|---|---|
| Mandat erfassen (IBAN, BIC, Kontoinhaber) | ✅ FE-Formular (⚠️ sensible Daten — TLS + Zugriffsschutz Pflicht) | ✅ BE-Eingabe | |
| Mandats-Status verwalten | | ✅ Select-Feld | |
| PDF-Mandatsformular generieren | | ✅ PHP + mPDF/TCPDF-Library | |
| Mandatsscan hochladen und archivieren | ✅ FAL-Upload | | |
| Erinnerung bei fehlendem/abgelaufenem Mandat | | ✅ Scheduler Cronjob + Mail | |
| Lastschriftlauf erstellen (Vorschau, Gesamtbetrag) | | ✅ BE-Modul mit PHP-Aggregation | |
| Export als SEPA XML pain.008 | | ✅ php-sepa-xml Library einbindbar (Extbase Command) | |
| Buchungsstatus je Einzel-Lastschrift | | ✅ Status-Feld, manuell | |
| Rücklastschrift erfassen + Benachrichtigung | ✅ FE-Meldung | ✅ BE-Eingabe + Mail | |

### FINANCE — Mitgliedsbeiträge & Finanzen

| Funktion | FE | BE | ❌ |
|---|---|---|---|
| Beitragskategorien definieren | | ✅ Custom Records | |
| Mitglied einer Kategorie zuordnen | | ✅ Relation-Feld | |
| Sonderregelungen (Geschwisterrabatt, Ermäßigung) | | ✅ Freitextfeld / Flag | |
| Beitragskonto Soll/Ist je Mitglied | | ✅ Aggregation aus Buchungen-Tabelle | |
| Manuelle Zahlung erfassen | | ✅ BE-Formular | |
| Automatische Sollstellung per Periode | | ✅ Scheduler Cronjob erzeugt Buchungs-Records | |
| Mahnwesen (Erinnerung → Mahnung → Eskalation) | | ✅ Scheduler Cronjob + Mail-Stufen | |
| Dienstgeld-Forderungen übernehmen (DUTIES ↔ FINANCE) | | ✅ PHP-Logik in Extbase Command | |
| Offene-Posten-Liste | ✅ FE für Mitglied | ✅ BE Gesamtübersicht | |
| Jahresübersicht Soll/Ist | | ✅ BE-Report mit PHP-Aggregation | |
| Export CSV / XLSX | | ✅ PHPSpreadsheet in BE-Modul | |

---

## Architektur-Optionen

### Option A — Alles in TYPO3 (Webhosting L reicht)
Alle Module als Extbase-Extensions, Tailwind CSS, fe_login für Auth, Cronjobs für Hintergrundaufgaben.

- **Pro:** Kein zweiter Stack, kein Hosting-Upgrade, bestehende Infrastruktur
- **Con:** Module DUTIES, TRANSPORT, EVENTS, SEPA, FINANCE werden sehr schmerzhaft oder gar nicht realisierbar
- **Realistisch für:** MVP mit CORE + MEMBERS + SCHEDULE + einfaches COMMS

### Option B — TYPO3 (public) + Laravel (intern) auf Space Server
Bestehende TYPO3-Site bleibt, TeamWERK läuft als separate Laravel-App auf demselben vServer.

- **Pro:** Saubere Trennung, voller Feature-Scope laut Konzept möglich, Laravel Queue Worker verfügbar
- **Con:** Hosting-Upgrade (~21 €/Monat), zwei Stacks zu pflegen
- **Realistisch für:** Vollausbau laut Konzept

### Option C — TYPO3 only, Queue-Worker via Cronjob-Workaround
Wie Option A, aber `artisan queue:work --stop-when-empty` jede Minute per Cronjob (nur wenn Laravel eingesetzt wird).

- **Pro:** Kein Hosting-Upgrade nötig
- **Con:** Fragil, Latenz bis zu 1 Minute, nicht für hohe Last geeignet

---

## Empfehlung

Für einen **realistischen MVP** (Login, Mitgliederverwaltung, Termine, Zu-/Absagen, Ankündigungen) ist **Option A** machbar — TYPO3 + fe_login + Extbase, Webhosting L bleibt.

Für den **vollen TeamWERK-Scope** aus dem Konzept (DUTIES, SEPA, FINANCE, Push) führt kein Weg an **Option B** vorbei: Space Server + Laravel neben TYPO3.

---

## Offene Fragen

- Gibt es auf dem Mittwald-Account bereits einen Space Server / vServer?
- Soll TeamWERK in die bestehende team-stuttgart.org-Site eingebettet sein (z.B. `/intern/`) oder als eigene Subdomain (`app.team-stuttgart.org`) laufen?
- Welche Module sind für Phase 1 wirklich zwingend erforderlich?
