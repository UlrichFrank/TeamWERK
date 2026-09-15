#!/usr/bin/env python3
"""Erzeugt die Schulungsfolien aus einer kleinen Text-DSL.

    docs/schulung/folien.txt      Texte der Folien       (wird bearbeitet)
    docs/schulung/tools/vorlage.html  CSS + JS + Rahmen  (nur fürs Aussehen)
 →  docs/schulung/folien/index.html   fertige Präsentation (generiert)

Aufruf: python3 docs/schulung/tools/folien.py   (oder `make folien`)
Nur Standardbibliothek. Fehler nennen die Zeile in folien.txt.
"""
import html
import math
import re
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent.parent
SRC = HERE / "folien.txt"
TPL = HERE / "tools" / "vorlage.html"
OUT = HERE / "folien" / "index.html"
IMG = HERE / "folien" / "img"
SHOTS = HERE / "shots.txt"

COPY_FIELDS = {"weg", "titel", "absicht", "rollen", "rollen-text", "rollen-zusatz", "hinweis"}
TYPES = {
    "titel": {"eyebrow", "titel", "untertitel", "chips", "fuss"},
    "kapitel": {"teil", "titel", "text"},
    "funktion": COPY_FIELDS | {"bild", "bild-ganz", "alt"},
    "handy": COPY_FIELDS | {"bilder"},
    "vorher-nachher": {"titel", "vorher", "nachher", "text"},
    "aussagen": {"titel"},
    "modell": {"titel", "notiz"},
    "vergleich": {"titel", "verbindung"},
    "raster": {"titel", "fuss"},
    "liste": {"titel"},
    "schritte": {"titel"},
}
ITEM_ATTRS = {"weg", "warn"}
ROLES = {
    "alle": "c-all", "spieler": "c-all",
    "trainer": "c-trainer", "sportliche leitung": "c-trainer", "sportl. leitung": "c-trainer",
    "vorstand": "c-vorstand", "kassierer": "c-kasse", "medien": "c-medien",
}
DEFAULT_HINWEIS = "Beispieldaten · Namen anonymisiert"


class DslError(Exception):
    pass


class Slide:
    def __init__(self, line):
        self.line = line
        self.fields = {}
        self.field_lines = {}
        self.items = []

    def f(self, key, default=""):
        return self.fields.get(key, default)

    def need(self, key):
        if not self.fields.get(key):
            raise DslError(f"folien.txt:{self.line}: Folie vom Typ „{self.fields.get('typ')}“ braucht „{key}:“")
        return self.fields[key]


# ── Parser ────────────────────────────────────────────────────────────────
def parse(text):
    slides, cur, last = [], None, None
    for no, raw in enumerate(text.split("\n"), 1):
        line = raw.rstrip()
        stripped = line.strip()
        if not stripped or line.startswith("#"):
            continue
        if stripped == "---":
            cur, last = Slide(no), None
            slides.append(cur)
            continue
        if cur is None:
            raise DslError(f"folien.txt:{no}: Text vor der ersten Folie — Folien beginnen mit „---“")
        if raw[0] in " \t":
            if isinstance(last, dict):
                m = re.match(r"([a-z]+):\s*(.*)$", stripped)
                if m and m.group(1) in ITEM_ATTRS:
                    last["attrs"][m.group(1)] = m.group(2)
                else:
                    last["lines"].append(stripped)
            elif isinstance(last, str):
                cur.fields[last] = (cur.fields[last] + "\n" + stripped).strip("\n")
            else:
                raise DslError(f"folien.txt:{no}: eingerückte Zeile ohne Feld oder Punkt davor")
        elif line.startswith("- "):
            last = {"text": line[2:].strip(), "lines": [], "attrs": {}, "line": no}
            cur.items.append(last)
        else:
            m = re.match(r"([a-z-]+):\s*(.*)$", line)
            if not m:
                raise DslError(f"folien.txt:{no}: unverständlich: „{line}“ (erwartet „feld: wert“ oder „- punkt“)")
            last = m.group(1)
            cur.fields[last] = m.group(2)
            cur.field_lines[last] = no
    for s in slides:
        typ = s.fields.get("typ")
        if typ not in TYPES:
            raise DslError(f"folien.txt:{s.line}: unbekannter oder fehlender typ „{typ or ''}“ — erlaubt: {', '.join(TYPES)}")
        unknown = sorted(set(s.fields) - TYPES[typ] - {"typ"}, key=lambda k: s.field_lines[k])
        if unknown:
            raise DslError(f"folien.txt:{s.field_lines[unknown[0]]}: Feld „{unknown[0]}“ passt nicht zu typ „{typ}“ — "
                           f"erlaubt: {', '.join(sorted(TYPES[typ]))}")
    return slides


# ── Text-Helfer ───────────────────────────────────────────────────────────
def inline(s):
    """HTML-sicher; lässt Entities wie &shy; durch und macht aus **x** fett."""
    out = html.escape(s, quote=False)
    out = re.sub(r"&amp;(#\d+|[a-zA-Z]+);", r"&\1;", out)
    return re.sub(r"\*\*(.+?)\*\*", r"<b>\1</b>", out)


def attr(s):
    return html.escape(re.sub(r"&(#\d+|[a-zA-Z]+);", "", s), quote=True)


def split_list(s):
    return [x.strip() for x in s.split(",") if x.strip()]


def split_label(text):
    head, sep, rest = text.partition(": ")
    return (head, rest) if sep else ("", text)


def item_text(it):
    return " ".join(it["lines"])


def weg(s):
    parts = [p.strip() for p in s.split("›")]
    lead = " › ".join(inline(p) for p in parts[:-1])
    return (lead + " › " if lead else "") + f"<b>{inline(parts[-1])}</b>"


def role_chip(name, line, strict=True):
    cls = ROLES.get(name.lower())
    if cls is None:
        if strict:
            raise DslError(f"folien.txt:{line}: unbekannte Rolle „{name}“ — erlaubt: {', '.join(sorted(ROLES))}")
        cls = "c-all"
    return f'<span class="chip {cls}">{inline(name)}</span>'


# ── Folientypen ───────────────────────────────────────────────────────────
def r_titel(s):
    chips = "".join(f"<span>{inline(c)}</span>" for c in split_list(s.f("chips")))
    sub = f" <span>{inline(s.f('untertitel'))}</span>" if s.f("untertitel") else ""
    return f"""<div class="slide s-yellow s-title">
    <div>
      {f'<p class="eyebrow"><b>{inline(s.f("eyebrow"))}</b></p>' if s.f("eyebrow") else ''}
      <h1>{inline(s.need("titel"))}{sub}</h1>
    </div>
    <img class="logo" src="logo.svg" alt="Team Stuttgart Handball">
    {f'<div class="roles">{chips}</div>' if chips else ''}
    {f'<p class="url">{inline(s.f("fuss"))}</p>' if s.f("fuss") else ''}
  </div>"""


def r_kapitel(s):
    return f"""<div class="slide s-dark s-section">
    <p class="part">{inline(s.f("teil"))}</p>
    <h2>{inline(s.need("titel"))}</h2>
    {f'<p>{inline(s.f("text"))}</p>' if s.f("text") else ''}
  </div>"""


def copy_block(s):
    points = "".join(f"<li>{inline(' '.join([it['text']] + it['lines']))}</li>" for it in s.items)
    chips = "".join(role_chip(r, s.line) for r in split_list(s.f("rollen")))
    extra = f'<span class="lbl">&nbsp;{inline(s.f("rollen-zusatz"))}</span>' if s.f("rollen-zusatz") else ""
    roles = (f'<div class="roles-row"><span class="lbl">{inline(s.f("rollen-text", "für"))}</span>{chips}{extra}</div>'
             if chips else "")
    return f"""<div class="copy">
      {f'<p class="eyebrow">{weg(s.f("weg"))}</p>' if s.f("weg") else ''}
      <h2>{inline(s.need("titel"))}</h2>
      {f'<p class="intent"><mark>{inline(s.f("absicht"))}</mark></p>' if s.f("absicht") else ''}
      {f'<ul class="points">{points}</ul>' if points else ''}
      {roles}
    </div>"""


def r_funktion(s):
    file, _, url = (x.strip() for x in s.need("bild").partition("|"))
    s.images.append(file)
    alt = s.f("alt") or f"Screenshot: {s.need('titel')}"
    full = ' style="width:100%;margin-left:0"' if s.f("bild-ganz").lower() in ("ja", "1", "true") else ""
    return f"""<div class="slide s-feature">
    {copy_block(s)}
    <figure class="shot"><div class="urlbar">{inline(url or "/")}</div><img src="img/{attr(file)}.jpg" alt="{attr(alt)}"{full} loading="lazy"></figure>
    <p class="demo">{inline(s.f("hinweis", DEFAULT_HINWEIS))}</p>
  </div>"""


def r_handy(s):
    files = split_list(s.need("bilder"))
    s.images.extend(files)
    phones = "".join(f'<div class="phone"><img src="img/{attr(f)}.jpg" alt="Handy-Ansicht {i}" loading="lazy"></div>'
                     for i, f in enumerate(files, 1))
    return f"""<div class="slide s-phones">
    {copy_block(s)}
    <div class="phones">{phones}</div>
    <p class="demo">{inline(s.f("hinweis", DEFAULT_HINWEIS))}</p>
  </div>"""


def r_vorher_nachher(s):
    before = "".join(f"<p>{inline(x)}</p>" for x in s.f("vorher").split("\n") if x)
    after = [x for x in s.f("nachher").split("\n") if x]
    big = "<br>".join(inline(x) for x in after[:-1])
    big += (("<br>" if big else "") + f"<span>{inline(after[-1])}</span>") if after else ""
    return f"""<div class="slide s-text">
    <h2>{inline(s.need("titel"))}</h2>
    <div class="before-after">
      <div class="before"><p class="eyebrow">Bisher</p>{before}</div>
      <div class="after">
        <p class="eyebrow">Jetzt</p>
        <p class="big">{big}</p>
        {f'<p>{inline(s.f("text"))}</p>' if s.f("text") else ''}
      </div>
    </div>
  </div>"""


def r_aussagen(s):
    gains = "".join(f'<div class="gain"><h3>{inline(it["text"])}</h3><p>{inline(item_text(it))}</p></div>'
                    for it in s.items)
    return f"""<div class="slide s-text">
    <h2>{inline(s.need("titel"))}</h2>
    <div class="gains">{gains}</div>
  </div>"""


def r_modell(s):
    boxes, cols = [], []
    for i, it in enumerate(s.items):
        label, title = split_label(it["text"])
        rows = [l.partition(" = ") for l in it["lines"] if " = " in l]
        paras = "".join(f"<p>{inline(l)}</p>" for l in it["lines"] if " = " not in l)
        lists = ("<div class=\"lists\">" + "".join(
            f'<div class="list"><b>{inline(a)}</b><span>{inline(b)}</span></div>' for a, _, b in rows) + "</div>"
                 if rows else "")
        cls = "box season" if i == 0 else "box"
        boxes.append(f'<div class="{cls}"><p class="eyebrow">{inline(label)}</p><h3>{inline(title)}</h3>{paras}{lists}</div>')
        cols.append("1.35fr" if rows else "1fr")
    arrow = '<div class="arrow" aria-hidden="true">→</div>'
    return f"""<div class="slide s-text">
    <h2>{inline(s.need("titel"))}</h2>
    <div class="model-wrap" style="--cols: {' auto '.join(cols)}">{arrow.join(boxes)}</div>
    {f'<p class="aside">{inline(s.f("notiz"))}</p>' if s.f("notiz") else ''}
  </div>"""


def r_vergleich(s):
    if len(s.items) < 2:
        raise DslError(f"folien.txt:{s.line}: vergleich braucht mindestens zwei Punkte (die beiden Kästen)")
    boxes = []
    for i, it in enumerate(s.items[:2]):
        label, title = split_label(it["text"])
        boxes.append(f'<div class="{"box season" if i else "box"}"><p class="eyebrow">{inline(label)}</p>'
                     f'<h3>{inline(title)}</h3><p>{inline(item_text(it))}</p></div>')
    link = f'<div class="link"><i></i>{inline(s.f("verbindung"))}</div>'
    cases = "".join(f'<div class="case"><b>{inline(it["text"])}</b><span>{inline(item_text(it))}</span></div>'
                    for it in s.items[2:])
    return f"""<div class="slide s-text">
    <h2>{inline(s.need("titel"))}</h2>
    <div class="pair">{boxes[0]}{link}{boxes[1]}</div>
    {f'<div class="cases">{cases}</div>' if cases else ''}
  </div>"""


def r_raster(s):
    cells = "".join(f'<div class="func">{role_chip(it["text"], it["line"], strict=False)}<p>{inline(item_text(it))}</p></div>'
                    for it in s.items)
    foot = "".join(f"<span>{inline(x)}</span>" for x in s.f("fuss").split("\n") if x)
    return f"""<div class="slide s-text">
    <h2>{inline(s.need("titel"))}</h2>
    <div class="funcs">{cells}</div>
    {f'<div class="foot">{foot}</div>' if foot else ''}
  </div>"""


def r_liste(s):
    cells = "".join(f'<div><p class="eyebrow">{inline(it["attrs"].get("weg", ""))}</p><b>{inline(it["text"])}</b>'
                    f'<span>{inline(item_text(it))}</span></div>' for it in s.items)
    return f"""<div class="slide s-text">
    <h2>{inline(s.need("titel"))}</h2>
    <div class="more">{cells}</div>
  </div>"""


def r_schritte(s):
    steps = []
    for it in s.items:
        sub = inline(it["attrs"].get("weg", ""))
        if it["attrs"].get("warn"):
            sub += (" " if sub else "") + f'<span class="warn">{inline(it["attrs"]["warn"])}</span>'
        steps.append(f"<li><b>{inline(it['text'])}</b><span>{sub}</span></li>")
    return f"""<div class="slide s-text">
    <h2>{inline(s.need("titel"))}</h2>
    <ol class="steps" style="--rows: {max(1, math.ceil(len(steps) / 2))}">{''.join(steps)}</ol>
  </div>"""


RENDER = {"titel": r_titel, "kapitel": r_kapitel, "funktion": r_funktion, "handy": r_handy,
          "vorher-nachher": r_vorher_nachher, "aussagen": r_aussagen, "modell": r_modell,
          "vergleich": r_vergleich, "raster": r_raster, "liste": r_liste, "schritte": r_schritte}


def main():
    try:
        slides = parse(SRC.read_text(encoding="utf-8"))
        parts = []
        for n, s in enumerate(slides, 1):
            s.images = []
            body = RENDER[s.fields["typ"]](s)
            label = re.sub(r"&\w+;", "", s.f("titel"))
            parts.append(f'  <!-- {n} · {s.fields["typ"]} · {label} (folien.txt:{s.line}) -->\n'
                         f'  <section class="slide-wrap">{body}</section>\n')
    except DslError as e:
        sys.exit(f"folien.py: {e}")

    template = TPL.read_text(encoding="utf-8")
    if template.count("<!-- FOLIEN -->") != 1:
        sys.exit("folien.py: vorlage.html braucht genau einen Platzhalter <!-- FOLIEN -->")
    OUT.write_text(template.replace("<!-- FOLIEN -->", "\n".join(parts)), encoding="utf-8")

    # Bilder: vorhanden? in shots.txt aufgeführt?
    listed = set()
    for l in SHOTS.read_text(encoding="utf-8").splitlines():
        cols = [c.strip() for c in l.split("|")]
        if len(cols) >= 3 and not l.lstrip().startswith("#"):
            listed.add(cols[2])
    used = [img for s in slides for img in s.images]
    for img in dict.fromkeys(used):
        if img not in listed:
            print(f"folien.py: Hinweis — Bild „{img}“ steht nicht in shots.txt (make schulung erzeugt es nicht)")
        elif not (IMG / f"{img}.jpg").exists():
            print(f"folien.py: Hinweis — Bild „{img}.jpg“ fehlt noch (make schulung)")
    print(f"folien.py: {len(slides)} Folien → {OUT.relative_to(HERE.parent.parent)}")


if __name__ == "__main__":
    main()
