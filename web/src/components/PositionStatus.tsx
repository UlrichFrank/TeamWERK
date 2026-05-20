interface Member {
  id: number
  name: string
  birth_year: number
  gender: string
  positions?: string | null
}

interface Position {
  name: string
  abbr: string
}

const POSITIONS: Position[] = [
  { name: 'Torwart', abbr: 'TW' },
  { name: 'Linksaußen', abbr: 'LA' },
  { name: 'Rechtsaußen', abbr: 'RA' },
  { name: 'Rückraum Links', abbr: 'RL' },
  { name: 'Rückraum Mitte', abbr: 'RM' },
  { name: 'Rückraum Rechts', abbr: 'RR' },
  { name: 'Kreisläufer', abbr: 'KL' },
]

function countMembersForPosition(members: Member[], positionName: string): number {
  return members.filter(m => {
    if (!m.positions) return false
    const positions = m.positions.split(',').map(p => p.trim())
    return positions.includes(positionName)
  }).length
}

function getCircleClass(count: number): string {
  if (count === 0) return 'border-2 border-red-500 bg-white'
  if (count === 1) return 'bg-brand-yellow'
  if (count === 2) return 'bg-brand-green'
  return 'bg-blue-500'
}

interface PositionStatusProps {
  members: Member[]
}

export default function PositionStatus({ members }: PositionStatusProps) {
  return (
    <div className="flex gap-3 items-end py-2 text-xs">
      {POSITIONS.map(pos => {
        const count = countMembersForPosition(members, pos.name)
        const circleClass = getCircleClass(count)
        const circleCount = count === 0 ? 1 : count === 1 ? 1 : Math.min(count, 3)

        return (
          <div key={pos.abbr} className="flex items-end gap-1">
            <span className="font-medium text-gray-700 whitespace-nowrap">{pos.abbr}</span>
            <div className="flex flex-col-reverse gap-0.5">
              {Array.from({ length: circleCount }).map((_, i) => (
                <div
                  key={i}
                  className={`w-2.5 h-2.5 rounded-full ${circleClass}`}
                />
              ))}
            </div>
          </div>
        )
      })}
    </div>
  )
}
