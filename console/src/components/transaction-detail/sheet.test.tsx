import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import * as sdk from "@/api/sdk.gen";
import { TransactionDetailSheet } from "@/components/transaction-detail/sheet";
import { sampleTransaction } from "@/test/fixtures";
import { mockApiResponse } from "@/test/mocks";
import { testRender } from "@/test/render";

describe("TransactionDetailSheet", () => {
  it("marks a transaction as one-off", async () => {
    const user = userEvent.setup();
    const onTransactionChange = vi.fn();
    const tx = sampleTransaction();
    vi.spyOn(sdk, "postTransactionOneOff").mockResolvedValue(
      mockApiResponse({ ...tx, one_off: true }),
    );

    testRender(
      <TransactionDetailSheet
        transaction={tx}
        open
        onOpenChange={() => undefined}
        onTransactionChange={onTransactionChange}
      />,
    );

    await user.click(
      screen.getByRole("button", { name: "Treat as one-off" }),
    );

    await waitFor(() => {
      expect(sdk.postTransactionOneOff).toHaveBeenCalledWith(
        expect.objectContaining({
          path: { transaction_id: tx.id },
          body: { one_off: true },
        }),
      );
    });
    await waitFor(() => {
      expect(onTransactionChange).toHaveBeenCalledWith(
        expect.objectContaining({ one_off: true }),
      );
    });
  });
});
