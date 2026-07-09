import { screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { TransactionDetailSheet } from "@/components/transaction-detail/sheet";
import { sampleTransaction } from "@/test/fixtures";
import { testRender } from "@/test/render";

describe("TransactionDetailSheet", () => {
  it("renders selected transaction fields", () => {
    testRender(
      <TransactionDetailSheet
        transaction={sampleTransaction({
          end_to_end_reference: "E2E-123",
          counterparty_iban: "DE89370400440532013000",
        })}
        open
        onOpenChange={vi.fn()}
      />,
    );

    const dialog = screen.getByRole("dialog");
    expect(dialog).toBeInTheDocument();
    expect(
      within(dialog).getByRole("heading", { name: "REWE Markt GmbH" }),
    ).toBeInTheDocument();
    expect(within(dialog).getByText("REWE Dortmund")).toBeInTheDocument();
    expect(within(dialog).getByText("CARD_PAYMENT")).toBeInTheDocument();
    expect(within(dialog).getByText("6011880043")).toBeInTheDocument();
    expect(within(dialog).getByText("E2E-123")).toBeInTheDocument();
    expect(within(dialog).getByText("DE89370400440532013000")).toBeInTheDocument();
    expect(within(dialog).getByText("-42.50")).toBeInTheDocument();
    expect(within(dialog).getByText("EUR")).toBeInTheDocument();
  });
});
