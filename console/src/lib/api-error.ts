export function apiErrorMessage(error: unknown, fallback: string): string {
  if (typeof error === "object" && error !== null) {
    const record = error as { message?: unknown };
    if (typeof record.message === "string" && record.message.trim()) {
      return record.message;
    }
  }
  if (error instanceof Error && error.message.trim()) {
    return error.message;
  }
  return fallback;
}
