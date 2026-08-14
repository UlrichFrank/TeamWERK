import { describe, test, expect, beforeEach } from 'vitest'
import { useRef } from 'react'
import { render, screen, waitFor, act } from '@testing-library/react'
import { MemoryRouter, Routes, Route, useNavigate } from 'react-router-dom'
import { useScrollRestoration } from './useScrollRestoration'

// Zurück-Navigation soll die Scroll-Position wiederherstellen, auch wenn der Nutzer
// nur gescrollt hat (ohne Karten-Klick, der ?focus=… setzt). Zielumgebung ist die
// iOS-Homescreen-PWA — Desktop-Browser können das von sich aus (siehe Hook-Kommentar),
// jsdom kann es gar nicht. Diese Tests prüfen deshalb die Logik des Hooks, nicht das
// Verhalten einer Engine.

/**
 * jsdom macht kein Layout: `scrollTop` ist dort ein No-Op und liest sich immer als 0.
 * Für den Test bekommt das Element deshalb ein echtes, klemmendes scrollTop —
 * `maxScroll` bildet die Inhaltshöhe nach (0 = Liste noch nicht geladen).
 */
let maxScroll = 1000

function patchScroll(el: HTMLElement) {
  if (Object.getOwnPropertyDescriptor(el, 'scrollTop')) return
  let value = 0
  Object.defineProperty(el, 'scrollTop', {
    configurable: true,
    get: () => value,
    set: (v: number) => {
      value = Math.max(0, Math.min(v, maxScroll))
      el.dispatchEvent(new Event('scroll'))
    },
  })
}

function Shell() {
  const mainRef = useRef<HTMLElement>(null)
  useScrollRestoration(mainRef)
  const navigate = useNavigate()
  return (
    <main
      data-testid="main"
      ref={(el) => {
        if (el) patchScroll(el)
        mainRef.current = el
      }}
    >
      <button onClick={() => navigate('/detail')}>zum Detail</button>
      <button onClick={() => navigate('/liste?past=1')}>Filter</button>
      <button onClick={() => navigate(-1)}>zurück</button>
      <Routes>
        <Route path="/liste" element={<div>Liste</div>} />
        <Route path="/detail" element={<div>Detail</div>} />
      </Routes>
    </main>
  )
}

function renderApp(initial = '/liste') {
  render(
    <MemoryRouter initialEntries={[initial]}>
      <Shell />
    </MemoryRouter>,
  )
  return screen.getByTestId('main')
}

function scrollTo(main: HTMLElement, top: number) {
  act(() => {
    main.scrollTop = top
  })
}

beforeEach(() => {
  sessionStorage.clear()
  maxScroll = 1000
})

describe('useScrollRestoration', () => {
  test('stellt die Scroll-Position beim Zurück wieder her', async () => {
    const main = renderApp()
    scrollTo(main, 300)

    screen.getByText('zum Detail').click()
    await screen.findByText('Detail')

    screen.getByText('zurück').click()
    await screen.findByText('Liste')

    await waitFor(() => expect(main.scrollTop).toBe(300))
  })

  test('wartet auf nachgeladenen Inhalt, bevor die Position sitzt', async () => {
    const main = renderApp()
    scrollTo(main, 400)

    screen.getByText('zum Detail').click()
    await screen.findByText('Detail')

    // Zurück auf eine Seite, deren Liste noch am API-Call hängt: der Container ist
    // leer und lässt sich (noch) gar nicht scrollen.
    maxScroll = 0
    screen.getByText('zurück').click()
    await screen.findByText('Liste')
    expect(main.scrollTop).toBe(0)

    // Daten treffen ein → der Restore-Versuch greift nachträglich.
    maxScroll = 1000
    await waitFor(() => expect(main.scrollTop).toBe(400))
  })

  test('neue Seite (anderer Pfad) beginnt oben', async () => {
    const main = renderApp()
    scrollTo(main, 300)

    screen.getByText('zum Detail').click()
    await screen.findByText('Detail')

    expect(main.scrollTop).toBe(0)
  })

  test('reine Query-Änderung auf derselben Seite lässt die Position stehen', async () => {
    const main = renderApp()
    scrollTo(main, 250)

    screen.getByText('Filter').click()
    await waitFor(() => expect(main.scrollTop).toBe(250))
  })

  test('stellt nichts wieder her, wenn zum History-Key eine andere URL gehört', async () => {
    // Nach einem Reload der iOS-PWA kann derselbe Key ('default' für den ersten
    // Eintrag) auf einer anderen Seite landen — dann darf die alte Position nicht
    // auf die neue Seite angewendet werden.
    sessionStorage.setItem(
      'scroll-positions',
      JSON.stringify({ default: { top: 500, url: '/eine-ganz-andere-seite' } }),
    )
    const main = renderApp()

    await new Promise(r => setTimeout(r, 60))
    expect(main.scrollTop).toBe(0)
  })

  test('bei ?focus= bleibt das Scrollen dem Fokus-Mechanismus überlassen', async () => {
    const main = renderApp('/liste?focus=game-1')
    scrollTo(main, 300)

    screen.getByText('zum Detail').click()
    await screen.findByText('Detail')

    screen.getByText('zurück').click()
    await screen.findByText('Liste')

    // Kein Restore — sonst würde er sich mit dem Fokus-Scroll um die Position streiten.
    await new Promise(r => setTimeout(r, 60))
    expect(main.scrollTop).toBe(0)
  })
})
