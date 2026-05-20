interface Member {
  id: number
  name: string
  birth_year: number
  gender: string
  positions?: string[]
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
  return members.filter(m => m.positions?.includes(positionName)).length
}

function getCircleColor(count: number): string {
  if (count === 0) return 'bg-red-500'
  if (count === 1) return 'bg-brand-yellow'
  if (count === 2) return 'bg-brand-green'
  return 'bg-blue-500'
}

interface PositionStatusProps {
  members: Member[]
}

export default function PositionStatus({ members }: PositionStatusProps) {
  return (
    <div className="flex gap-3 items-start py-2 text-xs">
      {POSITIONS.map(pos => {
        const count = countMembersForPosition(members, pos.name)
        const circleColor = getCircleColor(count)
        const circleCount = count === 0 ? 1 : count === 1 ? 1 : Math.min(count, 3)

        return (
          <div key={pos.abbr} className="flex items-center gap-1">
            <span className="font-medium text-gray-700 whitespace-nowrap">{pos.abbr}</span>
            <div className="flex flex-col gap-1">
              {Array.from({ length: circleCount }).map((_, i) => (
                <div
                  key={i}
                  className={`w-3.5 h-3.5 rounded-full ${circleColor}`}
                />
              ))}
            </div>
          </div>
        )
      })}
    </div>
  )
}
