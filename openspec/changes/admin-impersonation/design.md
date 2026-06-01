## Context

TeamWERK verwendet JWT Access Tokens (15 min, HS256, in-memory) kombiniert mit einem opaquen Refresh Token (7 Tage, HttpOnly Cookie). Die Middleware liest Claims aus dem Bearer-Token und entscheidet über Zugriff. Das Rollen-Modell ist zweiteilig: `users.role` (`admin`|`standard`) plus `member_club_functions` (mehrwertig: `spieler`, `trainer`, `vorstand`, `vorstand_beisitzer`). Beide Dimensionen landen im JWT (`role`, `club_functions`, `is_parent`) und steuern sowohl Backend-Guards als auch Frontend-UI.

## Goals / Non-Goals

**Goals:**
- Admin kann in der Nutzerverwaltung einen beliebigen User auswählen und dessen Session-Sicht übernehmen
- Alle API-Calls laufen authentisch mit dem Ziel-JWT (nicht nur UI-Simulation)
- Rückkehr zur Admin-Session ohne separaten Logout/Login-Zyklus
- Klare visuelle Kennzeichnung der aktiven Impersonation

**Non-Goals:**
- Impersonation von anderen Admins (nur Standard-User)
- Audit-Log der Aktionen während Impersonation
- Persistenz der Impersonation über Page-Reload hinaus
- Token-Verlängerung während Impersonation (15 min sind ausreichend für Testzwecke)

## Decisions

### D1: Access-Token-Swap statt Session-Duplizierung

**Entscheidung:** Der Impersonation-Endpoint gibt ein neues, gültig signiertes JWT zurück, das die Claims des Ziel-Users trägt. Der bestehende Refresh-Cookie des Admins bleibt unverändert.

**Rationale:** Der Refresh-Cookie ist der "Anker" zur Admin-Session. Da er HttpOnly und nie ausgetauscht wird, ist "Beenden" = normaler `/auth/refresh` — kein eigener Stop-Endpoint, kein gespeicherter Original-Token nötig. Einfachstmögliche Implementierung ohne neuen DB-State.

**Alternativen:** Refresh-Cookie tauschen + separaten Restore-Endpoint → unnötige Komplexität. Original-Token in sessionStorage sichern → unnötig, da Cookie bereits als Restore-Mechanismus fungiert.

### D2: Gleiche TTL wie normale Access Tokens (15 min)

**Entscheidung:** Das Impersonation-JWT bekommt die Standard-TTL von 15 Minuten.

**Rationale:** Testsessions sind kurz. Bei Ablauf greift der Auto-Refresh-Interceptor, der den unverändertem Admin-Cookie verwendet und damit automatisch zur Admin-Session zurückkehrt — ein akzeptables Verhalten.

**Bekannte Einschränkung:** Nach automatischem Refresh bleibt der `impersonating`-State im AuthContext stehen, bis der Admin manuell auf "Beenden" klickt. Nicht sicherheitsrelevant, nur kosmetisch.

### D3: Kein `impersonated_by`-Claim im JWT

**Entscheidung:** Das ausgestellte JWT enthält keine Markierung, dass es ein Impersonation-Token ist.

**Rationale:** Die Middleware muss nicht wissen, ob ein Token aus Impersonation stammt. Admin-only Endpoints sind durch `RequireRole("admin")` geschützt — ein Impersonation-Token mit `role=standard` kommt dort ohnehin nicht durch. Ein zusätzliches Claim würde nur `ParseAccessToken` verkomplizieren.

### D4: Impersonation nur auf Standard-User beschränken

**Entscheidung:** `Impersonate`-Handler lehnt ab, wenn der Ziel-User `role=admin` hat.

**Rationale:** Privilege-Eskalation vermeiden. Ein Admin kann sich nicht in einen anderen Admin "verstecken". Hauptanwendungsfall (Vereinsfunktionen testen) benötigt nur Standard-User.

## Risks / Trade-offs

- **Token-Expiry-Desync** → Nach Auto-Refresh sieht der Admin wieder Admin-Content, aber Banner zeigt weiterhin Impersonation. Mitigation: Akzeptierter Kompromiss für Testzweck; kann später durch einen `onRefresh`-Callback in `api.ts` behoben werden.
- **Kein Audit-Trail** → Admins könnten im Namen von Usern Aktionen ausführen, ohne dass dies protokolliert wird. Mitigation: Bewusst außerhalb des Scope — das Feature ist ausschließlich für lesende UI-Tests gedacht; destruktive Aktionen sind über normale Rollenprüfung eingeschränkt.

## Migration Plan

Keine DB-Migration nötig. Deployment: normales `make deploy` genügt.
