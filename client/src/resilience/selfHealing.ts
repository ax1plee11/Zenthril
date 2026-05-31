import {
  checkServerHealth,
  getSelectedServer,
  loadServers,
  setSelectedServer,
  type ZenthrilServer,
} from "../config/servers";

export type RecoveryKind = "primary" | "backup" | "custom";

export interface RecoveryAttempt {
  kind: RecoveryKind;
  label: string;
  server?: ZenthrilServer;
}

export async function buildRecoveryPlan(): Promise<RecoveryAttempt[]> {
  const servers = await loadServers();
  const selected = getSelectedServer(servers);
  const ordered = [selected, ...servers.filter(server => server.id !== selected.id)];
  const plan: RecoveryAttempt[] = [];

  for (const server of ordered) {
    plan.push({
      kind: server.custom ? "custom" : server.id.includes(":backup:") ? "backup" : "primary",
      label: server.name,
      server,
    });
  }

  return plan;
}

// CONNECTIVITY: backward-compatible export name; the plan now contains only
// administrator-configured endpoints and explicit custom servers.
export const buildFallbackPlan = buildRecoveryPlan;

export async function recoverReachableServer(): Promise<ZenthrilServer | null> {
  const plan = await buildRecoveryPlan();
  for (const attempt of plan) {
    if (!attempt.server) continue;
    // RESILIENCE: self-healing only selects reachable configured servers.
    // P2P direct messaging must remain an explicit user action in higher layers.
    if (await checkServerHealth(attempt.server)) {
      setSelectedServer(attempt.server.id);
      return attempt.server;
    }
  }
  return null;
}
