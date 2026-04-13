/**
 * API клиент — fetch к бэкенду.
 * Локально: тот же хост, порт 8080. Продакшен: задайте `VITE_API_BASE` при сборке (см. docs/DEPLOYMENT.md).
 */

function trimTrailingSlash(s: string): string {
  return s.replace(/\/+$/, "");
}

/**
 * Origin бэкенда без завершающего слэша.
 * `VITE_API_BASE` — полный origin, например `https://api.example.com` (встраивается на этапе `vite build`).
 */
export function getBackendOrigin(): string {
  const raw = import.meta.env.VITE_API_BASE?.trim();
  if (raw) {
    return trimTrailingSlash(raw);
  }
  return `${window.location.protocol}//${window.location.hostname}:8080`;
}

/** WebSocket к тому же API-origin (или к хосту страницы :8080 в dev). */
export function getWebSocketUrl(path = "/ws"): string {
  const raw = import.meta.env.VITE_API_BASE?.trim();
  if (raw) {
    const base = trimTrailingSlash(raw);
    let u: URL;
    try {
      u = new URL(base);
    } catch {
      return `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.hostname}:8080${path}`;
    }
    const wsProto = u.protocol === "https:" ? "wss:" : "ws:";
    const p = path.startsWith("/") ? path : `/${path}`;
    return `${wsProto}//${u.host}${p}`;
  }
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${window.location.hostname}:8080${path}`;
}

const BASE_URL = getBackendOrigin();
const TOKEN_KEY = "veltrix_token";

function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  auth = true,
): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };

  if (auth) {
    const token = getToken();
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }
  }

  const res = await fetch(`${BASE_URL}${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "unknown" }));
    throw Object.assign(new Error(err.message || "Request failed"), {
      status: res.status,
      code: err.error,
    });
  }

  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

// ─── Auth ─────────────────────────────────────────────────────────────────────

export interface RegisterResponse {
  user_id: string;
  token: string;
}

export interface LoginResponse {
  token: string;
  user: {
    id: string;
    username: string;
    public_key: string;
    created_at: string;
  };
}

export const api = {
  auth: {
    register: (username: string, password: string, public_key: string) =>
      request<RegisterResponse>("POST", "/api/v1/auth/register", {
        username,
        password,
        public_key,
      }, false),

    login: (username: string, password: string) =>
      request<LoginResponse>("POST", "/api/v1/auth/login", {
        username,
        password,
      }, false),

    logout: () => request<void>("POST", "/api/v1/auth/logout"),

    /** Одноразовый билет для WebSocket (не передавать JWT в URL) */
    wsTicket: () =>
      request<{ ticket: string }>("POST", "/api/v1/auth/ws-ticket"),
  },

  guilds: {
    list: () => request<GuildAPI[]>("GET", "/api/v1/guilds"),
    create: (name: string) =>
      request<GuildAPI>("POST", "/api/v1/guilds", { name }),
    channels: (guildId: string) =>
      request<ChannelAPI[]>("GET", `/api/v1/guilds/${guildId}/channels`),
    createChannel: (guildId: string, name: string, type: "text" | "voice") =>
      request<ChannelAPI>("POST", `/api/v1/guilds/${guildId}/channels`, {
        name,
        type,
      }),
    createInvite: (guildId: string) =>
      request<{ code: string }>("POST", `/api/v1/guilds/${guildId}/invites`),
    joinByInvite: (code: string) =>
      request<GuildAPI>("POST", `/api/v1/invites/${code}/join`),
    members: (guildId: string) =>
      request<{ id: string; username: string }[]>("GET", `/api/v1/guilds/${guildId}/members`),
    banMember: (guildId: string, userId: string) =>
      request<void>("POST", `/api/v1/guilds/${guildId}/members/${userId}/ban`),
    kickMember: (guildId: string, userId: string) =>
      request<void>("DELETE", `/api/v1/guilds/${guildId}/members/${userId}`),
  },

  admin: {
    globalBan: (userId: string, reason: string) =>
      request<void>("POST", `/api/v1/admin/users/${userId}/ban`, { reason }),
    globalUnban: (userId: string) =>
      request<void>("DELETE", `/api/v1/admin/users/${userId}/ban`),
  },

  users: {
    search: (q: string) =>
      request<UserSearchResult[]>("GET", `/api/v1/users/search?q=${encodeURIComponent(q)}`),
  },

  friends: {
    list: () => request<FriendUser[]>("GET", "/api/v1/friends"),
    sendRequest: (userId: string) =>
      request<void>("POST", "/api/v1/friends/request", { user_id: userId }),
    accept: (userId: string) =>
      request<void>("POST", `/api/v1/friends/${userId}/accept`),
    decline: (userId: string) =>
      request<void>("DELETE", `/api/v1/friends/${userId}`),
  },

  messages: {
    history: (channelId: string, before?: string) => {
      const qs = before ? `?before=${before}&limit=50` : "?limit=50";
      return request<MessageAPI[]>(
        "GET",
        `/api/v1/channels/${channelId}/messages${qs}`,
      );
    },
    send: (channelId: string, payload: EncryptedPayloadAPI) =>
      request<MessageAPI>("POST", `/api/v1/channels/${channelId}/messages`, {
        payload,
      }),
    edit: (messageId: string, payload: EncryptedPayloadAPI) =>
      request<MessageAPI>("PATCH", `/api/v1/messages/${messageId}`, {
        payload,
      }),
    delete: (messageId: string) =>
      request<void>("DELETE", `/api/v1/messages/${messageId}`),
  },
};

// ─── API типы (snake_case от бэкенда) ────────────────────────────────────────

export interface EncryptedPayloadAPI {
  ciphertext: string;
  iv: string;
  key_id: string;
  tag?: string;
}

export interface GuildAPI {
  id: string;
  name: string;
  owner_id: string;
  node_id: string;
  created_at: string;
}

export interface ChannelAPI {
  id: string;
  guild_id: string;
  name: string;
  type: "text" | "voice";
  position: number;
  created_at: string;
}

export interface UserSearchResult {
  id: string;
  username: string;
}

export interface FriendAPI {
  id: string;
  username: string;
  status?: "online" | "dnd" | "offline";
}

export interface FriendUser {
  id: string;
  username: string;
  status: "pending" | "accepted";
  direction?: "incoming" | "outgoing";
}

export interface MessageAPI {
  id: string;
  channel_id: string;
  author_id: string;
  payload: EncryptedPayloadAPI;
  edited: boolean;
  deleted: boolean;
  created_at: string;
  updated_at: string;
  // enriched on client
  author_username?: string;
  decryptedContent?: string;
}
