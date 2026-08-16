import { describe, expect, it } from "vitest";

import { initialsFromName, primaryDisplayName } from "@/lib/user-display";

describe("user-display", () => {
  it("prefers given name", () => {
    expect(
      primaryDisplayName({
        givenName: "Ada",
        displayName: "Ada Lovelace",
        email: "ada@example.com",
      }),
    ).toBe("Ada");
  });

  it("falls back to first display token then email", () => {
    expect(
      primaryDisplayName({ displayName: "Ada Lovelace", email: "x@y.z" }),
    ).toBe("Ada");
    expect(primaryDisplayName({ displayName: "", email: "ada@example.com" })).toBe(
      "ada",
    );
  });

  it("builds initials", () => {
    expect(initialsFromName("Ada Lovelace")).toBe("AL");
    expect(initialsFromName("Ada")).toBe("AD");
  });
});
