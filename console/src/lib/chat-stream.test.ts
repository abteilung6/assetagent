import { describe, expect, it, vi } from "vitest";

import { streamChat, type ChatStreamEvent } from "@/lib/chat-stream";

function mockStreamResponse(body: string, status = 200) {
  const encoder = new TextEncoder();
  const stream = new ReadableStream({
    start(controller) {
      controller.enqueue(encoder.encode(body));
      controller.close();
    },
  });

  return new Response(stream, {
    status,
    headers: { "Content-Type": "text/event-stream" },
  });
}

describe("streamChat", () => {
  it("parses delta, tool, and done events", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      mockStreamResponse(
        [
          'event: tool_start',
          'data: {"name":"get_cashflow","input":{"from":"2025-12-01","to":"2025-12-31"}}',
          "",
          'event: tool_result',
          'data: {"name":"get_cashflow","result":{"expenses":"17247.91","currency":"EUR"}}',
          "",
          'event: delta',
          'data: {"content":"You spent "}',
          "",
          'event: delta',
          'data: {"content":"€17,247.91."}',
          "",
          'event: done',
          'data: {"answer":"You spent €17,247.91.","tool_calls":[{"name":"get_cashflow","input":{"from":"2025-12-01","to":"2025-12-31"},"result":{"expenses":"17247.91","currency":"EUR"}}]}',
          "",
        ].join("\n"),
      ),
    );

    const events: ChatStreamEvent[] = [];
    await streamChat({
      messages: [{ role: "user", content: "How much did I spend?" }],
      selection: { provider: "openrouter", model: "openai/gpt-4o-mini" },
      onEvent: (event) => {
        events.push(event);
      },
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/chat/stream",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          messages: [{ role: "user", content: "How much did I spend?" }],
          provider: "openrouter",
          model: "openai/gpt-4o-mini",
        }),
      }),
    );

    expect(events).toEqual([
      {
        type: "tool_start",
        name: "get_cashflow",
        input: { from: "2025-12-01", to: "2025-12-31" },
      },
      {
        type: "tool_result",
        name: "get_cashflow",
        result: { expenses: "17247.91", currency: "EUR" },
      },
      { type: "delta", content: "You spent " },
      { type: "delta", content: "€17,247.91." },
      {
        type: "done",
        answer: "You spent €17,247.91.",
        toolCalls: [
          {
            name: "get_cashflow",
            input: { from: "2025-12-01", to: "2025-12-31" },
            result: { expenses: "17247.91", currency: "EUR" },
          },
        ],
      },
    ]);
  });

  it("emits error events for failed responses", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ message: "Provider unavailable" }), {
        status: 503,
        headers: { "Content-Type": "application/json" },
      }),
    );

    const events: ChatStreamEvent[] = [];
    await expect(
      streamChat({
        messages: [{ role: "user", content: "Hello" }],
        selection: { provider: "ollama", model: "gemma4:12b" },
        onEvent: (event) => {
          events.push(event);
        },
      }),
    ).rejects.toThrow("Provider unavailable");

    expect(events).toEqual([
      { type: "error", message: "Provider unavailable" },
    ]);
  });
});
