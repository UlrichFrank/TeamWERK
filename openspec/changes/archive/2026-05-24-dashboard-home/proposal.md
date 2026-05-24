# Proposal: Dashboard Home Page

**Status:** Proposed  
**Created:** 2026-05-22  
**Owner:** —

---

## Problem

Derzeit navigiert das Logo links oben (AppShell) zu `/mitglieder` als Default-Seite. Das ist für Trainer und Eltern nicht die richtige Entry Point — sie brauchen eine **Übersicht über ihre unmittelbaren Aufgaben und Events diese Woche**, nicht eine Mitgliederliste.

**Current flow:**
- Logo-Click → `/mitglieder` (hardcoded redirect in App.tsx)
- Trainer sieht Liste aller Mitglieder statt: "Was muss ich diese Woche tun?"
- Eltern sehen auch das statt: "Spiele meines Kindes? Dienste zu erfüllen?"

## Solution

Neue **Dashboard-Seite** (`/dashboard` oder `/`) mit:

1. **Accordion-Layout** (Mobile-first: eine Sektion offen, Rest eingeklappt)
2. **Action-Center oben** ("⚡ Diese Woche") — immer offen, rollen-spezifisch
3. **Weitere Sektionen expandierbar:**
   - 📅 Nächste Spiele
   - 🏠 Dein Konto / Team-Stats
   - 👥 Dein Team
   - 🚗 Fahrtgemeinschaften

4. **Rollen-Differentiation:**
   - **Trainer:** Sieht Team-Status, offene Dienste zu verwalten, Spielplan; Dienstkonto (gezählt, kein Ziel)
   - **Elternteil:** Sieht Kind-Status, Dienste zu erfüllen, nächste Spiele; Dienstkonto (Ziel: 5 × Anzahl Kinder)
   - **Spieler:** Dienstkonto (Ziel: 5/Saison)
   - **Admin/Vorstand:** Dashboard-Zugang + Dienstkonto (gezählt, kein Ziel) + Export-Button

5. **Backend-Aggregation:** Neuer `/api/dashboard`-Endpoint liefert rollen-spezifische Daten (Actions, nächste Events, Statistiken)

---

## Scope

### In Scope
- ✅ DashboardPage React-Komponente mit Accordion-UI
- ✅ `/api/dashboard` Endpoint (Go)
- ✅ Action-Berechnung (welche Tasks pro Rolle?)
- ✅ Mobile-optimiert (< 640px = eine Section offen)
- ✅ Logo-Click führt zu Dashboard
- ✅ `duty_types.target_role` — Klassifizierung welche Rolle den Dienst erbringt (Migration)
- ✅ Dienstkonto für **alle** eingeloggten Rollen (count-based; Soll nur für Elternteil + Spieler)
- ✅ DutyAccountsPage (`/dienstkonten`) entfernen + Nav-Eintrag "Dienstkonten" entfernen

### Out of Scope
- ❌ Inline-Actions (alles führt zu Detail-Seiten)
- ❌ Real-time Updates (1-2x weekly Daten OK)
- ❌ Separates Admin/Vorstand-spezifisches Dashboard (Admin+Vorstand sehen standard-Dashboard mit Dienstkonto ohne Ziel)
- ❌ Offline-Caching für Dashboard

---

## User Stories

### Trainer
> Als Trainer öffne ich die App und sehe sofort:
> - "2 Dienste diese Woche nicht besetzt — bitte zuweisen"
> - "Nächstes Spiel SA 10:00 vs. SG Feuerbach"
> - "Fahrzeug: Auswärts braucht noch 2 Plätze"
>
> Ein Click auf die Action führt mich zur jeweiligen Detail-Seite (Dienst-Planung, Spielplan, etc.)

### Elternteil
> Als Elternteil öffne ich die App und sehe:
> - "Dienst 'Getränke' SA 10:00 — wir brauchen dich! [→ Zu Diensten]"
> - "Fahrzeug: Hast du 3 Plätze für Auswärts? [→ Zu meinen Angaben]"
> - "Nächstes Spiel: SA 10:00, mein Kind spielt"
> - "Dienstkonto: 3 von 10 Diensten geleistet" (aufklappbar)
>
> Ein Click auf jede Action bringt mich zur passenden Seite, wo ich handeln kann.

---

## Key Decisions

| Decision | Rationale |
|----------|-----------|
| **Accordion, nicht Tabs** | Mobile: eine Section offen spart Platz. Web: alle sichtbar (scrollbar OK) |
| **"/api/dashboard" statt Frontend-Aggregation** | Backend hat Kontext (Saison, Rollen), kann rollen-spezifische Logic machen. Mobile: weniger Requests |
| **Links statt Inline-Actions** | Klare Separation — Dashboard = Übersicht, Actions = Detail-Seiten. Trainer braucht Kontext. |
| **"Diese Woche" hardcodiert** | 1-2x weekly Updates. "Diese Woche" ist Geschäfts-Rhythmus. Saison-Kontext vom Backend. |
| **Rollen-spezifische Cards** | Trainer/Eltern-Bedarf zu unterschiedlich für einheitliche Cards. Admin/Vorstand: später. |

---

## Design: Accordion Sections

```
WEB (Desktop)                              MOBILE (< 640px)
────────────────────────────────────      ────────────────────
⚡ DIESE WOCHE (offen, rollen-spezifisch) ⚡ DIESE WOCHE ▾
  □ Dienst "Getränke" [→ Dienstbörse]     □ Dienst [→ Börse]
  □ Fahrzeug eintragen [→ Profil]         □ Fahrzeug [→ Profil]
                                          
📅 NÄCHSTE SPIELE      (offen)            📅 SPIELE    ▸
  SA 10:00 vs. SG...                      
  DI 20:00 vs. HC...                      
                                          
🏠 KONTO / TEAM-STATS  (offen)            🏠 KONTO     ▸
  Dienstkonto: Soll 25h, Ist 12h          
  (Nur Eltern; Trainer sieht Team-Stats)  
                                          
👥 DEIN TEAM           (offen)            👥 TEAM      ▸
  18/20 Spieler aktiv, 2 verletzt         
                                          
🚗 FAHRTGEMEINSCHAFTEN (offen)            🚗 FAHRT     ▸
  SA: 4 Plätze gemeldet (Danke!)          
  DI: Brauchen noch 2 Plätze              
```

---

## Implementation Notes

### Backend (`/api/dashboard`)

**Endpoint:** `GET /api/dashboard`  
**Auth:** Authenticated (any role)  
**Response:**
```json
{
  "currentSeason": { "name": "2025/26", "isActive": true },
  "nextGameDate": "2026-05-24T08:00:00Z",
  "actions": [
    {
      "id": "duty-1",
      "type": "duty",
      "text": "Dienst 'Getränke' SA 10:00 — wir brauchen dich!",
      "link": "/dienste",
      "dueDate": "2026-05-24"
    },
    {
      "id": "vehicle-1",
      "type": "vehicle",
      "text": "Fahrzeug: Hast du 3 Plätze für Auswärts DI 20:00?",
      "link": "/profil",
      "actionNeeded": true
    }
  ],
  "nextGames": [
    {
      "id": 1,
      "date": "2026-05-24T10:00:00Z",
      "opponent": "SG Feuerbach",
      "isHome": true,
      "team": "U16 Männlich",
      "slotsCount": 6,
      "slotsFilled": 5
    }
  ],
  "teamStats": {
    "team": "U16 Männlich",
    "activeMembers": 18,
    "totalMembers": 20,
    "injuredCount": 2
  },
  "dutyAccount": {
    "season": "2025/26",
    "ist": 3,
    "soll": 10,
    "children": 2
  },
  "vehicleInfo": {
    "seats": 4,
    "upToDate": true
  }
}
```

**Action-Logik (Backend zu berechnen):**
- Für TRAINER und VORSTAND: "X offene Dienste diese Woche" (Slots ohne Zuweisungen in seinem Team)
- Für ELTERNTEIL: "Dienst X — wir brauchen dich!" (Slot mit `duty_types.target_role='elternteil'`, passt zu child, noch nicht zugewiesen)
- Für SPIELER: Offene Dienste mit `target_role='spieler'` in seinen Teams
- Fahrzeug: Für nächsten Auswärts-Spieltag, checke Fahrtgemeinschafts-Status

### Frontend

**File:** `web/src/pages/DashboardPage.tsx`

- Fetch `/api/dashboard`
- Render Accordion (Desktop: alle offen, Mobile: nur "Diese Woche")
- Rollen-spezifische Sektion-Inhalte (z.B. "🏠 KONTO" vs. "🏠 TEAM-STATS")
- Links führen zu bestehenden Seiten (Dienstbörse, Spielplan, Profil, etc.)

**Update:** `App.tsx`
- `Route index` zu DashboardPage statt `/mitglieder`-Redirect

**Update:** `AppShell.tsx`
- Logo-Click zur Dashboard (default, bereits enthalten mit index-Route)

---

## Risks & Unknowns

| Risk | Mitigation |
|------|-----------|
| "Diese Woche"-Logik zu simpel? | Action-Berechnung iterativ testen; Trainer/Eltern-Feedback. |
| Mobile: Zu viele Sections? | Priorität: "DIESE WOCHE" oben, Rest accordion. Kann später trimmen. |
| Saison-Abhängigkeit | Wenn keine aktive Saison: leeres Dashboard mit Hinweis (Admin → Saison aktivieren) |
| Fahrtgemeinschafts-Matching | Komplex: Spieltag ↔ Fahrtgemeinschaft ↔ Fahrzeuginfo. Backend-Logik zur Prüfung nötig. |

---

## Success Criteria

- ✅ Dashboard lädt in < 1s (Mobile)
- ✅ Logo führt zum Dashboard
- ✅ "DIESE WOCHE" zeigt mindestens 1 relevante Action (für aktive Saison)
- ✅ Alle Links funktionieren + führen zur richtigen Seite
- ✅ Accordion funktioniiert (Mobile: 1 offen, Web: alle)
- ✅ Rollen-Differenzierung funktioniert (Trainer sieht anderes als Eltern)
- ✅ Tests: Unit-Tests für Action-Berechnung (Backend)

---

## Timeline Estimate

- Backend `/api/dashboard` Endpoint: **3–4h**
- Frontend Dashboard-Komponente: **2–3h**
- Testing & Mobile-Polish: **1–2h**
- **Total: ~6–9h**

---

## Deployment Notes

- Migration `006_duty_types_target_role` erforderlich (neues `target_role`-Feld auf `duty_types`)
- `/api/dashboard` wird direkt nach Release verfügbar
- DutyAccountsPage (`/dienstkonten`) wird entfernt — keine Breaking Change für externe Clients
- Dashboard-Seite ist eine neue Route, keine Breaking Changes
- Logo-Click ändert sich automatisch (Index-Route Redirect)

---

## Follow-ups (Out of Scope)

- Admin/Vorstand Dashboard (späterer Task)
- "Anheften"-Feature (Favorite Sections merken)
- Notifications/Badges (z.B. rote Badge auf "DIESE WOCHE" wenn ungelesen)
