/**
 * Notifications — системные уведомления
 * Использует Tauri notification API если доступен, иначе Web Notifications API.
 * Требования: 10.8
 */

import { isTauri } from "../crypto/index";

/**
 * Запрашивает разрешение на показ уведомлений (Web Notifications API).
 * В Tauri-окружении разрешение управляется через плагин — вызов не нужен.
 */
export async function requestNotificationPermission(): Promise<boolean> {
  if (isTauri()) {
    // Tauri plugin-notification управляет разрешениями через tauri.conf.json
    return true;
  }
  if (!("Notification" in window)) {
    return false;
  }
  if (Notification.permission === "granted") {
    return true;
  }
  if (Notification.permission === "denied") {
    return false;
  }
  const result = await Notification.requestPermission();
  return result === "granted";
}

/**
 * Показывает системное уведомление.
 * В Tauri-окружении использует invoke("show_notification"),
 * иначе Web Notifications API.
 */
export async function showNotification(
  title: string,
  body: string,
): Promise<void> {
  if (isTauri()) {
    try {
      const { invoke } = await import("@tauri-apps/api/core");
      await invoke("show_notification", { title, body });
    } catch (err) {
      console.warn("[Notifications] Tauri notification failed:", err);
    }
    return;
  }

  if (!("Notification" in window)) {
    return;
  }

  if (Notification.permission !== "granted") {
    const granted = await requestNotificationPermission();
    if (!granted) return;
  }

  try {
    new Notification(title, { body });
  } catch (err) {
    console.warn("[Notifications] Web Notification failed:", err);
  }
}
