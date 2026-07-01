import { describe, test, expect } from 'vitest'
import { render } from '@testing-library/react'
import MarkdownRenderer from './MarkdownRenderer'

describe('MarkdownRenderer', () => {
  test('renders headings from Markdown', () => {
    const { container } = render(<MarkdownRenderer markdown="## Vorbereitung" />)
    const h2 = container.querySelector('h2')
    expect(h2).not.toBeNull()
    expect(h2?.textContent).toBe('Vorbereitung')
  })

  test('sanitizes disallowed html (script tag stripped)', () => {
    const { container } = render(
      <MarkdownRenderer markdown={`Ok\n\n<script>alert(1)</script>`} />
    )
    expect(container.querySelector('script')).toBeNull()
  })

  test('keeps relative image src pointing at /dokumente/datei/{id}', () => {
    const { container } = render(
      <MarkdownRenderer markdown="![Kasse](/dokumente/datei/123)" />
    )
    const img = container.querySelector('img')
    expect(img).not.toBeNull()
    expect(img?.getAttribute('src')).toBe('/dokumente/datei/123')
    expect(img?.getAttribute('alt')).toBe('Kasse')
  })
})
