import type React from "react";
import { useEffect, useMemo, useRef } from "react";
import { useNavigate } from "@tanstack/react-router";

import { Composer } from "@/components/chat/composer";
import { MessageList } from "@/components/chat/message-list";
import {
  defaultStarters,
  followUpsForTool,
  parseChatContextParam,
  promoteStarters,
} from "@/components/chat/starters";
import { useChat } from "@/hooks/use-chat";
import { useCurrentBaseline } from "@/hooks/use-baseline";
import { useClassificationQueue } from "@/hooks/use-classification-queue";
import { useLLMModels } from "@/hooks/use-llm-models";
import { useUncertainRecurring } from "@/hooks/use-recurring-uncertain";
import { useTransferCandidates } from "@/hooks/use-transfer-candidates";
import { chatRoute } from "@/router";

const ChatPage: React.FC = () => {
  const navigate = useNavigate();
  const search = chatRoute.useSearch();
  const {
    options,
    selection,
    setModel,
    canChat,
    isLoading: modelsLoading,
    isError: modelsError,
    showModelSelect,
  } = useLLMModels();
  const {
    messages,
    send,
    stop,
    isPending,
    isStreaming,
    streamingContent,
    activeTool,
    error,
    pageContext,
    setPageContext,
  } = useChat(selection);

  const transfersQuery = useTransferCandidates();
  const categoriesQuery = useClassificationQueue();
  const recurringQuery = useUncertainRecurring();
  const baselineQuery = useCurrentBaseline();

  const needsReviewTotal =
    (transfersQuery.data?.data.length ?? 0) +
    (categoriesQuery.data?.data.length ?? 0) +
    (recurringQuery.data?.data.length ?? 0);

  const starters = useMemo(
    () =>
      promoteStarters(defaultStarters(), {
        needsReviewTotal,
        baselineStatus: baselineQuery.data?.status ?? null,
      }),
    [needsReviewTotal, baselineQuery.data?.status],
  );

  const lastAssistant = [...messages]
    .reverse()
    .find((m) => m.role === "assistant");
  const lastToolName = lastAssistant?.toolCalls?.at(-1)?.name;
  const followUps = useMemo(
    () =>
      messages.length > 0 && !isPending ? followUpsForTool(lastToolName) : [],
    [messages.length, isPending, lastToolName],
  );

  const handoffHandled = useRef(false);
  useEffect(() => {
    if (handoffHandled.current) {
      return;
    }
    const prompt =
      typeof search.prompt === "string" ? search.prompt.trim() : "";
    if (!prompt || !selection || modelsLoading || !canChat || isPending) {
      return;
    }
    handoffHandled.current = true;
    const context = parseChatContextParam(search.context);
    if (context) {
      setPageContext(context);
    }
    void navigate({
      to: "/chat",
      search: {},
      replace: true,
    });
    void send(prompt, context);
  }, [
    search.prompt,
    search.context,
    selection,
    modelsLoading,
    canChat,
    isPending,
    send,
    setPageContext,
    navigate,
  ]);

  const composerDisabled = !canChat || modelsLoading;
  const showModelsError = modelsError || (!modelsLoading && !canChat);
  const showSendError = Boolean(error);

  const onSend = (content: string) => {
    void send(content, pageContext ?? undefined);
  };

  return (
    <div className="-m-4 flex min-h-0 flex-1 flex-col overflow-hidden">
      <MessageList
        messages={messages}
        isPending={isPending}
        streamingContent={streamingContent}
        activeTool={activeTool}
        starters={starters}
        onStarter={onSend}
        followUps={followUps}
        onFollowUp={onSend}
        startersDisabled={composerDisabled || isPending}
      />
      {showModelsError ? (
        <p className="shrink-0 px-4 pb-2 text-sm text-destructive">
          Assistant isn&apos;t configured. Check the API server settings.
        </p>
      ) : null}
      {showSendError ? (
        <p className="shrink-0 px-4 pb-2 text-sm text-destructive">
          The assistant is temporarily unavailable. Try again in a moment.
        </p>
      ) : null}
      {pageContext?.route ? (
        <p className="shrink-0 px-4 pb-1 text-xs text-muted-foreground">
          Viewing context: {pageContext.route}
        </p>
      ) : null}
      <Composer
        onSend={onSend}
        onStop={stop}
        isStreaming={isStreaming}
        disabled={composerDisabled}
        modelOptions={options}
        modelSelection={selection}
        onModelChange={setModel}
        showModelSelect={showModelSelect}
      />
    </div>
  );
};

export default ChatPage;
