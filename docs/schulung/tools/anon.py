"""Anonymisiert eine Kopie der TeamWERK-DB für Schulungs-Screenshots.

Aufruf: python3 anon.py <demo.db> <ids.json>

Struktur (Saisons, Teams, Kader, Spiele, Dienste, Zahlen) bleibt echt, Personen
werden ersetzt: Namen, E-Mails, Telefon, Adressen, Geburtstage (Jahr bleibt),
Chat-Texte, Kommentare, Gründe. Alle Logins bekommen das Passwort
"Schulung2026!"; zwei Personas bekommen feste Adressen:

    vorstand@beispiel.de   (Nutzer-ID aus SCHULUNG_VORSTAND, Default 10)
    trainer@beispiel.de    (Nutzer-ID aus SCHULUNG_TRAINER,  Default 11)

Schreibt <ids.json> mit den IDs, die shots.txt als Platzhalter nutzt, und bricht
mit Exit 1 ab, wenn danach noch ein echter Voll-Name in einem Freitext steht.
"""
import json
import os
import re
import sqlite3
import sys

DB, IDS = sys.argv[1], sys.argv[2]
if "/var/lib/teamwerk" in os.path.abspath(DB):
    sys.exit("anon.py: verweigert — Pfad zeigt auf das Prod-Verzeichnis")

# bcrypt-Hash von "Schulung2026!" (nur für die Wegwerf-Demo-DB)
PW = "$2a$10$bx7pJwWTjWI4iBHi1.OpgOuijnNRrq04fTIfha7cSsJp3BjvVWvb."
VORSTAND = int(os.environ.get("SCHULUNG_VORSTAND", "10"))
TRAINER = int(os.environ.get("SCHULUNG_TRAINER", "11"))

M = ["Lukas", "Jonas", "Felix", "Paul", "Leon", "Noah", "Elias", "Ben", "Finn", "Moritz",
     "Emil", "Anton", "Jakob", "David", "Samuel", "Linus", "Tim", "Nico", "Max", "Tom",
     "Julian", "Simon", "Luis", "Oskar", "Henry", "Mats", "Levi", "Karl", "Theo", "Vincent",
     "Matteo", "Johann", "Leo", "Philipp", "Jan", "Fabian", "Markus", "Stefan", "Thomas", "Andreas"]
F = ["Anna", "Lena", "Mia", "Emma", "Sophie", "Lea", "Clara", "Marie", "Hannah", "Laura",
     "Emilia", "Ida", "Frieda", "Nele", "Paula", "Greta", "Johanna", "Lina", "Ella", "Luisa",
     "Sabine", "Katrin", "Julia", "Nina", "Carina", "Sandra", "Petra", "Miriam", "Tanja", "Eva"]
L = ["Müller", "Schmid", "Keller", "Wagner", "Becker", "Hofmann", "Schäfer", "Koch", "Richter",
     "Bauer", "Klein", "Wolf", "Neumann", "Schwarz", "Zimmermann", "Braun", "Krüger", "Hartmann",
     "Lange", "Werner", "Krause", "Lehmann", "Maier", "Köhler", "Walter", "Kaiser", "Fuchs",
     "Peters", "Lang", "Scholz", "Möller", "Weiß", "Jung", "Hahn", "Vogel", "Friedrich", "Seidel",
     "Engel", "Roth", "Brandt", "Haas", "Schreiber", "Graf", "Dietrich", "Kühn", "Pohl", "Sommer",
     "Kraus", "Ludwig", "Winter", "Berger", "Frank", "Beck", "Lorenz", "Baumann", "Albrecht",
     "Franke", "Ernst", "Simon", "Böhm", "Arnold", "Horn", "Kuhn", "Busch", "Martin", "Voigt"]
SAMPLES = ["Bin dabei!", "Wer kann am Samstag fahren?", "Training fällt heute aus, Halle belegt.",
           "Danke fürs Organisieren", "Ich bringe einen Kuchen mit.", "Treffpunkt 45 min vor Anpfiff.",
           "Trikots sind gewaschen.", "Kann jemand meinen Dienst übernehmen?", "Super Spiel heute!",
           "Bitte bis Freitag zu- oder absagen."]


def slug(s):
    s = s.lower().replace("ä", "ae").replace("ö", "oe").replace("ü", "ue").replace("ß", "ss")
    return re.sub(r"[^a-z]", "", s)


db = sqlite3.connect(DB)
c = db.cursor()
real_full = set()   # echte Voll-Namen, für die Abschlussprüfung
name_map = {}       # echter Name/Nachname/Vorname -> Ersatz

# --- Mitglieder ----------------------------------------------------------
user_fake = {}
for mid, fn, ln, g, uid in c.execute("SELECT id, first_name, last_name, gender, user_id FROM members").fetchall():
    nfn = (F if g == "f" else M)[(mid * 7) % len(F if g == "f" else M)]
    nln = L[(mid * 13 + 5) % len(L)]
    if fn and ln:
        real_full.add(f"{fn} {ln}")
        name_map[f"{fn} {ln}"] = f"{nfn} {nln}"
        if len(ln) > 3:
            name_map.setdefault(ln, nln)
    if fn and len(fn) > 3:
        name_map.setdefault(fn, nfn)
    c.execute("""UPDATE members SET first_name=?, last_name=?, street=?, zip='70190', city='Stuttgart',
                 pass_number=NULL, handball_360_id=NULL, sepa_mandat_path=NULL WHERE id=?""",
              (nfn, nln, f"Musterweg {mid % 90 + 1}", mid))
    if uid:
        user_fake[uid] = (nfn, nln)

# --- Nutzer --------------------------------------------------------------
for uid, fn, ln in c.execute("SELECT id, first_name, last_name FROM users").fetchall():
    nfn, nln = user_fake.get(uid, ((M if uid % 2 else F)[(uid * 3) % 30], L[(uid * 11 + 2) % len(L)]))
    if fn and ln:
        real_full.add(f"{fn} {ln}")
        name_map.setdefault(f"{fn} {ln}", f"{nfn} {nln}")
        if len(ln) > 3:
            name_map.setdefault(ln, nln)
    email = {VORSTAND: "vorstand@beispiel.de", TRAINER: "trainer@beispiel.de"}.get(
        uid, f"{slug(nfn)}.{slug(nln)}{uid}@beispiel.de")
    c.execute("""UPDATE users SET first_name=?, last_name=?,
                 email=CASE WHEN ? IN (?, ?) OR (email IS NOT NULL AND email<>'') THEN ? ELSE email END,
                 login_name=CASE WHEN login_name IS NULL THEN NULL ELSE ? END, recovery_email=NULL,
                 street=NULL, zip=NULL, city=NULL, photo_path=NULL, password=?,
                 can_login=CASE WHEN ? IN (?, ?) THEN 1 ELSE can_login END,
                 failed_login_count=0, locked_until=NULL WHERE id=?""",
              (nfn, nln, uid, VORSTAND, TRAINER, email, f"{slug(nfn)}{uid}", PW, uid, VORSTAND, TRAINER, uid))

# Geburtstage: Jahr bleibt (Jahrgang im Kader), Tag und Monat werden verschoben
for t, k in (("members", 7), ("users", 5)):
    c.execute(f"""UPDATE {t} SET date_of_birth = substr(date_of_birth,1,4) || '-' ||
                  printf('%02d', 1 + (id*{k})%12) || '-' || printf('%02d', 1 + (id*13)%28)
                  WHERE date_of_birth IS NOT NULL AND length(date_of_birth) >= 4""")
c.execute("UPDATE member_phones SET number='+49 170 ' || (1000000 + id * 7919 % 8999999)")
c.execute("UPDATE user_phones SET number='+49 170 ' || (1000000 + id * 7919 % 8999999)")

# Übungsgruppen tragen oft Spitznamen der Trainer im Namen
for i, (kid, kname) in enumerate(c.execute("SELECT id, name FROM kader WHERE kind='practice' ORDER BY id").fetchall()):
    new = "Frühtraining Dienstag" if i == 0 else f"Übungsgruppe {i + 1}"
    if kname:
        name_map[kname] = new
    c.execute("UPDATE kader SET name=? WHERE id=?", (new, kid))

# --- Freitexte: Namen ersetzen (lange Schlüssel zuerst, nur ganze Wörter) --
keys = sorted(name_map, key=len, reverse=True)
pat = re.compile(r"(?<!\w)(" + "|".join(re.escape(k) for k in keys) + r")(?!\w)")


def scrub(s):
    return pat.sub(lambda m: name_map[m.group(1)], s) if isinstance(s, str) else s


TEXT = [("user_events", "title"), ("user_events", "body"), ("conversations", "name"),
        ("files", "original_name"), ("file_folders", "name"), ("duty_slots", "event_name"),
        ("carpooling_events", "actor_name"), ("games", "note"), ("training_sessions", "note"),
        ("training_sessions", "title"), ("training_series", "name"), ("training_series", "note"),
        ("match_reports", "title"), ("match_reports", "body_md"), ("videos", "title"),
        ("team_penalties", "reason"), ("team_cashbook_entries", "note"),
        ("pending_event_notes_push", "note_text"), ("member_series_unavailabilities", "reason")]
for t, col in TEXT:
    for rid, val in c.execute(f"SELECT rowid, {col} FROM {t}").fetchall():
        nv = scrub(val)
        if nv != val:
            c.execute(f"UPDATE {t} SET {col}=? WHERE rowid=?", (nv, rid))

# --- Inhalte, die man nicht zuverlässig scrubben kann: komplett ersetzen --
for (mid,) in c.execute("SELECT id FROM messages").fetchall():
    c.execute("UPDATE messages SET body=? WHERE id=?", (SAMPLES[mid % len(SAMPLES)], mid))
c.execute("UPDATE broadcasts SET body='Info vom Vorstand: Mitgliederversammlung am 10.10., 19 Uhr.'")
c.execute("UPDATE duty_assignment_comments SET body='Übernehme gern.'")
for t in ("training_responses", "game_responses"):
    c.execute(f"UPDATE {t} SET reason=CASE WHEN reason IS NULL OR reason='' THEN reason ELSE 'privat verhindert' END")
c.execute("UPDATE member_absences SET note=''")
c.execute("UPDATE training_diary_entries SET note=''")
c.execute("UPDATE vehicle_info SET notes=''")
c.execute("UPDATE training_sessions SET cancel_reason=CASE WHEN cancel_reason IS NULL THEN NULL ELSE 'Halle belegt' END")
c.execute("UPDATE membership_requests SET first_name='Alex', last_name='Beispiel', email='alex.beispiel'||id||'@beispiel.de', parent_email=NULL, comment=NULL")
c.execute("UPDATE invitation_tokens SET first_name='Alex', last_name='Beispiel', email='einladung'||id||'@beispiel.de', comment=NULL")
for t in ("member_change_drafts", "email_change_tokens", "refresh_tokens", "push_subscriptions"):
    c.execute(f"DELETE FROM {t}")
db.commit()

# --- Abschlussprüfung: kein echter Voll-Name mehr in einem Freitext -------
fake_full = {f"{a} {b}" for t in ("members", "users")
             for a, b in c.execute(f"SELECT first_name, last_name FROM {t}") if a and b}
real_only = sorted(real_full - fake_full, key=len, reverse=True)
left = []
if real_only:
    rpat = re.compile(r"(?<!\w)(" + "|".join(re.escape(n) for n in real_only) + r")(?!\w)")
    for t, col in TEXT:
        for (val,) in c.execute(f"SELECT {col} FROM {t}"):
            if isinstance(val, str) and rpat.search(val):
                left.append(f"{t}.{col}")
if left:
    sys.exit(f"anon.py: {len(left)} echte Namen übrig in {sorted(set(left))} — Skript erweitern")

# --- IDs für die Platzhalter in shots.txt ---------------------------------
def one(sql, *args):
    row = c.execute(sql, args).fetchone()
    return row[0] if row else None


def functions(uid):
    return one("""SELECT group_concat(f.function) FROM members m JOIN member_club_functions f ON f.member_id=m.id
                  WHERE m.user_id=?""", uid) or ""


ids = {
    "vorstand_member": one("SELECT id FROM members WHERE user_id=? ORDER BY id LIMIT 1", VORSTAND),
    "trainer_team": one("""SELECT k.team_id FROM kader_trainers kt JOIN kader k ON k.id=kt.kader_id
                           JOIN seasons s ON s.id=k.season_id AND s.is_active=1
                           JOIN members m ON m.id=kt.member_id
                           WHERE m.user_id=? AND k.team_id IS NOT NULL ORDER BY k.id LIMIT 1""", TRAINER),
}
ids["trainer_game"] = one("""SELECT g.id FROM games g JOIN game_teams gt ON gt.game_id=g.id
                             WHERE gt.team_id=? AND substr(g.date,1,10) >= date('now')
                             AND g.event_type <> 'generisch' ORDER BY g.date LIMIT 1""", ids["trainer_team"])
with open(IDS, "w") as f:
    json.dump(ids, f, indent=2)

for label, uid, need in (("Vorstand", VORSTAND, "vorstand"), ("Trainer", TRAINER, "trainer")):
    fx = functions(uid)
    warn = "" if need in fx.split(",") else f"  ACHTUNG: Funktion '{need}' fehlt"
    print(f"anon.py: Persona {label}: Nutzer {uid} ({fx or 'keine Funktion'}){warn}")
missing = [k for k, v in ids.items() if v is None]
if missing:
    sys.exit(f"anon.py: keine ID für {missing} — Persona-IDs per SCHULUNG_VORSTAND/SCHULUNG_TRAINER anpassen")
print(f"anon.py: {len(name_map)} Namen ersetzt, IDs {ids}")
