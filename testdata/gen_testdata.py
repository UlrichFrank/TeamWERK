#!/usr/bin/env python3
"""Generate test CSV files for TeamWERK member import."""
import csv
import random
from datetime import date, timedelta

random.seed(42)

# ── Data pools ──────────────────────────────────────────────────────────────

LAST_NAMES = [
    "Müller","Schmidt","Schneider","Fischer","Weber","Meyer","Wagner","Becker",
    "Schulz","Hoffmann","Schäfer","Koch","Bauer","Richter","Klein","Wolf",
    "Schröder","Neumann","Schwarz","Zimmermann","Braun","Krüger","Hofmann",
    "Hartmann","Lange","Schmitt","Werner","Schmitz","Krause","Meier","Lehmann",
    "Köhler","Herrmann","König","Walter","Huber","Kaiser","Fuchs","Peters",
    "Lang","Scholz","Möller","Weiß","Jung","Hahn","Schubert","Vogel",
    "Friedrich","Berger","Keller","Engel","Arnold","Pfeiffer","Roth","Frank",
    "Albert","Zimmer","Hesse","Brandt","Kühn","Haas","Sommer","Winter",
    "Böhm","Lorenz","Schulte","Maier","Steiner","Heller","Busch","Seidel",
    "Vogt","Kraft","Beck","Riedel","Lenz","Graf","Kroll","Hammer","Winkler",
    "Brandl","Haug","Ziegler","Frey","Hinz","Baumann","Altmann","Ritter",
    "Knoll","Groß","Böhme","Voss","Pfaff","Thiel","Seitz","Kurz","Schenk",
    "Heinrich","Stein","Ernst","Amann","Geiger","Kling","Moser","Held",
    "Decker","Voigt","Beyer","Wimmer","Löffler","Probst","Kramer","Wendt",
    "Götz","Brenner","Fink","Albrecht","Lutz","Steinbach","Römer","Finkbeiner",
    "Güntner","Waldmann","Hübner","Schilling","Renner","Bader","Schuster",
    "Nuss","Ruppert","Endres","Hölzle","Schempp","Mauch","Brenneis","Boldt",
    "Schreiber","Schmid","Häfner","Metzger","Gauger","Faigle","Kuhn","Kraut",
    "Burger","Hug","Hutt","Straub","Brand","Lipp","Nuber","Gaiser",
]

MALE_NAMES = [
    "Lukas","Jonas","Felix","Leon","Finn","Paul","Elias","Noah","Niklas",
    "Tobias","Tim","Jan","Philipp","Julian","Alexander","Maximilian","Daniel",
    "Luca","Simon","David","Fabian","Stefan","Andreas","Christian","Thomas",
    "Martin","Robert","Sebastian","Florian","Patrick","Benjamin","Markus",
    "Lars","Sven","Kai","Timo","Eric","Kevin","Marc","Dennis","Niko",
    "Moritz","Henrik","Ole","Ben","Tom","Max","Nico","Dominic","Rafael",
    "Mats","Till","Hannes","Lennart","Valentin","Jannik","Lasse","Benedikt",
    "Cedric","Dario","Emre","Gabriel","Hugo","Igor","Johann","Karl",
]

FEMALE_NAMES = [
    "Emma","Mia","Hannah","Lea","Lena","Anna","Sophie","Sarah","Laura","Maria",
    "Julia","Katharina","Lisa","Sandra","Jana","Nina","Eva","Franziska",
    "Johanna","Melanie","Nadine","Jessica","Vanessa","Stefanie","Christina",
    "Elena","Alina","Lara","Nora","Leonie","Marie","Luisa","Charlotte",
    "Emilia","Klara","Maja","Ida","Amelie","Friederike","Jasmin","Paula",
    "Greta","Zoe","Anni","Hanna","Maren","Ines","Veronika","Bettina",
    "Sabine","Andrea","Petra","Silke","Anke","Heike","Monika","Kerstin",
    "Renate","Brigitte","Ingrid","Walburga","Edith","Hildegard","Elfriede",
    "Carolin","Miriam","Tanja","Anja","Bianca","Denise","Fiona","Gina",
]

POSITIONS = [
    "Torwart","Linksaußen","Rechtsaußen",
    "Rückraum Links","Rückraum Mitte","Rückraum Rechts","Kreisspieler",
]

STAMMVEREINE = [
    "HSG Cannstatt/Münster/Max-Eyth-See",
    "HSG Gablenberg-Gaisburg",
    "HSG Oberer Neckar",
]

ADDRESSES = [
    ("Königstraße",        "70173", "Stuttgart"),
    ("Bahnhofstraße",      "70372", "Stuttgart"),
    ("Schillerstraße",     "70374", "Stuttgart"),
    ("Cannstatter Straße", "70376", "Stuttgart"),
    ("Münsterstraße",      "70378", "Stuttgart"),
    ("Zuffenhäuser Straße","70435", "Stuttgart"),
    ("Feuerbacher Weg",    "70469", "Stuttgart"),
    ("Möhringer Allee",    "70619", "Stuttgart"),
    ("Obertürkheimer Str", "70327", "Stuttgart"),
    ("Hauptstraße",        "70192", "Stuttgart"),
    ("Marktstraße",        "70806", "Kornwestheim"),
    ("Bahnhofplatz",       "70825", "Korntal-Münchingen"),
    ("Lange Straße",       "71332", "Waiblingen"),
    ("Gaisburger Straße",  "70186", "Stuttgart"),
    ("Obere Neckarstraße", "70379", "Stuttgart"),
    ("Steinhaldenstraße",  "70563", "Stuttgart"),
    ("Vaihinger Straße",   "70567", "Stuttgart"),
    ("Pliensaustraße",     "73728", "Esslingen"),
    ("Plochinger Straße",  "73730", "Esslingen"),
    ("Rathausplatz",       "70771", "Leinfelden-Echterdingen"),
]

EMAIL_DOMAINS = ["gmail.com","gmx.de","web.de","t-online.de","outlook.de","yahoo.de"]

MEMBER_FIELDS = [
    "Name","Vorname","Mitgliedsnummer","Email","Email 2","Telefon","Telefon 2",
    "Geschlecht","Adresse","PLZ","Ort","Mitglied","Stammverein","Status",
    "geboren am","Welcomemail","SEPA Mandat","Kontoinhaber","IBAN","Position",
]

PARENT_FIELDS = ["Name","Vorname","Email","Kind"]

# ── Helpers ──────────────────────────────────────────────────────────────────

def slug(s):
    return (s.lower()
            .replace("ä","ae").replace("ö","oe").replace("ü","ue")
            .replace("ß","ss").replace(" ","").replace("-",""))

def rand_date(y0, m0, y1, m1):
    a = date(y0, m0, 1)
    b = date(y1, m1, 28)
    return a + timedelta(days=random.randint(0, (b - a).days))

def rand_birth(year):
    return date(year, random.randint(1, 12), random.randint(1, 28))

def make_phone():
    prefix = random.choice(["0711","0172","0173","0174","0160","0162","0176","0177"])
    n = random.randint(1000000, 9999999)
    return f"{prefix} {n}"

def make_iban():
    bank = random.choice(["37040044","20050550","10050000","43060967","20080000","60050101"])
    acct = random.randint(1000000000, 9999999999)
    return f"DE{random.randint(10,99)}{bank}{acct}"

# ── Generator ────────────────────────────────────────────────────────────────

members = []
parents = []
last_name_idx = 0
male_idx = 0
female_idx = 0
sv_cycle = 0

def next_last():
    global last_name_idx
    n = LAST_NAMES[last_name_idx % len(LAST_NAMES)]
    last_name_idx += 1
    return n

def next_male():
    global male_idx
    n = MALE_NAMES[male_idx % len(MALE_NAMES)]
    male_idx += 1
    return n

def next_female():
    global female_idx
    n = FEMALE_NAMES[female_idx % len(FEMALE_NAMES)]
    female_idx += 1
    return n

def next_sv():
    global sv_cycle
    sv = STAMMVEREINE[sv_cycle % 3]
    sv_cycle += 1
    return sv

def add_member(idx, vorname, name, gender, birth_year, status, stammverein,
               position, join_y0, join_y1):
    addr = random.choice(ADDRESSES)
    street, plz, ort = addr
    house = random.randint(1, 120)
    email = f"{slug(vorname)}.{slug(name)}@{random.choice(EMAIL_DOMAINS)}"
    email2 = ""
    phone = make_phone()
    phone2 = make_phone() if random.random() < 0.25 else ""
    birth = rand_birth(birth_year)
    mitglied = rand_date(join_y0, 1, join_y1, 12)
    iban = make_iban()
    members.append({
        "Name": name,
        "Vorname": vorname,
        "Mitgliedsnummer": f"TS-{idx:04d}",
        "Email": email,
        "Email 2": email2,
        "Telefon": phone,
        "Telefon 2": phone2,
        "Geschlecht": gender,
        "Adresse": f"{street} {house}",
        "PLZ": plz,
        "Ort": ort,
        "Mitglied": mitglied.strftime("%d.%m.%Y"),
        "Stammverein": stammverein,
        "Status": status,
        "geboren am": birth.strftime("%d.%m.%Y"),
        "Welcomemail": "Ja" if random.random() < 0.9 else "Nein",
        "SEPA Mandat": "Ja" if random.random() < 0.85 else "Nein",
        "Kontoinhaber": f"{vorname} {name}",
        "IBAN": iban,
        "Position": position,
    })
    return f"{vorname} {name}", name

def add_parent(kind_full, kind_last, parent_gender):
    pv = next_female() if parent_gender == "f" else next_male()
    email = f"{slug(pv)}.{slug(kind_last)}@{random.choice(EMAIL_DOMAINS)}"
    parents.append({"Name": kind_last, "Vorname": pv, "Email": email, "Kind": kind_full})

# Age group: (label, birth_years, join_range, parent_count)
# Born 2007-2008 → A-Jugend 2025/26 (U18/U19)
# Born 2009-2010 → B-Jugend 2025/26 (U16/U17)
# Born 2011-2012 → C-Jugend 2025/26 (U14/U15)
# Born 2013-2014 → D-Jugend 2025/26 (U12/U13)

idx = 1
GROUPS = [
    # (gender, birth_years, join_y0, join_y1, parent_count, count)
    ("m", [2007, 2008], 2018, 2022, 1, 15),   # A-Jugend m
    ("f", [2007, 2008], 2018, 2022, 1, 15),   # A-Jugend f
    ("m", [2009, 2010], 2019, 2023, 1, 15),   # B-Jugend m
    ("f", [2009, 2010], 2019, 2023, 1, 15),   # B-Jugend f
    ("m", [2011, 2012], 2020, 2024, 2, 15),   # C-Jugend m
    ("f", [2011, 2012], 2020, 2024, 2, 15),   # C-Jugend f
    ("m", [2013, 2014], 2021, 2025, 2, 15),   # D-Jugend m
    ("f", [2013, 2014], 2021, 2025, 2, 15),   # D-Jugend f
]

for (gender, birth_years, join_y0, join_y1, n_parents, count) in GROUPS:
    for i in range(count):
        vorname = next_male() if gender == "m" else next_female()
        name = next_last()
        by = birth_years[i % 2]
        pos = POSITIONS[i % len(POSITIONS)]
        sv = next_sv()
        kind_full, kind_last = add_member(
            idx, vorname, name, gender, by, "aktiv", sv, pos, join_y0, join_y1
        )
        # Add parents (younger age groups always get 2 parents)
        actual_parents = n_parents if random.random() < 0.6 else 1
        add_parent(kind_full, kind_last, "f")
        if actual_parents >= 2:
            add_parent(kind_full, kind_last, "m")
        idx += 1

# 30 passive members — various ages, position mix, optional Stammverein
passiv_birth_years = (
    [random.randint(1970, 1990) for _ in range(10)] +
    [random.randint(1990, 2005) for _ in range(20)]
)
random.shuffle(passiv_birth_years)

for i in range(30):
    gender = "m" if i % 2 == 0 else "f"
    vorname = next_male() if gender == "m" else next_female()
    name = next_last()
    by = passiv_birth_years[i]
    pos = POSITIONS[i % len(POSITIONS)]
    sv = "" if random.random() < 0.3 else random.choice(STAMMVEREINE)
    add_member(idx, vorname, name, gender, by, "passiv", sv, pos, 2018, 2024)
    idx += 1

# ── Write CSVs ───────────────────────────────────────────────────────────────

with open("test_mitglieder.csv", "w", newline="", encoding="utf-8") as f:
    w = csv.DictWriter(f, fieldnames=MEMBER_FIELDS, delimiter=";",
                       quoting=csv.QUOTE_MINIMAL)
    w.writeheader()
    w.writerows(members)

with open("test_eltern.csv", "w", newline="", encoding="utf-8") as f:
    w = csv.DictWriter(f, fieldnames=PARENT_FIELDS, delimiter=";",
                       quoting=csv.QUOTE_MINIMAL)
    w.writeheader()
    w.writerows(parents)

print(f"Mitglieder: {len(members)}")
print(f"Elternteile: {len(parents)}")

# Verify all positions covered per age group
print("\nPositionsabdeckung je Jahrgang:")
groups_check = [
    ("A-Jugend", range(0, 30)),
    ("B-Jugend", range(30, 60)),
    ("C-Jugend", range(60, 90)),
    ("D-Jugend", range(90, 120)),
]
for label, r in groups_check:
    covered = {members[i]["Position"] for i in r}
    missing = set(POSITIONS) - covered
    status = "OK" if not missing else f"FEHLT: {missing}"
    print(f"  {label}: {status}")
