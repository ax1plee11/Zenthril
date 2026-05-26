export const isTauri =
  typeof window !== "undefined" && "__TAURI_INTERNALS__" in window;

export async function openUrl(url: string): Promise<void> {
  if (isTauri) {
    const { open } = await import("@tauri-apps/plugin-shell");
    await open(url);
  } else {
    window.open(url, "_blank", "noopener,noreferrer");
  }
}
