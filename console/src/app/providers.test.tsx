import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import * as sdk from "@/api/sdk.gen";
import { AppProviders } from "@/app/providers";
import { HealthStatus } from "@/components/health-status";
import { ThemeProvider } from "@/components/theme-provider";
import { mockApiResponse } from "@/test/mocks";

describe("AppProviders", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("provides TanStack Query context for child hooks", async () => {
    vi.spyOn(sdk, "getHealth").mockResolvedValue(
      mockApiResponse({ status: "ok" }),
    );

    render(
      <ThemeProvider>
        <AppProviders>
          <HealthStatus />
        </AppProviders>
      </ThemeProvider>,
    );

    await waitFor(() => {
      expect(screen.getByTestId("health-status")).toHaveTextContent("API ok");
    });
  });
});
