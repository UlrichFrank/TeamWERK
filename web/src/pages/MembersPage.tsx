import { Link } from 'react-router-dom'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import { usePaginatedFetch } from '../lib/usePaginatedFetch'
import MobileCard from '../components/MobileCard'

interface Member {
  id: number; first_name: string; last_name: string
  status: string; pass_number?: string; position?: string
}

const statusBadgeStyles = (status: string) => {
  if (status === 'aktiv') return 'bg-brand-black text-white'
  if (status === 'verletzt') return 'bg-brand-yellow text-brand-black'
  return 'bg-gray-200 text-gray-600'
}

export default function MembersPage() {
  const { user } = useAuth()
  const { items, total, loading, setSearch, loadMore } = usePaginatedFetch<Member>('/members')
  const isAdmin = user?.role === 'admin'

  return (
    <div>
      <div className="sticky top-0 z-10 bg-brand-white pb-4 mb-4 sm:bg-transparent sm:pb-6 sm:mb-0 sm:static sm:z-auto">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 sm:gap-0">
          <h1 className="text-2xl font-bold">Mitglieder</h1>
          <div className="flex flex-col sm:flex-row gap-2">
            <input
              type="search"
              placeholder="Suchen…"
              onChange={e => setSearch(e.target.value)}
              className="border border-gray-300 rounded-md px-3 py-2.5 sm:py-1.5 text-sm w-full sm:w-auto"
            />
            {isAdmin && (
              <>
                <Link
                  to="/mitglieder/neu"
                  className="text-sm bg-brand-yellow text-brand-black border border-brand-yellow rounded-md px-3 py-2.5 sm:py-1.5 font-medium hover:bg-brand-black hover:text-brand-yellow hover:border-brand-black transition-colors text-center"
                >
                  + Neu
                </Link>
                <button
                  onClick={() => api.get('/members/export', { responseType: 'blob' }).then(r => {
                    const url = URL.createObjectURL(r.data)
                    const a = document.createElement('a'); a.href = url; a.download = 'mitglieder.csv'; a.click()
                  })}
                  className="text-sm border border-brand-black text-brand-black rounded-md px-3 py-2.5 sm:py-1.5 hover:bg-brand-yellow hover:border-brand-yellow transition-colors"
                >
                  Export CSV
                </button>
              </>
            )}
          </div>
        </div>
      </div>

      {/* Mobile: Cards */}
      <div className="sm:hidden space-y-0">
        {items.map(m => (
          <Link key={m.id} to={`/mitglieder/${m.id}`} className="block">
            <MobileCard
              title={`${m.last_name}, ${m.first_name}`}
              subtitle={m.position || '–'}
              badge={{ label: m.status, variant: m.status === 'aktiv' ? 'blue' : m.status === 'verletzt' ? 'yellow' : 'red' }}
            />
          </Link>
        ))}
      </div>

      {/* Desktop: Table */}
      <div className="hidden sm:block bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-gray-500 text-xs uppercase">
            <tr>
              <th className="px-4 py-3 text-left">Name</th>
              <th className="px-4 py-3 text-left">Passnummer</th>
              <th className="px-4 py-3 text-left">Position</th>
              <th className="px-4 py-3 text-left">Status</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {items.map(m => (
              <tr key={m.id} className="hover:bg-gray-50">
                <td className="px-4 py-3 font-medium">
                  <Link to={`/mitglieder/${m.id}`} className="hover:text-brand-yellow transition-colors">
                    {m.last_name}, {m.first_name}
                  </Link>
                </td>
                <td className="px-4 py-3 text-gray-500">{m.pass_number || '–'}</td>
                <td className="px-4 py-3 text-gray-500">{m.position || '–'}</td>
                <td className="px-4 py-3">
                  <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusBadgeStyles(m.status)}`}>
                    {m.status}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Load More Button */}
      {items.length < total && (
        <div className="mt-6 text-center">
          <button
            onClick={loadMore}
            disabled={loading}
            className="px-6 py-2.5 sm:py-2 bg-brand-yellow text-brand-black rounded font-medium hover:bg-brand-black hover:text-brand-yellow disabled:opacity-50 transition-colors"
          >
            {loading ? 'Lädt…' : `Mehr laden (${items.length}/${total})`}
          </button>
        </div>
      )}
    </div>
  )
}
