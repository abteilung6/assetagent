import type React from "react";

import { Composer } from "@/components/chat/composer";
import { MessageList } from "@/components/chat/message-list";
import { useChat } from "@/hooks/use-chat";
import { useLLMModels } from "@/hooks/use-llm-models";

const ChatPage: React.FC = () => {
  const {
    options,
    selection,
    setModel,
    canChat,
    isLoading: modelsLoading,
    isError: modelsError,
    showModelSelect,
  } = useLLMModels();
  const { messages, send, isPending, error } = useChat(selection);

  const composerDisabled = !canChat || modelsLoading || isPending;
  const showModelsError = modelsError || (!modelsLoading && !canChat);
  const showSendError = Boolean(error);

  return (
    <div className="-m-4 flex min-h-0 flex-1 flex-col overflow-hidden">
      <MessageList messages={messages} isPending={isPending} />
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
      <Composer
        onSend={send}
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
