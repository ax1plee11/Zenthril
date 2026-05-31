import {
  checkServerHealth,
  getSelectedServer,
  loadServers,
  setSelectedServer,
  type ZenthrilServer,
} from "../config/servers";

export type FallbackKind = "primary" | "mirror" | "bridge" | "tor" | "p2p";

export interface FallbackAttempt {
  kind: FallbackKind;
  label: string;
  server?: ZenthrilServer;
}

export async function buildFallbackPlan(): Promise<FallbackAttempt[]> {
  const servers = await loadServers();
  const selected = getSelectedServer(servers);
  const ordered = [selected, ...servers.filter(server => server.id !== selected.id)];
  const plan: FallbackAttempt[] = [];

  for (const server of ordered) {
    plan.push({
      kind: server.transport === "tor" ? "tor" : server.id.includes(":mirror:") ? "mirror" : "primary",
      label: server.name,
      server,
    });
    for (const bridge of server.bridges) {
      plan.push({ kind: "bridge", label: bridge });
    }
  }

  plan.push({ kind: "p2p", label: "WebRTC direct fallback" });
  return plan;
}

export async function recoverReachableServer(): Promise<ZenthrilServer | null> {
  const plan = await buildFallbackPlan();
  for (const attempt of plan) {
    if (!attempt.server) continue;
    // RESILIENCE: self-healing prefers reachable configured servers before
    // escalating to bridge metadata or P2P-only modes.
    if (await checkServerHealth(attempt.server)) {
      setSelectedServer(attempt.server.id);
      return attempt.server;
    }
  }
  return null;
}
