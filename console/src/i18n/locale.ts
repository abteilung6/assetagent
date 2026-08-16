import { readStorage, writeStorage } from "@/lib/storage";

export const LOCALES = ["de", "en"] as const;

export type Locale = (typeof LOCALES)[number];

export const DEFAULT_LOCALE: Locale = "de";

export const LOCALE_STORAGE_KEY = "assetagent.locale";

export function isLocale(value: string | null | undefined): value is Locale {
  return value === "de" || value === "en";
}

/** Map BCP-47-ish tags onto product locales; unknown → DEFAULT_LOCALE. */
export function normalizeLocale(raw: string | null | undefined): Locale {
  const tag = (raw ?? "").trim().toLowerCase();
  if (!tag) {
    return DEFAULT_LOCALE;
  }
  const primary = tag.split(/[-_]/)[0] ?? "";
  if (primary === "en") {
    return "en";
  }
  if (primary === "de") {
    return "de";
  }
  return DEFAULT_LOCALE;
}

export function readStoredLocale(): Locale | null {
  const stored = readStorage(LOCALE_STORAGE_KEY);
  return isLocale(stored) ? stored : null;
}

export function persistLocale(locale: Locale): void {
  writeStorage(LOCALE_STORAGE_KEY, locale);
}

/**
 * Resolve UI locale before /api/me is available.
 * Order: localStorage → navigator → default.
 */
export function resolveInitialLocale(
  navigatorLanguage: string | undefined = typeof navigator !== "undefined"
    ? navigator.language
    : undefined,
): Locale {
  return readStoredLocale() ?? normalizeLocale(navigatorLanguage);
}
