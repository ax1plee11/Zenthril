/** Поля распространённых ошибок API-клиента (не гарантирует наличие обоих полей). */
export function apiErrorFields(err: unknown): {
  code?: string;
  message?: string;
} {
  if (typeof err !== "object" || err === null) {
    return {};
  }
  const o = err as Record<string, unknown>;
  const code = typeof o.code === "string" ? o.code : undefined;
  const message =
    typeof o.message === "string"
      ? o.message
      : err instanceof Error
        ? err.message
        : undefined;
  return { code, message };
}
