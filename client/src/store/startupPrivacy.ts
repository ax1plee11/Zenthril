const AUTO_CONNECT_KEY = "zenthril_auto_connect_on_startup";

export function loadAutoConnectOnStartup(): boolean {
  return localStorage.getItem(AUTO_CONNECT_KEY) === "true";
}

export function saveAutoConnectOnStartup(enabled: boolean): void {
  localStorage.setItem(AUTO_CONNECT_KEY, enabled ? "true" : "false");
}

export function shouldStartOnline(hasStoredSession: boolean): boolean {
  // PRIVACY: default is fail-closed for network activity at startup.
  return hasStoredSession && loadAutoConnectOnStartup();
}

