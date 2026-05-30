## Context

Das Change-Draft-System speichert Änderungsanfragen von Mitgliedern in `member_change_drafts` mit einem `field_name`-Schlüssel. Persönliche Daten nutzen bereits einen kombinierten Draft (`field_name='profil'`, enthält name + adresse). Bankdaten hingegen haben zwei separate Drafts (`iban` und `account_holder`), die unabhängig akzeptiert/abgelehnt werden können — was semantisch falsch ist, weil IBAN und Kontoinhaber immer zusammen validiert werden sollten.

Die Mitgliederliste zeigt aktuell nur ein generisches Boolean-Flag (`has_pending_drafts`), das keine Unterscheidung nach Typ erlaubt.

## Goals / Non-Goals

**Goals:**
- Bankdaten atomar behandeln: IBAN und Kontoinhaber als ein einziger `bankdaten`-Draft
- Admin-Workflow vereinfachen: eine Genehmigung für alle Bankdaten
- Mitgliederliste mit aussagekräftigen Typ-Indikatoren (Persönliche Daten vs. Bankdaten)
- Backend sauber abgrenzen: `allowedFields` nur noch `profil` und `bankdaten`

**Non-Goals:**
- Automatische Migration bestehender `iban`/`account_holder`-Drafts in der DB
- Granulare Einzelfeld-Genehmigung innerhalb von Bankdaten
- Änderung der `profil`-Draft-Struktur (bleibt `{first_name, last_name, street, zip, city}`)

## Decisions

**D1: Neuer `bankdaten`-Draft statt Umbenennung**

`field_name='bankdaten'` mit `new_value: {iban: string, account_holder: string}`. Alternativ hätte man `iban` behalten und `account_holder` darin einbetten können — aber ein eigener semantischer Name ist klarer und konsistent mit `profil`.

**D2: Keine DB-Migration für Altdaten**

Bestehende `iban`/`account_holder`-Drafts bleiben in der DB, werden aber nicht mehr vom Frontend erzeugt. Der Admin sieht sie noch in der Detail-Ansicht und kann sie ablehnen. Nach Ablehnung verschwinden sie. Kein Migrationsscript nötig — der Impact ist minimal (wenige aktive Drafts in einer internen App).

**D3: Zwei getrennte Boolean-Felder in der Mitgliederliste**

`has_pending_profil_draft` und `has_pending_bank_draft` statt einer Liste von field_names. Einfacher zu konsumieren im Frontend, kein Breaking Change an der Aggregationslogik.

**D4: Backend `allowedFields` bereinigen**

`iban` und `account_holder` aus `allowedFields` entfernen. Bestehende Drafts dieser Typen können noch akzeptiert/abgelehnt werden (der Accept/Reject-Handler prüft `field_name` nicht gegen `allowedFields`), aber neue können nicht mehr erzeugt werden.

## Risks / Trade-offs

- **Altdaten-Drafts** → Mitigation: Admin-Hinweis in der Detail-Ansicht; alte `iban`/`account_holder`-Drafts erscheinen noch in der Karte, können abgelehnt werden. `has_pending_bank_draft` filtert nur auf `bankdaten`, nicht auf alte Typen — d.h. alte Drafts erzeugen kein Icon in der Liste (akzeptabler temporärer Zustand).
- **Frontend zeigt alten Draft-Typ nicht im kombinierten Icon** → Mitigation: In `MemberKontaktTab` werden weiterhin alle Draft-Typen (iban, account_holder, bankdaten) gerendert, solange sie in der DB existieren.

## Migration Plan

1. Backend deployen (neuer `bankdaten`-Handler, neue Listenfelder)
2. Frontend deployen (ProfileBankTab, MemberKontaktTab, MembersPage)
3. Bestehende `iban`/`account_holder`-Drafts: Admin lehnt sie manuell ab (oder sie werden ignoriert bis Mitglied neu einreicht)
4. Rollback: Kein DB-Schema-Change → einfaches Revert des Deployments
