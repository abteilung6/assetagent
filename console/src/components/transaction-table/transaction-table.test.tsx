import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { TransactionTable } from "@/components/transaction-table/transaction-table";
import { sampleTransaction } from "@/test/fixtures";
import { testRender } from "@/test/render";

describe("TransactionTable", () => {
  it("renders transaction rows", () => {
    testRender(
      <TransactionTable
        transactions={[
          sampleTransaction(),
          sampleTransaction({
            id: "b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22",
            purpose: "Salary January",
            counterparty: "Employer GmbH",
            amount: "3200.00",
          }),
        ]}
      />,
    );

    expect(screen.getByText("REWE Dortmund")).toBeInTheDocument();
    expect(screen.getByText("Salary January")).toBeInTheDocument();
    expect(screen.getByText("-42.50 EUR")).toBeInTheDocument();
    expect(screen.getByText("3200.00 EUR")).toBeInTheDocument();
    expect(screen.getByText("REWE Markt GmbH")).toBeInTheDocument();
    expect(screen.getByRole("columnheader", { name: "Date" })).toBeInTheDocument();
    expect(screen.getByRole("columnheader", { name: "Amount" })).toBeInTheDocument();
    expect(
      screen.queryByRole("columnheader", { name: "Account" }),
    ).not.toBeInTheDocument();
  });
});
