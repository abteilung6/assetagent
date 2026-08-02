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
    expect(screen.getByText(/confirm to save/i)).toBeInTheDocument();
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
    expect(screen.getByLabelText("Account name")).toHaveValue("DE89…3000");
    expect(
      screen.getByRole("button", { name: "Confirm import" }),
    ).toBeEnabled();
  });

  it("commits after naming the account", async () => {
    const user = userEvent.setup();
    vi.spyOn(sdk, "postImportsPreview").mockResolvedValue(
      mockApiResponse(sampleImportPreview()),
    );
    const commitSpy = vi.spyOn(sdk, "postImports").mockResolvedValue(
      mockApiResponse({
        import_run_id: "11111111-1111-1111-1111-111111111111",
        account_id: "22222222-2222-2222-2222-222222222222",
        account_name: "Sparkasse",
        rows: 6,
        inserted: 6,
        duplicates: 0,
      }),
    );
    vi.spyOn(sdk, "getTransactions").mockResolvedValue(
      mockApiResponse({
        data: [],
        pagination: { limit: 50, offset: 0, total: 0 },
      }),
    );

    testRender({ route: "/imports" });
    expect(
      await screen.findByText(/Drop a Sparkasse CSV here/i),
    ).toBeInTheDocument();

    await user.upload(
      screen.getByTestId("import-file-input"),
      new File(["csv"], "minimal.csv", { type: "text/csv" }),
    );

    const accountInput = await screen.findByLabelText("Account name");
    await user.clear(accountInput);
    await user.type(accountInput, "Sparkasse");
    await user.click(screen.getByRole("button", { name: "Confirm import" }));

    await waitFor(() => {
      expect(commitSpy).toHaveBeenCalled();
    });

    expect(await screen.findByText("Import complete")).toBeInTheDocument();
    expect(screen.getByText("New transactions")).toBeInTheDocument();
    expect(screen.getByText("Sparkasse")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "View transactions" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Prepare first Money Review" }),
    ).toBeDisabled();
  });

  it("shows duplicate messaging when nothing new is inserted", async () => {
    const user = userEvent.setup();
    vi.spyOn(sdk, "postImportsPreview").mockResolvedValue(
      mockApiResponse(sampleImportPreview()),
    );
    vi.spyOn(sdk, "postImports").mockResolvedValue(
      mockApiResponse({
        import_run_id: "11111111-1111-1111-1111-111111111111",
        account_id: "22222222-2222-2222-2222-222222222222",
        account_name: "Sparkasse",
        rows: 6,
        inserted: 0,
        duplicates: 6,
      }),
    );

    testRender({ route: "/imports" });
    expect(
      await screen.findByText(/Drop a Sparkasse CSV here/i),
    ).toBeInTheDocument();
    await user.upload(
      screen.getByTestId("import-file-input"),
      new File(["csv"], "minimal.csv", { type: "text/csv" }),
    );
    await user.click(
      await screen.findByRole("button", { name: "Confirm import" }),
    );

    expect(
      await screen.findByText("Nothing new to import"),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/already in your ledger/i),
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

  it("shows commit errors on the preview step", async () => {
    const user = userEvent.setup();
    vi.spyOn(sdk, "postImportsPreview").mockResolvedValue(
      mockApiResponse(sampleImportPreview()),
    );
    vi.spyOn(sdk, "postImports").mockRejectedValue({
      error: "validation_failed",
      message: "preview_hash does not match uploaded file",
    });

    testRender({ route: "/imports" });
    expect(
      await screen.findByText(/Drop a Sparkasse CSV here/i),
    ).toBeInTheDocument();
    await user.upload(
      screen.getByTestId("import-file-input"),
      new File(["csv"], "minimal.csv", { type: "text/csv" }),
    );
    await user.click(
      await screen.findByRole("button", { name: "Confirm import" }),
    );

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "preview_hash does not match uploaded file",
    );
    expect(screen.getByText("Sample rows")).toBeInTheDocument();
  });
});
