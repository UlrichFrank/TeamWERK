# Validierung: deploy-update-reliability

Schrittweise Prüfung nach dem Deploy. Reihenfolge einhalten.

> Domain noch ausstehend — falls `internal.team-stuttgart.org` noch nicht
> aufgelöst wird, statt der URL `https://217.160.118.39` verwenden (Zertifikat
> selbstsigniert → `curl -k`).

## 0. Deploy

```bash
make deploy   # baut Frontend + Binary, rsync auf VPS, migrate up, systemctl restart
```

## 1. Backend-Header (sofort nach Deploy, kein PWA nötig)

```bash
# index.html → immer revalidieren, ETag vorhanden
curl -sI https://internal.team-stuttgart.org/ | grep -iE 'cache-control|etag'
# erwartet: Cache-Control: no-cache, must-revalidate  +  ETag: "<hash>-xxxxxxxx"

# sw.js → ebenso
curl -sI https://internal.team-stuttgart.org/sw.js | grep -iE 'cache-control|etag'

# hashed Asset → immutable (Dateinamen vorher aus den DevTools/Network kopieren)
curl -sI https://internal.team-stuttgart.org/assets/index-XXXX.js | grep -i cache-control
# erwartet: Cache-Control: public, max-age=31536000, immutable

# 304-Revalidierung
ETAG=$(curl -sI https://internal.team-stuttgart.org/ | grep -i etag | sed 's/^[Ee]tag: //' | tr -d '\r')
curl -sI -H "If-None-Match: $ETAG" https://internal.team-stuttgart.org/ | head -1
# erwartet: HTTP/2 304
```

> Hinweis: `curl -I` (HEAD) liefert beim SPA-Handler **405** — Header trotzdem
> sichtbar mit `curl -sI` ist ok, aber falls 405 stört, GET nehmen:
> `curl -s -o /dev/null -D - https://internal.team-stuttgart.org/`.

## 2. Service Worker / Precache (Chrome DevTools → Application)

1. Seite einmal normal laden (eingeloggt).
2. **Application → Cache Storage**:
   - `app-shell` existiert nach der ersten Navigation und enthält **1 Eintrag** (die Shell).
   - `workbox-precache-*` enthält JS/CSS/Icons, **keine** `.html`-Datei.
3. **Application → Service Workers**: aktiver SW zeigt den neuen Stand.

## 3. Update-Pfad „Jetzt laden" (Chrome Desktop)

1. App mit **alter** Version offen lassen.
2. `make deploy` (oder es wurde gerade deployed).
3. Innerhalb ~30 s erscheint der Banner „Neue Version verfügbar".
4. „Jetzt laden" klicken → Seite lädt, neuer Commit-Hash im Sidebar-Footer.

## 4. iOS-PWA Cold-Start (Vorstand-Tester, ohne Banner)

1. Installierte PWA komplett schließen (aus dem App-Switcher wischen).
2. Nach dem Deploy wieder vom Homescreen öffnen.
3. **Erwartung:** neuer Hash im Sidebar-Footer **ohne** Logout und **ohne**
   Banner-Klick — die NetworkFirst-Shell holt die frische `index.html`.

## 5. Offline-Probe

1. App einmal erfolgreich laden (füllt `app-shell`).
2. DevTools → Network → **Offline**, dann neu laden.
3. **Erwartung:** App startet aus `app-shell` (gewohnte Shell oder Offline-Hinweis),
   kein harter Browser-Fehler.

## Wenn etwas hängt

- Header fehlen → prüfen ob wirklich der neue Binary läuft
  (`ssh vServer 'systemctl status teamwerk'`).
- Alte Shell bleibt beim **ersten** Rollout einmalig (alter SW kennt die
  NetworkFirst-Regel noch nicht) — self-healing nach einem Reload-Zyklus.
- Notnagel zum Erzwingen: im Banner „Jetzt laden" — löscht Precache,
  `app-shell`, `api-cache` und lädt neu.

## Rollback

`git revert <commit>` der vier Commits (`505c988`, `7bdfa4e`, `ea2b857`,
`8bd604c`) + `make deploy`. Backend-Header wirken sofort; die SW-Änderung
braucht einen weiteren Reload-Zyklus.
