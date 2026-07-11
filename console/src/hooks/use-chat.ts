import { useCallback, useRef, useState } from "react";

import type { ChatMessage, ChatToolCall, LlmModelSelection } from "@/api/types.gen";
import { streamChat } from "@/lib/chat-stream";

export type ChatUIMessage = {
  id: string;
  role: "user" | "assistant";
  content: string;
  toolCalls?: ChatToolCall[];
};

function toApiMessages(messages: ChatUIMessage[]): ChatMessage[] {
  return messages.map((message) => ({
    role: message.role,
    content: message.content,
  }));
}

export function useChat(selection: LlmModelSelection | null) {
  const [messages, setMessages] = useState<ChatUIMessage[]>([]);
  const [isPending, setIsPending] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [streamingContent, setStreamingContent] = useState<string | null>(null);
  const [activeTool, setActiveTool] = useState<string | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const stop = useCallback(() => {
    abortRef.current?.abort();
  }, []);

  const send = useCallback(
    async (content: string) => {
      const trimmed = content.trim();
      if (!trimmed || isPending || !selection) {
        return;
      }

      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;

      const userMessage: ChatUIMessage = {
        id: crypto.randomUUID(),
        role: "user",
        content: trimmed,
      };
      const nextMessages = [...messages, userMessage];
      setMessages(nextMessages);
      setIsPending(true);
      setError(null);
      setStreamingContent("");
      setActiveTool(null);

      let answer = "";
      let finalized = false;
      const pendingToolCalls: ChatToolCall[] = [];

      const finalizeAssistant = (
        content: string,
        toolCalls?: ChatToolCall[],
      ) => {
        if (finalized) {
          return;
        }
        finalized = true;
        setMessages((current) => [
          ...current,
          {
            id: crypto.randomUUID(),
            role: "assistant",
            content,
            toolCalls: toolCalls?.length ? toolCalls : undefined,
          },
        ]);
      };

      try {
        await streamChat({
          messages: toApiMessages(nextMessages),
          selection,
          signal: controller.signal,
          onEvent: (event) => {
            switch (event.type) {
              case "delta":
                answer += event.content;
                setStreamingContent(answer);
                setActiveTool(null);
                break;
              case "tool_start":
                setActiveTool(event.name);
                pendingToolCalls.push({
                  name: event.name,
                  input: event.input,
                  result: {},
                });
                break;
              case "tool_result": {
                setActiveTool(null);
                let index = -1;
                for (let i = pendingToolCalls.length - 1; i >= 0; i -= 1) {
                  if (pendingToolCalls[i]?.name === event.name) {
                    index = i;
                    break;
                  }
                }
                if (index >= 0) {
                  pendingToolCalls[index] = {
                    ...pendingToolCalls[index],
                    result: event.result,
                  };
                }
                break;
              }
              case "done":
                answer = event.answer || answer;
                finalizeAssistant(
                  answer,
                  event.toolCalls.length > 0
                    ? event.toolCalls
                    : pendingToolCalls,
                );
                setStreamingContent(null);
                setActiveTool(null);
                break;
              case "error":
                setError(new Error(event.message));
                break;
            }
          },
        });

        if (!finalized && answer.trim()) {
          finalizeAssistant(answer, pendingToolCalls);
        }
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") {
          if (answer.trim()) {
            finalizeAssistant(answer, pendingToolCalls);
          }
        } else {
          setError(
            err instanceof Error
              ? err
              : new Error("The assistant is temporarily unavailable."),
          );
        }
      } finally {
        setIsPending(false);
        setStreamingContent(null);
        setActiveTool(null);
        if (abortRef.current === controller) {
          abortRef.current = null;
        }
      }
    },
    [messages, isPending, selection],
  );

  return {
    messages,
    send,
    stop,
    isPending,
    isStreaming: isPending,
    streamingContent,
    activeTool,
    error,
  };
}
