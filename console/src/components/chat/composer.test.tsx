import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import * as sdk from "@/api/sdk.gen";
import { Composer } from "@/components/chat/composer";
import { mockApiResponse } from "@/test/mocks";
import { testRender } from "@/test/render";

function mockStreamResponse(body: string) {
  const encoder = new TextEncoder();
  const stream = new ReadableStream({
    start(controller) {
      controller.enqueue(encoder.encode(body));
      controller.close();
    },
  });

  return new Response(stream, {
    status: 200,
    headers: { "Content-Type": "text/event-stream" },
  });
}

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

  it("shows a stop button while streaming", async () => {
    const user = userEvent.setup();
    const onStop = vi.fn();

    testRender(
      <Composer onSend={vi.fn()} onStop={onStop} isStreaming disabled />,
    );

    await user.click(screen.getByRole("button", { name: "Stop response" }));

    expect(onStop).toHaveBeenCalledTimes(1);
  });
});

describe("ChatPage", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.spyOn(sdk, "getHealth").mockResolvedValue(
      mockApiResponse({ status: "ok" }),
    );
    vi.spyOn(sdk, "getLlmModels").mockResolvedValue(
      mockApiResponse({
        default: { provider: "ollama", model: "gemma4:12b" },
        options: [
          {
            provider: "ollama",
            model: "gemma4:12b",
            label: "Gemma 4 12B",
          },
        ],
      }),
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("submits a message to the streaming chat API", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      mockStreamResponse(
        [
          'event: tool_start',
          'data: {"name":"get_cashflow","input":{"from":"2025-12-01","to":"2025-12-31"}}',
          "",
          'event: tool_result',
          'data: {"name":"get_cashflow","result":{"income":"3200.00","expenses":"15.98","net":"3184.02","currency":"EUR"}}',
          "",
          'event: delta',
          'data: {"content":"You spent 15.98 EUR in December."}',
          "",
          'event: done',
          'data: {"answer":"You spent 15.98 EUR in December.","tool_calls":[{"name":"get_cashflow","input":{"from":"2025-12-01","to":"2025-12-31"},"result":{"income":"3200.00","expenses":"15.98","net":"3184.02","currency":"EUR"}}]}',
          "",
        ].join("\n"),
      ),
    );

    testRender({ route: "/chat" });

    const input = await screen.findByPlaceholderText("Ask about your finances…");
    await waitFor(() => {
      expect(input).not.toBeDisabled();
    });
    await user.type(input, "How much did I spend in December?");
    await user.click(screen.getByRole("button", { name: "Send message" }));

    expect(
      await screen.findByText("You spent 15.98 EUR in December."),
    ).toBeInTheDocument();
    expect(screen.getByText("Based on your data")).toBeInTheDocument();
    expect(screen.getByText("Spending summary")).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /view transactions/i }),
    ).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/chat/stream",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          messages: [
            {
              role: "user",
              content: "How much did I spend in December?",
            },
          ],
          provider: "ollama",
          model: "gemma4:12b",
        }),
      }),
    );
  });
});
