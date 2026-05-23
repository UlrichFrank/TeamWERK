import { ChevronsLeft, ChevronsRight } from 'lucide-react'

interface Props {
  currentPage: number
  totalPages: number
  onPageChange: (page: number) => void
}

interface SlotDef {
  type: 'first' | 'page' | 'last'
  target: number | null
  label: string
  isActive: boolean
}

function buildSlots(currentPage: number, totalPages: number): SlotDef[] {
  const inRange = (n: number) => n >= 1 && n <= totalPages ? n : null

  return [
    { type: 'first', target: currentPage === 1 ? null : 1, label: '«', isActive: false },
    { type: 'page', target: inRange(currentPage - 3), label: inRange(currentPage - 3) !== null ? String(currentPage - 3) : '–', isActive: false },
    { type: 'page', target: inRange(currentPage - 1), label: inRange(currentPage - 1) !== null ? String(currentPage - 1) : '–', isActive: false },
    { type: 'page', target: currentPage, label: String(currentPage), isActive: true },
    { type: 'page', target: inRange(currentPage + 1), label: inRange(currentPage + 1) !== null ? String(currentPage + 1) : '–', isActive: false },
    { type: 'page', target: inRange(currentPage + 3), label: inRange(currentPage + 3) !== null ? String(currentPage + 3) : '–', isActive: false },
    { type: 'last', target: currentPage === totalPages ? null : totalPages, label: '»', isActive: false },
  ]
}

const BASE = 'w-10 h-10 flex items-center justify-center rounded-full text-sm font-medium'
const ACTIVE = `${BASE} bg-brand-black text-white font-semibold`
const NAVIGABLE = `${BASE} bg-brand-yellow text-brand-black transition-colors hover:bg-brand-black hover:text-brand-yellow cursor-pointer`
const DISABLED = `${BASE} bg-brand-yellow text-brand-black opacity-30 cursor-not-allowed`

export default function Pagination({ currentPage, totalPages, onPageChange }: Props) {
  if (totalPages <= 1) return null

  const slots = buildSlots(currentPage, totalPages)

  return (
    <nav aria-label="Seitennavigation" className="flex justify-center items-center gap-2 mt-8 mb-4">
      {slots.map((slot, i) => {
        const content = slot.type === 'first'
          ? <ChevronsLeft className="w-4 h-4" />
          : slot.type === 'last'
            ? <ChevronsRight className="w-4 h-4" />
            : slot.label

        if (slot.isActive) {
          return <span key={i} className={ACTIVE}>{content}</span>
        }
        if (slot.target !== null) {
          return (
            <button key={i} className={NAVIGABLE} onClick={() => onPageChange(slot.target!)}>
              {content}
            </button>
          )
        }
        return <span key={i} className={DISABLED}>{content}</span>
      })}
    </nav>
  )
}
