import { describe, expect, it, beforeEach } from "vitest";

import {
  normalizeLocale,
  resolveInitialLocale,
  persistLocale,
  readStoredLocale,
  LOCALE_STORAGE_KEY,
} from "@/i18n/locale";

describe("locale", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("normalizes BCP-47 tags", () => {
    expect(normalizeLocale("en-US")).toBe("en");
    expect(normalizeLocale("de_AT")).toBe("de");
    expect(normalizeLocale("fr")).toBe("de");
    expect(normalizeLocale("")).toBe("de");
  });

  it("prefers stored locale over navigator", () => {
    persistLocale("en");
    expect(resolveInitialLocale("de-DE")).toBe("en");
    expect(readStoredLocale()).toBe("en");
    expect(window.localStorage.getItem(LOCALE_STORAGE_KEY)).toBe("en");
  });

  it("falls back to navigator then default", () => {
    expect(resolveInitialLocale("en-GB")).toBe("en");
    expect(resolveInitialLocale("ja")).toBe("de");
  });
});
