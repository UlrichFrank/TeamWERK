import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'

interface Member {
  id: number; first_name: string; last_name: string
  status: string; pass_number?: string; position?: string
}

export default function MembersPage() {
  const { user } = useAuth()
  const [members, setMembers] = useState<Member[]>([])
  const [search, setSearch] = useState('')
  const isAdmin = user?.role === 'admin'

  useEffect(() => { api.get('/members').then(r => setMembers(r.data ?? [])) }, [])

  const filtered = members.filter(m =>
    `${m.first_name} ${m.last_name}`.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">Mitglieder</h1>
        <div className="flex gap-2">
          <input
            type="search" placeholder="Suchen…" value={search} onChange={e => setSearch(e.target.value)}
            className="border border-gray-300 rounded-md px-3 py-1.5 text-sm"
          />
          {isAdmin && (
            <>
              <Link
                to="/mitglieder/neu"
                className="text-sm bg-brand-blue text-white border border-brand-blue rounded-md px-3 py-1.5 hover:bg-brand-blue-dark"
              >
                + Neu
              </Link>
              <button
                onClick={() => api.get('/members/export', { responseType: 'blob' }).then(r => {
                  const url = URL.createObjectURL(r.data)
                  const a = document.createElement('a'); a.href = url; a.download = 'mitglieder.csv'; a.click()
                })}
                className="text-sm text-brand-blue border border-brand-blue rounded-md px-3 py-1.5 hover:bg-brand-blue hover:text-white"
              >
                Export CSV
              </button>
            </>
          )}
        </div>
      </div>
      <div className="bg-white rounded-xl shadow overflow-hidden">
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
            {filtered.map(m => (
              <tr key={m.id} className="hover:bg-gray-50">
                <td className="px-4 py-3 font-medium">
                  <Link to={`/mitglieder/${m.id}`} className="hover:text-brand-blue">
                    {m.last_name}, {m.first_name}
                  </Link>
                </td>
                <td className="px-4 py-3 text-gray-500">{m.pass_number || '–'}</td>
                <td className="px-4 py-3 text-gray-500">{m.position || '–'}</td>
                <td className="px-4 py-3">
                  <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
                    m.status === 'aktiv' ? 'bg-green-100 text-green-700' :
                    m.status === 'verletzt' ? 'bg-yellow-100 text-yellow-700' :
                    'bg-gray-100 text-gray-600'
                  }`}>{m.status}</span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
