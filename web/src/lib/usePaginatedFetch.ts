import { useEffect, useState, useRef } from 'react'
import { api } from './api'

interface PaginatedResponse<T> {
  items: T[]
  total: number
}

interface UsePaginatedFetchState<T> {
  items: T[]
  total: number
  offset: number
  loading: boolean
  error: string | null
}

interface UsePaginatedFetchActions {
  loadMore: () => void
  setSearch: (search: string) => void
  reset: () => void
}

export function usePaginatedFetch<T>(endpoint: string, limit = 50): UsePaginatedFetchState<T> & UsePaginatedFetchActions {
  const [state, setState] = useState<UsePaginatedFetchState<T>>({
    items: [],
    total: 0,
    offset: 0,
    loading: false,
    error: null,
  })
  const [search, setSearchImmediate] = useState('')
  const searchTimeoutRef = useRef<ReturnType<typeof setTimeout>>()

  const fetchData = async (searchVal: string, offsetVal: number) => {
    setState(prev => ({ ...prev, loading: true, error: null }))
    try {
      const params = new URLSearchParams()
      if (searchVal) params.append('search', searchVal)
      params.append('limit', String(limit))
      params.append('offset', String(offsetVal))

      const response = await api.get<PaginatedResponse<T>>(`${endpoint}?${params}`)
      const { items, total } = response.data

      setState(prev => ({
        ...prev,
        items: offsetVal === 0 ? items : [...prev.items, ...items],
        total,
        offset: offsetVal + items.length,
        loading: false,
      }))
    } catch (err) {
      setState(prev => ({
        ...prev,
        loading: false,
        error: err instanceof Error ? err.message : 'Unknown error',
      }))
    }
  }

  const setSearch = (newSearch: string) => {
    setSearchImmediate(newSearch)
    if (searchTimeoutRef.current) {
      clearTimeout(searchTimeoutRef.current)
    }
    searchTimeoutRef.current = setTimeout(() => {
      setState(prev => ({ ...prev, items: [], total: 0, offset: 0 }))
      fetchData(newSearch, 0)
    }, 300)
  }

  const loadMore = () => {
    fetchData(search, state.offset)
  }

  const reset = () => {
    setState({
      items: [],
      total: 0,
      offset: 0,
      loading: false,
      error: null,
    })
    setSearchImmediate('')
    fetchData('', 0)
  }

  useEffect(() => {
    fetchData(search, 0)
  }, [])

  return {
    ...state,
    setSearch,
    loadMore,
    reset,
  }
}
