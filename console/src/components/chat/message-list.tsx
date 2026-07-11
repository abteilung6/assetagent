import type React from "react";
import { useEffect, useRef } from "react";

import { cn } from "@/lib/utils";

import type { ChatUIMessage } from "@/hooks/use-chat";

import { ThinkingIndicator } from "./thinking-indicator";
import { ToolCallIndicator } from "./tool-call-indicator";
import { ToolEvidence } from "./tool-evidence";

type MessageListProps = {
  messages: ChatUIMessage[];
  isPending?: boolean;
  streamingContent?: string | null;
  activeTool?: string | null;
};

export const MessageList: React.FC<MessageListProps> = ({
  messages,
  isPending = false,
  streamingContent = null,
  activeTool = null,
}) => {
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView?.({ behavior: "smooth", block: "end" });
  }, [messages, isPending, streamingContent, activeTool]);

  const showThinking =
    isPending && !activeTool && streamingContent !== null && streamingContent.length === 0;
  const showToolIndicator = isPending && Boolean(activeTool);
  const showStreamingText =
    isPending && !activeTool && streamingContent !== null && streamingContent.length > 0;

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-y-auto overscroll-contain px-4 py-6">
      {messages.length === 0 && !isPending ? (
        <div className="m-auto max-w-lg text-center text-muted-foreground">
          <p className="text-sm">
            Ask about your spending, income, or transactions. Answers are
            grounded in your imported bank data.
          </p>
        </div>
      ) : (
        <div className="mx-auto flex w-full max-w-3xl flex-col gap-4">
          {messages.map((message) => (
            <article
              key={message.id}
              className={cn(
                "max-w-[85%] rounded-2xl px-4 py-3 text-sm",
                message.role === "user"
                  ? "ml-auto bg-primary text-primary-foreground whitespace-pre-wrap"
                  : "bg-muted text-foreground",
              )}
            >
              <div className="whitespace-pre-wrap">{message.content}</div>
              {message.role === "assistant" && message.toolCalls?.length ? (
                <ToolEvidence toolCalls={message.toolCalls} />
              ) : null}
            </article>
          ))}
          {isPending ? (
            <article className="max-w-[85%] rounded-2xl bg-muted px-4 py-3">
              {showToolIndicator && activeTool ? (
                <ToolCallIndicator toolName={activeTool} />
              ) : null}
              {showThinking ? <ThinkingIndicator /> : null}
              {showStreamingText ? (
                <div className="whitespace-pre-wrap text-sm">{streamingContent}</div>
              ) : null}
            </article>
          ) : null}
          <div ref={bottomRef} className="h-px shrink-0" aria-hidden />
        </div>
      )}
    </div>
  );
};
