import type { EventLine } from './staffeln'

/**
 * Ableitung des Torverlaufs aus den Ereignissen eines Spielberichts.
 *
 * Reine Funktionen ohne Abruf: `GET /api/bwhv-games/{id}/report` liefert die
 * Ereignisse bereits vollständig, Lauf und Spielsituation stehen darin nur
 * nicht ausgerechnet. Deshalb gibt es dafür keine eigene Route.
 */

/** Spielsituation der werfenden Mannschaft NACH ihrem Tor. */
export type Situation = 'fuehrung' | 'gleichstand' | 'rueckstand'

export interface Tor {
  seq: number
  /** Spielzeit in Sekunden, kumulativ über beide Halbzeiten. */
  gameSecond: number
  side: 'home' | 'guest'
  scorer: string
  sevenMeter: boolean
  scoreHome: number
  scoreGuest: number
  /** Das wievielte Tor in Folge ohne Gegentreffer. */
  lauf: number
  situation: Situation
}

/**
 * Die Tore eines Spiels, mit mitgezähltem Spielstand, Lauf und Situation.
 *
 * Der Spielstand wird MITGEZÄHLT, nicht aus `scoreHome`/`scoreGuest` gelesen:
 * das Feld ist nur bei Toren gefüllt, und die Kreuzprobe beim Einlesen
 * garantiert für jeden ausgewerteten Bericht, dass die Summe der Tor-Ereignisse
 * den Endstand ergibt. Das Zählen kann also nicht auseinanderlaufen.
 *
 * Nur `goal` und `seven_m_goal` sind Tore — ein verworfener Siebenmeter, eine
 * Zeitstrafe oder eine Auszeit erzeugt keinen Punkt.
 */
export function torereihe(events: EventLine[]): Tor[] {
  const tore: Tor[] = []
  let home = 0
  let gast = 0
  let letzteSeite: 'home' | 'guest' | null = null
  let lauf = 0

  for (const e of events) {
    if (e.kind !== 'goal' && e.kind !== 'seven_m_goal') continue
    if (e.side !== 'home' && e.side !== 'guest') continue

    if (e.side === 'home') home += 1
    else gast += 1

    lauf = e.side === letzteSeite ? lauf + 1 : 1
    letzteSeite = e.side

    const eigene = e.side === 'home' ? home : gast
    const fremde = e.side === 'home' ? gast : home

    tore.push({
      seq: e.seq,
      gameSecond: e.gameSecond,
      side: e.side,
      scorer: e.playerName ?? '',
      sevenMeter: e.kind === 'seven_m_goal',
      scoreHome: home,
      scoreGuest: gast,
      lauf,
      situation: eigene > fremde ? 'fuehrung' : eigene === fremde ? 'gleichstand' : 'rueckstand',
    })
  }
  return tore
}

/**
 * Index des ersten Tores der ZWEITEN Halbzeit — oder `null`, wenn kein
 * Halbzeitstand vorliegt.
 *
 * Die Grenze kommt aus dem Halbzeitstand des BERICHTSKOPFES, nie aus einer
 * angenommenen Spieldauer. Der Bericht trägt keine Halbzeitmarke im Verlauf:
 * die Spieluhr steht in der Pause still, und die Pause ist nur in der Uhrzeit
 * sichtbar — als Trennsignal unzuverlässig, eine lange Auszeit sieht genauso
 * aus. Der Halbzeitstand dagegen ist eine Aussage über DIESES Spiel, und die
 * Reihenfolge der Tore ist eindeutig.
 *
 * Gemessen an der Fixture `spielbericht_905272`: der Halbzeitstand 14:13 fällt
 * bei Spielzeit 21:57, das nächste Ereignis steht bei 25:27 — die Halbzeit
 * endet also NICHT beim letzten Tor, und ein Zeitvergleich mit einer geratenen
 * Dauer träfe die falschen Tore.
 */
export function halbzeitGrenze(
  tore: Tor[],
  homeGoalsHt: number | null,
  guestGoalsHt: number | null,
): number | null {
  if (homeGoalsHt === null || guestGoalsHt === null) return null
  for (let i = 0; i < tore.length; i++) {
    if (tore[i].scoreHome === homeGoalsHt && tore[i].scoreGuest === guestGoalsHt) {
      return i + 1
    }
  }
  // Der Halbzeitstand taucht im mitgezählten Verlauf nicht auf (0:0 zur Pause
  // trifft das nicht, weil dann gar kein Tor davor liegt). Für 0:0 ist die
  // erste Halbzeit leer, sonst ist der Bericht in sich uneinheitlich und eine
  // durchgehende Achse die ehrlichere Antwort.
  return homeGoalsHt === 0 && guestGoalsHt === 0 ? 0 : null
}

/** Auf das nächste Vielfache von 5 gerundet (kaufmännisch). */
function auf5(minuten: number): number {
  return Math.max(5, Math.round(minuten / 5) * 5)
}

/**
 * Halbzeitdauer in Minuten für die Achsen.
 *
 * `konfiguriert` ist die für die Altersklasse gepflegte Halbzeitdauer
 * (`age_class_game_rules`). Sie wird übernommen, aber nicht blind: sie kann
 * fehlen (die Tabelle kennt nur A- bis D-Jugend und ist nicht vorbefüllt) oder
 * dem Dokument widersprechen. Der zweite Fall ist nicht theoretisch — die
 * Fixture `905272` ist eine `mB-RL-BW` mit 25-Minuten-Halbzeiten, und
 * „B-Jugend = 30" ist ein naheliegender Eintrag. Mit 30 gezeichnet läge das
 * erste Tor der zweiten Halbzeit (25:27) bei Minute −4:33, also VOR dem Beginn
 * seiner eigenen Achse.
 *
 * Geprüft werden deshalb genau die beiden Arten, auf die eine Achse unmöglich
 * würde: ein Tor der ersten Halbzeit jenseits ihres Endes, oder eines der
 * zweiten vor ihrem Anfang. Was sie durchlassen, ist zeichenbar.
 *
 * WICHTIG: Welches Tor zu welcher Halbzeit gehört, hängt an dieser Zahl NICHT
 * (siehe `halbzeitGrenze`). Ein falsch gepflegter Wert verschiebt höchstens
 * Positionen auf der Achse.
 */
export function halbzeitMinuten(
  tore: Tor[],
  grenze: number | null,
  konfiguriert: number | null,
): number {
  const maxSekunde = tore.length > 0 ? Math.max(...tore.map((t) => t.gameSecond)) : 0
  const letzteHz1 = grenze !== null && grenze > 0 ? tore[grenze - 1].gameSecond : 0
  const ersteHz2 = grenze !== null && grenze < tore.length ? tore[grenze].gameSecond : Infinity

  const abgeleitet = Math.max(
    auf5(Math.ceil(maxSekunde / 60) / 2),
    Math.ceil(letzteHz1 / 60),
  )
  if (konfiguriert === null || konfiguriert <= 0) return abgeleitet
  const grenzSekunde = konfiguriert * 60
  if (letzteHz1 > grenzSekunde || ersteHz2 < grenzSekunde) return abgeleitet
  return konfiguriert
}

export interface Halbzeit {
  /** 1 oder 2; bei fehlender Halbzeitgrenze gibt es nur die 1. */
  nummer: number
  /** Beschriftung der Achse in Spielminuten. */
  vonMinute: number
  bisMinute: number
  tore: Tor[]
}

/**
 * Die Tore auf ihre Halbzeit-Achsen verteilt.
 *
 * Die Achse darf zu lang sein, nie zu kurz: `bisMinute` wird notfalls über die
 * Halbzeitdauer hinaus gedehnt, damit kein Tor außerhalb liegt. Eine zu lange
 * Achse staucht die Kreise; ein abgeschnittenes Tor wäre ein verschwundener
 * Datenpunkt.
 */
export function achsen(tore: Tor[], grenze: number | null, halbzeit: number): Halbzeit[] {
  const minute = (t: Tor) => t.gameSecond / 60
  if (grenze === null) {
    const bis = Math.max(halbzeit * 2, ...tore.map(minute))
    return [{ nummer: 1, vonMinute: 0, bisMinute: Math.ceil(bis), tore }]
  }
  const hz1 = tore.slice(0, grenze)
  const hz2 = tore.slice(grenze)
  return [
    {
      nummer: 1,
      vonMinute: 0,
      bisMinute: Math.ceil(Math.max(halbzeit, ...hz1.map(minute))),
      tore: hz1,
    },
    {
      nummer: 2,
      vonMinute: halbzeit,
      bisMinute: Math.ceil(Math.max(halbzeit * 2, ...hz2.map(minute))),
      tore: hz2,
    },
  ]
}
