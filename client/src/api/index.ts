import type {
  DeviceAPI,
  KeyBundleAPI,
  RegisterDeviceRequest,
} from "../features/e2ee/types";
import {
  getSelectedServer,
  loadServers,
  setSelectedServer,
  type ZenthrilServer,
} from "../config/servers";
import {
  loadAccessToken,
  notifySessionExpired,
  saveAccessToken,
} from "../store/auth";

/**
 * API клиент — fetch к бэкенду.
 * Локально: тот же хост, порт 8080. Продакшен: задайте `VITE_API_BASE` при сборке (см. docs/DEPLOYMENT.md).
 */

function trimTrailingSlash(s: string): string {
  return s.replace(/\/+$/, "");
}

let activeServer: ZenthrilServer | null = null;
let serverLoadPromise: Promise<ZenthrilServer[]> | null = null;

async function getServerPool(): Promise<ZenthrilServer[]> {
  if (!serverLoadPromise) {
    serverLoadPromise = loadServers();
  }
  return serverLoadPromise;
}

export async function reloadServerPool(): Promise<ZenthrilServer[]> {
  serverLoadPromise = loadServers();
  const servers = await serverLoadPromise;
  activeServer = getSelectedServer(servers);
  return servers;
}

export async function getActiveServer(): Promise<ZenthrilServer> {
  const servers = await getServerPool();
  if (!activeServer) {
    activeServer = getSelectedServer(servers);
  }
  return activeServer;
}

export function clearActiveServer(): void {
  activeServer = null;
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
  // When no explicit API base is configured the frontend and backend are served
  // from the same origin (e.g. both behind the same ngrok / reverse-proxy URL).
  return trimTrailingSlash(window.location.origin);
}

/** WebSocket к тому же API-origin (или к тому же origin страницы в dev). */
export function getWebSocketUrl(path = "/ws"): string {
  const raw = import.meta.env.VITE_API_BASE?.trim();
  if (raw) {
    const base = trimTrailingSlash(raw);
    let u: URL;
    try {
      u = new URL(base);
    } catch {
      return `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.host}${path}`;
    }
    const wsProto = u.protocol === "https:" ? "wss:" : "ws:";
    const p = path.startsWith("/") ? path : `/${path}`;
    return `${wsProto}//${u.host}${p}`;
  }
  // Same-origin fallback: use the page's own host (works with ngrok / reverse proxy).
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${window.location.host}${path}`;
}

export async function getActiveWebSocketUrl(path = "/ws"): Promise<string> {
  const server = await getActiveServer();
  const p = path.startsWith("/") ? path : `/${path}`;
  return `${server.wsBase}${p}`;
}

let refreshInFlight: Promise<string | null> | null = null;

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  auth = true,
): Promise<T> {
  const servers = await getServerPool();
  const selected = await getActiveServer();
  const orderedServers = [
    selected,
    ...servers.filter(server => server.apiBase !== selected.apiBase),
  ];
  let lastError: unknown;

  for (const server of orderedServers) {
    try {
      return await requestFromServer<T>(server, method, path, body, auth);
    } catch (err) {
      lastError = err;
      if (!isNetworkLikeError(err)) throw err;
      // CONNECTIVITY: when a server is unreachable, try the next configured backup endpoint.
      continue;
    }
  }

  throw lastError instanceof Error ? lastError : new Error("All configured servers are unavailable");
}

async function requestFromServer<T>(
  server: ZenthrilServer,
  method: string,
  path: string,
  body?: unknown,
  auth = true,
): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    // Skip the ngrok browser-warning page when the backend is tunneled via ngrok.
    "ngrok-skip-browser-warning": "true",
  };

  if (auth) {
    const token = loadAccessToken();
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }
  }

  const init: RequestInit = {
    method,
    headers,
    // SECURITY: send HttpOnly refresh cookies to the selected API origin; the
    // refresh token itself remains unavailable to JavaScript.
    credentials: "include",
  };
  if (body !== undefined) {
    init.body = JSON.stringify(body);
  }

  const res = await fetch(`${server.apiBase}${path}`, init);

  if (res.status === 401 && auth && path !== "/api/v1/auth/refresh") {
    const refreshedToken = await refreshAccessToken(server);
    if (refreshedToken) {
      return requestFromServer<T>(server, method, path, body, auth);
    }
  }

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "unknown" }));
    throw Object.assign(new Error(err.message || "Request failed"), {
      status: res.status,
      code: err.error,
    });
  }

  if (!activeServer || activeServer.apiBase !== server.apiBase) {
    activeServer = server;
    setSelectedServer(server.id);
  }

  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

async function refreshAccessToken(server: ZenthrilServer): Promise<string | null> {
  if (!refreshInFlight) {
    refreshInFlight = requestFromServer<RefreshResponse>(
      server,
      "POST",
      "/api/v1/auth/refresh",
      undefined,
      false,
    )
      .then(response => {
        saveAccessToken(response.access_token ?? response.token);
        return response.access_token ?? response.token;
      })
      .catch(() => {
        notifySessionExpired();
        return null;
      })
      .finally(() => {
        refreshInFlight = null;
      });
  }
  return refreshInFlight;
}

export async function restoreAccessTokenFromRefreshCookie(): Promise<string | null> {
  const server = await getActiveServer();
  // SECURITY-HARDENING: restore a memory-only access token from the HttpOnly
  // refresh cookie without exposing refresh-token material to JavaScript.
  return refreshAccessToken(server);
}

function isNetworkLikeError(err: unknown): boolean {
  return err instanceof TypeError || err instanceof DOMException;
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

export interface RefreshResponse {
  access_token: string;
  refresh_token?: string;
  expires_in: number;
  token: string;
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
    logoutAll: () => request<void>("POST", "/api/v1/auth/logout-all"),

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
    devices: (userId: string) =>
      request<{ devices: DeviceAPI[] }>(
        "GET",
        `/api/v1/users/${encodeURIComponent(userId)}/devices`,
      ),
  },

  devices: {
    listOwn: () => request<{ devices: DeviceAPI[] }>("GET", "/api/v1/devices/"),
    register: (body: RegisterDeviceRequest) =>
      request<DeviceAPI>("POST", "/api/v1/devices/register", body),
    revoke: (deviceId: string) =>
      request<void>("DELETE", `/api/v1/devices/${encodeURIComponent(deviceId)}`),
  },

  keyBundles: {
    claim: (userId: string, deviceId: string) =>
      request<KeyBundleAPI>("POST", "/api/v1/key-bundles/claim", {
        user_id: userId,
        device_id: deviceId,
      }),
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
  tag: string;
  protocol_version: number;
  channel_id?: string;
  sender_user_id?: string;
  sender_device_id?: string;
  session_id?: string;
  client_message_id?: string;
  cipher_suite?: string;
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
