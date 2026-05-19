import { useEffect, useState } from 'react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import MobileCard from '../components/MobileCard'

interface Account { user_id: number; name: string; soll: number; ist: number; balance: number }

export default function DutyAccountsPage() {
  const { user } = useAuth()
  const [accounts, setAccounts] = useState<Account[]>([])

  useEffect(() => { api.get('/duty-accounts').then(r => setAccounts(r.data ?? [])) }, [])

  const exportCSV = () =>
    api.get('/admin/duty-accounts/export', { responseType: 'blob' }).then(r => {
      const url = URL.createObjectURL(r.data)
      const a = document.createElement('a'); a.href = url; a.download = 'dienstkonten.csv'; a.click()
    })

  return (
    <div>
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between mb-6 gap-3 sm:gap-0">
        <h1 className="text-2xl font-bold">Dienstkonten</h1>
        {user?.role === 'admin' && (
          <button onClick={exportCSV} className="text-sm border border-black text-black rounded-md px-3 py-2.5 sm:py-1.5 hover:bg-brand-yellow hover:border-brand-yellow transition-colors">
            Export CSV
          </button>
        )}
      </div>

      {/* Mobile: Cards */}
      <div className="sm:hidden space-y-0">
        {accounts.map(a => (
          <MobileCard
            key={a.user_id}
            title={a.name}
            subtitle={`${a.soll.toFixed(1)}h Soll · ${a.ist.toFixed(1)}h Ist`}
            badge={{
              label: `${a.balance > 0 ? '+' : ''}${a.balance.toFixed(1)}h`,
              variant: a.balance > 0 ? 'red' : 'green',
            }}
          />
        ))}
      </div>

      {/* Desktop: Table */}
      <div className="hidden sm:block bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-gray-500 text-xs uppercase">
            <tr>
              {user?.role === 'admin' && <th className="px-4 py-3 text-left">Name</th>}
              <th className="px-4 py-3 text-right">Soll (h)</th>
              <th className="px-4 py-3 text-right">Ist (h)</th>
              <th className="px-4 py-3 text-right">Saldo</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {accounts.map(a => (
              <tr key={a.user_id} className="hover:bg-gray-50">
                {user?.role === 'admin' && <td className="px-4 py-3 font-medium">{a.name}</td>}
                <td className="px-4 py-3 text-right">{a.soll.toFixed(1)}</td>
                <td className="px-4 py-3 text-right">{a.ist.toFixed(1)}</td>
                <td className={`px-4 py-3 text-right font-medium ${a.balance > 0 ? 'text-brand-error' : 'text-brand-success'}`}>
                  {a.balance.toFixed(1)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
