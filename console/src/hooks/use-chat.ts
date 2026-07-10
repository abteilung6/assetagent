import { useMutation } from "@tanstack/react-query";
import { useCallback, useState } from "react";

import { postChatMutation } from "@/api/@tanstack/react-query.gen";
import type { ChatMessage, ChatToolCall } from "@/api/types.gen";

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

export function useChat() {
  const [messages, setMessages] = useState<ChatUIMessage[]>([]);
  const mutation = useMutation(postChatMutation());

  const send = useCallback(
    async (content: string) => {
      const trimmed = content.trim();
      if (!trimmed || mutation.isPending) {
        return;
      }

      const userMessage: ChatUIMessage = {
        id: crypto.randomUUID(),
        role: "user",
        content: trimmed,
      };
      const nextMessages = [...messages, userMessage];
      setMessages(nextMessages);

      const response = await mutation.mutateAsync({
        body: { messages: toApiMessages(nextMessages) },
      });

      setMessages((current) => [
        ...current,
        {
          id: crypto.randomUUID(),
          role: "assistant",
          content: response.answer,
          toolCalls: response.tool_calls,
        },
      ]);
    },
    [messages, mutation],
  );

  return {
    messages,
    send,
    isPending: mutation.isPending,
    error: mutation.error,
  };
}
