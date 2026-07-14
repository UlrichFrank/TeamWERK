import {
  useState,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useCallback,
} from "react";
import { useSearchParams } from "react-router-dom";
import {
  Send,
  Plus,
  LogOut,
  MessageSquare,
  Megaphone,
  X,
  Search,
  Users,
  Trash2,
  CornerUpLeft,
  Pencil,
  SmilePlus,
  Copy,
  Paperclip,
} from "lucide-react";
import { api } from "../lib/api";
import { compressImage } from "../lib/imageCompress";
import AuthImage from "../components/AuthImage";
import { buildTeamShortNames } from "../lib/teamName";
import {
  conversationTimeLabel,
  daySeparatorLabel,
  shouldRenderSeparator,
} from "../lib/chatDateFormat";
import { errorMessage } from "../lib/errors";
import { DaySeparator } from "../components/DaySeparator";
import WindowedRows from "../components/WindowedRows";
import { useAuth } from "../contexts/AuthContext";
import { useChatEvents } from "../hooks/useChatEvents";
import ConversationParticipantsModal from "../components/ConversationParticipantsModal";
import CreatorExitChoiceModal from "../components/CreatorExitChoiceModal";

interface ConvMember {
  id: number;
  name: string;
}
interface LastMessage {
  body: string;
  sentAt: string;
}
interface Conversation {
  id: number;
  type: "direct" | "group";
  name: string | null;
  createdBy: number;
  unreadCount: number;
  lastMessage: LastMessage | null;
  members: ConvMember[];
}
interface Reaction {
  emoji: string;
  count: number;
  userNames: string[];
  myReaction: boolean;
}

interface Message {
  id: number;
  senderId: number;
  senderName: string;
  // Liste liefert nur einen gekürzten Preview (≤280 Zeichen) + truncated-Flag;
  // der Volltext wird bei Bedarf über GET /chat/messages/{id} nachgeladen.
  preview: string;
  truncated: boolean;
  sentAt: string;
  replyToId: number | null;
  replyToBody: string | null;
  replyToSenderName: string | null;
  editedAt: string | null;
  deletedAt: string | null;
  isSystem: boolean;
  mediaId: number | null;
  mediaUrl: string | null;
  reactions: Reaction[];
}

const REACTION_EMOJIS = ["👍", "👎", "❤️", "😂", "😮", "😢", "🙌", "🔥"];
// Serverseitige Seitengröße von GET /chat/conversations/{id}/messages —
// eine volle Seite heißt: es kann noch ältere Nachrichten geben (?before=).
const MESSAGE_PAGE_SIZE = 100;
interface Broadcast {
  id: number;
  senderName: string;
  body: string;
  sentAt: string;
  isRead: boolean;
  isSent: boolean;
  editedAt: string | null;
  mediaId: number | null;
  mediaUrl: string | null;
}
interface ChatUser {
  id: number;
  name: string;
}

interface TeamGroup {
  teamId: number;
  displayShort: string;
  kind: "trainer" | "spieler" | "eltern" | "alle_trainer";
  count: number;
}

const TEAM_GROUP_KIND_LABEL: Record<TeamGroup["kind"], string> = {
  trainer: "Trainer",
  spieler: "Spieler",
  eltern: "Eltern",
  alle_trainer: "Alle Trainer",
};

type Tab = "chats" | "broadcasts";

// Nicht abgesendete Nachrichten-Entwürfe pro Konversation überleben Reload,
// Tab-Wechsel und App-Schließen via localStorage (pro Nutzer gescoped, damit
// Entwürfe auf einem geteilten Gerät nicht zwischen Konten lecken).
function loadChatDrafts(key: string | null): Map<number, string> {
  if (!key) return new Map();
  try {
    const raw = localStorage.getItem(key);
    if (!raw) return new Map();
    const obj = JSON.parse(raw) as Record<string, string>;
    return new Map(
      Object.entries(obj)
        .filter(([, v]) => typeof v === "string" && v.length > 0)
        .map(([k, v]) => [Number(k), v]),
    );
  } catch {
    return new Map();
  }
}

function saveChatDrafts(key: string | null, map: Map<number, string>): void {
  if (!key) return;
  try {
    if (map.size === 0) {
      localStorage.removeItem(key);
      return;
    }
    const obj: Record<string, string> = {};
    map.forEach((v, k) => {
      obj[String(k)] = v;
    });
    localStorage.setItem(key, JSON.stringify(obj));
  } catch {
    /* localStorage nicht verfügbar (Privatmodus/Quota) → Entwurf bleibt in-memory */
  }
}

interface ContextMenuState {
  x: number;
  y: number;
  message: Message;
  selectedText?: string;
}

export default function ChatPage() {
  const { user, hasCapability } = useAuth();
  const [searchParams, setSearchParams] = useSearchParams();
  const [tab, setTab] = useState<Tab>("chats");
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [broadcasts, setBroadcasts] = useState<Broadcast[]>([]);
  const [activeConv, setActiveConv] = useState<Conversation | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  // Cache für nachgeladene Volltexte gekürzter Nachrichten (id → body).
  const [fullBodies, setFullBodies] = useState<Record<number, string>>({});
  const [msgInput, setMsgInput] = useState("");
  const [sending, setSending] = useState(false);
  const [showNewModal, setShowNewModal] = useState(false);
  const [showBroadcastModal, setShowBroadcastModal] = useState(false);
  const [showParticipants, setShowParticipants] = useState(false);
  const [showCreatorExit, setShowCreatorExit] = useState(false);
  const [toast, setToast] = useState<string | null>(null);
  const [activeBroadcast, setActiveBroadcast] = useState<Broadcast | null>(
    null,
  );
  const [showBroadcastEdit, setShowBroadcastEdit] = useState(false);
  const [replyTo, setReplyTo] = useState<Message | null>(null);
  const [editingMessage, setEditingMessage] = useState<Message | null>(null);
  // Ausgewähltes, noch nicht gesendetes Bild (Chat-Tab) inkl. lokaler Vorschau.
  const [pendingImage, setPendingImage] = useState<{
    file: File;
    previewUrl: string;
  } | null>(null);
  // Bild im Vollbild-Overlay (Lightbox), url ohne /api-Prefix.
  const [lightboxUrl, setLightboxUrl] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null);
  const [mobileOverlay, setMobileOverlay] = useState<{
    message: Message;
    isOwn: boolean;
  } | null>(null);
  const [emojiPickerMsgId, setEmojiPickerMsgId] = useState<number | null>(null);
  const [hasOlder, setHasOlder] = useState(false);
  const [loadingOlder, setLoadingOlder] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesBoxRef = useRef<HTMLDivElement>(null);
  const suppressAutoScrollRef = useRef(false);
  // Erzwingt einmalig einen Scroll ans Ende beim nächsten messages-Update
  // (nach eigenem Senden / Öffnen einer Konversation), unabhängig davon, ob
  // der Nutzer gerade hochgescrollt ist.
  const forceScrollToEndRef = useRef(false);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const contextMenuRef = useRef<HTMLDivElement>(null);
  const draftsRef = useRef<Map<number, string>>(new Map());
  const draftsKey = user ? `teamwerk:chat-drafts:${user.id}` : null;

  // Persistierte Entwürfe laden, sobald der Nutzer feststeht (Reload/App-Start).
  useEffect(() => {
    draftsRef.current = loadChatDrafts(draftsKey);
  }, [draftsKey]);

  // Aktuellen Entwurf der aktiven Konversation bei jeder Änderung persistieren,
  // damit er auch ohne Konversationswechsel Reload/App-Schließen überlebt.
  // Während einer Edit-Session hält msgInput den Nachrichtentext, keinen Entwurf.
  useEffect(() => {
    if (!activeConv || editingMessage) return;
    if (msgInput) draftsRef.current.set(activeConv.id, msgInput);
    else draftsRef.current.delete(activeConv.id);
    saveChatDrafts(draftsKey, draftsRef.current);
  }, [msgInput, activeConv, editingMessage, draftsKey]);

  useEffect(() => {
    const el = inputRef.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = `${Math.min(el.scrollHeight, 160)}px`;
  }, [msgInput]);

  const isMobile = window.innerWidth < 640;
  const [mobileShowChat, setMobileShowChat] = useState(false);

  const canBroadcast = hasCapability("broadcast_messages");

  const loadConversations = useCallback(async () => {
    try {
      const r = await api.get("/chat/conversations");
      setConversations(r.data ?? []);
    } catch {}
  }, []);

  const reloadActiveConv = useCallback(async (convId: number) => {
    try {
      const r = await api.get("/chat/conversations");
      const updated = (r.data ?? []).find((c: Conversation) => c.id === convId);
      if (updated) setActiveConv(updated);
      setConversations(r.data ?? []);
    } catch {}
  }, []);

  const loadBroadcasts = useCallback(async () => {
    try {
      const r = await api.get("/chat/broadcasts");
      setBroadcasts(r.data ?? []);
    } catch {}
  }, []);

  const loadMessages = async (convId: number) => {
    try {
      const r = await api.get(`/chat/conversations/${convId}/messages`);
      const msgs: Message[] = r.data ?? [];
      setMessages(msgs);
      setHasOlder(msgs.length === MESSAGE_PAGE_SIZE);
      setEmojiPickerMsgId(null);
      await api.post(`/chat/conversations/${convId}/read`);
      loadConversations();
    } catch {}
  };

  // Inkrementelles Nachladen (incremental-sync, id-Cursor): holt per
  // ?after=<letzte bekannte id> nur die neueren Nachrichten und hängt sie an,
  // statt die ganze Konversation neu zu laden. Ein leeres Delta heißt: das
  // Event kam von einer Mutation einer bestehenden Nachricht (Edit/Löschung/
  // Reaktion — der Server sendet dafür dasselbe chat:new-message-Event) →
  // dann ist der Voll-Reload weiterhin nötig.
  const appendNewMessages = async (convId: number) => {
    const lastId = messages.length > 0 ? messages[messages.length - 1].id : 0;
    if (lastId === 0) {
      await loadMessages(convId);
      return;
    }
    try {
      const r = await api.get(`/chat/conversations/${convId}/messages`, {
        params: { after: lastId },
      });
      const newMsgs: Message[] = r.data ?? [];
      if (newMsgs.length === 0) {
        await loadMessages(convId);
        return;
      }
      setMessages((prev) => {
        const seen = new Set(prev.map((m) => m.id));
        return [...prev, ...newMsgs.filter((m) => !seen.has(m.id))];
      });
      await api.post(`/chat/conversations/${convId}/read`);
      loadConversations();
    } catch {}
  };

  // Verlaufs-Scroll: lädt per ?before=<älteste bekannte id> die vorherige
  // Seite und stellt sie voran; Scroll-Position bleibt stabil.
  const loadOlderMessages = async () => {
    if (!activeConv || loadingOlder || messages.length === 0) return;
    setLoadingOlder(true);
    try {
      const r = await api.get(`/chat/conversations/${activeConv.id}/messages`, {
        params: { before: messages[0].id },
      });
      const older: Message[] = r.data ?? [];
      setHasOlder(older.length === MESSAGE_PAGE_SIZE);
      if (older.length > 0) {
        // Scroll-Container ist der interne WindowedRows-Div (Windowing, ⑤):
        // Position beim Voranstellen erhalten, damit die Ansicht nicht springt.
        const box =
          messagesBoxRef.current?.querySelector<HTMLDivElement>(
            "[data-windowed-scroll]",
          ) ?? null;
        const prevHeight = box?.scrollHeight ?? 0;
        suppressAutoScrollRef.current = true;
        setMessages((prev) => [...older, ...prev]);
        requestAnimationFrame(() => {
          if (box) box.scrollTop += box.scrollHeight - prevHeight;
        });
      }
    } catch {
    } finally {
      setLoadingOlder(false);
    }
  };

  const openConversation = async (conv: Conversation) => {
    // Aktuellen Entwurf für die bisherige Konversation sichern, damit er beim
    // Zurückwechseln erhalten bleibt. Während einer Edit-Session enthält
    // msgInput den bearbeiteten Nachrichtentext, nicht den Entwurf → nicht
    // überschreiben.
    if (activeConv && activeConv.id !== conv.id && !editingMessage) {
      if (msgInput) draftsRef.current.set(activeConv.id, msgInput);
      else draftsRef.current.delete(activeConv.id);
    }
    setActiveConv(conv);
    setMobileShowChat(true);
    setReplyTo(null);
    setEditingMessage(null);
    setMsgInput(draftsRef.current.get(conv.id) ?? "");
    // Beim Öffnen einer Konversation den Sticky-Guard einmalig überstimmen,
    // damit der loadMessages-State-Update in jedem Fall ans Ende scrollt.
    forceScrollToEndRef.current = true;
    await loadMessages(conv.id);
  };

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-/Abhängigkeits-getrieben), kein Ableitungs-Bug
    loadConversations();
    loadBroadcasts();
  }, [loadConversations, loadBroadcasts]);

  useEffect(() => {
    const openUser = searchParams.get("openUser");
    if (!openUser) return;
    setSearchParams({}, { replace: true });
    api
      .post("/chat/conversations", { type: "direct", userId: Number(openUser) })
      .then((r) => {
        const conv: Conversation = r.data;
        setConversations((prev) =>
          prev.some((c) => c.id === conv.id) ? prev : [conv, ...prev],
        );
        setActiveConv(conv);
        setTab("chats");
        if (isMobile) setMobileShowChat(true);
        loadMessages(conv.id);
      })
      .catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Deep-Link aus Chat-Push: /chat?conv=<id> öffnet die Unterhaltung,
  // /chat?tab=broadcasts springt in den Broadcasts-Tab. Einmalig konsumieren
  // (Guard-Ref), sonst würde jeder conversations-Reload via SSE die Auswahl
  // erneut umschalten.
  const deepLinkConsumed = useRef(false);
  useEffect(() => {
    if (deepLinkConsumed.current) return;
    const tabParam = searchParams.get("tab");
    if (tabParam === "broadcasts") {
      deepLinkConsumed.current = true;
      // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-/Abhängigkeits-getrieben), kein Ableitungs-Bug
      setTab("broadcasts");
      setSearchParams({}, { replace: true });
      return;
    }
    const convParam = searchParams.get("conv");
    if (!convParam) return;
    const conv = conversations.find((c) => c.id === Number(convParam));
    if (!conv) return; // conversations noch nicht geladen → bei nächstem Reload erneut prüfen
    deepLinkConsumed.current = true;
    setTab("chats");
    setSearchParams({}, { replace: true });
    openConversation(conv);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams, conversations]);

  useChatEvents((event) => {
    if (event.startsWith("chat:new-message")) {
      const parts = event.split(":");
      const convId = parseInt(parts[2]);
      if (activeConv?.id === convId) {
        // hängt das Delta an (?after=) und aktualisiert die Liste mit;
        // Voll-Reload nur noch als Fallback bei leerem Delta
        appendNewMessages(convId);
      } else {
        loadConversations();
      }
    }
    if (event.startsWith("chat:member-left")) {
      loadConversations();
      const parts = event.split(":");
      const convId = parseInt(parts[2]);
      if (activeConv?.id === convId) loadMessages(convId);
    }
    if (event.startsWith("chat:conv-updated")) {
      const parts = event.split(":");
      const convId = parseInt(parts[2]);
      if (activeConv?.id === convId) {
        reloadActiveConv(convId);
      } else {
        loadConversations();
      }
    }
    if (event.startsWith("chat:conv-deleted")) {
      const parts = event.split(":");
      const convId = parseInt(parts[2]);
      setConversations((prev) => prev.filter((c) => c.id !== convId));
      if (activeConv?.id === convId) {
        setActiveConv(null);
        setMobileShowChat(false);
        setShowParticipants(false);
        setToast("Die Gruppe wurde gelöscht");
        setTimeout(() => setToast(null), 4000);
      }
    }
    if (event === "chat:new-broadcast") loadBroadcasts();
  });

  const toggleReaction = async (msgId: number, emoji: string) => {
    try {
      await api.post(`/chat/messages/${msgId}/reactions`, { emoji });
      if (activeConv) loadMessages(activeConv.id);
    } catch {}
  };

  useEffect(() => {
    // Beim Voranstellen älterer Nachrichten (?before=) nicht ans Ende springen
    if (suppressAutoScrollRef.current) {
      suppressAutoScrollRef.current = false;
      return;
    }
    // Sticky-to-Bottom: nur automatisch ans Ende scrollen, wenn der Nutzer
    // eh in der Nähe des Endes steht (oder wir explizit forcieren, z. B. beim
    // eigenen Senden / beim Öffnen einer Konversation). Ohne diesen Guard reißt
    // jeder Reactions-Toggle / eingehende SSE-Message den hochgescrollten
    // Nutzer zurück ans Ende (Symptom „Position springt ständig") — messages
    // ändert sich häufig, auch ohne dass etwas Neues am Ende dazugekommen ist.
    if (!forceScrollToEndRef.current) {
      const box = messagesBoxRef.current?.querySelector<HTMLDivElement>(
        "[data-windowed-scroll]",
      );
      if (box) {
        const distanceFromBottom =
          box.scrollHeight - box.scrollTop - box.clientHeight;
        if (distanceFromBottom > 100) return;
      }
    }
    forceScrollToEndRef.current = false;
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  // Clamp context menu to viewport after render (runs before paint → no flicker)
  useLayoutEffect(() => {
    if (!contextMenu || !contextMenuRef.current) return;
    const el = contextMenuRef.current;
    const rect = el.getBoundingClientRect();
    const margin = 8;
    const x = Math.max(
      margin,
      Math.min(contextMenu.x, window.innerWidth - rect.width - margin),
    );
    const y = Math.max(
      margin,
      Math.min(contextMenu.y, window.innerHeight - rect.height - margin),
    );
    if (x !== contextMenu.x || y !== contextMenu.y) {
      setContextMenu((prev) => (prev ? { ...prev, x, y } : null));
    }
  }, [contextMenu]);

  // Close context menu and emoji picker on outside click/tap or Escape
  useEffect(() => {
    if (!contextMenu && !emojiPickerMsgId) return;
    const close = () => {
      setContextMenu(null);
      setEmojiPickerMsgId(null);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setContextMenu(null);
        setEmojiPickerMsgId(null);
        setMobileOverlay(null);
      }
    };
    document.addEventListener("mousedown", close);
    document.addEventListener("touchstart", close);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", close);
      document.removeEventListener("touchstart", close);
      document.removeEventListener("keydown", onKey);
    };
  }, [contextMenu, emojiPickerMsgId]);

  const clearPendingImage = useCallback(() => {
    setPendingImage((prev) => {
      if (prev) URL.revokeObjectURL(prev.previewUrl);
      return null;
    });
  }, []);

  const setPendingFromFile = useCallback((file: File) => {
    if (!file.type.startsWith("image/")) return;
    setPendingImage((prev) => {
      if (prev) URL.revokeObjectURL(prev.previewUrl);
      return { file, previewUrl: URL.createObjectURL(file) };
    });
  }, []);

  // Bei Konversationswechsel ein noch nicht gesendetes Bild verwerfen, damit es
  // nicht versehentlich in eine andere Konversation gerät.
  useEffect(() => {
    clearPendingImage();
  }, [activeConv?.id, clearPendingImage]);

  // uploadImage verkleinert das Bild clientseitig (≤ 1 MB) und lädt es hoch;
  // liefert die media-ID oder null bei Fehler.
  const uploadImage = async (file: File): Promise<number | null> => {
    try {
      const { blob, fileName } = await compressImage(file);
      const form = new FormData();
      form.append("image", blob, fileName);
      const r = await api.post("/media/upload", form);
      return r.data?.mediaId ?? null;
    } catch {
      setToast("Bild konnte nicht hochgeladen werden");
      return null;
    }
  };

  const sendMessage = async () => {
    if (!activeConv || sending) return;
    const hasText = msgInput.trim().length > 0;

    // Bearbeiten ändert nur Text; Bilder sind hier nicht Teil des Edit-Flows.
    if (editingMessage) {
      if (!hasText) return;
      setSending(true);
      try {
        await api.put(`/chat/messages/${editingMessage.id}`, {
          body: msgInput.trim(),
        });
        setEditingMessage(null);
        setMsgInput("");
        draftsRef.current.delete(activeConv.id);
        await loadMessages(activeConv.id);
      } catch {
      } finally {
        setSending(false);
      }
      return;
    }

    if (!hasText && !pendingImage) return;
    setSending(true);
    try {
      let mediaId: number | null = null;
      if (pendingImage) {
        mediaId = await uploadImage(pendingImage.file);
        if (mediaId === null) return; // Upload fehlgeschlagen → Abbruch
      }
      await api.post(`/chat/conversations/${activeConv.id}/messages`, {
        body: msgInput.trim(),
        replyToId: replyTo?.id ?? null,
        mediaId,
      });
      setReplyTo(null);
      setMsgInput("");
      clearPendingImage();
      draftsRef.current.delete(activeConv.id);
      // Nach dem eigenen Senden soll die eigene Nachricht in den Blick — auch
      // wenn der Nutzer kurz vorher hochgescrollt hatte.
      forceScrollToEndRef.current = true;
      await appendNewMessages(activeConv.id);
    } catch {
    } finally {
      setSending(false);
    }
  };

  const startReply = (msg: Message) => {
    setReplyTo(msg);
    setEditingMessage(null);
    setMsgInput("");
    setContextMenu(null);
    inputRef.current?.focus();
  };

  // bodyOf liefert den bereits verfügbaren Text: den nachgeladenen Volltext, sonst
  // den Preview. Für nicht-gekürzte Nachrichten ist der Preview der Volltext.
  const bodyOf = (msg: Message) => fullBodies[msg.id] ?? msg.preview;

  // fetchFullBody lädt bei gekürzten Nachrichten den Volltext über den Einzel-Pfad
  // nach (und cached ihn). Gibt den Volltext zurück; bei Fehlern den Preview.
  const fetchFullBody = async (msg: Message): Promise<string> => {
    if (!msg.truncated || fullBodies[msg.id] !== undefined) return bodyOf(msg);
    try {
      const r = await api.get(`/chat/messages/${msg.id}`);
      const body: string = r.data?.body ?? msg.preview;
      setFullBodies((prev) => ({ ...prev, [msg.id]: body }));
      return body;
    } catch {
      return msg.preview;
    }
  };

  const startEdit = async (msg: Message) => {
    setEditingMessage(msg);
    setReplyTo(null);
    // Beim Bearbeiten den Volltext ins Eingabefeld — nicht den gekürzten Preview.
    setMsgInput(msg.truncated ? await fetchFullBody(msg) : msg.preview);
    setContextMenu(null);
    inputRef.current?.focus();
  };

  const cancelReplyOrEdit = () => {
    setReplyTo(null);
    setEditingMessage(null);
    setMsgInput("");
  };

  const copyMsgToClipboard = async (msg: Message, selectedText?: string) => {
    const text =
      selectedText ?? (msg.truncated ? await fetchFullBody(msg) : msg.preview);
    navigator.clipboard.writeText(text).catch(() => {});
    setContextMenu(null);
  };

  const deleteMsg = async (msg: Message) => {
    setContextMenu(null);
    try {
      await api.delete(`/chat/messages/${msg.id}`);
      if (activeConv) await loadMessages(activeConv.id);
    } catch {}
  };

  const handleContextMenu = (e: React.MouseEvent, msg: Message) => {
    if (msg.deletedAt) return;
    e.preventDefault();
    if (isMobile) return;
    const sel = window.getSelection();
    const selectedText =
      sel && sel.toString().trim() ? sel.toString() : undefined;
    setContextMenu({ x: e.clientX, y: e.clientY, message: msg, selectedText });
  };

  const handleLongPress = (msg: Message, _x: number, _y: number) => {
    if (msg.deletedAt) return;
    if (isMobile) {
      setMobileOverlay({ message: msg, isOwn: msg.senderId === user?.id });
    } else {
      const sel = window.getSelection();
      const selectedText =
        sel && sel.toString().trim() ? sel.toString() : undefined;
      setContextMenu({ x: _x, y: _y, message: msg, selectedText });
    }
  };

  const leaveGroup = async () => {
    if (!activeConv || activeConv.type !== "group") return;
    if (activeConv.createdBy === user?.id) {
      setShowCreatorExit(true);
      return;
    }
    if (!confirm("Gruppe verlassen?")) return;
    await api.delete(`/chat/conversations/${activeConv.id}/members/me`);
    setActiveConv(null);
    setMobileShowChat(false);
    loadConversations();
  };

  const deleteConversation = async (conv: Conversation) => {
    if (!confirm("Gespräch löschen?")) return;
    await api.delete(`/chat/conversations/${conv.id}`).catch(() => {});
    if (activeConv?.id === conv.id) {
      setActiveConv(null);
      setMobileShowChat(false);
    }
    loadConversations();
  };

  const deleteBroadcast = async (bc: Broadcast) => {
    if (!confirm("Mitteilung löschen?")) return;
    await api.delete(`/chat/broadcasts/${bc.id}`).catch(() => {});
    if (activeBroadcast?.id === bc.id) {
      setActiveBroadcast(null);
      setMobileShowChat(false);
    }
    loadBroadcasts();
  };

  const openBroadcast = async (bc: Broadcast) => {
    setActiveBroadcast(bc);
    setMobileShowChat(true);
    if (!bc.isRead && !bc.isSent) {
      await api.post(`/chat/broadcasts/${bc.id}/read`).catch(() => {});
      loadBroadcasts();
    }
  };

  const convName = (conv: Conversation) => {
    if (conv.name) return conv.name;
    const others = conv.members.filter((m) => m.id !== user?.id);
    return others.map((m) => m.name).join(", ") || "Konversation";
  };

  const totalUnread =
    conversations.reduce((s, c) => s + c.unreadCount, 0) +
    broadcasts.filter((b) => !b.isRead && !b.isSent).length;

  const canDelete = (msg: Message) =>
    msg.senderId === user?.id || hasCapability("moderate_chat");

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold text-brand-text flex items-center gap-2">
          Nachrichten
          {totalUnread > 0 && (
            <span className="bg-brand-yellow text-brand-black text-xs font-bold rounded-full px-2 py-0.5">
              {totalUnread}
            </span>
          )}
        </h1>
      </div>

      <div className="flex flex-1 min-h-0 gap-4">
        {/* Left panel: list */}
        <div
          className={`${isMobile && mobileShowChat ? "hidden" : "flex"} flex-col w-full sm:w-72 bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden`}
        >
          {/* Tabs */}
          <div className="flex border-b border-brand-border-subtle">
            <button
              onClick={() => setTab("chats")}
              className={`flex-1 py-3 text-sm font-medium flex items-center justify-center gap-1.5 transition-colors ${tab === "chats" ? "text-brand-text border-b-2 border-brand-yellow" : "text-brand-text-muted hover:text-brand-text"}`}
            >
              <MessageSquare className="w-4 h-4" />
              Chats
            </button>
            <button
              onClick={() => setTab("broadcasts")}
              className={`flex-1 py-3 text-sm font-medium flex items-center justify-center gap-1.5 transition-colors ${tab === "broadcasts" ? "text-brand-text border-b-2 border-brand-yellow" : "text-brand-text-muted hover:text-brand-text"}`}
            >
              <Megaphone className="w-4 h-4" />
              Mitteilungen
              {broadcasts.filter((b) => !b.isRead && !b.isSent).length > 0 && (
                <span className="bg-brand-yellow text-brand-black text-xs font-bold rounded-full px-1.5">
                  {broadcasts.filter((b) => !b.isRead && !b.isSent).length}
                </span>
              )}
            </button>
          </div>

          {tab === "chats" && (
            <>
              <div className="p-3 border-b border-brand-border-subtle">
                <button
                  onClick={() => setShowNewModal(true)}
                  className="w-full bg-brand-yellow text-brand-black rounded-md px-3 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors flex items-center justify-center gap-1.5"
                >
                  <Plus className="w-4 h-4" />
                  Neues Gespräch
                </button>
              </div>
              <div className="flex-1 overflow-y-auto">
                {conversations.length === 0 && (
                  <p className="text-brand-text-muted text-sm p-4 text-center">
                    Noch keine Gespräche
                  </p>
                )}
                {conversations.map((conv) => (
                  <div
                    key={conv.id}
                    className={`flex items-center border-b border-brand-border-subtle hover:bg-brand-table-select transition-colors ${activeConv?.id === conv.id ? "bg-brand-table-select" : ""}`}
                  >
                    <button
                      onClick={() => openConversation(conv)}
                      className="flex-1 min-w-0 text-left px-4 py-3"
                    >
                      <div className="flex items-center justify-between gap-2">
                        <span className="text-sm font-medium text-brand-text truncate">
                          {convName(conv)}
                        </span>
                        {conv.lastMessage && (
                          <span className="text-xs text-brand-text-muted shrink-0">
                            {conversationTimeLabel(
                              new Date(conv.lastMessage.sentAt),
                              new Date(),
                            )}
                          </span>
                        )}
                      </div>
                      {conv.lastMessage && (
                        <div className="flex items-center justify-between gap-2 mt-0.5">
                          <p className="text-xs text-brand-text-muted truncate">
                            {conv.lastMessage.body}
                          </p>
                          {conv.unreadCount > 0 && (
                            <span className="bg-brand-yellow text-brand-black text-xs font-bold rounded-full px-1.5 shrink-0">
                              {conv.unreadCount}
                            </span>
                          )}
                        </div>
                      )}
                    </button>
                    <button
                      onClick={() => deleteConversation(conv)}
                      className="shrink-0 px-3 py-3 text-brand-text-subtle hover:text-brand-danger transition-colors"
                      aria-label="Gespräch löschen"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                ))}
              </div>
            </>
          )}

          {tab === "broadcasts" && (
            <>
              {canBroadcast && (
                <div className="p-3 border-b border-brand-border-subtle">
                  <button
                    onClick={() => setShowBroadcastModal(true)}
                    className="w-full bg-brand-yellow text-brand-black rounded-md px-3 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors flex items-center justify-center gap-1.5"
                  >
                    <Megaphone className="w-4 h-4" />
                    Mitteilung senden
                  </button>
                </div>
              )}
              <div className="flex-1 overflow-y-auto">
                {broadcasts.length === 0 && (
                  <p className="text-brand-text-muted text-sm p-4 text-center">
                    Keine Mitteilungen
                  </p>
                )}
                {broadcasts.map((bc) => (
                  <div
                    key={bc.id}
                    className={`flex items-center border-b border-brand-border-subtle hover:bg-brand-table-select transition-colors ${activeBroadcast?.id === bc.id ? "bg-brand-table-select" : ""}`}
                  >
                    <button
                      onClick={() => openBroadcast(bc)}
                      className="flex-1 min-w-0 text-left px-4 py-3"
                    >
                      <div className="flex items-center justify-between gap-2">
                        <span
                          className={`text-sm truncate ${!bc.isRead && !bc.isSent ? "font-semibold text-brand-text" : "font-medium text-brand-text-muted"}`}
                        >
                          {bc.isSent ? "Gesendet" : bc.senderName}
                        </span>
                        {!bc.isRead && !bc.isSent && (
                          <span className="w-2 h-2 rounded-full bg-brand-yellow shrink-0" />
                        )}
                      </div>
                      <p className="text-xs text-brand-text-muted truncate mt-0.5">
                        {bc.body}
                      </p>
                    </button>
                    <button
                      onClick={() => deleteBroadcast(bc)}
                      className="shrink-0 px-3 py-3 text-brand-text-subtle hover:text-brand-danger transition-colors"
                      aria-label="Mitteilung löschen"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                ))}
              </div>
            </>
          )}
        </div>

        {/* Right panel: active chat or broadcast */}
        <div
          className={`${isMobile && !mobileShowChat ? "hidden" : "flex"} flex-col flex-1 min-w-0 bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden`}
        >
          {activeConv && tab === "chats" && (
            <>
              <div className="flex items-center justify-between px-4 py-3 border-b border-brand-border-subtle">
                <div className="flex items-center gap-2 min-w-0">
                  {isMobile && (
                    <button
                      onClick={() => setMobileShowChat(false)}
                      className="text-brand-text-muted hover:text-brand-text mr-1"
                      aria-label="Zurück"
                    >
                      <X className="w-5 h-5" />
                    </button>
                  )}
                  <span className="font-semibold text-brand-text truncate">
                    {convName(activeConv)}
                  </span>
                  {activeConv.type === "group" && (
                    <button
                      onClick={() => setShowParticipants(true)}
                      className="text-xs text-brand-text-subtle hover:text-brand-text shrink-0 inline-flex items-center"
                      aria-label="Teilnehmer anzeigen"
                    >
                      <Users className="w-3.5 h-3.5 mr-0.5" />
                      {activeConv.members.length}
                    </button>
                  )}
                </div>
                {activeConv.type === "group" && (
                  <div className="flex items-center gap-2 shrink-0">
                    <button
                      onClick={leaveGroup}
                      className="text-brand-text-muted hover:text-brand-danger transition-colors"
                      aria-label="Gruppe verlassen"
                    >
                      <LogOut className="w-4 h-4" />
                    </button>
                  </div>
                )}
              </div>

              {/* messagesBoxRef umschließt Kopf + gefensterte Liste; die
                  Scroll-Positions-Erhaltung beim Voranstellen (⑥) greift auf den
                  inneren WindowedRows-Container (`[data-windowed-scroll]`) zu. */}
              <div
                ref={messagesBoxRef}
                className="flex-1 flex flex-col min-h-0"
              >
                {/* „Ältere laden" (⑥, ?before=-Cursor) als gepinnter Kopf über der
                    gefensterten Liste. */}
                {hasOlder && (
                  <div className="px-4 pt-2 flex justify-center shrink-0">
                    <button
                      onClick={loadOlderMessages}
                      disabled={loadingOlder}
                      className="bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                    >
                      {loadingOlder ? "Lade…" : "Ältere Nachrichten laden"}
                    </button>
                  </div>
                )}
                {/* Chat-Bubbles haben extrem variable Höhen (~30 px kurzer Text
                    bis 300 px+ mit Bild/Reply/Reaktionen). Windowing mit fixer
                    estimatedRowHeight verschiebt beim Scrollen scrollHeight
                    gegenüber der Realität → Sichtbereich springt. Bei einer
                    100er-Seite bleibt der reine DOM ohnehin billig, deshalb
                    hier über die threshold explizit deaktiviert.
                    WindowedRows bleibt als Wrapper (setzt data-windowed-scroll,
                    das die Sticky-Scroll- und loadOlderMessages-Logik nutzt). */}
                <WindowedRows
                  items={messages}
                  estimatedRowHeight={64}
                  threshold={Number.MAX_SAFE_INTEGER}
                  className="flex-1 overflow-y-auto overflow-x-hidden p-4 flex flex-col gap-2"
                  footer={<div ref={messagesEndRef} />}
                  renderRow={(msg, index) => {
                    const now = new Date();
                    const prevSentAt =
                      index > 0 ? messages[index - 1].sentAt : null;
                    const sep = shouldRenderSeparator(
                      prevSentAt,
                      msg.sentAt,
                    ) ? (
                      <DaySeparator
                        key={`sep-${msg.id}`}
                        label={daySeparatorLabel(new Date(msg.sentAt), now)}
                      />
                    ) : null;
                    if (msg.isSystem) {
                      return (
                        <div key={msg.id} className="contents">
                          {sep}
                          <div className="flex justify-center my-1">
                            <span className="text-xs text-brand-text-muted bg-brand-surface-card px-3 py-1 rounded-full">
                              {msg.senderName} {msg.preview}
                            </span>
                          </div>
                        </div>
                      );
                    }
                    const isOwn = msg.senderId === user?.id;
                    return (
                      <div key={msg.id} className="contents">
                        {sep}
                        <MessageBubble
                          msg={msg}
                          body={bodyOf(msg)}
                          showExpand={
                            msg.truncated && fullBodies[msg.id] === undefined
                          }
                          onExpand={() => {
                            void fetchFullBody(msg);
                          }}
                          isOwn={isOwn}
                          onContextMenu={handleContextMenu}
                          onSwipeReply={startReply}
                          onLongPress={handleLongPress}
                          isPickerOpen={emojiPickerMsgId === msg.id}
                          onOpenPicker={(e) => {
                            e.stopPropagation();
                            setEmojiPickerMsgId(msg.id);
                          }}
                          onClosePicker={() => setEmojiPickerMsgId(null)}
                          onToggleReaction={toggleReaction}
                          onImageClick={() => {
                            if (msg.mediaUrl) setLightboxUrl(msg.mediaUrl);
                          }}
                        />
                      </div>
                    );
                  }}
                />
              </div>

              {/* Reply / Edit bar */}
              {(replyTo || editingMessage) && (
                <div className="px-4 py-2 border-t border-brand-border-subtle bg-white flex items-center gap-2">
                  {replyTo ? (
                    <CornerUpLeft className="w-4 h-4 text-brand-text-muted shrink-0" />
                  ) : (
                    <Pencil className="w-4 h-4 text-brand-text-muted shrink-0" />
                  )}
                  <div className="flex-1 min-w-0">
                    <p className="text-xs font-medium text-brand-text">
                      {replyTo
                        ? `Antwort auf ${replyTo.senderName}`
                        : "Nachricht bearbeiten"}
                    </p>
                    {replyTo && (
                      <p className="text-xs text-brand-text-muted truncate">
                        {bodyOf(replyTo).slice(0, 60)}
                      </p>
                    )}
                  </div>
                  <button
                    onClick={cancelReplyOrEdit}
                    aria-label="Abbrechen"
                    className="text-brand-text-muted hover:text-brand-text shrink-0"
                  >
                    <X className="w-4 h-4" />
                  </button>
                </div>
              )}

              {/* Bild-Vorschau vor dem Senden */}
              {pendingImage && !editingMessage && (
                <div className="px-4 py-2 border-t border-brand-border-subtle bg-white flex items-center gap-3">
                  <img
                    src={pendingImage.previewUrl}
                    alt="Vorschau"
                    className="h-16 w-16 object-cover rounded-md border border-brand-border-subtle"
                  />
                  <span className="flex-1 min-w-0 text-xs text-brand-text-muted truncate">
                    {pendingImage.file.name}
                  </span>
                  <button
                    onClick={clearPendingImage}
                    aria-label="Bild entfernen"
                    className="text-brand-text-muted hover:text-brand-text shrink-0"
                  >
                    <X className="w-4 h-4" />
                  </button>
                </div>
              )}

              <div className="px-4 py-3 border-t border-brand-border-subtle flex gap-2 items-end">
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="image/*"
                  className="hidden"
                  onChange={(e) => {
                    const file = e.target.files?.[0];
                    e.target.value = "";
                    if (file) setPendingFromFile(file);
                  }}
                />
                {!editingMessage && (
                  <button
                    onClick={() => fileInputRef.current?.click()}
                    disabled={sending}
                    className="text-brand-text-muted hover:text-brand-text transition-colors shrink-0 py-2 disabled:opacity-40 disabled:cursor-not-allowed"
                    aria-label="Bild anhängen"
                  >
                    <Paperclip className="w-5 h-5" />
                  </button>
                )}
                <textarea
                  ref={inputRef}
                  value={msgInput}
                  onChange={(e) => setMsgInput(e.target.value)}
                  onPaste={(e) => {
                    const img = Array.from(e.clipboardData.files).find((f) =>
                      f.type.startsWith("image/"),
                    );
                    if (img && !editingMessage) {
                      e.preventDefault();
                      setPendingFromFile(img);
                    }
                  }}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && !e.nativeEvent.isComposing) {
                      if (isMobile) return;
                      if (e.altKey || e.ctrlKey) {
                        e.preventDefault();
                        const ta = e.currentTarget;
                        const start = ta.selectionStart ?? msgInput.length;
                        const end = ta.selectionEnd ?? msgInput.length;
                        const newValue =
                          msgInput.slice(0, start) + "\n" + msgInput.slice(end);
                        setMsgInput(newValue);
                        requestAnimationFrame(() => {
                          ta.selectionStart = ta.selectionEnd = start + 1;
                        });
                        return;
                      }
                      if (!e.shiftKey) {
                        e.preventDefault();
                        sendMessage();
                      }
                    }
                  }}
                  placeholder="Nachricht schreiben…"
                  maxLength={2000}
                  rows={1}
                  enterKeyHint={isMobile ? "enter" : "send"}
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow resize-none overflow-y-auto leading-5"
                />
                <button
                  onClick={sendMessage}
                  disabled={(!msgInput.trim() && !pendingImage) || sending}
                  className="bg-brand-yellow text-brand-black rounded-md px-3 py-2 hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                  aria-label={editingMessage ? "Speichern" : "Senden"}
                >
                  {editingMessage ? (
                    <Pencil className="w-4 h-4" />
                  ) : (
                    <Send className="w-4 h-4" />
                  )}
                </button>
              </div>
            </>
          )}

          {activeBroadcast && tab === "broadcasts" && (
            <div className="flex-1 overflow-y-auto p-6">
              <div className="flex items-center gap-2 mb-1">
                {isMobile && (
                  <button
                    onClick={() => {
                      setActiveBroadcast(null);
                      setMobileShowChat(false);
                    }}
                    className="text-brand-text-muted hover:text-brand-text mr-1"
                    aria-label="Zurück"
                  >
                    <X className="w-5 h-5" />
                  </button>
                )}
                <span className="font-semibold text-brand-text flex-1">
                  {activeBroadcast.isSent
                    ? "Gesendet von mir"
                    : activeBroadcast.senderName}
                </span>
                {activeBroadcast.isSent && (
                  <button
                    onClick={() => setShowBroadcastEdit(true)}
                    className="text-brand-text-muted hover:text-brand-text transition-colors"
                    aria-label="Mitteilung bearbeiten"
                  >
                    <Pencil className="w-4 h-4" />
                  </button>
                )}
              </div>
              <p className="text-xs text-brand-text-muted mb-4">
                {new Date(activeBroadcast.sentAt).toLocaleString("de-DE")}
                {activeBroadcast.editedAt && (
                  <span className="ml-2">(bearbeitet)</span>
                )}
              </p>
              <p className="text-sm text-brand-text whitespace-pre-wrap break-words">
                {renderWithLinks(activeBroadcast.body, false)}
              </p>
              {activeBroadcast.mediaUrl && (
                <AuthImage
                  url={activeBroadcast.mediaUrl}
                  alt="Bild der Mitteilung"
                  className="mt-3 max-w-xs rounded-lg cursor-pointer"
                  onClick={() => setLightboxUrl(activeBroadcast.mediaUrl)}
                />
              )}
            </div>
          )}

          {!activeConv && !activeBroadcast && (
            <div className="flex-1 flex items-center justify-center text-brand-text-muted text-sm">
              Gespräch oder Mitteilung auswählen
            </div>
          )}
        </div>
      </div>

      {/* Context Menu */}
      {contextMenu && (
        <div
          ref={contextMenuRef}
          className="fixed z-50 bg-white rounded-lg shadow-lg border border-brand-border py-1 min-w-[160px]"
          style={{ left: contextMenu.x, top: contextMenu.y }}
          onMouseDown={(e) => e.stopPropagation()}
          onTouchStart={(e) => e.stopPropagation()}
        >
          {/* Emoji reaction row */}
          {!contextMenu.message.deletedAt && (
            <div className="flex gap-0.5 px-2 py-2 border-b border-brand-border-subtle">
              {REACTION_EMOJIS.map((emoji) => (
                <button
                  key={emoji}
                  onClick={() => {
                    toggleReaction(contextMenu.message.id, emoji);
                    setContextMenu(null);
                  }}
                  className={`text-lg p-1 rounded-full transition-transform hover:scale-125 ${
                    contextMenu.message.reactions?.some(
                      (r) => r.emoji === emoji && r.myReaction,
                    )
                      ? "bg-brand-yellow/30"
                      : "hover:bg-brand-border-subtle"
                  }`}
                >
                  {emoji}
                </button>
              ))}
            </div>
          )}
          <button
            onClick={() => startReply(contextMenu.message)}
            className="w-full flex items-center gap-2 px-4 py-2 text-sm text-brand-text hover:bg-brand-table-select transition-colors"
          >
            <CornerUpLeft className="w-4 h-4" />
            Antworten
          </button>
          <button
            onClick={() =>
              copyMsgToClipboard(contextMenu.message, contextMenu.selectedText)
            }
            className="w-full flex items-center gap-2 px-4 py-2 text-sm text-brand-text hover:bg-brand-table-select transition-colors"
          >
            <Copy className="w-4 h-4" />
            {contextMenu.selectedText ? "Auswahl kopieren" : "Kopieren"}
          </button>
          {contextMenu.message.senderId === user?.id && (
            <button
              onClick={() => startEdit(contextMenu.message)}
              className="w-full flex items-center gap-2 px-4 py-2 text-sm text-brand-text hover:bg-brand-table-select transition-colors"
            >
              <Pencil className="w-4 h-4" />
              Bearbeiten
            </button>
          )}
          {canDelete(contextMenu.message) && (
            <button
              onClick={() => deleteMsg(contextMenu.message)}
              className="w-full flex items-center gap-2 px-4 py-2 text-sm text-brand-danger hover:bg-brand-danger-light transition-colors"
            >
              <Trash2 className="w-4 h-4" />
              Löschen
            </button>
          )}
        </div>
      )}

      {mobileOverlay && (
        <MobileMessageActionOverlay
          overlay={mobileOverlay}
          body={bodyOf(mobileOverlay.message)}
          onCopyFull={() => fetchFullBody(mobileOverlay.message)}
          onClose={() => setMobileOverlay(null)}
          onReply={(msg) => {
            startReply(msg);
            setMobileOverlay(null);
          }}
          onEdit={(msg) => {
            startEdit(msg);
            setMobileOverlay(null);
          }}
          onDelete={(msg) => {
            deleteMsg(msg);
            setMobileOverlay(null);
          }}
          onToggleReaction={(msgId, emoji) => {
            toggleReaction(msgId, emoji);
            setMobileOverlay(null);
          }}
          canDeleteMsg={canDelete}
          userId={user?.id}
        />
      )}

      {showNewModal && (
        <NewConversationModal
          onClose={() => setShowNewModal(false)}
          onCreated={(conv) => {
            setShowNewModal(false);
            loadConversations();
            openConversation(conv);
            setTab("chats");
          }}
        />
      )}

      {showBroadcastModal && (
        <BroadcastModal
          onClose={() => setShowBroadcastModal(false)}
          onSent={() => {
            setShowBroadcastModal(false);
            loadBroadcasts();
          }}
          isAdmin={hasCapability("broadcast_all")}
        />
      )}

      {showBroadcastEdit && activeBroadcast && (
        <BroadcastEditModal
          broadcast={activeBroadcast}
          onClose={() => setShowBroadcastEdit(false)}
          onSaved={async () => {
            setShowBroadcastEdit(false);
            await loadBroadcasts();
            const r = await api
              .get("/chat/broadcasts")
              .catch(() => ({ data: [] }));
            const updated = (r.data ?? []).find(
              (b: Broadcast) => b.id === activeBroadcast.id,
            );
            if (updated) setActiveBroadcast(updated);
          }}
        />
      )}

      {showParticipants && activeConv && activeConv.type === "group" && (
        <ConversationParticipantsModal
          convId={activeConv.id}
          initialName={activeConv.name}
          createdBy={activeConv.createdBy}
          members={activeConv.members}
          onClose={() => setShowParticipants(false)}
          onChanged={() => reloadActiveConv(activeConv.id)}
        />
      )}

      {showCreatorExit && activeConv && activeConv.type === "group" && (
        <CreatorExitChoiceModal
          convId={activeConv.id}
          ownerId={activeConv.createdBy}
          members={activeConv.members}
          onClose={() => setShowCreatorExit(false)}
          onDone={() => {
            setShowCreatorExit(false);
            setActiveConv(null);
            setMobileShowChat(false);
            loadConversations();
          }}
        />
      )}

      {lightboxUrl && (
        <div
          className="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4"
          onClick={() => setLightboxUrl(null)}
        >
          <button
            onClick={() => setLightboxUrl(null)}
            aria-label="Schließen"
            className="absolute top-4 right-4 text-white hover:text-brand-yellow transition-colors"
          >
            <X className="w-7 h-7" />
          </button>
          <AuthImage
            url={lightboxUrl}
            alt="Bild"
            className="max-h-[90vh] max-w-full object-contain rounded-lg"
          />
        </div>
      )}

      {toast && (
        <div className="fixed bottom-4 left-1/2 -translate-x-1/2 z-50 bg-brand-text text-white text-sm rounded-md shadow-lg px-4 py-2">
          {toast}
        </div>
      )}
    </div>
  );
}

function renderWithLinks(body: string, isOwn: boolean) {
  const parts = body.split(/(https?:\/\/[^\s]+)/g);
  return parts.map((part, i) =>
    /^https?:\/\//.test(part) ? (
      <a
        key={i}
        href={part}
        target="_blank"
        rel="noopener noreferrer"
        className={`underline break-all ${isOwn ? "opacity-75" : "text-brand-blue"}`}
        onClick={(e) => e.stopPropagation()}
      >
        {part}
      </a>
    ) : (
      part
    ),
  );
}

// --- Message Bubble ---
function MessageBubble({
  msg,
  body,
  showExpand,
  onExpand,
  isOwn,
  onContextMenu,
  onSwipeReply,
  onLongPress,
  isPickerOpen,
  onOpenPicker,
  onClosePicker,
  onToggleReaction,
  onImageClick,
}: {
  msg: Message;
  body: string;
  showExpand: boolean;
  onExpand: () => void;
  isOwn: boolean;
  onContextMenu: (e: React.MouseEvent, msg: Message) => void;
  onSwipeReply: (msg: Message) => void;
  onLongPress: (msg: Message, x: number, y: number) => void;
  isPickerOpen: boolean;
  onOpenPicker: (e: React.MouseEvent) => void;
  onClosePicker: () => void;
  onToggleReaction: (msgId: number, emoji: string) => void;
  onImageClick: () => void;
}) {
  const wrapperRef = useRef<HTMLDivElement>(null);
  const touchStartX = useRef(0);
  const touchStartY = useRef(0);
  const longPressTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [swipeDelta, setSwipeDelta] = useState(0);
  const [showReplyIcon, setShowReplyIcon] = useState(false);

  const cancelLongPress = () => {
    if (longPressTimer.current) {
      clearTimeout(longPressTimer.current);
      longPressTimer.current = null;
    }
  };

  const onTouchStart = (e: React.TouchEvent) => {
    touchStartX.current = e.touches[0].clientX;
    touchStartY.current = e.touches[0].clientY;
    if (!msg.deletedAt) {
      const x = e.touches[0].clientX;
      const y = e.touches[0].clientY;
      longPressTimer.current = setTimeout(() => {
        longPressTimer.current = null;
        onLongPress(msg, x, y);
      }, 500);
    }
  };

  const onTouchMove = (e: React.TouchEvent) => {
    const dx = e.touches[0].clientX - touchStartX.current;
    const dy = e.touches[0].clientY - touchStartY.current;
    if (Math.abs(dx) > 8 || Math.abs(dy) > 8) cancelLongPress();
    if (Math.abs(dy) > Math.abs(dx) || dx < 0) return;
    const delta = Math.min(dx, 70);
    setSwipeDelta(delta);
    setShowReplyIcon(delta > 20);
  };

  const onTouchEnd = () => {
    cancelLongPress();
    if (swipeDelta >= 60 && !msg.deletedAt) {
      onSwipeReply(msg);
    }
    setSwipeDelta(0);
    setShowReplyIcon(false);
  };

  if (msg.deletedAt) {
    return (
      <div className={`flex flex-col ${isOwn ? "items-end" : "items-start"}`}>
        <div className="flex items-center gap-1.5 px-3 py-2 rounded-xl bg-brand-surface-card border border-brand-border-subtle text-brand-text-subtle text-sm italic">
          <Trash2 className="w-3.5 h-3.5 shrink-0" />
          Nachricht gelöscht
        </div>
        <span className="text-xs text-brand-text-subtle mt-0.5">
          {new Date(msg.sentAt).toLocaleTimeString("de-DE", {
            hour: "2-digit",
            minute: "2-digit",
          })}
        </span>
      </div>
    );
  }

  const reactions = msg.reactions ?? [];

  return (
    <div
      className={`flex items-center gap-1 ${isOwn ? "flex-row-reverse" : "flex-row"} group/msg`}
    >
      {/* Swipe reply icon */}
      <div
        className="shrink-0 transition-opacity duration-100"
        style={{ opacity: showReplyIcon ? 1 : 0 }}
      >
        <CornerUpLeft className="w-4 h-4 text-brand-text-muted" />
      </div>

      {/* Smiley button — hover only, mobile uses long-press context menu */}
      <div
        className="relative shrink-0 self-end mb-1"
        onMouseDown={(e) => e.stopPropagation()}
      >
        <button
          className="opacity-0 group-hover/msg:opacity-100 transition-opacity p-1 rounded-full hover:bg-brand-border-subtle text-brand-text-muted hidden sm:block"
          onClick={onOpenPicker}
          aria-label="Reaktion hinzufügen"
        >
          <SmilePlus className="w-4 h-4" />
        </button>
        {isPickerOpen && (
          <div
            className={`absolute bottom-full mb-1 z-50 bg-white rounded-full shadow-xl border border-brand-border-subtle flex gap-0.5 px-2 py-1.5 ${isOwn ? "right-0" : "left-0"}`}
            onMouseDown={(e) => e.stopPropagation()}
          >
            {REACTION_EMOJIS.map((emoji) => (
              <button
                key={emoji}
                onClick={() => {
                  onToggleReaction(msg.id, emoji);
                  onClosePicker();
                }}
                className={`text-lg p-1 rounded-full transition-transform hover:scale-125 ${
                  reactions.some((r) => r.emoji === emoji && r.myReaction)
                    ? "bg-brand-yellow/30"
                    : "hover:bg-brand-border-subtle"
                }`}
              >
                {emoji}
              </button>
            ))}
          </div>
        )}
      </div>

      <div
        ref={wrapperRef}
        className={`flex flex-col ${isOwn ? "items-end" : "items-start"} flex-1 select-none`}
        style={{
          transform: `translateX(${isOwn ? -swipeDelta : swipeDelta}px)`,
          transition: swipeDelta === 0 ? "transform 0.2s ease" : "none",
        }}
        onContextMenu={(e) => onContextMenu(e, msg)}
        onTouchStart={onTouchStart}
        onTouchMove={onTouchMove}
        onTouchEnd={onTouchEnd}
      >
        {!isOwn && (
          <span className="text-xs text-brand-text-muted mb-0.5">
            {msg.senderName}
          </span>
        )}

        <div
          className={`max-w-xs sm:max-w-sm rounded-xl px-3 py-2 text-sm select-text ${isOwn ? "bg-brand-yellow text-brand-black" : "bg-white border border-brand-border text-brand-text"}`}
        >
          {/* Reply quote */}
          {msg.replyToId && (
            <div
              className={`mb-1.5 pl-2 border-l-2 ${isOwn ? "border-brand-black/40" : "border-brand-yellow"} text-xs opacity-80`}
            >
              <p className="font-semibold">{msg.replyToSenderName}</p>
              <p className="truncate">{(msg.replyToBody ?? "").slice(0, 60)}</p>
            </div>
          )}
          {body && (
            <span className="whitespace-pre-wrap break-words">
              {renderWithLinks(body, isOwn)}
            </span>
          )}
          {msg.mediaUrl && (
            <AuthImage
              url={msg.mediaUrl}
              alt="Bild"
              className={`${body ? "mt-2 " : ""}max-w-full rounded-lg cursor-pointer`}
              onClick={onImageClick}
            />
          )}
          {showExpand && (
            <button
              onClick={onExpand}
              className={`block mt-1 text-xs font-medium underline ${isOwn ? "text-brand-black/70" : "text-brand-info"}`}
            >
              Mehr anzeigen
            </button>
          )}
        </div>

        {/* Reaction chips */}
        {reactions.length > 0 && (
          <div className="flex flex-wrap gap-1 mt-1">
            {reactions.map((r) => (
              <div key={r.emoji} className="relative group/reaction">
                <button
                  onClick={() => onToggleReaction(msg.id, r.emoji)}
                  className={`flex items-center gap-0.5 rounded-full border px-1.5 py-0.5 text-sm leading-none transition-colors ${
                    r.myReaction
                      ? "bg-brand-yellow/20 border-brand-yellow text-brand-text"
                      : "bg-white border-brand-border-subtle text-brand-text hover:bg-brand-border-subtle"
                  }`}
                >
                  <span>{r.emoji}</span>
                  <span className="text-xs font-medium ml-0.5">{r.count}</span>
                </button>
                {/* Tooltip */}
                <div
                  className={`pointer-events-none absolute bottom-full mb-1.5 hidden group-hover/reaction:block z-50 ${isOwn ? "right-0" : "left-0"}`}
                >
                  <div className="bg-brand-text text-white text-xs rounded px-2 py-1.5 text-left min-w-max max-w-[200px]">
                    {r.userNames.map((name) => (
                      <div key={name}>{name}</div>
                    ))}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        <div className="flex items-center gap-1 mt-0.5">
          <span className="text-xs text-brand-text-subtle">
            {new Date(msg.sentAt).toLocaleTimeString("de-DE", {
              hour: "2-digit",
              minute: "2-digit",
            })}
          </span>
          {msg.editedAt && (
            <span className="text-xs text-brand-text-subtle">(bearbeitet)</span>
          )}
        </div>
      </div>
    </div>
  );
}

// --- Mobile Message Action Overlay ---
function MobileMessageActionOverlay({
  overlay,
  body,
  onCopyFull,
  onClose,
  onReply,
  onEdit,
  onDelete,
  onToggleReaction,
  canDeleteMsg,
  userId,
}: {
  overlay: { message: Message; isOwn: boolean };
  body: string;
  onCopyFull: () => Promise<string>;
  onClose: () => void;
  onReply: (msg: Message) => void;
  onEdit: (msg: Message) => void;
  onDelete: (msg: Message) => void;
  onToggleReaction: (msgId: number, emoji: string) => void;
  canDeleteMsg: (msg: Message) => boolean;
  userId: number | undefined;
}) {
  const { message: msg, isOwn } = overlay;

  const copyText = async () => {
    const sel = window.getSelection();
    // Bei gekürzten Nachrichten ohne aktive Selektion den Volltext nachladen.
    const text =
      sel && sel.toString().trim()
        ? sel.toString()
        : msg.truncated
          ? await onCopyFull()
          : body;
    navigator.clipboard.writeText(text).catch(() => {});
    onClose();
  };

  return (
    <div
      className="fixed inset-0 z-50 flex flex-col items-center justify-center gap-4 p-6 bg-black/50 backdrop-blur-sm"
      onTouchStart={onClose}
    >
      <div
        className="flex flex-col gap-3 w-full max-w-xs max-h-[calc(100vh-3rem)] overflow-y-auto overflow-x-hidden"
        onTouchStart={(e) => e.stopPropagation()}
      >
        {/* Emoji row */}
        <div className="flex justify-center gap-0.5 bg-white rounded-full px-3 py-2 shadow-xl self-center select-none">
          {REACTION_EMOJIS.map((emoji) => (
            <button
              key={emoji}
              className="text-xl p-1.5"
              onClick={() => onToggleReaction(msg.id, emoji)}
            >
              {emoji}
            </button>
          ))}
        </div>

        {/* Message bubble — select-text für OS-Textselektion */}
        <div
          className={`rounded-xl px-3 py-2.5 text-sm select-text shadow-xl ${isOwn ? "bg-brand-yellow text-brand-black self-end" : "bg-white border border-brand-border text-brand-text self-start"}`}
        >
          {msg.replyToId && (
            <div
              className={`mb-1.5 pl-2 border-l-2 ${isOwn ? "border-brand-black/40" : "border-brand-yellow"} text-xs opacity-80`}
            >
              <p className="font-semibold">{msg.replyToSenderName}</p>
              <p className="truncate">{(msg.replyToBody ?? "").slice(0, 60)}</p>
            </div>
          )}
          <span className="whitespace-pre-wrap break-words">
            {renderWithLinks(body, isOwn)}
          </span>
        </div>

        {/* Action buttons */}
        <div className="bg-white rounded-xl shadow-xl overflow-hidden select-none self-center max-w-[210px] w-full">
          <button
            onClick={() => onReply(msg)}
            className="w-full flex items-center gap-3 px-4 py-3.5 text-sm text-brand-text border-b border-brand-border-subtle"
          >
            <CornerUpLeft className="w-4 h-4 text-brand-text-muted shrink-0" />
            Antworten
          </button>
          <button
            onClick={copyText}
            className="w-full flex items-center gap-3 px-4 py-3.5 text-sm text-brand-text border-b border-brand-border-subtle"
          >
            <Copy className="w-4 h-4 text-brand-text-muted shrink-0" />
            Kopieren
          </button>
          {msg.senderId === userId && (
            <button
              onClick={() => onEdit(msg)}
              className="w-full flex items-center gap-3 px-4 py-3.5 text-sm text-brand-text border-b border-brand-border-subtle"
            >
              <Pencil className="w-4 h-4 text-brand-text-muted shrink-0" />
              Bearbeiten
            </button>
          )}
          {canDeleteMsg(msg) && (
            <button
              onClick={() => onDelete(msg)}
              className="w-full flex items-center gap-3 px-4 py-3.5 text-sm text-brand-danger"
            >
              <Trash2 className="w-4 h-4 shrink-0" />
              Löschen
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

// --- New Conversation Modal ---
function NewConversationModal({
  onClose,
  onCreated,
}: {
  onClose: () => void;
  onCreated: (conv: Conversation) => void;
}) {
  const [type, setType] = useState<"direct" | "group">("direct");
  const [query, setQuery] = useState("");
  const [users, setUsers] = useState<ChatUser[]>([]);
  const [selected, setSelected] = useState<ChatUser[]>([]);
  const [groupName, setGroupName] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [teamGroups, setTeamGroups] = useState<TeamGroup[]>([]);
  const [resolvingTag, setResolvingTag] = useState<string | null>(null);
  const [pickedTags, setPickedTags] = useState<Set<string>>(new Set());

  const tagKey = (tg: { teamId: number; kind: TeamGroup["kind"] }) =>
    `${tg.teamId}:${tg.kind}`;

  useEffect(() => {
    const t = setTimeout(async () => {
      try {
        const r = await api.get("/chat/users", { params: { q: query } });
        setUsers(r.data ?? []);
      } catch {}
    }, 200);
    return () => clearTimeout(t);
  }, [query]);

  useEffect(() => {
    if (type !== "group") return;
    api
      .get("/chat/team-groups")
      .then((r) => setTeamGroups(r.data ?? []))
      .catch(() => {});
  }, [type]);

  const visibleTeamGroups = useMemo(() => {
    if (type !== "group") return [];
    const q = query.trim().toLowerCase();
    return teamGroups.filter((tg) => {
      if (pickedTags.has(tagKey(tg))) return false;
      if (!q) return true;
      const short = tg.displayShort.toLowerCase();
      const kindLabel = TEAM_GROUP_KIND_LABEL[tg.kind].toLowerCase();
      return short.includes(q) || kindLabel.includes(q);
    });
  }, [type, teamGroups, query, pickedTags]);

  const addTeamGroup = async (tg: TeamGroup) => {
    const key = tagKey(tg);
    if (resolvingTag) return;
    setResolvingTag(key);
    try {
      const r = await api.get(
        `/chat/team-groups/${tg.teamId}/${tg.kind}/members`,
      );
      const incoming: ChatUser[] = r.data ?? [];
      setSelected((prev) => {
        const seen = new Set(prev.map((p) => p.id));
        const merged = [...prev];
        for (const u of incoming) {
          if (!seen.has(u.id)) {
            merged.push(u);
            seen.add(u.id);
          }
        }
        return merged;
      });
      setPickedTags((prev) => new Set(prev).add(key));
    } catch (e) {
      setError(errorMessage(e, "Gruppe konnte nicht aufgelöst werden"));
    } finally {
      setResolvingTag(null);
    }
  };

  const toggleUser = (u: ChatUser) => {
    if (type === "direct") {
      setSelected([u]);
    } else {
      setSelected((prev) =>
        prev.find((p) => p.id === u.id)
          ? prev.filter((p) => p.id !== u.id)
          : [...prev, u],
      );
    }
  };

  const submit = async () => {
    if (selected.length === 0) return;
    setLoading(true);
    setError("");
    try {
      const payload =
        type === "direct"
          ? { type, userId: selected[0].id }
          : { type, name: groupName, memberIds: selected.map((u) => u.id) };
      const r = await api.post("/chat/conversations", payload);
      onCreated(r.data);
    } catch (e) {
      setError(errorMessage(e, "Fehler beim Erstellen"));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-md">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold text-brand-text">Neues Gespräch</h2>
          <button onClick={onClose} aria-label="Schließen">
            <X className="w-5 h-5 text-brand-text-muted" />
          </button>
        </div>

        <div className="flex gap-2 mb-4">
          {(["direct", "group"] as const).map((t) => (
            <button
              key={t}
              onClick={() => {
                setType(t);
                setSelected([]);
              }}
              className={`flex-1 py-2 rounded-md text-sm font-medium transition-colors ${type === t ? "bg-brand-yellow text-brand-black" : "bg-brand-surface-card text-brand-text-muted hover:text-brand-text"}`}
            >
              {t === "direct" ? "Direkt" : "Gruppe"}
            </button>
          ))}
        </div>

        {type === "group" && (
          <input
            type="text"
            placeholder="Gruppenname"
            value={groupName}
            onChange={(e) => setGroupName(e.target.value)}
            className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow mb-3"
          />
        )}

        <div className="relative mb-3">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-brand-text-subtle" />
          <input
            type="text"
            placeholder="Person suchen…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="w-full border border-brand-border rounded-md pl-9 pr-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
          />
        </div>

        {selected.length > 0 && (
          <div className="flex flex-wrap gap-1.5 mb-3">
            {selected.map((u) => (
              <span
                key={u.id}
                className="flex items-center gap-1 bg-brand-yellow/20 text-brand-text text-xs rounded-full px-2 py-0.5"
              >
                {u.name}
                <button
                  onClick={() =>
                    setSelected((prev) => prev.filter((p) => p.id !== u.id))
                  }
                  aria-label="Entfernen"
                >
                  <X className="w-3 h-3" />
                </button>
              </span>
            ))}
          </div>
        )}

        <div className="max-h-64 overflow-y-auto border border-brand-border-subtle rounded-md mb-4">
          {type === "group" && visibleTeamGroups.length > 0 && (
            <>
              <div className="px-3 py-1.5 text-xs uppercase text-brand-text-muted bg-brand-surface-card border-b border-brand-border-subtle">
                Standard-Gruppen
              </div>
              {visibleTeamGroups.map((tg) => {
                const key = tagKey(tg);
                const short = tg.displayShort || `Team ${tg.teamId}`;
                const label =
                  tg.kind === "alle_trainer"
                    ? TEAM_GROUP_KIND_LABEL[tg.kind]
                    : `${TEAM_GROUP_KIND_LABEL[tg.kind]} ${short}`;
                return (
                  <button
                    key={key}
                    onClick={() => addTeamGroup(tg)}
                    disabled={resolvingTag === key}
                    className="w-full flex items-center justify-between gap-2 text-left px-3 py-2 text-sm text-brand-text hover:bg-brand-table-select transition-colors disabled:opacity-50"
                  >
                    <span className="flex items-center gap-2">
                      <Users className="w-4 h-4 text-brand-text-muted" />
                      {label}
                    </span>
                    <span className="text-xs text-brand-text-muted">
                      {tg.count}
                    </span>
                  </button>
                );
              })}
              <div className="px-3 py-1.5 text-xs uppercase text-brand-text-muted bg-brand-surface-card border-y border-brand-border-subtle">
                Personen
              </div>
            </>
          )}
          {users.map((u) => (
            <button
              key={u.id}
              onClick={() => toggleUser(u)}
              className={`w-full text-left px-3 py-2 text-sm hover:bg-brand-table-select transition-colors ${selected.find((s) => s.id === u.id) ? "bg-brand-yellow/10 font-medium" : "text-brand-text"}`}
            >
              {u.name}
            </button>
          ))}
          {users.length === 0 && visibleTeamGroups.length === 0 && (
            <p className="text-brand-text-muted text-sm p-3 text-center">
              Keine Ergebnisse
            </p>
          )}
        </div>

        {error && <p className="text-brand-danger text-sm mb-3">{error}</p>}

        <button
          onClick={submit}
          disabled={
            loading ||
            selected.length === 0 ||
            (type === "group" && !groupName.trim())
          }
          className="w-full bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {loading ? "Erstelle…" : "Gespräch starten"}
        </button>
      </div>
    </div>
  );
}

// --- Broadcast Modal ---
function BroadcastModal({
  onClose,
  onSent,
  isAdmin,
}: {
  onClose: () => void;
  onSent: () => void;
  isAdmin: boolean;
}) {
  const [body, setBody] = useState("");
  const [targetType, setTargetType] = useState<"all" | "team" | "role">(
    isAdmin ? "all" : "team",
  );
  const [teams, setTeams] = useState<
    {
      id: number;
      name: string;
      age_class: string;
      gender: string;
      team_number: number;
      group_count: number;
    }[]
  >([]);
  const teamShortNames = useMemo(() => buildTeamShortNames(teams), [teams]);
  const [targetId, setTargetId] = useState(0);
  const [targetRole, setTargetRole] = useState("spieler");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [image, setImage] = useState<{ file: File; previewUrl: string } | null>(
    null,
  );
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    api
      .get("/teams")
      .then((r) => setTeams(r.data ?? []))
      .catch(() => {});
  }, []);

  useEffect(() => {
    return () => {
      if (image) URL.revokeObjectURL(image.previewUrl);
    };
  }, [image]);

  const pickImage = (file: File) => {
    if (!file.type.startsWith("image/")) return;
    setImage((prev) => {
      if (prev) URL.revokeObjectURL(prev.previewUrl);
      return { file, previewUrl: URL.createObjectURL(file) };
    });
  };

  const submit = async () => {
    if (!body.trim() && !image) return;
    setLoading(true);
    setError("");
    try {
      let mediaId: number | null = null;
      if (image) {
        const { blob, fileName } = await compressImage(image.file);
        const form = new FormData();
        form.append("image", blob, fileName);
        const r = await api.post("/media/upload", form);
        mediaId = r.data?.mediaId ?? null;
        if (mediaId === null) throw new Error("upload failed");
      }
      await api.post("/chat/broadcasts", {
        body: body.trim(),
        targetType,
        targetId,
        targetRole,
        mediaId,
      });
      onSent();
    } catch (e) {
      setError(errorMessage(e, "Fehler beim Senden"));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-md">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold text-brand-text">
            Mitteilung senden
          </h2>
          <button onClick={onClose} aria-label="Schließen">
            <X className="w-5 h-5 text-brand-text-muted" />
          </button>
        </div>

        <label className="block text-sm font-medium text-brand-text mb-1">
          Zielgruppe
        </label>
        <select
          value={targetType}
          onChange={(e) =>
            setTargetType(e.target.value as "all" | "team" | "role")
          }
          className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow mb-3"
        >
          {isAdmin && <option value="all">Alle Mitglieder</option>}
          <option value="team">Team</option>
          {isAdmin && <option value="role">Rolle</option>}
        </select>

        {targetType === "team" && (
          <select
            value={targetId}
            onChange={(e) => setTargetId(Number(e.target.value))}
            className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow mb-3"
          >
            <option value={0}>Team wählen…</option>
            {teams.map((t) => (
              <option key={t.id} value={t.id}>
                {teamShortNames.get(t.id) ?? t.name}
              </option>
            ))}
          </select>
        )}

        {targetType === "role" && (
          <select
            value={targetRole}
            onChange={(e) => setTargetRole(e.target.value)}
            className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow mb-3"
          >
            <option value="spieler">Spieler</option>
            <option value="elternteil">Elternteile</option>
            <option value="trainer">Trainer</option>
          </select>
        )}

        <label className="block text-sm font-medium text-brand-text mb-1">
          Nachricht
        </label>
        <textarea
          value={body}
          onChange={(e) => setBody(e.target.value)}
          maxLength={2000}
          rows={5}
          placeholder="Deine Mitteilung…"
          className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow resize-none mb-1"
        />
        <p className="text-xs text-brand-text-subtle text-right mb-2">
          {body.length}/2000
        </p>

        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          className="hidden"
          onChange={(e) => {
            const file = e.target.files?.[0];
            e.target.value = "";
            if (file) pickImage(file);
          }}
        />
        {image ? (
          <div className="flex items-center gap-3 mb-3">
            <img
              src={image.previewUrl}
              alt="Vorschau"
              className="h-16 w-16 object-cover rounded-md border border-brand-border-subtle"
            />
            <span className="flex-1 min-w-0 text-xs text-brand-text-muted truncate">
              {image.file.name}
            </span>
            <button
              onClick={() => setImage(null)}
              aria-label="Bild entfernen"
              className="text-brand-text-muted hover:text-brand-text shrink-0"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        ) : (
          <button
            onClick={() => fileInputRef.current?.click()}
            className="flex items-center gap-2 text-sm text-brand-text-muted hover:text-brand-text transition-colors mb-3"
          >
            <Paperclip className="w-4 h-4" />
            Bild anhängen
          </button>
        )}

        {error && <p className="text-brand-danger text-sm mb-3">{error}</p>}

        <button
          onClick={submit}
          disabled={
            loading ||
            (!body.trim() && !image) ||
            (targetType === "team" && !targetId)
          }
          className="w-full bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {loading ? "Sende…" : "Mitteilung senden"}
        </button>
      </div>
    </div>
  );
}

// --- Broadcast Edit Modal ---
function BroadcastEditModal({
  broadcast,
  onClose,
  onSaved,
}: {
  broadcast: Broadcast;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [body, setBody] = useState(broadcast.body);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const submit = async () => {
    if (!body.trim()) return;
    setLoading(true);
    setError("");
    try {
      await api.put(`/chat/broadcasts/${broadcast.id}`, { body: body.trim() });
      onSaved();
    } catch (e) {
      setError(errorMessage(e, "Fehler beim Speichern"));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-md">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold text-brand-text">
            Mitteilung bearbeiten
          </h2>
          <button onClick={onClose} aria-label="Schließen">
            <X className="w-5 h-5 text-brand-text-muted" />
          </button>
        </div>

        <textarea
          value={body}
          onChange={(e) => setBody(e.target.value)}
          maxLength={2000}
          rows={5}
          className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow resize-none mb-1"
        />
        <p className="text-xs text-brand-text-subtle text-right mb-3">
          {body.length}/2000
        </p>

        {error && <p className="text-brand-danger text-sm mb-3">{error}</p>}

        <button
          onClick={submit}
          disabled={loading || !body.trim()}
          className="w-full bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {loading ? "Speichere…" : "Speichern"}
        </button>
      </div>
    </div>
  );
}
