// Sitzungs-Cache für den entsperrten Gruppen-Privatschlüssel (Zero-Knowledge-Tresor, Modell B).
//
// Der Schlüssel liegt als nicht-exportierbares `CryptoKey`-Objekt in IndexedDB: Structured
// Clone speichert das Schlüssel-Handle, nicht das Schlüsselmaterial. Ein XSS kann den
// Schlüssel damit zwar *benutzen*, solange die Seite offen ist, aber nicht *exfiltrieren* —
// anders als beim früheren PKCS8-Base64 in `sessionStorage`, das jedes Skript auslesen konnte.
//
// Alle Operationen sind bewusst fehlertolerant: ohne IndexedDB (Safari-Private-Mode,
// blockierte Storage-APIs, Quota) lösen sie mit `null` bzw. no-op auf und werfen nie. Der
// Aufrufer fällt dann auf „Schlüssel nur im React-State" zurück — der Tresor ist nach einem
// Reload gesperrt, aber nichts geht kaputt.

const DB_NAME = 'teamwerk-vault'
const STORE = 'keys'
const KEY = 'groupPriv'

function openDB(): Promise<IDBDatabase | null> {
  return new Promise(resolve => {
    try {
      if (typeof indexedDB === 'undefined' || indexedDB === null) return resolve(null)
      const req = indexedDB.open(DB_NAME, 1)
      req.onupgradeneeded = () => {
        const db = req.result
        if (!db.objectStoreNames.contains(STORE)) db.createObjectStore(STORE)
      }
      req.onsuccess = () => resolve(req.result)
      req.onerror = () => resolve(null)
      req.onblocked = () => resolve(null)
    } catch {
      resolve(null)
    }
  })
}

// Eine Transaktion ausführen und immer auflösen — Fehler werden zu `fallback`.
function run<T>(
  mode: IDBTransactionMode,
  op: (store: IDBObjectStore) => IDBRequest,
  fallback: T,
): Promise<T> {
  return openDB().then(
    db =>
      new Promise<T>(resolve => {
        if (!db) return resolve(fallback)
        const done = (value: T) => {
          try {
            db.close()
          } catch {
            /* egal */
          }
          resolve(value)
        }
        try {
          const tx = db.transaction(STORE, mode)
          const req = op(tx.objectStore(STORE))
          req.onsuccess = () => done((req.result ?? fallback) as T)
          req.onerror = () => done(fallback)
          tx.onabort = () => done(fallback)
          tx.onerror = () => done(fallback)
        } catch {
          done(fallback)
        }
      }),
  )
}

export function put(key: CryptoKey): Promise<void> {
  return run<unknown>('readwrite', store => store.put(key, KEY), null).then(() => undefined)
}

export function get(): Promise<CryptoKey | null> {
  return run<unknown>('readonly', store => store.get(KEY), null).then(v =>
    // Defensiv: ein Eintrag aus einer früheren Version könnte etwas anderes als einen
    // CryptoKey enthalten — dann lieber „kein Schlüssel" als ein kaputtes Objekt.
    typeof v === 'object' && v !== null && 'algorithm' in v ? (v as CryptoKey) : null,
  )
}

export function clear(): Promise<void> {
  return run<unknown>('readwrite', store => store.delete(KEY), null).then(() => undefined)
}
