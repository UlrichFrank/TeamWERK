// Zentrale Berechnung der Chat-Ungelesen-Zahlen.
//
// Die Formel stand vorher an zwei Stellen (AppShell, ChatPage) und wäre mit dem
// Dashboard auf drei gewachsen. Driftgefährdet ist dabei vor allem die
// Broadcast-Bedingung: eine *eigene* Mitteilung (`isSent`) ist für den Absender
// nie ungelesen, auch wenn `isRead` false bleibt — das reproduziert ein dritter
// Aufschreiber nicht zuverlässig.
//
// Reine Funktion ohne React und ohne Netzwerk, damit sie ohne Component-Render
// testbar ist (analog `createThrottledProgress` in VideoUploadPage.tsx).

// Bewusst schmale Strukturtypen: die Aufrufstellen behalten ihre eigenen,
// vollständigen Interfaces und passen ohne Cast hier hinein.
export interface UnreadConversation {
  unreadCount: number
}

export interface UnreadBroadcast {
  isRead: boolean
  isSent: boolean
}

export interface ChatUnreadCounts {
  /** Summe der ungelesenen Nachrichten über alle Konversationen. */
  conversations: number
  /** Anzahl ungelesener Mitteilungen, die der Nutzer nicht selbst gesendet hat. */
  broadcasts: number
  /** Summe beider Anteile — die Zahl, die für „Nachrichten" als Ganzes steht. */
  total: number
}

export function chatUnreadCounts(
  conversations: readonly UnreadConversation[] | null | undefined,
  broadcasts: readonly UnreadBroadcast[] | null | undefined,
): ChatUnreadCounts {
  const conv = (conversations ?? []).reduce((sum, c) => sum + (c.unreadCount ?? 0), 0)
  const bc = (broadcasts ?? []).filter(b => !b.isRead && !b.isSent).length
  return { conversations: conv, broadcasts: bc, total: conv + bc }
}
