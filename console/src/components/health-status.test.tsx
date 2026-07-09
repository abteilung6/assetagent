import { screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import * as sdk from "@/api/sdk.gen";
import { HealthStatus } from "@/components/health-status";
import { mockApiResponse } from "@/test/mocks";
import { testRender } from "@/test/render";

describe("HealthStatus", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("shows API status when getHealth succeeds", async () => {
    vi.spyOn(sdk, "getHealth").mockResolvedValue(
      mockApiResponse({ status: "ok" }),
    );

    testRender(<HealthStatus />);

    await waitFor(() => {
      expect(screen.getByTestId("health-status")).toHaveTextContent("API ok");
    });
  });

  it("shows offline when getHealth fails", async () => {
    vi.spyOn(sdk, "getHealth").mockRejectedValue(new Error("network"));

    testRender(<HealthStatus />);

    await waitFor(() => {
      expect(screen.getByTestId("health-status")).toHaveTextContent(
        "API offline",
      );
    });
  });
});
