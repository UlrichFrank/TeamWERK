import { useEffect, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { api } from './api'

interface PaginatedResponse<T> {
  items: T[]
  total: number
}

export function usePagination<T>(endpoint: string, limit = 20, extraParams: Record<string, string> = {}) {
  const [searchParams, setSearchParams] = useSearchParams()
  const currentPage = Math.max(1, Number(searchParams.get('page') || '1'))
  const currentSearch = searchParams.get('search') || ''

  const [items, setItems] = useState<T[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [refreshTick, setRefreshTick] = useState(0)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)

  const extraParamsKey = Object.entries(extraParams).map(([k, v]) => `${k}=${v}`).sort().join('&')

  const totalPages = total > 0 ? Math.ceil(total / limit) : 1

  useEffect(() => {
    let cancelled = false

    async function fetchData() {
      setLoading(true)
      setError(null)
      try {
        const params = new URLSearchParams()
        if (currentSearch) params.append('search', currentSearch)
        params.append('limit', String(limit))
        params.append('offset', String((currentPage - 1) * limit))
        for (const [k, v] of Object.entries(extraParams)) {
          if (v) params.append(k, v)
        }

        const res = await api.get<PaginatedResponse<T>>(`${endpoint}?${params}`)
        if (cancelled) return

        const { items: fetched, total: fetchedTotal } = res.data
        setItems(fetched)
        setTotal(fetchedTotal)

        const newTotalPages = fetchedTotal > 0 ? Math.ceil(fetchedTotal / limit) : 1
        if (currentPage > newTotalPages && newTotalPages > 0) {
          setSearchParams({ page: String(newTotalPages), search: currentSearch })
        }
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Unknown error')
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    fetchData()
    return () => { cancelled = true }
    // extraParamsKey serialisiert extraParams stabil (statt Objektidentität); endpoint/limit/setSearchParams gelten als stabil
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentPage, currentSearch, refreshTick, extraParamsKey])

  function setSearch(val: string) {
    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => {
      setSearchParams({ page: '1', search: val })
    }, 300)
  }

  function goToPage(page: number) {
    setSearchParams({ page: String(page), search: currentSearch })
  }

  function refresh() {
    setRefreshTick(t => t + 1)
  }

  return { items, total, currentPage, totalPages, loading, error, setSearch, goToPage, refresh }
}
