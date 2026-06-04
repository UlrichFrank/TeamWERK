type EventType = 'training' | 'heim' | 'auswärts' | 'generisch'

const EVENT_COLORS: Record<EventType, {
  card: { border: string; bg: string; icon: string }
  filter: string
  pill: string
  pillIcon: string
}> = {
  training: {
    card: { border: 'border-brand-green', bg: 'bg-brand-green/10', icon: 'text-brand-green' },
    filter: 'bg-brand-green text-white border-brand-green',
    pill: 'bg-brand-green/10 hover:bg-brand-green/20 border-brand-green/30',
    pillIcon: 'text-brand-green',
  },
  heim: {
    card: { border: 'border-brand-yellow', bg: 'bg-brand-yellow/15', icon: 'text-brand-yellow' },
    filter: 'bg-brand-yellow text-brand-black border-brand-yellow',
    pill: 'bg-brand-yellow/15 hover:bg-brand-yellow/25 border-brand-yellow/40',
    pillIcon: 'text-brand-yellow',
  },
  'auswärts': {
    card: { border: 'border-brand-blue', bg: 'bg-brand-blue/10', icon: 'text-brand-blue' },
    filter: 'bg-brand-blue text-white border-brand-blue',
    pill: 'bg-brand-blue/10 hover:bg-brand-blue/20 border-brand-blue/30',
    pillIcon: 'text-brand-blue',
  },
  generisch: {
    card: { border: 'border-brand-text-muted', bg: 'bg-brand-gray/40', icon: 'text-brand-text-muted' },
    filter: 'bg-brand-gray text-brand-black border-brand-gray',
    pill: 'bg-brand-gray/60 hover:bg-brand-gray border-brand-border',
    pillIcon: 'text-brand-text-muted',
  },
}

export function getEventColors(type: string) {
  return EVENT_COLORS[type as EventType] ?? EVENT_COLORS.generisch
}

export { EVENT_COLORS }
export type { EventType }
