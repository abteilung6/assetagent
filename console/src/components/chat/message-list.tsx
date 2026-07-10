import type React from "react";

import { cn } from "@/lib/utils";

import type { ChatUIMessage } from "@/hooks/use-chat";

import { ToolEvidence } from "./tool-evidence";

type MessageListProps = {
  messages: ChatUIMessage[];
  isPending?: boolean;
};

export const MessageList: React.FC<MessageListProps> = ({
  messages,
  isPending = false,
}) => {
  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-y-auto px-4 py-6">
      {messages.length === 0 ? (
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
            <article className="max-w-[85%] rounded-2xl bg-muted px-4 py-3 text-sm text-muted-foreground">
              Thinking…
            </article>
          ) : null}
        </div>
      )}
    </div>
  );
};
