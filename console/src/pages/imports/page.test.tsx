import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import * as sdk from "@/api/sdk.gen";
import { sampleImportPreview } from "@/test/fixtures";
import { mockApiResponse } from "@/test/mocks";
import { testRender } from "@/test/render";

describe("ImportsPage", () => {
  beforeEach(() => {
    vi.spyOn(sdk, "getHealth").mockResolvedValue(
      mockApiResponse({ status: "ok" }),
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("renders empty upload state", async () => {
    testRender({ route: "/imports" });

    expect(
      await screen.findByRole("heading", { name: "Import" }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Drop a Sparkasse CSV here/i),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/preview only, no import yet/i),
    ).toBeInTheDocument();
  });

  it("previews a selected CSV via the API", async () => {
    const user = userEvent.setup();
    const preview = sampleImportPreview();
    const spy = vi
      .spyOn(sdk, "postImportsPreview")
      .mockResolvedValue(mockApiResponse(preview));

    testRender({ route: "/imports" });
    expect(
      await screen.findByText(/Drop a Sparkasse CSV here/i),
    ).toBeInTheDocument();

    const input = screen.getByTestId("import-file-input");
    const file = new File(["Auftragskonto;..."], "minimal.csv", {
      type: "text/csv",
    });
    await user.upload(input, file);

    await waitFor(() => {
      expect(spy).toHaveBeenCalled();
    });

    expect(await screen.findByText("Sample rows")).toBeInTheDocument();
    expect(screen.getByText("PayPal Europe")).toBeInTheDocument();
    expect(screen.getByText("Valid")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Confirm import" }),
    ).toBeDisabled();
    expect(
      screen.getByText(/Confirming the import comes in the next step/i),
    ).toBeInTheDocument();
  });

  it("shows API errors without committing", async () => {
    const user = userEvent.setup();
    vi.spyOn(sdk, "postImportsPreview").mockRejectedValue({
      error: "validation_failed",
      message: "csv: empty file",
    });

    testRender({ route: "/imports" });
    expect(
      await screen.findByText(/Drop a Sparkasse CSV here/i),
    ).toBeInTheDocument();

    const input = screen.getByTestId("import-file-input");
    const file = new File(["x"], "bad.csv", { type: "text/csv" });
    await user.upload(input, file);

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "csv: empty file",
    );
    expect(screen.queryByText("Sample rows")).not.toBeInTheDocument();
  });
});
