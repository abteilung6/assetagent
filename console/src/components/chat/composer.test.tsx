import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import * as sdk from "@/api/sdk.gen";
import { Composer } from "@/components/chat/composer";
import { mockApiResponse } from "@/test/mocks";
import { testRender } from "@/test/render";

describe("Composer", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("calls onSend with trimmed input", async () => {
    const user = userEvent.setup();
    const onSend = vi.fn().mockResolvedValue(undefined);

    testRender(<Composer onSend={onSend} />);

    await user.type(
      screen.getByPlaceholderText("Ask about your finances…"),
      "  How much did I spend?  ",
    );
    await user.click(screen.getByRole("button", { name: "Send message" }));

    expect(onSend).toHaveBeenCalledWith("How much did I spend?");
  });
});

describe("ChatPage", () => {
  beforeEach(() => {
    vi.spyOn(sdk, "getHealth").mockResolvedValue(
      mockApiResponse({ status: "ok" }),
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("submits a message to the chat API", async () => {
    const user = userEvent.setup();
    const postChat = vi.spyOn(sdk, "postChat").mockResolvedValue(
      mockApiResponse({
        answer: "You spent 15.98 EUR in December.",
        tool_calls: [
          {
            name: "get_cashflow",
            input: { from: "2025-12-01", to: "2025-12-31" },
            result: {
              income: "3200.00",
              expenses: "15.98",
              net: "3184.02",
              currency: "EUR",
            },
          },
        ],
      }),
    );

    testRender({ route: "/chat" });

    const input = await screen.findByPlaceholderText("Ask about your finances…");
    await user.type(input, "How much did I spend in December?");
    await user.click(screen.getByRole("button", { name: "Send message" }));

    expect(await screen.findByText("You spent 15.98 EUR in December.")).toBeInTheDocument();
    expect(screen.getByText("Based on your data")).toBeInTheDocument();
    expect(screen.getByText("Spending summary")).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /view transactions/i }),
    ).toBeInTheDocument();
    expect(postChat).toHaveBeenCalledWith(
      expect.objectContaining({
        body: {
          messages: [
            {
              role: "user",
              content: "How much did I spend in December?",
            },
          ],
        },
      }),
    );
  });
});
