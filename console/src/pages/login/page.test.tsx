import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import * as sdk from "@/api/sdk.gen";
import { mockApiResponse } from "@/test/mocks";
import { testRender } from "@/test/render";

describe("LoginPage", () => {
  beforeEach(() => {
    vi.spyOn(sdk, "getMe").mockRejectedValue({
      error: "unauthorized",
      message: "not authenticated",
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("renders Google sign-in CTA", async () => {
    testRender({ route: "/login" });

    expect(
      await screen.findByRole("heading", { name: "assetagent" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Login with Google" }),
    ).toBeInTheDocument();
    expect(screen.queryByText(/Weitere Optionen folgen/i)).not.toBeInTheDocument();
  });

  it("navigates to Google start URL on click", async () => {
    const assign = vi.fn();
    vi.stubGlobal("location", {
      ...window.location,
      assign,
    });

    testRender({ route: "/login" });
    await screen.findByRole("button", { name: "Login with Google" });
    await userEvent.click(
      screen.getByRole("button", { name: "Login with Google" }),
    );

    expect(assign).toHaveBeenCalledWith(
      expect.stringMatching(/\/auth\/google\/start$/),
    );
  });
});

describe("AuthGate", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("redirects to /login when /api/me returns 401", async () => {
    vi.spyOn(sdk, "getMe").mockRejectedValue({
      error: "unauthorized",
      message: "not authenticated",
    });

    testRender({ route: "/chat" });

    await waitFor(() => {
      expect(
        screen.getByRole("heading", { name: "assetagent" }),
      ).toBeInTheDocument();
    });
    expect(
      screen.getByRole("button", { name: "Login with Google" }),
    ).toBeInTheDocument();
  });

  it("renders app when /api/me succeeds", async () => {
    vi.spyOn(sdk, "getMe").mockResolvedValue(
      mockApiResponse({
        user: {
          id: "00000000-0000-4000-8000-000000000001",
          display_name: "Ada",
          email: "ada@example.com",
        },
        household: {
          id: "00000000-0000-4000-8000-000000000002",
          name: "Ada household",
        },
        membership: { role: "owner" as const },
      }),
    );
    vi.spyOn(sdk, "getHealth").mockResolvedValue(
      mockApiResponse({ status: "ok" }),
    );
    vi.spyOn(sdk, "postChat").mockResolvedValue(
      mockApiResponse({ answer: "Hello", tool_calls: [] }),
    );

    testRender({ route: "/chat" });

    expect(
      await screen.findByRole("heading", { name: "Chat" }),
    ).toBeInTheDocument();
  });
});
