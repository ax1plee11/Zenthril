export interface ZenthrilServer {
  id: string;
  name: string;
  apiBase: string;
  wsBase: string;
  healthPath: string;
  mirrors: string[];
  transport: ServerTransport;
  onion?: boolean;
  custom?: boolean;
}

export type ServerTransport = "direct" | "tor" | "bridge";

interface ServerListFile {
  version?: number;
  servers?: Array<{
    id?: string;
    name?: string;
    api_base?: string;
    ws_base?: string;
    health_path?: string;
    mirrors?: string[];
    transport?: string;
    onion?: boolean;
  }>;
}

const SERVER_LIST_CACHE_KEY = "zenthril_server_list_cache_v1";
const SELECTED_SERVER_KEY = "zenthril_selected_server_id";
const CUSTOM_SERVERS_KEY = "zenthril_custom_servers_v1";
const DEFAULT_HEALTH_PATH = "/health";
const DEFAULT_DOH_ENDPOINT = "https://cloudflare-dns.com/dns-query";

export const LOCAL_SERVER: ZenthrilServer = {
  id: "local",
  name: "Local development",
  apiBase: "http://localhost:8080",
  wsBase: "ws://localhost:8080",
  healthPath: DEFAULT_HEALTH_PATH,
  mirrors: [],
  transport: "direct",
};

export function normalizeApiBase(value: string): string {
  const trimmed = value.trim().replace(/\/+$/, "");
  if (!trimmed) throw new Error("Server URL is required");
  const parsed = new URL(trimmed);
  if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
    throw new Error("Server URL must use http or https");
  }
  parsed.pathname = parsed.pathname.replace(/\/+$/, "");
  parsed.search = "";
  parsed.hash = "";
  return parsed.toString().replace(/\/+$/, "");
}

export function wsBaseFromApiBase(apiBase: string): string {
  const parsed = new URL(apiBase);
  parsed.protocol = parsed.protocol === "https:" ? "wss:" : "ws:";
  return parsed.toString().replace(/\/+$/, "");
}

export function serverFromApiBase(name: string, apiBase: string): ZenthrilServer {
  const normalized = normalizeApiBase(apiBase);
  return {
    id: `custom:${normalized}`,
    name: name.trim() || normalized,
    apiBase: normalized,
    wsBase: wsBaseFromApiBase(normalized),
    healthPath: DEFAULT_HEALTH_PATH,
    mirrors: [],
    transport: isOnionOrigin(normalized) ? "tor" : "direct",
    onion: isOnionOrigin(normalized),
    custom: true,
  };
}

export async function loadServers(): Promise<ZenthrilServer[]> {
  const fromEnv = parseEnvServers();
  const custom = loadCustomServers();
  if (fromEnv.length > 0) return uniqueServers([...fromEnv, ...custom]);

  try {
    const response = await fetchServerList();
    if (!response.ok) throw new Error(`servers.json HTTP ${response.status}`);
    const file = (await response.json()) as ServerListFile;
    const parsed = parseServerListFile(file);
    localStorage.setItem(SERVER_LIST_CACHE_KEY, JSON.stringify(parsed));
    return uniqueServers([...parsed, ...custom]);
  } catch {
    const cached = loadCachedServers();
    return uniqueServers([...(cached.length > 0 ? cached : [LOCAL_SERVER]), ...custom]);
  }
}

export async function resolveDoH(name: string, endpoint = DEFAULT_DOH_ENDPOINT): Promise<string[]> {
  const host = name.trim();
  if (!host || host.endsWith(".onion")) return [];

  const url = new URL(endpoint);
  url.searchParams.set("name", host);
  url.searchParams.set("type", "A");

  // ANTI-BLOCKING: DoH is used only as bootstrap metadata, because browsers do
  // not allow applications to override DNS resolution for fetch/WebSocket.
  const response = await fetch(url.toString(), {
    headers: { Accept: "application/dns-json" },
  });
  if (!response.ok) return [];

  const payload = await response.json() as {
    Answer?: Array<{ data?: string; type?: number }>;
  };
  return (payload.Answer ?? [])
    .filter(answer => answer.type === 1 && typeof answer.data === "string")
    .map(answer => answer.data!)
    .filter(Boolean);
}

export function getSelectedServer(servers: ZenthrilServer[]): ZenthrilServer {
  const selectedID = localStorage.getItem(SELECTED_SERVER_KEY);
  return servers.find(server => server.id === selectedID) ?? servers[0] ?? LOCAL_SERVER;
}

export function setSelectedServer(serverID: string): void {
  localStorage.setItem(SELECTED_SERVER_KEY, serverID);
  if (typeof window !== "undefined") {
    window.dispatchEvent(new CustomEvent("zenthril:server-changed", { detail: { serverID } }));
  }
}

export function loadCustomServers(): ZenthrilServer[] {
  try {
    const raw = localStorage.getItem(CUSTOM_SERVERS_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as ZenthrilServer[];
    return parsed.map(s => serverFromApiBase(s.name, s.apiBase));
  } catch {
    return [];
  }
}

export function addCustomServer(server: ZenthrilServer): void {
  const current = loadCustomServers().filter(item => item.apiBase !== server.apiBase);
  localStorage.setItem(CUSTOM_SERVERS_KEY, JSON.stringify([...current, server]));
}

export async function checkServerHealth(server: ZenthrilServer, timeoutMs = 4000): Promise<boolean> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    // ANTI-BLOCKING: health checks let the client skip blocked or unreachable servers before login.
    const response = await fetch(`${server.apiBase}${server.healthPath}`, {
      method: "GET",
      signal: controller.signal,
    });
    return response.ok || response.status === 401;
  } catch {
    return false;
  } finally {
    clearTimeout(timer);
  }
}

function parseEnvServers(): ZenthrilServer[] {
  const rawList = import.meta.env.VITE_SERVER_LIST?.trim();
  if (rawList) {
    try {
      const parsed = JSON.parse(rawList) as string[];
      return parsed.map((apiBase, index) => serverFromApiBase(`Server ${index + 1}`, apiBase));
    } catch {
      return [];
    }
  }

  const apiBase = import.meta.env.VITE_API_BASE?.trim();
  if (!apiBase) return [];
  return [serverFromApiBase("Configured server", apiBase)];
}

function parseServerListFile(file: ServerListFile): ZenthrilServer[] {
  const servers = file.servers ?? [];
  return servers.flatMap((server, index) => {
    try {
      const apiBase = normalizeApiBase(server.api_base ?? "");
      const primary: ZenthrilServer = {
        id: server.id?.trim() || `server-${index}`,
        name: server.name?.trim() || apiBase,
        apiBase,
        wsBase: server.ws_base?.trim().replace(/\/+$/, "") || wsBaseFromApiBase(apiBase),
        healthPath: server.health_path?.trim() || DEFAULT_HEALTH_PATH,
        mirrors: server.mirrors ?? [],
        transport: normalizeTransport(server.transport, apiBase),
        onion: server.onion ?? isOnionOrigin(apiBase),
      };
      // ANTI-BLOCKING: mirrors are promoted into normal fallback targets so a blocked primary
      // server does not require a separate app update or manual user intervention.
      return [primary, ...serverMirrors(primary)];
    } catch {
      return [];
    }
  });
}

function serverMirrors(primary: ZenthrilServer): ZenthrilServer[] {
  return primary.mirrors.flatMap((mirror, index) => {
    try {
      const apiBase = normalizeApiBase(mirror);
      return [{
        id: `${primary.id}:mirror:${index}`,
        name: `${primary.name} mirror ${index + 1}`,
        apiBase,
        wsBase: wsBaseFromApiBase(apiBase),
        healthPath: primary.healthPath,
        mirrors: [],
        transport: normalizeTransport(undefined, apiBase),
        onion: isOnionOrigin(apiBase),
      }];
    } catch {
      return [];
    }
  });
}

async function fetchServerList(): Promise<Response> {
  const serverListURL = import.meta.env.VITE_SERVER_LIST_URL?.trim();
  if (serverListURL) {
    return fetch(serverListURL, { cache: "no-store" });
  }

  const dohName = import.meta.env.VITE_SERVER_LIST_DOH_NAME?.trim();
  const dohTemplate = import.meta.env.VITE_SERVER_LIST_DOH_TEMPLATE?.trim();
  if (dohName && dohTemplate) {
    const addresses = await resolveDoH(dohName, import.meta.env.VITE_DOH_ENDPOINT?.trim() || DEFAULT_DOH_ENDPOINT);
    for (const address of addresses) {
      try {
        const url = dohTemplate.replace("{address}", address).replace("{name}", dohName);
        const response = await fetch(url, { cache: "no-store" });
        if (response.ok) return response;
      } catch {
        // continue to bundled fallback
      }
    }
  }

  return fetch("/servers.json", { cache: "no-store" });
}

function normalizeTransport(value: string | undefined, apiBase: string): ServerTransport {
  if (isOnionOrigin(apiBase)) return "tor";
  if (value === "tor" || value === "bridge") return value;
  return "direct";
}

function isOnionOrigin(value: string): boolean {
  try {
    return new URL(value).hostname.endsWith(".onion");
  } catch {
    return false;
  }
}

function loadCachedServers(): ZenthrilServer[] {
  try {
    const raw = localStorage.getItem(SERVER_LIST_CACHE_KEY);
    if (!raw) return [];
    return JSON.parse(raw) as ZenthrilServer[];
  } catch {
    return [];
  }
}

function uniqueServers(servers: ZenthrilServer[]): ZenthrilServer[] {
  const seen = new Set<string>();
  const out: ZenthrilServer[] = [];
  for (const server of servers) {
    if (seen.has(server.apiBase)) continue;
    seen.add(server.apiBase);
    out.push(server);
  }
  return out.length > 0 ? out : [LOCAL_SERVER];
}
