import type {
  ChatMessage,
  ChatPageContext,
  ChatToolCall,
  LlmModelSelection,
} from "@/api/types.gen";

export type ChatStreamEvent =
  | { type: "delta"; content: string }
  | { type: "tool_start"; name: string; input: Record<string, unknown> }
  | { type: "tool_result"; name: string; result: Record<string, unknown> }
  | { type: "done"; answer: string; toolCalls: ChatToolCall[]; traceId?: string }
  | { type: "error"; message: string };

export type StreamChatOptions = {
  messages: ChatMessage[];
  selection: LlmModelSelection;
  context?: ChatPageContext;
  signal?: AbortSignal;
  onEvent: (event: ChatStreamEvent) => void;
};

export async function streamChat({
  messages,
  selection,
  context,
  signal,
  onEvent,
}: StreamChatOptions): Promise<void> {
  const response = await fetch("/api/chat/stream", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      messages,
      provider: selection.provider,
      model: selection.model,
      ...(context ? { context } : {}),
    }),
    signal,
  });

  if (!response.ok) {
    let message = "The assistant is temporarily unavailable.";
    try {
      const payload = (await response.json()) as { message?: string };
      if (payload.message) {
        message = payload.message;
      }
    } catch {
      // ignore parse errors
    }
    onEvent({ type: "error", message });
    throw new Error(message);
  }

  if (!response.body) {
    const message = "Streaming is not supported in this browser.";
    onEvent({ type: "error", message });
    throw new Error(message);
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) {
        break;
      }

      buffer += decoder.decode(value, { stream: true });
      const blocks = buffer.split("\n\n");
      buffer = blocks.pop() ?? "";

      for (const block of blocks) {
        const event = parseSSEBlock(block);
        if (!event) {
          continue;
        }
        dispatchStreamEvent(event, onEvent);
      }
    }

    if (buffer.trim()) {
      const event = parseSSEBlock(buffer);
      if (event) {
        dispatchStreamEvent(event, onEvent);
      }
    }
  } finally {
    reader.releaseLock();
  }
}

type ParsedSSE = {
  event: string;
  data: string;
};

function parseSSEBlock(block: string): ParsedSSE | null {
  const lines = block.split("\n");
  let event = "";
  const dataLines: string[] = [];

  for (const line of lines) {
    if (line.startsWith("event:")) {
      event = line.slice(6).trim();
    } else if (line.startsWith("data:")) {
      dataLines.push(line.slice(5).trim());
    }
  }

  if (!event || dataLines.length === 0) {
    return null;
  }

  return { event, data: dataLines.join("\n") };
}

function dispatchStreamEvent(
  parsed: ParsedSSE,
  onEvent: (event: ChatStreamEvent) => void,
): void {
  let payload: Record<string, unknown>;
  try {
    payload = JSON.parse(parsed.data) as Record<string, unknown>;
  } catch {
    onEvent({ type: "error", message: "Invalid response from server." });
    return;
  }

  switch (parsed.event) {
    case "delta":
      onEvent({
        type: "delta",
        content: typeof payload.content === "string" ? payload.content : "",
      });
      return;
    case "tool_start":
      onEvent({
        type: "tool_start",
        name: typeof payload.name === "string" ? payload.name : "",
        input: asRecord(payload.input),
      });
      return;
    case "tool_result":
      onEvent({
        type: "tool_result",
        name: typeof payload.name === "string" ? payload.name : "",
        result: asRecord(payload.result),
      });
      return;
    case "done":
      onEvent({
        type: "done",
        answer: typeof payload.answer === "string" ? payload.answer : "",
        toolCalls: parseToolCalls(payload.tool_calls),
        traceId:
          typeof payload.trace_id === "string" ? payload.trace_id : undefined,
      });
      return;
    case "error":
      onEvent({
        type: "error",
        message:
          typeof payload.message === "string"
            ? payload.message
            : "The assistant is temporarily unavailable.",
      });
      return;
    default:
      return;
  }
}

function parseToolCalls(raw: unknown): ChatToolCall[] {
  if (!Array.isArray(raw)) {
    return [];
  }

  return raw
    .map((item) => {
      if (!item || typeof item !== "object") {
        return null;
      }
      const record = item as Record<string, unknown>;
      const name = readString(record, "name", "Name");
      if (!name) {
        return null;
      }
      return {
        name,
        input: asRecord(record.input ?? record.Input),
        result: asRecord(record.result ?? record.Result),
      };
    })
    .filter((item): item is ChatToolCall => item !== null);
}

function readString(
  record: Record<string, unknown>,
  ...keys: string[]
): string | undefined {
  for (const key of keys) {
    const value = record[key];
    if (typeof value === "string" && value.length > 0) {
      return value;
    }
  }
  return undefined;
}

function asRecord(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return {};
  }
  return value as Record<string, unknown>;
}
