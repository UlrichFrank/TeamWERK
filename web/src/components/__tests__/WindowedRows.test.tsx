import { describe, test, expect, beforeAll } from 'vitest'
import { render, screen, act } from '@testing-library/react'
import { fireEvent } from '@testing-library/react'
import WindowedRows from '../WindowedRows'

// jsdom liefert für clientHeight/scrollTop standardmäßig 0 → Layout muss für den
// Windowing-Test explizit simuliert werden. Wir verdrahten clientHeight auf einen
// festen Viewport und lassen scrollTop über ein zuweisbares Feld steuerbar sein.
const VIEWPORT = 300
const ROW_HEIGHT = 30

function installLayout(scrollTopBox: { value: number }) {
  Object.defineProperty(HTMLElement.prototype, 'clientHeight', {
    configurable: true,
    get() {
      return this.hasAttribute('data-windowed-scroll') ? VIEWPORT : 0
    },
  })
  Object.defineProperty(HTMLElement.prototype, 'scrollTop', {
    configurable: true,
    get() {
      return this.hasAttribute('data-windowed-scroll') ? scrollTopBox.value : 0
    },
    set(v: number) {
      if (this.hasAttribute('data-windowed-scroll')) scrollTopBox.value = v
    },
  })
}

const scrollBox = { value: 0 }

beforeAll(() => {
  // ResizeObserver ist in jsdom nicht vorhanden — no-op-Stub.
  if (typeof globalThis.ResizeObserver === 'undefined') {
    globalThis.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    } as unknown as typeof ResizeObserver
  }
  installLayout(scrollBox)
})

const ITEMS = Array.from({ length: 500 }, (_, i) => ({ id: i, label: `Zeile ${i}` }))

function renderList() {
  return render(
    <WindowedRows
      items={ITEMS}
      estimatedRowHeight={ROW_HEIGHT}
      overscan={2}
      renderRow={(item) => (
        <div key={item.id} data-testid="row" data-id={item.id}>
          {item.label}
        </div>
      )}
    />,
  )
}

describe('WindowedRows — renders_only_visible_rows', () => {
  test('rendert bei N ≫ Viewport nur die sichtbaren Zeilen (+ Puffer)', () => {
    scrollBox.value = 0
    renderList()

    const rendered = screen.getAllByTestId('row')
    // Viewport 300 / Zeile 30 = 10 sichtbare + 2 Overscan oben (am Anfang keine) + 2 unten.
    // Deutlich weniger als die 500 Einträge — konstant bzgl. Listengröße.
    expect(rendered.length).toBeLessThan(20)
    expect(rendered.length).toBeGreaterThan(0)
    // Erste Zeile ist im DOM, weit hinten liegende nicht.
    expect(screen.getByText('Zeile 0')).toBeInTheDocument()
    expect(screen.queryByText('Zeile 400')).toBeNull()
  })

  test('Scrollen tauscht die gerenderten Zeilen aus, ohne Einträge zu verlieren', () => {
    scrollBox.value = 0
    renderList()

    expect(screen.getByText('Zeile 0')).toBeInTheDocument()
    expect(screen.queryByText('Zeile 300')).toBeNull()

    // Weit nach unten scrollen: Zeile 300 liegt bei 300*30 = 9000px.
    const container = document.querySelector('[data-windowed-scroll]') as HTMLElement
    act(() => {
      scrollBox.value = 9000
      fireEvent.scroll(container)
    })

    // Jetzt ist Zeile 300 sichtbar, Zeile 0 wurde aus dem DOM ausgetauscht.
    expect(screen.getByText('Zeile 300')).toBeInTheDocument()
    expect(screen.queryByText('Zeile 0')).toBeNull()

    // Immer noch nur ein kleines Fenster im DOM (nicht die ganze Liste).
    expect(screen.getAllByTestId('row').length).toBeLessThan(20)
  })

  test('kurze Listen unterhalb der Schwelle werden vollständig gerendert', () => {
    const few = Array.from({ length: 5 }, (_, i) => ({ id: i, label: `Kurz ${i}` }))
    render(
      <WindowedRows
        items={few}
        estimatedRowHeight={ROW_HEIGHT}
        renderRow={(item) => <div key={item.id} data-testid="short-row">{item.label}</div>}
      />,
    )
    expect(screen.getAllByTestId('short-row').length).toBe(5)
  })
})
