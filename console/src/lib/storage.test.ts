import { describe, expect, it, beforeEach, afterEach, vi } from "vitest";

import {
  readStorage,
  writeStorage,
  removeStorage,
  readJSONStorage,
  writeJSONStorage,
} from "@/lib/storage";

describe("storage", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("round-trips string values", () => {
    writeStorage("k", "v");
    expect(readStorage("k")).toBe("v");
    removeStorage("k");
    expect(readStorage("k")).toBeNull();
  });

  it("round-trips JSON values", () => {
    writeJSONStorage("prefs", { locale: "en" });
    expect(readJSONStorage<{ locale: string }>("prefs")).toEqual({
      locale: "en",
    });
  });

  it("returns null for invalid JSON", () => {
    writeStorage("bad", "{");
    expect(readJSONStorage("bad")).toBeNull();
  });

  it("swallows localStorage write failures", () => {
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new Error("quota");
    });
    expect(() => writeStorage("k", "v")).not.toThrow();
  });
});
