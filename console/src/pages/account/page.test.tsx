import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import * as sdk from "@/api/sdk.gen";
import i18n from "@/i18n";
import { mockApiResponse } from "@/test/mocks";
import { testRender } from "@/test/render";

describe("AccountPage", () => {
  it("shows profile and switches language via PATCH /api/me", async () => {
    const user = userEvent.setup();
    vi.spyOn(sdk, "getMe").mockResolvedValue(
      mockApiResponse({
        user: {
          id: "00000000-0000-4000-8000-000000000001",
          display_name: "Ada Lovelace",
          given_name: "Ada",
          email: "ada@example.com",
          picture_url: "https://example.com/ada.png",
          preferred_locale: "en",
        },
        household: {
          id: "00000000-0000-4000-8000-000000000002",
          name: "Ada household",
        },
        membership: { role: "owner" as const },
      }),
    );
    const patch = vi.spyOn(sdk, "patchMe").mockResolvedValue(
      mockApiResponse({
        user: {
          id: "00000000-0000-4000-8000-000000000001",
          display_name: "Ada Lovelace",
          given_name: "Ada",
          email: "ada@example.com",
          picture_url: "https://example.com/ada.png",
          preferred_locale: "de",
        },
        household: {
          id: "00000000-0000-4000-8000-000000000002",
          name: "Ada household",
        },
        membership: { role: "owner" as const },
      }),
    );

    testRender({ route: "/account" });

    expect(await screen.findByText("Ada Lovelace")).toBeInTheDocument();
    expect(screen.getAllByText("ada@example.com").length).toBeGreaterThan(0);
    expect(
      screen.getByRole("button", { name: "Deutsch" }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Deutsch" }));

    await waitFor(() => {
      expect(patch).toHaveBeenCalled();
    });
    await waitFor(() => {
      expect(i18n.language).toBe("de");
    });
    expect(
      await screen.findByRole("heading", { name: "Konto" }),
    ).toBeInTheDocument();
  });
});
