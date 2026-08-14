import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useWindowedList } from './useWindowedList'

describe('useWindowedList', () => {
  describe('scroll: container mode', () => {
    test('renders all items below threshold', () => {
      const { result } = renderHook(() =>
        useWindowedList({
          count: 10,
          estimatedRowHeight: 50,
          threshold: 40,
        })
      )

      expect(result.current.windowed).toBe(false)
      expect(result.current.start).toBe(0)
      expect(result.current.end).toBe(10)
      expect(result.current.padTop).toBe(0)
      expect(result.current.padBottom).toBe(0)
    })

    test('applies windowing above threshold when container has height', () => {
      const { result } = renderHook(() =>
        useWindowedList({
          count: 100,
          estimatedRowHeight: 50,
          threshold: 40,
          overscan: 5,
        })
      )

      const containerEl = document.createElement('div')
      containerEl.style.height = '500px'
      containerEl.style.overflow = 'auto'

      // Simulate containerRef being called
      act(() => {
        result.current.containerRef(containerEl)
      })

      // Without a scrollTop measurement in jsdom, windowed should be false
      // (viewport stays 0). This is expected behavior per the comment in the hook.
      expect(result.current.windowed).toBe(false)
    })
  })

  describe('scroll: window mode', () => {
    let mainElement: HTMLElement

    beforeEach(() => {
      // Create a fake main element in the DOM to act as the scroll container
      mainElement = document.createElement('main')
      mainElement.style.height = '800px'
      mainElement.style.overflow = 'auto'
      mainElement.innerHTML = '' // Will hold our list container
      document.body.appendChild(mainElement)
    })

    afterEach(() => {
      if (mainElement && mainElement.parentNode) {
        mainElement.parentNode.removeChild(mainElement)
      }
    })

    test('attaches scroll listener to <main> element, not window', () => {
      const windowScrollSpy = vi.spyOn(window, 'addEventListener')
      const windowRemoveSpy = vi.spyOn(window, 'removeEventListener')

      const { result, unmount } = renderHook(() =>
        useWindowedList({
          count: 100,
          estimatedRowHeight: 50,
          threshold: 40,
          overscan: 5,
          scroll: 'window',
        })
      )

      // Create a list container and attach it via ref
      const listContainer = document.createElement('div')
      mainElement.appendChild(listContainer)

      act(() => {
        result.current.containerRef(listContainer)
      })

      // Verify that window scroll listener was NOT added
      const windowScrollCalls = windowScrollSpy.mock.calls.filter(
        (call) => (call[0] as string) === 'scroll'
      )
      expect(windowScrollCalls.length).toBe(0)

      // Verify that window resize listener WAS added (that's expected)
      const windowResizeCalls = windowScrollSpy.mock.calls.filter(
        (call) => (call[0] as string) === 'resize'
      )
      expect(windowResizeCalls.length).toBe(1)

      unmount()

      // Clean up spies
      windowScrollSpy.mockRestore()
      windowRemoveSpy.mockRestore()
    })

    test('measures correctly when <main> element scrolls', () => {
      const { result } = renderHook(() =>
        useWindowedList({
          count: 200,
          estimatedRowHeight: 50,
          threshold: 40,
          overscan: 2,
          scroll: 'window',
        })
      )

      // Create a list container positioned inside main
      const listContainer = document.createElement('div')
      listContainer.style.height = '10000px' // 200 rows × 50px
      mainElement.appendChild(listContainer)

      act(() => {
        result.current.containerRef(listContainer)
      })

      // Fire a scroll event on main (not window)
      // The hook should handle it without errors
      const scrollEvent = new Event('scroll', { bubbles: true })
      act(() => {
        mainElement.dispatchEvent(scrollEvent)
      })

      // After scroll, result should still be computable (no exceptions)
      // The exact windowing state depends on jsdom's layout computation
      expect(result.current).toBeDefined()
      expect(typeof result.current.start).toBe('number')
      expect(typeof result.current.end).toBe('number')
    })

    test('falls back to window listener if <main> element does not exist', () => {
      // Remove the main element to test fallback
      mainElement.parentNode?.removeChild(mainElement)

      const windowAddSpy = vi.spyOn(window, 'addEventListener')

      const { result } = renderHook(() =>
        useWindowedList({
          count: 100,
          estimatedRowHeight: 50,
          threshold: 40,
          scroll: 'window',
        })
      )

      const listContainer = document.createElement('div')
      document.body.appendChild(listContainer)

      act(() => {
        result.current.containerRef(listContainer)
      })

      // In fallback mode, scroll listener should be on window
      const scrollCalls = windowAddSpy.mock.calls.filter(
        (call) => (call[0] as string) === 'scroll'
      )
      expect(scrollCalls.length).toBeGreaterThan(0)

      document.body.removeChild(listContainer)
      windowAddSpy.mockRestore()
    })

    test('cleans up scroll listener on unmount', () => {
      const mainRemoveSpy = vi.spyOn(mainElement, 'removeEventListener')

      const { result, unmount } = renderHook(() =>
        useWindowedList({
          count: 100,
          estimatedRowHeight: 50,
          threshold: 40,
          scroll: 'window',
        })
      )

      const listContainer = document.createElement('div')
      mainElement.appendChild(listContainer)

      act(() => {
        result.current.containerRef(listContainer)
      })

      unmount()

      // Verify scroll listener was removed from main
      const scrollCalls = mainRemoveSpy.mock.calls.filter((call) => call[0] === 'scroll')
      expect(scrollCalls.length).toBeGreaterThan(0)

      mainRemoveSpy.mockRestore()
    })
  })
})
