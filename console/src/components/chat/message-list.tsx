import type React from "react";
import { useEffect, useRef } from "react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

import type { ChatUIMessage } from "@/hooks/use-chat";

import type { ChatStarter, FollowUpChip } from "./starters";
import { ThinkingIndicator } from "./thinking-indicator";
import { ToolCallIndicator } from "./tool-call-indicator";
import { ToolEvidence } from "./tool-evidence";

type MessageListProps = {
  messages: ChatUIMessage[];
  isPending?: boolean;
  streamingContent?: string | null;
  activeTool?: string | null;
  starters?: ChatStarter[];
  onStarter?: (prompt: string) => void;
  followUps?: FollowUpChip[];
  onFollowUp?: (prompt: string) => void;
  startersDisabled?: boolean;
};

export const MessageList: React.FC<MessageListProps> = ({
  messages,
  isPending = false,
  streamingContent = null,
  activeTool = null,
  starters = [],
  onStarter,
  followUps = [],
  onFollowUp,
  startersDisabled = false,
}) => {
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView?.({ behavior: "smooth", block: "end" });
  }, [messages, isPending, streamingContent, activeTool, followUps]);

  const showThinking =
    isPending && !activeTool && streamingContent !== null && streamingContent.length === 0;
  const showToolIndicator = isPending && Boolean(activeTool);
  const showStreamingText =
    isPending && !activeTool && streamingContent !== null && streamingContent.length > 0;
  const showFollowUps =
    !isPending && messages.length > 0 && followUps.length > 0 && Boolean(onFollowUp);

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-y-auto overscroll-contain px-4 py-6">
      {messages.length === 0 && !isPending ? (
        <div className="m-auto flex w-full max-w-lg flex-col items-center gap-6 text-center">
          <div className="space-y-2 text-muted-foreground">
            <p className="text-sm font-medium text-foreground">
              Ask about your money
            </p>
            <p className="text-sm">
              I answer from your imported bank data using trusted cashflow and
              Baseline tools.
            </p>
          </div>
          {starters.length > 0 && onStarter ? (
            <div className="flex w-full flex-col gap-2">
              {starters.map((starter) => (
                <Button
                  key={starter.id}
                  type="button"
                  variant="outline"
                  className="h-auto w-full justify-start whitespace-normal px-3 py-2.5 text-left text-sm"
                  disabled={startersDisabled}
                  onClick={() => onStarter(starter.prompt)}
                >
                  {starter.label}
                </Button>
              ))}
            </div>
          ) : null}
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
          {showFollowUps ? (
            <div className="flex flex-wrap gap-2" aria-label="Follow-up suggestions">
              {followUps.map((chip) => (
                <Button
                  key={chip.id}
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={startersDisabled}
                  onClick={() => onFollowUp?.(chip.prompt)}
                >
                  {chip.label}
                </Button>
              ))}
            </div>
          ) : null}
          <div ref={bottomRef} className="h-px shrink-0" aria-hidden />
        </div>
      )}
    </div>
  );
};
