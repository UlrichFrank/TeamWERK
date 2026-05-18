# Team Stuttgart TeamWERK (Where Engagement Really Klicks)
## Produktkonzept & Architekturübersicht

---

## 1. Ziele

### 1.1 Primäre Ziele

**Vereinsverwaltung vereinfachen**
Alle wiederkehrenden administrativen Aufgaben eines Handballvereins — Dienste, Hallenzeiten, Fahrdienste — an einem zentralen Ort bündeln, statt auf Excel-Tabellen, WhatsApp-Gruppen und E-Mail-Ketten angewiesen zu sein.

**Dienst- und Pflichtstunden transparent machen**
Jede Familie sieht jederzeit ihren Dienststand (Soll vs. Ist). Offene Dienste sind für alle sichtbar, die Zuteilung ist nachvollziehbar und fair.

**Kommunikation strukturieren**
Ankündigungen, Erinnerungen und Rückmeldungen laufen über das System — nicht über private Messenger-Kanäle, die einzelne Personen ausschließen oder überlasten.

**Terminplanung & Rückmeldungen digitalisieren**
Trainings- und Spieltermine werden zentral verwaltet. Zu- und Absagen von Spielern und Eltern erfolgen strukturiert mit Deadline und automatischer Erinnerung.

**Rollentrennung & Datenschutz sicherstellen**
Eltern sehen nur Daten ihrer eigenen Kinder. Trainer sehen nur ihre Mannschaft. Sensible Daten (Gesundheit, Finanzen) sind auf berechtigte Rollen beschränkt.

### 1.2 Sekundäre Ziele

- Aufwand für Teamleiter und Jugendleiter durch Automatisierung reduzieren
- Lizenz- und Dokumentenablauf automatisch überwachen und erinnern
- Hallenzeiten-Konflikte frühzeitig erkennen und vermeiden
- Als PWA ohne App-Store-Installation nutzbar sein

---

## 2. Nicht-Ziele

**Kein vollständiges Vereinsverwaltungssystem (ERP)**
Buchhaltung, Steuern oder Jahresabschlüsse sind nicht Kern des Systems. Mitgliedsbeiträge, SEPA-Mandate und Einverständniserklärungen werden jedoch aktiv verwaltet. Das System ist eine Ergänzung zu, kein Ersatz für Buchhaltungssoftware wie SEWOBE oder ClubDesk.

**Kein Schiedsrichterverwaltungssystem**
Die Verwaltung und Einteilung externer Schiedsrichter durch den Verband ist nicht Gegenstand dieser Anwendung. Lediglich vereinsinterne Schiedsrichter-Dienste (Zeitnehmer, Sekretär) werden abgebildet.

**Keine Echtzeit-Spielstatistiken**
Live-Ticker, detaillierte Spielerstatistiken oder Videoanalysen sind nicht vorgesehen.

**Keine Verbandsanbindung**
Eine direkte Zwei-Wege-Schnittstelle zum HVW/DHB (z. B. automatische Spielmeldungen, Pass-Beantragung) ist nicht Ziel dieser Phase. Nur Import von Spielplänen per iCal/CSV.

**Kein öffentliches Vereinsportal**
Die Anwendung ist eine interne Plattform für Mitglieder. Eine öffentliche Vereinswebsite mit News, Sponsoren und Kontaktformular ist nicht Teil des Systems.

**Keine native Mobile App**
Es wird keine native iOS- oder Android-App entwickelt. Die Anwendung ist als PWA (Progressive Web App) für mobile Browser optimiert.

---

## 3. Funktionsumfang

### Modul CORE — Administration
- Vereinsstammdaten, Logo und Saison-Konfiguration
- Teams und Altersklassen anlegen und verwalten
- Diensttypen definieren (Name, Pflicht-Stundenwert, Ersatzbetrag)
- Hallen mit Adressen, Kapazitäten und Zeitfenstern pflegen
- Benutzerrollen zuweisen und Einladungen versenden
- DSGVO-Einwilligungstexte verwalten
- Systemweite Benachrichtigungsregeln konfigurieren

### Modul MEMBERS — Mitgliederverwaltung
- Spielerprofile: Stammdaten, Passnummer, Trikotnummer, Position
- Eltern/Erziehungsberechtigte mit Spieler verknüpfen
- Fahrzeuginformationen für den Fahrdienst hinterlegen
- Lizenzen und Dokumente hochladen mit Ablaufdatum-Erinnerung
- Mitgliedsstatus verwalten (aktiv, verletzt, pausiert, ausgetreten)
- Zugehörigkeit zu Mannschaften (auch mehrfach möglich)
- CSV/XLSX-Import und -Export

### Modul SCHEDULE — Terminplanung
- Wiederkehrende Trainingstermine mit Hallenzuweisung anlegen
- Spielplan manuell erfassen oder per iCal importieren (HVW)
- Treffpunkte und Abfahrtszeiten für Auswärtsspiele definieren
- Zu- und Absagen mit konfigurierbarer Deadline einsammeln
- Erinnerungsnotifikationen vor Ablauf der Meldefrist
- Anwesenheitserfassung durch Trainer (auch offline-fähig)
- Kaderaufstellung und Spielbericht (Ergebnis, Tore, Karten, Zeitstrafen)
- Kalenderexport als iCal (Google Calendar, Apple Calendar)

### Modul DUTIES — Dienste
- Dienste an Spiele und Events knüpfen mit definierten Rollen und Anzahl
- Automatische Rotationsvorschläge basierend auf Dienstkontostand
- Dienstbörse: offene Dienste einsehen und selbst eintragen
- Diensttausch zwischen Familien anfragen und bestätigen
- Dienstkonto je Familie: Soll/Ist-Stand, Verlauf
- Ersatzzahlung als Alternative erfassen
- Export-Report für Kassenwart (offene Konten, Saison-Abschluss)

### Modul TRANSPORT — Fahrdienst
- Fahrer melden sich mit Fahrzeug und freien Sitzplätzen an
- Zuweisung von Kindern zu Fahrzeugen (visuelle Zuordnungsansicht)
- Automatische Benachrichtigung mit Fahrerkontakt an zugewiesene Familien
- Rückfahrt separat planbar
- Kilometererfassung je Fahrt (Aufwandsentschädigung)

### Modul HALLS — Hallenzeiten
- Wochenkalender je Halle mit gebuchten und freien Zeitfenstern
- Buchungsanfragen stellen und durch Hallenwart genehmigen
- Konflikterkennung bei Doppelbuchungen
- Kosten je Zeitfenster für interne Abrechnung

### Modul EVENTS — Turniere & Sonderveranstaltungen
- Heimturniere anlegen mit Datum, Halle und Teilnehmerzahl
- Externe Teams einladen und anmelden
- Automatischer Spielplan-Generator (Gruppen und K.O.-Runde)
- Ergebniserfassung und Live-Tabelle (Anzeigetafel-Modus)
- Helferkoordination verknüpft mit Dienste-Modul

### Modul COMMS — Kommunikation
- Gruppen-Nachrichten an Teams, Altersklassen oder alle Mitglieder
- Ankündigungen mit optionaler Lesebestätigung
- Push-Benachrichtigungen (Web Push) und E-Mail-Fallback
- Automatische Systemnachrichten (Dienst-Erinnerung, Lizenz läuft ab, Rückmeldefrist)

### Modul DOCS — Einverständniserklärungen & Dokumente

**Vorlagen verwalten (Admin)**
- Dokumentvorlagen anlegen mit Versionierung (z. B. „Foto-Einwilligung 2025", „Datenschutzerklärung v3")
- Vorlagen als PDF hinterlegen oder als strukturiertes Formular im System definieren
- Zuordnung: welche Vorlage gilt für welche Rolle / Altersgruppe / Mannschaft
- Pflichtdokumente kennzeichnen: Mitgliedschaft oder Spielberechtigung erst nach Unterzeichnung aktiv

**Einholung & Unterzeichnung**
- Digitale Unterzeichnung direkt im System (Checkbox mit Zeitstempel und IP-Protokollierung)
- Alternativ: PDF-Upload des handschriftlich signierten Dokuments
- Anforderung per E-Mail mit personalisierten Links (kein Login erforderlich für Einmalunterzeichnung)
- Erinnerungsmail an ausstehende Unterzeichner (konfigurierbare Intervalle)

**Statusübersicht (Admin / Teamleiter)**
- Pro Mitglied: welche Dokumente liegen vor, welche fehlen, welche sind abgelaufen
- Gesamtübersicht: Ampel-Status je Dokument und Team
- Ablaufdatum-Tracking (z. B. jährlich erneuerte Einwilligungen)
- Export-Liste fehlender Einwilligungen

**Archiv & DSGVO-Konformität**
- Unveränderliche Ablage aller unterzeichneten Versionen mit Zeitstempel
- Widerruf einer Einwilligung dokumentieren (Datum, Konsequenzen vermerken)
- Löschanfragen: DSGVO-konforme Anonymisierung bei Vereinsaustritt
- Zugriffsprotokoll: wer hat wann welches Dokument eingesehen

---

### Modul SEPA — Mandate & Lastschriften

**Mandatsverwaltung**
- SEPA-Lastschriftmandat je Mitglied / Familie erfassen
  - Gläubiger-ID, Mandatsreferenz (eindeutig, automatisch generiert)
  - IBAN, BIC, Kontoinhaber, Unterzeichnungsdatum
- Mandat-Status: aktiv / widerrufen / abgelaufen
- PDF-Mandatsformular generieren und zum Unterschreiben versenden
- Unterzeichnetes Mandat als Scan hochladen und archivieren
- Erinnerung bei fehlendem oder abgelaufenem Mandat

**Lastschriftläufe**
- Lastschriftlauf für Mitgliedsbeiträge erstellen (manuell oder automatisch je Fälligkeit)
- Lastschriftlauf für Dienstgeld-Nachzahlungen aus Modul DUTIES
- Vorschau: welche Mitglieder sind enthalten, Gesamtbetrag, offene Mandate
- Export als SEPA XML pain.008 (kompatibel mit allen deutschen Banken)
- Status je Buchung: ausstehend / eingereicht / gebucht / Rücklastschrift
- Rücklastschrift erfassen und Mitglied benachrichtigen

---

### Modul FINANCE — Mitgliedsbeiträge & Finanzen

**Beitragsmodelle**
- Beitragskategorien definieren (Jugendlicher, Erwachsener, Passiv, Familie, Ermäßigt)
- Beitrag je Kategorie und Saison / Jahr festlegen
- Mitglied einer Kategorie zuordnen (mit Gültigkeitsdatum)
- Sonderregelungen: Geschwisterrabatt, Ehrenmitglied, soziale Ermäßigung (intern vermerkt)

**Beitragsverwaltung**
- Beitragskonto je Mitglied: Soll-Beträge je Periode, eingegangene Zahlungen, Saldo
- Manuelle Zahlung erfassen (Barzahlung, Überweisung)
- Automatische Sollstellung bei Periodenanfang (monatlich / quartalsweise / jährlich)
- Mahnwesen: Zahlungserinnerung nach konfigurierbaren Fristen (1. Erinnerung, Mahnung, Vereinsleitung)
- Beitragsbefreiung oder -stundung dokumentieren

**Dienstgeld-Integration**
- Offene Dienst-Ersatzzahlungen aus Modul DUTIES als Forderung übernehmen
- Gemeinsamer Lastschriftlauf für Beitrag + Dienstgeld möglich

**Kassenwart-Auswertungen**
- Offene Posten je Mitglied und gesamt
- Eingänge im Zeitraum (Monat / Quartal / Saison)
- Beitragsrückstände-Liste (sortierbar, exportierbar)
- Jahresübersicht Soll / Ist / Differenz
- Export als CSV / XLSX für Buchhaltungsprogramm

---

## 4. Architekturübersicht

### 4.1 Systemübersicht

```
┌─────────────────────────────────────────────────────────┐
│                      Browser / PWA                      │
│              (Vue 3 + Tailwind CSS)                     │
└──────────────────────┬──────────────────────────────────┘
                       │ HTTPS / REST + WebSocket
┌──────────────────────▼──────────────────────────────────┐
│                   API Server                            │
│            (Laravel 11 / PHP 8.3)                       │
│                                                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐               │
│  │  CORE    │  │ SCHEDULE │  │  DUTIES  │  ...Module    │
│  └──────────┘  └──────────┘  └──────────┘               │
│                                                         │
│  Policies & Gates (Rollenbasierte Zugriffskontrolle)    │
└───────┬──────────────┬────────────────┬─────────────────┘
        │              │                │
┌───────▼──────┐ ┌─────▼───────┐ ┌──────▼──────┐
│  PostgreSQL  │ │   Redis     │ │   Storage   │
│  (Daten)     │ │  (Cache,    │ │  (Dokumente,│
│              │ │   Queue,    │ │   Bilder)   │
│              │ │   Sessions) │ │             │
└──────────────┘ └─────────────┘ └─────────────┘
```

### 4.2 Datenschichten

```
Frontend (Vue 3)
  └── Pinia Store (lokaler State)
  └── API Client (Axios / Fetch)

Backend (Laravel)
  └── Routes → Controller → Service → Repository
  └── Eloquent ORM → PostgreSQL
  └── Jobs & Queues → Redis (Benachrichtigungen, Mails)
  └── Policies → Rollenbasierte Zugriffsprüfung
  └── Events & Listeners (Domänenereignisse)
```

### 4.3 Mandantenfähigkeit (Multi-Tenancy)

Jeder Verein ist ein **Tenant**. Die Isolation erfolgt auf Datenbankebene über eine `club_id` auf allen relevanten Tabellen. Für spätere Skalierung kann auf Schema-Trennung (PostgreSQL Schemas pro Verein) umgestellt werden.

---

## 5. Technologie-Stack

### 5.1 Backend

| Komponente | Technologie | Begründung |
|---|---|---|
| Framework | **Laravel 11** (PHP 8.3) | Weit verbreitet, hervorragendes Ökosystem, schnelle Entwicklung |
| Datenbank | **PostgreSQL 16** | ACID-konform, JSON-Support, starke Volltextsuche |
| Cache / Queue | **Redis** | Session-Store, Job-Queue für Mails & Push, Echtzeit-Events |
| Auth | **Laravel Sanctum** | SPA-Auth mit CSRF-Schutz, API-Token für mobile Clients |
| Berechtigungen | **Spatie Laravel Permission** | Bewährtes Rollen- & Permissions-Paket |
| Mailing | **Laravel Mail + Mailpit** (Dev) / **Postmark** (Prod) | Transaktionsmails mit Tracking |
| Push | **Web Push (VAPID)** via `laravel-notification-channels/webpush` | Browserbasierte Push-Notifikationen ohne App |
| Datei-Storage | **Laravel Storage + S3-kompatibler Speicher** | Dokumente, Bilder, Exporte |
| Task Scheduling | **Laravel Scheduler** | Erinnerungen, Lizenz-Checks, Dienst-Rotationen |

### 5.2 Frontend

| Komponente | Technologie | Begründung |
|---|---|---|
| Framework | **Vue 3** (Composition API) | Reaktiv, komponentenbasiert, gute TypeScript-Integration |
| Build | **Vite** | Schnelle HMR, modernes Tooling |
| State Management | **Pinia** | Offizieller Vue-Store, einfach und typsicher |
| UI-Bibliothek | **Tailwind CSS + shadcn-vue** | Utility-first, konsistentes Design ohne Overhead |
| Kalender | **FullCalendar** (Vue-Wrapper) | Ausgereifte Kalender-Komponente |
| Formulare | **VeeValidate + Zod** | Typsichere Formularvalidierung |
| HTTP Client | **Axios** | Interceptors für Auth-Header und Error-Handling |
| Routing | **Vue Router 4** | Client-seitiges Routing, Lazy Loading per Modul |
| PWA | **Vite PWA Plugin** | Service Worker, App-Manifest, Offline-Cache |

### 5.3 Entwicklung & Betrieb

| Bereich | Technologie |
|---|---|
| Lokale Entwicklung | **Laravel Herd** oder **Docker (Laravel Sail)** |
| Testing Backend | **PestPHP** (Unit + Feature Tests) |
| Testing Frontend | **Vitest** + **Vue Testing Library** |
| API-Dokumentation | **Scribe** (automatisch aus Laravel-Routes) |
| CI/CD | **GitHub Actions** |
| Hosting | **Railway / Render** (einfach) oder **Hetzner VPS + Coolify** (günstig, selbst gehostet) |
| Monitoring | **Sentry** (Fehler) + **Laravel Telescope** (Dev) |

---

## 6. Rollenmodell

```
Vereinsadmin
  Vollzugriff auf alle Module und alle Teams

Jugendleiter / Abteilungsleiter
  Alle Teams seiner Abteilung, Dienst- und Hallenplanung

Teamleiter (pro Mannschaft)
  Mitglieder seines Teams, Dienste, Fahrdienst, Termine

Trainer (pro Mannschaft)
  Termine seines Teams, Anwesenheitserfassung, Spielbericht

Elternteil
  Eigene Kinder, Zu-/Absagen, Dienstbörse, Dienstkonto

Spieler (ab ~14 Jahren)
  Eigenes Profil, Termine, Zu-/Absagen
```

---

## 7. Phasenplan (MVP → Vollausbau)

### Phase 1 — MVP (Kern)
- [ ] Authentifizierung & Rollenverwaltung (CORE)
- [ ] Mitgliederverwaltung (MEMBERS)
- [ ] Trainings- und Spieltermine mit Zu-/Absagen (SCHEDULE)
- [ ] Dienste-Planung und Dienstkonten (DUTIES)
- [ ] Fahrdienst-Verwaltung (TRANSPORT)
- [ ] Basis-Kommunikation: Ankündigungen & E-Mail-Benachrichtigungen (COMMS)

### Phase 2 — Betrieb
- [ ] Hallenzeiten-Kalender (HALLS)
- [ ] Push-Benachrichtigungen & PWA-Manifest
- [ ] Einverständniserklärungen: Vorlagen, digitale Unterzeichnung, Statusübersicht (DOCS)
- [ ] SEPA-Mandate erfassen, PDF-Generierung, XML-Export pain.008 (SEPA)
- [ ] Mitgliedsbeiträge: Kategorien, Konten, manuelle Zahlungserfassung (FINANCE)
- [ ] Mahnwesen & Beitragsrückstände-Report (FINANCE)
- [ ] Dienstgeld-Integration in Lastschriftlauf (DUTIES ↔ SEPA)
- [ ] Lizenz-Ablauf-Erinnerungen

### Phase 3 — Vollausbau
- [ ] Turnier-Modul (EVENTS)
- [ ] Spielbericht & Statistiken (SCHEDULE)
- [ ] Diensttauschbörse (DUTIES)
- [ ] Rücklastschrift-Verwaltung & erweitertes Mahnwesen (SEPA / FINANCE)
- [ ] DSGVO-Widerruf & Löschanfragen-Workflow (DOCS)
- [ ] Kassenwart-Dashboard: Jahresübersicht, CSV/XLSX-Export (FINANCE)
- [ ] Mehrsprachigkeit (i18n)
